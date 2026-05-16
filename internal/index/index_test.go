// SPDX-License-Identifier: AGPL-3.0-or-later

package index

import (
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
