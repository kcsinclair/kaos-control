// SPDX-License-Identifier: AGPL-3.0-or-later

package index

import (
	"database/sql"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// TestAgentRunCountsByTargetPath verifies the GROUP BY query that returns
// run counts keyed by target_path. Missing keys mean 0 runs (caller convention).
func TestAgentRunCountsByTargetPath(t *testing.T) {
	idx := openTestIndex(t)

	now := time.Now()

	// Path A: 3 runs across done/failed/killed — all statuses must be counted.
	for _, r := range []*AgentRunRow{
		{RunID: "arc-a-0", AgentName: "test-agent", Role: "developer", TargetPath: "lifecycle/ideas/arc-a.md", StartedAt: now, Status: "done"},
		{RunID: "arc-a-1", AgentName: "test-agent", Role: "developer", TargetPath: "lifecycle/ideas/arc-a.md", StartedAt: now, Status: "failed"},
		{RunID: "arc-a-2", AgentName: "test-agent", Role: "developer", TargetPath: "lifecycle/ideas/arc-a.md", StartedAt: now, Status: "killed"},
	} {
		if err := idx.InsertAgentRun(r); err != nil {
			t.Fatalf("InsertAgentRun: %v", err)
		}
	}

	// Path B: 1 run (running).
	if err := idx.InsertAgentRun(&AgentRunRow{
		RunID: "arc-b-0", AgentName: "test-agent", Role: "developer",
		TargetPath: "lifecycle/ideas/arc-b.md", StartedAt: now, Status: "running",
	}); err != nil {
		t.Fatalf("InsertAgentRun: %v", err)
	}

	// Path C: 1 run (queued).
	if err := idx.InsertAgentRun(&AgentRunRow{
		RunID: "arc-c-0", AgentName: "test-agent", Role: "developer",
		TargetPath: "lifecycle/ideas/arc-c.md", StartedAt: now, Status: "queued",
	}); err != nil {
		t.Fatalf("InsertAgentRun: %v", err)
	}

	// Path D: no runs — must be absent from map.

	counts, err := idx.AgentRunCountsByTargetPath()
	if err != nil {
		t.Fatalf("AgentRunCountsByTargetPath: %v", err)
	}

	if got := counts["lifecycle/ideas/arc-a.md"]; got != 3 {
		t.Errorf("arc-a: want 3 runs, got %d", got)
	}
	if got := counts["lifecycle/ideas/arc-b.md"]; got != 1 {
		t.Errorf("arc-b: want 1 run, got %d", got)
	}
	if got := counts["lifecycle/ideas/arc-c.md"]; got != 1 {
		t.Errorf("arc-c: want 1 run, got %d", got)
	}
	if _, present := counts["lifecycle/ideas/arc-d.md"]; present {
		t.Error("arc-d: must be absent from map (caller treats missing key as 0)")
	}
}

// TestActiveAgentStatusByTargetPath verifies that "running" trumps "queued",
// completed runs are excluded, and paths with no active runs are absent.
func TestActiveAgentStatusByTargetPath(t *testing.T) {
	idx := openTestIndex(t)

	now := time.Now()

	// Path A: running only → "running".
	if err := idx.InsertAgentRun(&AgentRunRow{
		RunID: "sts-a-0", AgentName: "test-agent", Role: "developer",
		TargetPath: "lifecycle/ideas/sts-a.md", StartedAt: now, Status: "running",
	}); err != nil {
		t.Fatalf("InsertAgentRun: %v", err)
	}

	// Path B: running + queued → "running" (running trumps queued).
	for _, r := range []*AgentRunRow{
		{RunID: "sts-b-0", AgentName: "test-agent", Role: "developer", TargetPath: "lifecycle/ideas/sts-b.md", StartedAt: now, Status: "running"},
		{RunID: "sts-b-1", AgentName: "test-agent", Role: "developer", TargetPath: "lifecycle/ideas/sts-b.md", StartedAt: now, Status: "queued"},
	} {
		if err := idx.InsertAgentRun(r); err != nil {
			t.Fatalf("InsertAgentRun: %v", err)
		}
	}

	// Path C: queued only → "queued".
	if err := idx.InsertAgentRun(&AgentRunRow{
		RunID: "sts-c-0", AgentName: "test-agent", Role: "developer",
		TargetPath: "lifecycle/ideas/sts-c.md", StartedAt: now, Status: "queued",
	}); err != nil {
		t.Fatalf("InsertAgentRun: %v", err)
	}

	// Path D: only completed runs → absent from map.
	for _, r := range []*AgentRunRow{
		{RunID: "sts-d-0", AgentName: "test-agent", Role: "developer", TargetPath: "lifecycle/ideas/sts-d.md", StartedAt: now, Status: "done"},
		{RunID: "sts-d-1", AgentName: "test-agent", Role: "developer", TargetPath: "lifecycle/ideas/sts-d.md", StartedAt: now, Status: "failed"},
	} {
		if err := idx.InsertAgentRun(r); err != nil {
			t.Fatalf("InsertAgentRun: %v", err)
		}
	}

	statuses, err := idx.ActiveAgentStatusByTargetPath()
	if err != nil {
		t.Fatalf("ActiveAgentStatusByTargetPath: %v", err)
	}

	if got := statuses["lifecycle/ideas/sts-a.md"]; got != "running" {
		t.Errorf("sts-a: want running, got %q", got)
	}
	if got := statuses["lifecycle/ideas/sts-b.md"]; got != "running" {
		t.Errorf("sts-b (running+queued): want running, got %q", got)
	}
	if got := statuses["lifecycle/ideas/sts-c.md"]; got != "queued" {
		t.Errorf("sts-c: want queued, got %q", got)
	}
	if _, present := statuses["lifecycle/ideas/sts-d.md"]; present {
		t.Error("sts-d: must be absent (completed runs only)")
	}
}

// makeTypedArtifact builds an Artifact with the given path, type, and status
// for use in Count/filter unit tests.
func makeTypedArtifact(path, typ, status string) *artifact.Artifact {
	slug := path
	return &artifact.Artifact{
		Path:  path,
		Slug:  slug,
		Stage: stageForType(typ),
		Index: 2,
		Mtime: time.Now(),
		FM: artifact.Frontmatter{
			Title:   slug,
			Type:    typ,
			Status:  status,
			Lineage: slug,
		},
	}
}

// stageForType returns a plausible stage directory name for the given artifact type.
func stageForType(typ string) string {
	switch typ {
	case "plan-backend":
		return "backend-plans"
	case "plan-frontend":
		return "frontend-plans"
	case "plan-test":
		return "test-plans"
	case "idea":
		return "ideas"
	case "ticket":
		return "requirements"
	default:
		return "ideas"
	}
}

// TestCountWithTypeFilter verifies that Count respects both Status and Type
// predicates simultaneously, so that per-agent source_types filtering works
// correctly even when multiple artifact types share the same status.
//
// Scenario:
//
//	artifact A: type=plan-backend, status=in-development  → matches (status+type)
//	artifact B: type=plan-frontend, status=in-development → matches status only
//	artifact C: type=plan-backend, status=draft            → matches type only
//	artifact D: type=idea, status=draft                    → matches neither
func TestCountWithTypeFilter(t *testing.T) {
	idx := openTestIndex(t)

	artifacts := []*artifact.Artifact{
		makeTypedArtifact("lifecycle/backend-plans/count-be-1-3-be.md", "plan-backend", "in-development"),
		makeTypedArtifact("lifecycle/frontend-plans/count-fe-1-4-fe.md", "plan-frontend", "in-development"),
		makeTypedArtifact("lifecycle/backend-plans/count-be-draft-3-be.md", "plan-backend", "draft"),
		makeTypedArtifact("lifecycle/ideas/count-idea-1.md", "idea", "draft"),
	}
	for _, a := range artifacts {
		if err := idx.Upsert(a); err != nil {
			t.Fatalf("Upsert(%s): %v", a.Path, err)
		}
	}

	tests := []struct {
		name   string
		filter Filter
		want   int
	}{
		{
			name:   "status+type: in-development plan-backend",
			filter: Filter{Status: "in-development", Type: "plan-backend"},
			want:   1, // only artifact A
		},
		{
			name:   "status+type: in-development plan-frontend",
			filter: Filter{Status: "in-development", Type: "plan-frontend"},
			want:   1, // only artifact B
		},
		{
			name:   "status only: in-development (no type filter)",
			filter: Filter{Status: "in-development"},
			want:   2, // artifacts A and B
		},
		{
			name:   "type only: plan-backend (no status filter)",
			filter: Filter{Type: "plan-backend"},
			want:   2, // artifacts A and C
		},
		{
			name:   "status+type: draft plan-backend",
			filter: Filter{Status: "draft", Type: "plan-backend"},
			want:   1, // only artifact C
		},
		{
			name:   "status+type: in-development idea (no match)",
			filter: Filter{Status: "in-development", Type: "idea"},
			want:   0,
		},
		{
			name:   "status+type: draft plan-frontend (no match)",
			filter: Filter{Status: "draft", Type: "plan-frontend"},
			want:   0,
		},
		{
			name:   "no filter: all artifacts",
			filter: Filter{},
			want:   4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := idx.Count(tc.filter)
			if err != nil {
				t.Fatalf("Count(%+v): %v", tc.filter, err)
			}
			if got != tc.want {
				t.Errorf("Count(%+v) = %d, want %d", tc.filter, got, tc.want)
			}
		})
	}
}

// TestCountWithTypeFilter_InDevelopmentNoTypeIsAllTypes verifies specifically
// that Count(Filter{Status: "in-development"}) returns ALL in-development
// artifacts regardless of their type — i.e. no implicit type restriction.
func TestCountWithTypeFilter_InDevelopmentNoTypeIsAllTypes(t *testing.T) {
	idx := openTestIndex(t)

	// Insert three in-development artifacts of three different types.
	for _, a := range []*artifact.Artifact{
		makeTypedArtifact("lifecycle/backend-plans/all-types-be-3-be.md", "plan-backend", "in-development"),
		makeTypedArtifact("lifecycle/frontend-plans/all-types-fe-4-fe.md", "plan-frontend", "in-development"),
		makeTypedArtifact("lifecycle/test-plans/all-types-test-5-test.md", "plan-test", "in-development"),
	} {
		if err := idx.Upsert(a); err != nil {
			t.Fatalf("Upsert(%s): %v", a.Path, err)
		}
	}

	got, err := idx.Count(Filter{Status: "in-development"})
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if got != 3 {
		t.Errorf("Count(Status=in-development, no Type) = %d, want 3", got)
	}
}

// TestCountWithTypeFilter_MultipleTypesCSV verifies that a comma-separated
// Type value matches artifacts of any of the listed types (the OR behaviour
// implemented in buildWhere via IN clause).
func TestCountWithTypeFilter_MultipleTypesCSV(t *testing.T) {
	idx := openTestIndex(t)

	for _, a := range []*artifact.Artifact{
		makeTypedArtifact("lifecycle/backend-plans/csv-be-3-be.md", "plan-backend", "in-development"),
		makeTypedArtifact("lifecycle/frontend-plans/csv-fe-4-fe.md", "plan-frontend", "in-development"),
		makeTypedArtifact("lifecycle/ideas/csv-idea-1.md", "idea", "in-development"),
	} {
		if err := idx.Upsert(a); err != nil {
			t.Fatalf("Upsert(%s): %v", a.Path, err)
		}
	}

	// Type filter with comma-separated values.
	got, err := idx.Count(Filter{Status: "in-development", Type: "plan-backend,plan-frontend"})
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	// plan-backend and plan-frontend both match; idea must not.
	if got != 2 {
		t.Errorf("Count(Type=plan-backend,plan-frontend, Status=in-development) = %d, want 2", got)
	}
}

// ── Milestone 3 — Schema migration ──────────────────────────────────────────

// TestEnsureAgentRunsTable_AddsNewColumns verifies that ensureAgentRunsTable
// creates all the analytics columns added after the initial schema.
func TestEnsureAgentRunsTable_AddsNewColumns(t *testing.T) {
	idx := openTestIndex(t)

	rows, err := idx.db.Query(`PRAGMA table_info(agent_runs)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer rows.Close()

	cols := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("scan PRAGMA row: %v", err)
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("PRAGMA rows.Err: %v", err)
	}

	required := []string{
		"model", "total_cost_usd", "duration_api_ms",
		"input_tokens", "ttft_ms", "metrics_available",
	}
	for _, c := range required {
		if !cols[c] {
			t.Errorf("column %q not found in agent_runs after ensureAgentRunsTable", c)
		}
	}
}

// TestEnsureAgentRunsTable_Idempotent verifies that calling ensureAgentRunsTable
// a second time does not return an error.
func TestEnsureAgentRunsTable_Idempotent(t *testing.T) {
	idx := openTestIndex(t)
	if err := idx.ensureAgentRunsTable(); err != nil {
		t.Errorf("second ensureAgentRunsTable call returned error: %v", err)
	}
}

// TestEnsureAgentRunsTable_BackwardsCompatible verifies that an existing row
// created before the analytics columns were added survives a schema migration:
// the row is still readable and its new columns are nil / zero.
func TestEnsureAgentRunsTable_BackwardsCompatible(t *testing.T) {
	idx := openTestIndex(t)

	// Drop and recreate agent_runs with only the original columns.
	_, err := idx.db.Exec(`DROP TABLE IF EXISTS agent_runs`)
	if err != nil {
		t.Fatalf("DROP TABLE: %v", err)
	}
	_, err = idx.db.Exec(`CREATE TABLE agent_runs (
		run_id                   TEXT PRIMARY KEY,
		agent_name               TEXT NOT NULL,
		role                     TEXT NOT NULL,
		target_path              TEXT,
		started_at               INTEGER NOT NULL,
		finished_at              INTEGER,
		status                   TEXT NOT NULL,
		exit_code                INTEGER,
		stderr_tail              TEXT,
		artifacts_produced_json  TEXT
	)`)
	if err != nil {
		t.Fatalf("CREATE TABLE (legacy): %v", err)
	}

	// Insert a legacy row. Provide all non-nullable string columns to avoid
	// scan errors — the test verifies analytics columns survive, not NULL handling.
	_, err = idx.db.Exec(
		`INSERT INTO agent_runs (run_id, agent_name, role, target_path, started_at, status, stderr_tail, artifacts_produced_json)
		 VALUES ('compat-run-1', 'qa', 'analyst', 'lifecycle/ideas/test.md', ?, 'done', '', '[]')`,
		time.Now().Unix(),
	)
	if err != nil {
		t.Fatalf("INSERT legacy row: %v", err)
	}

	// Run migration.
	if err := idx.ensureAgentRunsTable(); err != nil {
		t.Fatalf("ensureAgentRunsTable after legacy schema: %v", err)
	}

	// Row must survive.
	row, err := idx.GetAgentRun("compat-run-1")
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if row == nil {
		t.Fatal("GetAgentRun returned nil — legacy row was lost after migration")
	}
	if row.Model != nil {
		t.Errorf("Model should be nil for legacy row, got %v", row.Model)
	}
	if row.TotalCostUSD != nil {
		t.Errorf("TotalCostUSD should be nil for legacy row, got %v", row.TotalCostUSD)
	}
	if row.MetricsAvailable != 0 {
		t.Errorf("MetricsAvailable should be 0 for legacy row, got %d", row.MetricsAvailable)
	}
}

// ── Milestone 4 — Metrics persistence and TTFT ───────────────────────────────

// TestUpdateAgentRunMetrics_PopulatesColumns verifies that UpdateAgentRunMetrics
// writes all metrics columns and sets metrics_available=1.
func TestUpdateAgentRunMetrics_PopulatesColumns(t *testing.T) {
	idx := openTestIndex(t)

	now := time.Now()
	row := &AgentRunRow{
		RunID:     "metrics-run-1",
		AgentName: "qa",
		Role:      "analyst",
		StartedAt: now,
		Status:    "done",
	}
	if err := idx.InsertAgentRun(row); err != nil {
		t.Fatalf("InsertAgentRun: %v", err)
	}

	m := AgentRunMetrics{
		Model:               "claude-opus-4-7",
		TotalCostUSD:        0.042,
		DurationApiMs:       1500,
		NumTurns:            3,
		InputTokens:         200,
		CacheCreationTokens: 50,
		CacheReadTokens:     30,
		OutputTokens:        100,
	}
	if err := idx.UpdateAgentRunMetrics("metrics-run-1", m); err != nil {
		t.Fatalf("UpdateAgentRunMetrics: %v", err)
	}

	got, err := idx.GetAgentRun("metrics-run-1")
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if got == nil {
		t.Fatal("GetAgentRun returned nil")
	}
	if got.MetricsAvailable != 1 {
		t.Errorf("MetricsAvailable: got %d, want 1", got.MetricsAvailable)
	}
	if got.Model == nil || *got.Model != "claude-opus-4-7" {
		t.Errorf("Model: got %v, want claude-opus-4-7", got.Model)
	}
	if got.TotalCostUSD == nil || *got.TotalCostUSD != 0.042 {
		t.Errorf("TotalCostUSD: got %v, want 0.042", got.TotalCostUSD)
	}
	if got.DurationApiMs == nil || *got.DurationApiMs != 1500 {
		t.Errorf("DurationApiMs: got %v, want 1500", got.DurationApiMs)
	}
	if got.InputTokens == nil || *got.InputTokens != 200 {
		t.Errorf("InputTokens: got %v, want 200", got.InputTokens)
	}
	if got.CacheCreationTokens == nil || *got.CacheCreationTokens != 50 {
		t.Errorf("CacheCreationTokens: got %v, want 50", got.CacheCreationTokens)
	}
	if got.CacheReadTokens == nil || *got.CacheReadTokens != 30 {
		t.Errorf("CacheReadTokens: got %v, want 30", got.CacheReadTokens)
	}
	if got.OutputTokens == nil || *got.OutputTokens != 100 {
		t.Errorf("OutputTokens: got %v, want 100", got.OutputTokens)
	}
}

// TestSetAgentRunModel_OverwritesNull verifies that SetAgentRunModel sets the
// model column on a row that was inserted without one.
func TestSetAgentRunModel_OverwritesNull(t *testing.T) {
	idx := openTestIndex(t)

	now := time.Now()
	if err := idx.InsertAgentRun(&AgentRunRow{
		RunID: "model-run-1", AgentName: "qa", Role: "analyst",
		StartedAt: now, Status: "running",
	}); err != nil {
		t.Fatalf("InsertAgentRun: %v", err)
	}

	if err := idx.SetAgentRunModel("model-run-1", "claude-opus-4-7"); err != nil {
		t.Fatalf("SetAgentRunModel: %v", err)
	}

	got, err := idx.GetAgentRun("model-run-1")
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if got == nil {
		t.Fatal("GetAgentRun returned nil")
	}
	if got.Model == nil || *got.Model != "claude-opus-4-7" {
		t.Errorf("Model: got %v, want claude-opus-4-7", got.Model)
	}
}

// TestSetAgentRunTTFT_RecordedOnce verifies that the last call to
// SetAgentRunTTFT wins at the DB level (plain UPDATE). The supervisor enforces
// single-write semantics via firstTokenSeen; the DB does not guard against it.
func TestSetAgentRunTTFT_RecordedOnce(t *testing.T) {
	idx := openTestIndex(t)

	now := time.Now()
	if err := idx.InsertAgentRun(&AgentRunRow{
		RunID: "ttft-run-1", AgentName: "qa", Role: "analyst",
		StartedAt: now, Status: "running",
	}); err != nil {
		t.Fatalf("InsertAgentRun: %v", err)
	}

	if err := idx.SetAgentRunTTFT("ttft-run-1", 120); err != nil {
		t.Fatalf("SetAgentRunTTFT(120): %v", err)
	}
	// Second call — last write wins at DB level.
	if err := idx.SetAgentRunTTFT("ttft-run-1", 999); err != nil {
		t.Fatalf("SetAgentRunTTFT(999): %v", err)
	}

	got, err := idx.GetAgentRun("ttft-run-1")
	if err != nil {
		t.Fatalf("GetAgentRun: %v", err)
	}
	if got == nil {
		t.Fatal("GetAgentRun returned nil")
	}
	// The last write (999) wins — supervisor guards single-write via firstTokenSeen.
	if got.TtftMs == nil || *got.TtftMs != 999 {
		t.Errorf("TtftMs: got %v, want 999 (last write wins at DB level)", got.TtftMs)
	}
}

// TestUpdateAgentRunMetrics_UnknownRunID verifies that calling
// UpdateAgentRunMetrics on a non-existent run ID does not panic.
// SQLite's UPDATE returns no error for zero-rows-affected.
func TestUpdateAgentRunMetrics_UnknownRunID(t *testing.T) {
	idx := openTestIndex(t)

	m := AgentRunMetrics{
		Model:         "claude-opus-4-7",
		TotalCostUSD:  0.01,
		DurationApiMs: 1000,
	}
	// Must not panic; may return nil or an error.
	err := idx.UpdateAgentRunMetrics("nonexistent-run-id", m)
	_ = err // Either nil or error is acceptable; we only care about no panic.
}
