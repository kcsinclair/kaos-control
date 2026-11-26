// SPDX-License-Identifier: AGPL-3.0-or-later

package reports

import (
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/project"
)

// TestFailoverUsage_CompletedCycle verifies FR-10.2: a failover followed by
// a restore contributes one failover count, one restore count, the elapsed
// time on secondary, and a matching time-to-restore for the agent — and one
// failover count attributed to the provider that caused it.
func TestFailoverUsage_CompletedCycle(t *testing.T) {
	history := []project.FailoverHistoryEntry{
		{At: 1000, Agent: "analyst", Action: "failover", FromProvider: "anthropic-cloud", ToProvider: "gemini-cloud", Reason: "overloaded"},
		{At: 1300, Agent: "analyst", Action: "restore", FromProvider: "gemini-cloud", ToProvider: "anthropic-cloud"},
	}

	report := FailoverUsage(history, time.Unix(2000, 0))

	if len(report.PerAgent) != 1 {
		t.Fatalf("expected 1 agent summary, got %d", len(report.PerAgent))
	}
	a := report.PerAgent[0]
	if a.Agent != "analyst" || a.FailoverCount != 1 || a.RestoreCount != 1 {
		t.Fatalf("unexpected agent summary: %+v", a)
	}
	wantMs := int64(300 * 1000)
	if a.TimeOnSecondaryMs != wantMs {
		t.Errorf("time_on_secondary_ms: got %d, want %d", a.TimeOnSecondaryMs, wantMs)
	}
	if a.MeanTimeToRestoreMs != float64(wantMs) {
		t.Errorf("mean_time_to_restore_ms: got %v, want %v", a.MeanTimeToRestoreMs, wantMs)
	}
	if a.CurrentlyOnSecondary {
		t.Error("expected CurrentlyOnSecondary=false after restore")
	}

	if len(report.PerProvider) != 1 {
		t.Fatalf("expected 1 provider summary, got %d", len(report.PerProvider))
	}
	p := report.PerProvider[0]
	if p.Provider != "anthropic-cloud" || p.FailoverCount != 1 || len(p.Agents) != 1 || p.Agents[0] != "analyst" {
		t.Errorf("unexpected provider summary: %+v", p)
	}
}

// TestFailoverUsage_StillOnSecondary verifies an agent with no matching
// restore yet contributes an open-ended time-on-secondary duration up to
// the reference time, and is flagged as currently on secondary.
func TestFailoverUsage_StillOnSecondary(t *testing.T) {
	history := []project.FailoverHistoryEntry{
		{At: 1000, Agent: "analyst", Action: "failover", FromProvider: "anthropic-cloud", ToProvider: "gemini-cloud", Reason: "rate_limit"},
	}

	report := FailoverUsage(history, time.Unix(1500, 0))

	if len(report.PerAgent) != 1 {
		t.Fatalf("expected 1 agent summary, got %d", len(report.PerAgent))
	}
	a := report.PerAgent[0]
	if !a.CurrentlyOnSecondary {
		t.Error("expected CurrentlyOnSecondary=true with no matching restore")
	}
	wantMs := int64(500 * 1000)
	if a.TimeOnSecondaryMs != wantMs {
		t.Errorf("time_on_secondary_ms: got %d, want %d", a.TimeOnSecondaryMs, wantMs)
	}
	if a.MeanTimeToRestoreMs != 0 {
		t.Errorf("expected no time-to-restore yet, got %v", a.MeanTimeToRestoreMs)
	}
}

// TestFailoverUsage_MultipleAgentsSameCausingProvider verifies the
// per-provider aggregation counts failovers across multiple agents caused
// by the same failing provider (FR-3.1: project-wide failover).
func TestFailoverUsage_MultipleAgentsSameCausingProvider(t *testing.T) {
	history := []project.FailoverHistoryEntry{
		{At: 100, Agent: "analyst", Action: "failover", FromProvider: "anthropic-cloud", ToProvider: "gemini-cloud"},
		{At: 100, Agent: "backend-developer", Action: "failover", FromProvider: "anthropic-cloud", ToProvider: "gemini-cloud"},
	}

	report := FailoverUsage(history, time.Unix(200, 0))

	if len(report.PerProvider) != 1 {
		t.Fatalf("expected 1 provider summary, got %d", len(report.PerProvider))
	}
	p := report.PerProvider[0]
	if p.FailoverCount != 2 || len(p.Agents) != 2 {
		t.Errorf("expected 2 failovers across 2 agents caused by anthropic-cloud, got %+v", p)
	}
}

// TestFailoverUsage_Empty verifies an empty history produces an empty
// (not nil-panicking) report.
func TestFailoverUsage_Empty(t *testing.T) {
	report := FailoverUsage(nil, time.Now())
	if len(report.PerAgent) != 0 || len(report.PerProvider) != 0 {
		t.Errorf("expected empty report, got %+v", report)
	}
}
