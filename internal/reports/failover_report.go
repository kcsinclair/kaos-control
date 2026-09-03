// SPDX-License-Identifier: AGPL-3.0-or-later

package reports

import (
	"sort"
	"time"

	"github.com/kaos-control/kaos-control/internal/project"
)

// FailoverReport aggregates a project's operations.yaml failover history
// (agent-switchover-and-failover FR-10.2): per agent, how often it failed
// over and how long it spent on a secondary; per provider, how often its
// failure caused a failover and which agents it affected.
type FailoverReport struct {
	PerAgent    []FailoverAgentSummary    `json:"per_agent"`
	PerProvider []FailoverProviderSummary `json:"per_provider"`
}

// FailoverAgentSummary is one agent's aggregated failover activity.
type FailoverAgentSummary struct {
	Agent         string `json:"agent"`
	FailoverCount int    `json:"failover_count"`
	RestoreCount  int    `json:"restore_count"`
	// TimeOnSecondaryMs sums every interval the agent spent active on a
	// secondary provider, including any still-open interval (not yet
	// restored) up to the report's reference time.
	TimeOnSecondaryMs int64 `json:"time_on_secondary_ms"`
	// MeanTimeToRestoreMs averages the duration of each completed
	// failover-to-restore interval; 0 when none have completed yet.
	MeanTimeToRestoreMs  float64 `json:"mean_time_to_restore_ms"`
	CurrentlyOnSecondary bool    `json:"currently_on_secondary"`
}

// FailoverProviderSummary aggregates, per provider whose failure caused a
// failover (the "from" provider, not the secondary switched to), how many
// times it did so and which agents were affected.
type FailoverProviderSummary struct {
	Provider      string   `json:"provider"`
	FailoverCount int      `json:"failover_count"`
	Agents        []string `json:"agents"`
}

// FailoverUsage aggregates history (typically Operations.HistorySnapshot())
// into a FailoverReport. now is the reference time used to compute the
// still-open duration of any agent currently on a secondary (no matching
// "restore" entry yet).
func FailoverUsage(history []project.FailoverHistoryEntry, now time.Time) FailoverReport {
	sorted := make([]project.FailoverHistoryEntry, len(history))
	copy(sorted, history)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].At < sorted[j].At })

	type agentAcc struct {
		failoverCount      int
		restoreCount       int
		timeOnSecondaryMs  int64
		restoreDurationsMs []int64
		onSecondary        bool
		secondarySinceAt   int64
	}
	agents := map[string]*agentAcc{}
	agentOf := func(name string) *agentAcc {
		a, ok := agents[name]
		if !ok {
			a = &agentAcc{}
			agents[name] = a
		}
		return a
	}

	providerCauses := map[string]int{}
	providerAgents := map[string]map[string]bool{}

	for _, e := range sorted {
		switch e.Action {
		case "failover":
			a := agentOf(e.Agent)
			a.failoverCount++
			a.onSecondary = true
			a.secondarySinceAt = e.At
			if e.FromProvider != "" {
				providerCauses[e.FromProvider]++
				if providerAgents[e.FromProvider] == nil {
					providerAgents[e.FromProvider] = map[string]bool{}
				}
				providerAgents[e.FromProvider][e.Agent] = true
			}
		case "restore":
			a := agentOf(e.Agent)
			a.restoreCount++
			if a.onSecondary {
				durMs := (e.At - a.secondarySinceAt) * 1000
				if durMs < 0 {
					durMs = 0
				}
				a.timeOnSecondaryMs += durMs
				a.restoreDurationsMs = append(a.restoreDurationsMs, durMs)
			}
			a.onSecondary = false
			a.secondarySinceAt = 0
		}
	}

	nowUnix := now.Unix()
	perAgent := make([]FailoverAgentSummary, 0, len(agents))
	for name, a := range agents {
		if a.onSecondary {
			openMs := (nowUnix - a.secondarySinceAt) * 1000
			if openMs > 0 {
				a.timeOnSecondaryMs += openMs
			}
		}
		var meanRestoreMs float64
		if len(a.restoreDurationsMs) > 0 {
			var sum int64
			for _, d := range a.restoreDurationsMs {
				sum += d
			}
			meanRestoreMs = float64(sum) / float64(len(a.restoreDurationsMs))
		}
		perAgent = append(perAgent, FailoverAgentSummary{
			Agent:                name,
			FailoverCount:        a.failoverCount,
			RestoreCount:         a.restoreCount,
			TimeOnSecondaryMs:    a.timeOnSecondaryMs,
			MeanTimeToRestoreMs:  meanRestoreMs,
			CurrentlyOnSecondary: a.onSecondary,
		})
	}
	sort.Slice(perAgent, func(i, j int) bool { return perAgent[i].Agent < perAgent[j].Agent })

	perProvider := make([]FailoverProviderSummary, 0, len(providerCauses))
	for provider, count := range providerCauses {
		agentNames := make([]string, 0, len(providerAgents[provider]))
		for name := range providerAgents[provider] {
			agentNames = append(agentNames, name)
		}
		sort.Strings(agentNames)
		perProvider = append(perProvider, FailoverProviderSummary{
			Provider:      provider,
			FailoverCount: count,
			Agents:        agentNames,
		})
	}
	sort.Slice(perProvider, func(i, j int) bool { return perProvider[i].Provider < perProvider[j].Provider })

	return FailoverReport{PerAgent: perAgent, PerProvider: perProvider}
}
