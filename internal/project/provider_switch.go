// SPDX-License-Identifier: AGPL-3.0-or-later

package project

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
)

// botGitName and botGitEmail identify the automated commits made by the
// provider-switching engine (manual switch, restore, failover, template
// apply) — a fixed system identity distinct from the acting agent or user,
// per the plan's git-audit requirement.
const (
	botGitName  = "kaos-control bot"
	botGitEmail = "bot@kaos-control.local"
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
// (isFailover=true). On the first failover for an agent still on its normal
// provider, the current provider/model are stashed as primary_provider/
// primary_model so RestoreAgentProvider can revert later; a manual switch of
// an already-failed-over agent leaves the stashed primary untouched.
func (p *Project) SwitchAgentProvider(agentName, newProvider, newModel, reason string, isFailover bool) error {
	p.configWriteMu.Lock()
	defer p.configWriteMu.Unlock()

	cfg := p.Config()
	ag, ok := findAgentConfig(cfg, agentName)
	if !ok {
		return fmt.Errorf("agent %q not found", agentName)
	}

	patch := config.AgentProviderPatch{
		AgentName: agentName,
		Provider:  newProvider,
		Model:     newModel,
	}
	primaryProvider := ag.PrimaryProvider
	primaryModel := ag.PrimaryModel
	if isFailover && ag.PrimaryProvider == "" {
		primaryProvider = ag.Provider
		primaryModel = ag.Model
		patch.PrimaryProvider = &primaryProvider
		patch.PrimaryModel = &primaryModel
	}

	if err := config.PatchAgentProviders(p.Entry.Path, []config.AgentProviderPatch{patch}); err != nil {
		return fmt.Errorf("patching agent provider: %w", err)
	}
	if err := p.ReloadConfig(); err != nil {
		return fmt.Errorf("reloading config after provider switch: %w", err)
	}

	var commitMsg, summary string
	if isFailover {
		commitMsg = fmt.Sprintf("failover(agent): %s %s -> %s (reason: %s)", agentName, primaryProvider, newProvider, reason)
		summary = fmt.Sprintf("Agent %s failed over from %s to %s/%s (reason: %s)", agentName, primaryProvider, newProvider, newModel, reason)
	} else {
		commitMsg = fmt.Sprintf("switch(agent): %s -> %s/%s (reason: %s)", agentName, newProvider, newModel, reason)
		summary = fmt.Sprintf("Agent %s manually switched to %s/%s (reason: %s)", agentName, newProvider, newModel, reason)
	}
	p.commitConfigChange(commitMsg)
	p.insertFeedEvent("provider_switched", summary)
	p.Hub.Broadcast(hub.Event{
		Type: "provider.switched",
		Payload: map[string]any{
			"agent":            agentName,
			"provider":         newProvider,
			"model":            newModel,
			"reason":           reason,
			"is_failover":      isFailover,
			"primary_provider": primaryProvider,
			"primary_model":    primaryModel,
		},
	})
	return nil
}

// RestoreAgentProvider reverts agentName from its current (failover)
// provider back to its stashed primary provider/model, clearing the
// primary_provider/primary_model fields. Returns an error if the agent is
// not currently in a failover state.
func (p *Project) RestoreAgentProvider(agentName string) error {
	p.configWriteMu.Lock()
	defer p.configWriteMu.Unlock()

	cfg := p.Config()
	ag, ok := findAgentConfig(cfg, agentName)
	if !ok {
		return fmt.Errorf("agent %q not found", agentName)
	}
	if ag.PrimaryProvider == "" {
		return fmt.Errorf("agent %q is not currently in a failover state", agentName)
	}

	empty := ""
	patch := config.AgentProviderPatch{
		AgentName:       agentName,
		Provider:        ag.PrimaryProvider,
		Model:           ag.PrimaryModel,
		PrimaryProvider: &empty,
		PrimaryModel:    &empty,
	}
	if err := config.PatchAgentProviders(p.Entry.Path, []config.AgentProviderPatch{patch}); err != nil {
		return fmt.Errorf("patching agent provider: %w", err)
	}
	if err := p.ReloadConfig(); err != nil {
		return fmt.Errorf("reloading config after provider restore: %w", err)
	}

	commitMsg := fmt.Sprintf("restore(agent): %s restored to %s", agentName, ag.PrimaryProvider)
	p.commitConfigChange(commitMsg)
	p.insertFeedEvent("provider_restored", fmt.Sprintf("Agent %s restored to primary provider %s/%s", agentName, ag.PrimaryProvider, ag.PrimaryModel))
	p.Hub.Broadcast(hub.Event{
		Type: "provider.restored",
		Payload: map[string]any{
			"agent":    agentName,
			"provider": ag.PrimaryProvider,
			"model":    ag.PrimaryModel,
		},
	})
	return nil
}

// ApplyProviderTemplate batch-switches every agent mapped in the named
// provider template, applying all agent patches in a single atomic config
// write, then reloads config, commits, and broadcasts config.reloaded. It
// returns the number of agents updated.
func (p *Project) ApplyProviderTemplate(templateName string) (int, error) {
	p.configWriteMu.Lock()
	defer p.configWriteMu.Unlock()

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

	patches := make([]config.AgentProviderPatch, 0, len(tpl.Agents))
	for agentName, binding := range tpl.Agents {
		patches = append(patches, config.AgentProviderPatch{
			AgentName: agentName,
			Provider:  binding.Provider,
			Model:     binding.Model,
		})
	}

	if err := config.PatchAgentProviders(p.Entry.Path, patches); err != nil {
		return 0, fmt.Errorf("applying provider template: %w", err)
	}
	if err := p.ReloadConfig(); err != nil {
		return 0, fmt.Errorf("reloading config after template apply: %w", err)
	}

	p.commitConfigChange(fmt.Sprintf("template(provider): applied %s", templateName))
	p.insertFeedEvent("provider_template_applied", fmt.Sprintf("Provider template %q applied to %d agent(s)", templateName, len(patches)))
	p.Hub.Broadcast(hub.Event{
		Type: "config.reloaded",
		Payload: map[string]any{
			"reason":         "provider_template_applied",
			"template":       templateName,
			"updated_agents": len(patches),
		},
	})
	return len(patches), nil
}

// commitConfigChange commits the current lifecycle/config.yaml under the
// fixed bot identity, when the project is backed by a git repo. Failures are
// logged, not returned — the config mutation itself has already succeeded
// and reloaded, and an uncommitted change is preferable to reporting the
// whole operation as failed.
func (p *Project) commitConfigChange(msg string) {
	if p.Git == nil {
		return
	}
	if _, err := p.Git.AddAndCommit([]string{"lifecycle/config.yaml"}, msg, botGitName, botGitEmail); err != nil {
		slog.Warn("project: committing provider-switch config change", "name", p.Entry.Name, "err", err)
	}
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
