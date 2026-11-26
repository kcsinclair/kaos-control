// SPDX-License-Identifier: AGPL-3.0-or-later

package index

import (
	"testing"
	"time"
)

// TestAgentRunsTable_HasDriverProviderColumns verifies that a fresh index
// exposes the driver and provider columns added by the
// agent-logging-provider-driver migration (Milestone 1).
func TestAgentRunsTable_HasDriverProviderColumns(t *testing.T) {
	idx := openTestIndex(t)

	rows, err := idx.db.Query(`PRAGMA table_info(agent_runs)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer rows.Close()

	found := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("scanning table_info row: %v", err)
		}
		found[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterating table_info rows: %v", err)
	}

	if !found["driver"] {
		t.Error("agent_runs missing 'driver' column")
	}
	if !found["provider"] {
		t.Error("agent_runs missing 'provider' column")
	}
}

// TestAgentRunsTable_ReopenIsIdempotent verifies that reopening an existing
// index (simulating a pre-existing DB predating this migration) does not
// error and does not rebuild/lose existing agent_runs rows.
func TestAgentRunsTable_ReopenIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/reopen.db"

	idx, err := Open(dbPath, dir, nil)
	if err != nil {
		t.Fatalf("Open (first): %v", err)
	}
	if err := idx.InsertAgentRun(&AgentRunRow{
		RunID: "reopen-run-1", AgentName: "test-agent", Role: "developer",
		TargetPath: "lifecycle/ideas/reopen.md", StartedAt: time.Now(), Status: "done",
		Driver: "openai-compatible", Provider: "prov-a",
	}); err != nil {
		t.Fatalf("InsertAgentRun: %v", err)
	}
	if err := idx.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Reopen the same DB — this re-runs ensureAgentRunsTable's idempotent
	// ALTER TABLE migrations against a DB that already has the columns.
	idx2, err := Open(dbPath, dir, nil)
	if err != nil {
		t.Fatalf("Open (reopen): %v", err)
	}
	t.Cleanup(func() { idx2.Close() })

	row, err := idx2.GetAgentRun("reopen-run-1")
	if err != nil {
		t.Fatalf("GetAgentRun after reopen: %v", err)
	}
	if row == nil {
		t.Fatal("run row did not survive reopen")
	}
	if row.Driver != "openai-compatible" || row.Provider != "prov-a" {
		t.Errorf("driver/provider after reopen: got %q/%q, want openai-compatible/prov-a", row.Driver, row.Provider)
	}
}

// TestInsertAgentRun_DriverProviderRoundTrip verifies that Driver/Provider
// set at insert are returned unchanged by GetAgentRun, and that an empty
// provider round-trips as "" rather than panicking on a NULL scan.
func TestInsertAgentRun_DriverProviderRoundTrip(t *testing.T) {
	idx := openTestIndex(t)
	now := time.Now()

	if err := idx.InsertAgentRun(&AgentRunRow{
		RunID: "rt-api-driver", AgentName: "test-agent", Role: "developer",
		TargetPath: "lifecycle/ideas/rt-api.md", StartedAt: now, Status: "running",
		Driver: "openai-compatible", Provider: "gemini-cloud",
	}); err != nil {
		t.Fatalf("InsertAgentRun (api driver): %v", err)
	}
	if err := idx.InsertAgentRun(&AgentRunRow{
		RunID: "rt-cli-driver", AgentName: "test-agent", Role: "developer",
		TargetPath: "lifecycle/ideas/rt-cli.md", StartedAt: now, Status: "running",
		Driver: "gemini-cli", Provider: "",
	}); err != nil {
		t.Fatalf("InsertAgentRun (cli driver): %v", err)
	}

	apiRow, err := idx.GetAgentRun("rt-api-driver")
	if err != nil {
		t.Fatalf("GetAgentRun (api driver): %v", err)
	}
	if apiRow == nil {
		t.Fatal("GetAgentRun (api driver) returned nil")
	}
	if apiRow.Driver != "openai-compatible" || apiRow.Provider != "gemini-cloud" {
		t.Errorf("api driver row: got driver=%q provider=%q, want openai-compatible/gemini-cloud", apiRow.Driver, apiRow.Provider)
	}

	cliRow, err := idx.GetAgentRun("rt-cli-driver")
	if err != nil {
		t.Fatalf("GetAgentRun (cli driver): %v", err)
	}
	if cliRow == nil {
		t.Fatal("GetAgentRun (cli driver) returned nil")
	}
	if cliRow.Driver != "gemini-cli" {
		t.Errorf("cli driver row: got driver=%q, want gemini-cli", cliRow.Driver)
	}
	if cliRow.Provider != "" {
		t.Errorf("cli driver row: got provider=%q, want empty string (not NULL panic)", cliRow.Provider)
	}

	// ListAgentRuns must expose the same fields as GetAgentRun.
	listed, err := idx.ListAgentRuns("", 0)
	if err != nil {
		t.Fatalf("ListAgentRuns: %v", err)
	}
	byID := map[string]*AgentRunRow{}
	for _, r := range listed {
		byID[r.RunID] = r
	}
	if r := byID["rt-api-driver"]; r == nil || r.Driver != "openai-compatible" || r.Provider != "gemini-cloud" {
		t.Errorf("ListAgentRuns api driver row: got %+v", r)
	}
	if r := byID["rt-cli-driver"]; r == nil || r.Driver != "gemini-cli" || r.Provider != "" {
		t.Errorf("ListAgentRuns cli driver row: got %+v", r)
	}
}

// TestAgentRun_DriverProviderImmutable verifies that no update path
// (SetAgentRunModel, UpdateAgentRunMetrics, UpdateAgentRun) mutates the
// driver/provider recorded at insert (FR-3), while model — the contrast
// case — is allowed to change.
func TestAgentRun_DriverProviderImmutable(t *testing.T) {
	idx := openTestIndex(t)
	now := time.Now()

	const runID = "immutable-run-1"
	if err := idx.InsertAgentRun(&AgentRunRow{
		RunID: runID, AgentName: "test-agent", Role: "developer",
		TargetPath: "lifecycle/ideas/immutable.md", StartedAt: now, Status: "running",
		Driver: "openai-compatible", Provider: "prov-a",
	}); err != nil {
		t.Fatalf("InsertAgentRun: %v", err)
	}

	if err := idx.SetAgentRunModel(runID, "model-x"); err != nil {
		t.Fatalf("SetAgentRunModel: %v", err)
	}
	if err := idx.UpdateAgentRunMetrics(runID, AgentRunMetrics{
		Model: "model-y", TotalCostUSD: 0.5, DurationApiMs: 100, NumTurns: 1,
		InputTokens: 10, OutputTokens: 20,
	}); err != nil {
		t.Fatalf("UpdateAgentRunMetrics: %v", err)
	}
	finishedAt := time.Now()
	if err := idx.UpdateAgentRun(&AgentRunRow{
		RunID: runID, Status: "done", FinishedAt: &finishedAt,
	}); err != nil {
		t.Fatalf("UpdateAgentRun: %v", err)
	}

	row, err := idx.GetAgentRun(runID)
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if row == nil {
		t.Fatal("GetAgentRun returned nil")
	}
	if row.Driver != "openai-compatible" {
		t.Errorf("driver mutated by update paths: got %q, want openai-compatible", row.Driver)
	}
	if row.Provider != "prov-a" {
		t.Errorf("provider mutated by update paths: got %q, want prov-a", row.Provider)
	}
	// Contrast: model IS allowed to change, proving the test actually
	// exercises the update paths rather than trivially passing.
	if row.Model == nil || *row.Model != "model-y" {
		t.Errorf("model: got %v, want model-y (UpdateAgentRunMetrics should have applied)", row.Model)
	}
	if row.Status != "done" {
		t.Errorf("status: got %q, want done", row.Status)
	}
}
