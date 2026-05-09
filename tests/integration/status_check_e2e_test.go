// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// End-to-end scenario tests for the lineage status checker — Milestone 5.
//
// These tests exercise the full flow from API setup through to disk state
// verification, including concurrent access and edge-case topologies.

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestStatusCheckE2E_FullFlow creates a stale lineage via the artifact API,
// calls GET /status-check to discover staleness, advances all stale artifacts
// via POST /status-check/advance, and verifies the final disk state has the
// correct statuses in each file's frontmatter.
func TestStatusCheckE2E_FullFlow(t *testing.T) {
	// Seed a lineage with a stale parent.
	// idea (draft) → requirement (planning)
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/e2e-full.md",
			content: makeArtifact("E2E Full Idea", "idea", "draft", "e2e-full", "", "Originating idea."),
		},
		{
			relPath: "lifecycle/requirements/e2e-full-2.md",
			content: makeArtifact("E2E Full Req", "ticket", "planning", "e2e-full",
				"lifecycle/ideas/e2e-full.md", "Requirement."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Step 1: Discover stale artifacts.
	checkResp := env.doRequest("GET", "/api/p/testproject/status-check?lineage=e2e-full", nil)
	requireStatus(t, checkResp, 200)
	checkData := readJSON(t, checkResp)
	entries := decodeStaleEntries(t, checkData)

	if len(entries) == 0 {
		t.Fatal("expected at least one stale artifact; status-check returned empty")
	}

	// Collect advanceable paths.
	var paths []string
	for _, e := range entries {
		if e.CanAdvance {
			paths = append(paths, e.Path)
		}
	}
	if len(paths) == 0 {
		t.Fatal("no stale artifacts are advanceable for admin user")
	}

	// Step 2: Advance all stale artifacts.
	advResp := env.doRequest("POST", "/api/p/testproject/status-check/advance", map[string]any{
		"paths": paths,
	})
	requireStatus(t, advResp, 200)
	advData := readJSON(t, advResp)
	results := decodeAdvanceResults(t, advData)

	for _, r := range results {
		if r.Outcome != "advanced" && r.Outcome != "skipped" {
			t.Errorf("unexpected outcome %q for path %q: %s", r.Outcome, r.Path, r.Reason)
		}
	}

	// Step 3: Verify the idea is now at "planning" on disk.
	const ideaPath = "lifecycle/ideas/e2e-full.md"
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, ideaPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: planning") {
		t.Errorf("idea frontmatter should contain 'status: planning' after full advance flow; got:\n%s", raw)
	}

	// Step 4: Re-run status-check; the lineage should now be clean.
	checkResp2 := env.doRequest("GET", "/api/p/testproject/status-check?lineage=e2e-full", nil)
	requireStatus(t, checkResp2, 200)
	checkData2 := readJSON(t, checkResp2)
	entries2 := decodeStaleEntries(t, checkData2)

	if len(entries2) != 0 {
		t.Errorf("after advancing all stale artifacts, status-check should return empty; got %d entries", len(entries2))
	}
}

// TestStatusCheckE2E_ConcurrentAdvance verifies that when two clients concurrently
// call POST /advance on the same artifact, the transition happens exactly once
// and neither request returns an error. The second request should see that the
// artifact is already at the correct status and return outcome "skipped".
func TestStatusCheckE2E_ConcurrentAdvance(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/e2e-concurrent.md",
			content: makeArtifact("E2E Concurrent Idea", "idea", "draft", "e2e-concurrent", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/e2e-concurrent-2.md",
			content: makeArtifact("E2E Concurrent Req", "ticket", "clarifying", "e2e-concurrent",
				"lifecycle/ideas/e2e-concurrent.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const ideaPath = "lifecycle/ideas/e2e-concurrent.md"

	// Launch two goroutines that each call advance simultaneously.
	type callResult struct {
		results []advanceResult
		err     string
	}

	var wg sync.WaitGroup
	outcomes := make([]callResult, 2)
	ready := make(chan struct{})

	for i := 0; i < 2; i++ {
		idx := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ready // wait for both goroutines to be ready before firing
			resp := env.doRequest("POST", "/api/p/testproject/status-check/advance", map[string]any{
				"paths": []string{ideaPath},
			})
			if resp.StatusCode != 200 {
				raw, _ := readBodyString(resp)
				outcomes[idx] = callResult{err: raw}
				return
			}
			data := readJSON(t, resp)
			outcomes[idx] = callResult{results: decodeAdvanceResults(t, data)}
		}()
	}

	close(ready) // release both goroutines simultaneously
	wg.Wait()

	// Neither call should have produced an error.
	for i, o := range outcomes {
		if o.err != "" {
			t.Errorf("goroutine %d: HTTP error: %s", i, o.err)
		}
	}

	// Count how many "advanced" outcomes occurred — must be exactly 1.
	advancedCount := 0
	for _, o := range outcomes {
		for _, r := range o.results {
			if r.Outcome == "advanced" {
				advancedCount++
			}
		}
	}
	if advancedCount != 1 {
		t.Errorf("expected exactly 1 'advanced' outcome across both concurrent calls; got %d", advancedCount)
	}

	// Verify the artifact is now at clarifying — one transition, no more.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, ideaPath))
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(raw), "status: clarifying") {
		t.Errorf("idea should be at 'clarifying' after concurrent advance; disk content:\n%s", raw)
	}
}

// TestStatusCheckE2E_SingleArtifactLineage verifies that calling status-check
// on a lineage whose root has no children returns an empty stale list.
// A single-artifact lineage cannot be stale because there are no children to
// compare against.
func TestStatusCheckE2E_SingleArtifactLineage(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/e2e-solo.md",
			content: makeArtifact("E2E Solo Idea", "idea", "draft", "e2e-solo", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/status-check?lineage=e2e-solo", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	entries := decodeStaleEntries(t, data)

	if len(entries) != 0 {
		t.Errorf("single-artifact lineage should produce no stale entries; got %d", len(entries))
	}
}

// TestStatusCheckE2E_TerminalParentIgnored verifies that when a lineage's root
// artifact is in a terminal status (rejected), the status checker does not
// include it in the stale results even if its children have advanced.
// Rejected/abandoned/done artifacts are considered final; advancing them would
// be incorrect.
func TestStatusCheckE2E_TerminalParentIgnored(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/e2e-terminal.md",
			content: makeArtifact("E2E Terminal Idea", "idea", "rejected", "e2e-terminal", "", "Body."),
		},
		{
			relPath: "lifecycle/requirements/e2e-terminal-2.md",
			content: makeArtifact("E2E Terminal Req", "ticket", "planning", "e2e-terminal",
				"lifecycle/ideas/e2e-terminal.md", "Body."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/status-check?lineage=e2e-terminal", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	entries := decodeStaleEntries(t, data)

	// The rejected idea should NOT appear in the stale list.
	for _, e := range entries {
		if e.Path == "lifecycle/ideas/e2e-terminal.md" {
			t.Errorf("rejected artifact should not appear in stale list, but got entry: %+v", e)
		}
	}
}

// ── shared helper ─────────────────────────────────────────────────────────────

// readBodyString reads and closes the response body, returning it as a string.
// Used to capture error detail when we don't want to fully decode JSON.
func readBodyString(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}
