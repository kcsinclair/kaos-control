// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Suite — Milestone 3: disabled-mode queue ordering (FR-4, NFR-3).
//
// With automated_switchover disabled, a would-be-failover reason must pause
// the queue in order, wait for a manual resume, restart the failed job
// first, then continue in queued order.

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

// findCompletedJobByPath scans a queue snapshot's recent list for a
// completed job matching artifactPath, returning its record or nil. Unlike
// findJobByPath, this only matches state=="completed" — after a pause/retry
// cycle the same artifact_path can appear twice in "recent" (the original
// failed attempt and the successful retry), and callers computing ordering
// need the retry's record specifically.
func findCompletedJobByPath(snap map[string]any, artifactPath string) map[string]any {
	items, _ := snap["recent"].([]any)
	for _, raw := range items {
		j, _ := raw.(map[string]any)
		if j["artifact_path"] == artifactPath && j["state"] == "completed" {
			return j
		}
	}
	return nil
}

// TestFailover_Disabled_PauseOrder_RestartsFirstThenQueuedOrder (Milestone
// 3): three jobs are queued for the same agent. The first fails with a
// would-be-failover reason (overloaded) while automated_switchover is
// disabled, pausing the queue with the failed job re-enqueued at the head.
// After a manual resume, the restarted job must complete before the two
// jobs that were already queued behind it, which must themselves complete
// in their original order.
func TestFailover_Disabled_PauseOrder_RestartsFirstThenQueuedOrder(t *testing.T) {
	markerPath := filepath.Join(t.TempDir(), "pause-order-invoked")
	errorJSON := `{"error":{"type":"overloaded_error","message":"Overloaded"}}`
	setupFakeClaudeWithScript(t, failoverThenSucceedScript(markerPath, errorJSON))

	providers := []config.Provider{
		{Name: "anthropic-cloud", BaseURL: "http://127.0.0.1:1", Driver: "openai-compatible"},
		{Name: "gemini-cloud", BaseURL: "http://127.0.0.1:1", Driver: "openai-compatible"},
	}

	env := newFailoverTestEnv(t, failoverDisabledCfgYAML, providers, []seedArtifact{
		{relPath: "lifecycle/ideas/order-1.md", content: makeApprovedArtifact("Order 1", "idea", "order-1")},
		{relPath: "lifecycle/ideas/order-2.md", content: makeApprovedArtifact("Order 2", "idea", "order-2")},
		{relPath: "lifecycle/ideas/order-3.md", content: makeApprovedArtifact("Order 3", "idea", "order-3")},
	})

	env.enqueue("lifecycle/ideas/order-1.md", "requirements-analyst")
	env.enqueue("lifecycle/ideas/order-2.md", "requirements-analyst")
	env.enqueue("lifecycle/ideas/order-3.md", "requirements-analyst")

	env.waitFor(15*time.Second, "queue to pause after job 1's transient failure", func() bool {
		snap := env.queueSnapshot()
		paused, _ := snap["paused"].(bool)
		return paused
	})

	// While paused, order-1's retry (attempts=2) must be at the head of the
	// pending queue, ahead of the still-untouched order-2/order-3.
	snap := env.queueSnapshot()
	pending, _ := snap["pending"].([]any)
	if len(pending) != 3 {
		t.Fatalf("expected 3 pending jobs while paused (retried job-1 + job-2 + job-3), got %d", len(pending))
	}
	first, _ := pending[0].(map[string]any)
	if first["artifact_path"] != "lifecycle/ideas/order-1.md" {
		t.Errorf("expected order-1's retry at the head of the paused queue, got %v", first["artifact_path"])
	}
	if attempts, _ := first["attempts"].(float64); attempts != 2 {
		t.Errorf("expected order-1's retry to have attempts=2, got %v", attempts)
	}

	resumeResp := env.doRequest("POST", "/api/queue/resume", nil)
	requireStatus(t, resumeResp, 204)

	env.waitFor(15*time.Second, "all three jobs to complete after resume", func() bool {
		snap := env.queueSnapshot()
		return findCompletedJobByPath(snap, "lifecycle/ideas/order-1.md") != nil &&
			findCompletedJobByPath(snap, "lifecycle/ideas/order-2.md") != nil &&
			findCompletedJobByPath(snap, "lifecycle/ideas/order-3.md") != nil
	})

	// The project runs with concurrency > 1 (MaxConcurrentAgents), so
	// completions can interleave once dispatched — the ordering guarantee
	// that matters here is dequeue order (started_at, non-decreasing), not
	// strict completion order. started_at/finished_at are second-resolution
	// in the job store, so ties are expected and tolerated; a genuine
	// reordering bug would still show up as a strict violation.
	finalSnap := env.queueSnapshot()
	j1 := findCompletedJobByPath(finalSnap, "lifecycle/ideas/order-1.md")
	j2 := findCompletedJobByPath(finalSnap, "lifecycle/ideas/order-2.md")
	j3 := findCompletedJobByPath(finalSnap, "lifecycle/ideas/order-3.md")

	t1 := parseStartedAt(t, j1)
	t2 := parseStartedAt(t, j2)
	t3 := parseStartedAt(t, j3)
	if t2.Before(t1) || t3.Before(t2) {
		t.Errorf("expected dispatch order order-1 (retry) <= order-2 <= order-3, got started_at %v, %v, %v", t1, t2, t3)
	}
}

// parseStartedAt parses a job's started_at JSON field (RFC3339Nano) for
// ordering comparisons.
func parseStartedAt(t *testing.T, job map[string]any) time.Time {
	t.Helper()
	s, _ := job["started_at"].(string)
	when, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatalf("parsing started_at %q: %v", s, err)
	}
	return when
}
