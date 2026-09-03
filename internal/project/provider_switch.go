// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"fmt"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
)

// findAgentConfig returns a copy of the named agent's config, or false if unknown.
func findAgentConfig(cfg *config.Project, agentName string) (config.AgentConfig, bool) {
	for i := range cfg.Agents {
		if cfg.Agents[i].Name == agentName {
			return cfg.Agents[i], true
		}
	}
	return config.AgentConfig{}, false
}

// SwitchAgentProvider changes agentName's active provider/model, either as a
// manual operator switch (isFailover=false) or an automated failover
// (isFailover=true). It records the change in operations.yaml only —
// lifecycle/config.yaml (declared intent) is never touched and no git
// commit is made. The agent's primary is always its current config-declared
// {provider, model}, snapshotted into the operations record at switch time;
// this is what makes the operation immune to the old
// "fallback_provider must differ from provider" config-validation failure
// mode, since no failover path writes config anymore.
func (p *Project) SwitchAgentProvider(agentName, newProvider, newModel, reason string, isFailover bool) error {
	p.switchMu.Lock()
	defer p.switchMu.Unlock()

	cfg := p.Config()
	ag, ok := findAgentConfig(cfg, agentName)
	if !ok {
		return fmt.Errorf("agent %q not found", agentName)
	}

	primary := ProviderModel{Provider: ag.Provider, Model: ag.Model}
	now := time.Now()
	state := AgentOperationalState{
		Agent:      agentName,
		Primary:    primary,
		Active:     ProviderModel{Provider: newProvider, Model: newModel},
		SwitchedAt: now.Unix(),
		Reason:     reason,
	}
	if err := p.Operations().SetAgentState(state); err != nil {
		return fmt.Errorf("recording provider switch: %w", err)
	}

	action := "switch"
	var summary string
	if isFailover {
		action = "failover"
		summary = fmt.Sprintf("Agent %s failed over from %s to %s/%s (reason: %s)", agentName, primary.Provider, newProvider, newModel, reason)
	} else {
		summary = fmt.Sprintf("Agent %s manually switched to %s/%s (reason: %s)", agentName, newProvider, newModel, reason)
	}
	_ = p.Operations().AppendHistory(FailoverHistoryEntry{
		At: now.Unix(), Agent: agentName, Action: action,
		FromProvider: primary.Provider, ToProvider: newProvider, Reason: reason,
	})

	p.insertFeedEvent("provider_switched", summary)
	p.Hub.Broadcast(hub.Event{
		Type: "provider.switched",
		Payload: map[string]any{
			"agent":            agentName,
			"provider":         newProvider,
			"model":            newModel,
			"reason":           reason,
			"is_failover":      isFailover,
			"primary_provider": primary.Provider,
			"primary_model":    primary.Model,
		},
	})
	return nil
}

// RestoreAgentProvider reverts agentName from its current (failover)
// provider back to its primary provider/model, clearing its operations.yaml
// override. Returns an error if the agent has no recorded override (i.e. is
// not currently in a failover/switched state).
func (p *Project) RestoreAgentProvider(agentName string) error {
	p.switchMu.Lock()
	defer p.switchMu.Unlock()

	cfg := p.Config()
	ag, ok := findAgentConfig(cfg, agentName)
	if !ok {
		return fmt.Errorf("agent %q not found", agentName)
	}

	state, hadState := p.Operations().AgentState(agentName)
	if !hadState {
		return fmt.Errorf("agent %q is not currently in a failover state", agentName)
	}

	if err := p.Operations().ClearAgentState(agentName); err != nil {
		return fmt.Errorf("restoring provider: %w", err)
	}
	_ = p.Operations().AppendHistory(FailoverHistoryEntry{
		At: time.Now().Unix(), Agent: agentName, Action: "restore",
		FromProvider: state.Active.Provider, ToProvider: ag.Provider,
	})

	p.insertFeedEvent("provider_restored", fmt.Sprintf("Agent %s restored to primary provider %s/%s", agentName, ag.Provider, ag.Model))
	p.Hub.Broadcast(hub.Event{
		Type: "provider.restored",
		Payload: map[string]any{
			"agent":    agentName,
			"provider": ag.Provider,
			"model":    ag.Model,
		},
	})
	return nil
}

// ApplyProviderTemplate batch-switches every agent mapped in the named
// provider template, recording each agent's override in operations.yaml.
// It returns the number of agents updated.
func (p *Project) ApplyProviderTemplate(templateName string) (int, error) {
	p.switchMu.Lock()
	defer p.switchMu.Unlock()

	cfg := p.Config()
	var tpl *config.ProviderTemplate
	for i := range cfg.ProviderTemplates {
		if cfg.ProviderTemplates[i].Name == templateName {
			tpl = &cfg.ProviderTemplates[i]
			break
		}
	}
	if tpl == nil {
		return 0, fmt.Errorf("provider template %q not found", templateName)
	}

	now := time.Now()
	reason := fmt.Sprintf("template:%s", templateName)
	count := 0
	for agentName, binding := range tpl.Agents {
		ag, ok := findAgentConfig(cfg, agentName)
		if !ok {
			continue // template may reference an agent not configured in this project
		}
		state := AgentOperationalState{
			Agent:      agentName,
			Primary:    ProviderModel{Provider: ag.Provider, Model: ag.Model},
			Active:     ProviderModel{Provider: binding.Provider, Model: binding.Model},
			SwitchedAt: now.Unix(),
			Reason:     reason,
		}
		if err := p.Operations().SetAgentState(state); err != nil {
			return count, fmt.Errorf("applying provider template: %w", err)
		}
		_ = p.Operations().AppendHistory(FailoverHistoryEntry{
			At: now.Unix(), Agent: agentName, Action: "switch",
			FromProvider: ag.Provider, ToProvider: binding.Provider, Reason: reason,
		})
		count++
	}

	p.insertFeedEvent("provider_template_applied", fmt.Sprintf("Provider template %q applied to %d agent(s)", templateName, count))
	p.Hub.Broadcast(hub.Event{
		Type: "provider.switched",
		Payload: map[string]any{
			"reason":         "provider_template_applied",
			"template":       templateName,
			"updated_agents": count,
		},
	})
	return count, nil
}

// SwitchAllAgentProviders batch-switches every agent currently configured
// with provider fromProvider to toProvider/toModel, recording each agent's
// override in operations.yaml. Returns the number of agents switched
// (0 when no agent matches fromProvider — not an error).
func (p *Project) SwitchAllAgentProviders(fromProvider, toProvider, toModel, reason string) (int, error) {
	p.switchMu.Lock()
	defer p.switchMu.Unlock()

	cfg := p.Config()
	now := time.Now()
	count := 0
	for _, ag := range cfg.Agents {
		if ag.Provider != fromProvider {
			continue
		}
		state := AgentOperationalState{
			Agent:      ag.Name,
			Primary:    ProviderModel{Provider: ag.Provider, Model: ag.Model},
			Active:     ProviderModel{Provider: toProvider, Model: toModel},
			SwitchedAt: now.Unix(),
			Reason:     reason,
		}
		if err := p.Operations().SetAgentState(state); err != nil {
			return count, fmt.Errorf("switching all agent providers: %w", err)
		}
		_ = p.Operations().AppendHistory(FailoverHistoryEntry{
			At: now.Unix(), Agent: ag.Name, Action: "switch",
			FromProvider: fromProvider, ToProvider: toProvider, Reason: reason,
		})
		count++
	}
	if count == 0 {
		return 0, nil
	}

	p.insertFeedEvent("provider_switched", fmt.Sprintf("Switched %d agent(s) from %s to %s/%s (reason: %s)", count, fromProvider, toProvider, toModel, reason))
	p.Hub.Broadcast(hub.Event{
		Type: "provider.switched",
		Payload: map[string]any{
			"from_provider": fromProvider,
			"to_provider":   toProvider,
			"to_model":      toModel,
			"reason":        reason,
			"count":         count,
		},
	})
	return count, nil
}

// RestoreAllAgentProviders restores every agent currently in a failover
// state (an operations.yaml override recorded) back to its primary
// provider/model. Returns the number of agents restored (0 when no agent is
// in failover — not an error).
func (p *Project) RestoreAllAgentProviders() (int, error) {
	p.switchMu.Lock()
	defer p.switchMu.Unlock()

	cfg := p.Config()
	states := p.Operations().AllAgentStates()
	now := time.Now()
	var restoredNames []string
	for name, state := range states {
		ag, ok := findAgentConfig(cfg, name)
		if !ok {
			continue
		}
		if err := p.Operations().ClearAgentState(name); err != nil {
			return len(restoredNames), fmt.Errorf("restoring all agent providers: %w", err)
		}
		_ = p.Operations().AppendHistory(FailoverHistoryEntry{
			At: now.Unix(), Agent: name, Action: "restore",
			FromProvider: state.Active.Provider, ToProvider: ag.Provider,
		})
		restoredNames = append(restoredNames, name)
	}
	if len(restoredNames) == 0 {
		return 0, nil
	}

	p.insertFeedEvent("provider_restored", fmt.Sprintf("Restored %d agent(s) to primary provider", len(restoredNames)))
	p.Hub.Broadcast(hub.Event{
		Type: "provider.restored",
		Payload: map[string]any{
			"agents": restoredNames,
			"count":  len(restoredNames),
		},
	})
	return len(restoredNames), nil
}

// EffectiveAgentProvider returns agentName's current effective active
// provider/model: an operations.yaml override if recorded, else the
// agent's config-declared provider/model. ok is false when agentName is not
// a configured agent.
func (p *Project) EffectiveAgentProvider(agentName string) (provider, model string, ok bool) {
	ag, found := findAgentConfig(p.Config(), agentName)
	if !found {
		return "", "", false
	}
	if state, hasState := p.Operations().AgentState(agentName); hasState {
		return state.Active.Provider, state.Active.Model, true
	}
	return ag.Provider, ag.Model, true
}

// IsAgentFailedOver reports whether agentName currently has an
// operations.yaml override recording it as switched away from its primary
// (NFR-6's one-level cap check).
func (p *Project) IsAgentFailedOver(agentName string) bool {
	state, ok := p.Operations().AgentState(agentName)
	return ok && state.IsFailedOver()
}

// FailoverProviderWide performs FR-3.1: every agent whose effective active
// provider is fromProvider moves to its own configured fallback
// provider/model in a single action — agents may differ in secondary, so
// each uses its own fallback_provider/fallback_model. An agent with no
// fallback_provider configured cannot fail over; it is recorded as
// partially paused (FR-3.4) instead, so the queue dispatcher can pause its
// jobs while other agents proceed. An agent already in a failover state
// (the one-level cap, NFR-6) is left untouched — callers are expected to
// have already routed a repeated failure for an already-failed-over agent
// to pause_queue before calling this. Returns the agent names actually
// switched and the agent names left partially paused.
func (p *Project) FailoverProviderWide(fromProvider, reason string, resetsAtUnix int64, bucket string) (switched, noSecondary []string, err error) {
	p.switchMu.Lock()
	defer p.switchMu.Unlock()

	cfg := p.Config()
	now := time.Now()

	for _, ag := range cfg.Agents {
		state, hasState := p.Operations().AgentState(ag.Name)
		activeProvider := ag.Provider
		if hasState {
			activeProvider = state.Active.Provider
		}
		if activeProvider != fromProvider {
			continue
		}
		if hasState && state.IsFailedOver() {
			// One-level cap (NFR-6): already on a secondary — a further
			// failure for this agent must pause rather than fail over again.
			continue
		}

		if ag.FallbackProvider == "" {
			// FR-3.4: no secondary configured — partial pause while other
			// agents proceed. The active provider/model are left as the
			// (unreachable) primary; PartialPause records that its jobs
			// should be skipped by the dispatcher until restored.
			pausedState := AgentOperationalState{
				Agent:        ag.Name,
				Primary:      ProviderModel{Provider: ag.Provider, Model: ag.Model},
				Active:       ProviderModel{Provider: ag.Provider, Model: ag.Model},
				PartialPause: true,
				Reason:       reason,
				SwitchedAt:   now.Unix(),
			}
			if err := p.Operations().SetAgentState(pausedState); err != nil {
				return switched, noSecondary, fmt.Errorf("recording partial pause for %q: %w", ag.Name, err)
			}
			_ = p.Operations().AppendHistory(FailoverHistoryEntry{
				At: now.Unix(), Agent: ag.Name, Action: "pause_queue",
				FromProvider: fromProvider, Reason: reason,
			})
			noSecondary = append(noSecondary, ag.Name)
			continue
		}

		primary := ProviderModel{Provider: ag.Provider, Model: ag.Model}
		newState := AgentOperationalState{
			Agent:        ag.Name,
			Primary:      primary,
			Active:       ProviderModel{Provider: ag.FallbackProvider, Model: ag.FallbackModel},
			SwitchedAt:   now.Unix(),
			Reason:       reason,
			ResetsAtUnix: resetsAtUnix,
			Bucket:       bucket,
		}
		if err := p.Operations().SetAgentState(newState); err != nil {
			return switched, noSecondary, fmt.Errorf("failing over %q: %w", ag.Name, err)
		}
		_ = p.Operations().AppendHistory(FailoverHistoryEntry{
			At: now.Unix(), Agent: ag.Name, Action: "failover",
			FromProvider: primary.Provider, ToProvider: ag.FallbackProvider, Reason: reason,
		})

		p.insertFeedEvent("provider_switched", fmt.Sprintf(
			"Agent %s failed over from %s to %s/%s (reason: %s)", ag.Name, primary.Provider, ag.FallbackProvider, ag.FallbackModel, reason))
		p.Hub.Broadcast(hub.Event{
			Type: "provider.switched",
			Payload: map[string]any{
				"agent":            ag.Name,
				"provider":         ag.FallbackProvider,
				"model":            ag.FallbackModel,
				"reason":           reason,
				"is_failover":      true,
				"primary_provider": primary.Provider,
				"primary_model":    primary.Model,
			},
		})
		switched = append(switched, ag.Name)
	}

	if len(switched) > 0 || len(noSecondary) > 0 {
		p.insertFeedEvent("provider_failover_project_wide", fmt.Sprintf(
			"Project-wide failover from %s (reason: %s): %d agent(s) switched, %d agent(s) paused (no secondary)",
			fromProvider, reason, len(switched), len(noSecondary)))
		p.Hub.Broadcast(hub.Event{
			Type: "provider.failover_project_wide",
			Payload: map[string]any{
				"from_provider": fromProvider,
				"reason":        reason,
				"switched":      switched,
				"no_secondary":  noSecondary,
			},
		})
	}

	return switched, noSecondary, nil
}

// PartiallyPausedAgents returns the names of agents currently recorded with
// PartialPause set (FR-3.4: bound to a failed provider with no secondary to
// fail over to) — used by the queue dispatcher to skip their jobs while
// other agents' work proceeds.
func (p *Project) PartiallyPausedAgents() []string {
	var names []string
	for name, state := range p.Operations().AllAgentStates() {
		if state.PartialPause {
			names = append(names, name)
		}
	}
	return names
}

// insertFeedEvent records a system-actor feed event and broadcasts feed.new,
// mirroring the pattern used by the agent run supervisor.
func (p *Project) insertFeedEvent(eventType, summary string) {
	feedEvent := &index.EventRow{
		EventType: eventType,
		Timestamp: time.Now().Unix(),
		Actor:     "system",
		Summary:   summary,
	}
	if err := p.Idx.InsertEvent(feedEvent); err == nil {
		p.Hub.Broadcast(hub.Event{Type: "feed.new", Payload: feedEvent})
	}
}
