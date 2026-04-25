//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestPriorityFullRoundTrip creates an artifact with priority: low, PATCHes it
// to high, then verifies the updated priority is visible in both the graph API
// and on disk, with all other frontmatter fields unchanged.
func TestPriorityFullRoundTrip(t *testing.T) {
	const priority0 = "low"
	const priority1 = "high"

	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rtrip-full.md",
			content: makeArtifact(
				"Round Trip Full", "idea", "draft", "rtrip-full", "", "Round-trip body.",
				"tag-a", "tag-b",
			),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/rtrip-full.md"

	// Seed the initial priority via a PATCH to priority0 so the file is in a
	// known state before the main transition we are testing.
	setupResp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": priority0,
	})
	requireStatus(t, setupResp, 200)
	setupResp.Body.Close()

	// PATCH from priority0 → priority1.
	patchResp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": priority1,
	})
	requireStatus(t, patchResp, 200)
	patchResp.Body.Close()

	// 1. Verify via graph API.
	graphData := graphResponseForProject(t, env)
	nodes := decodeGraphNodes(t, graphData)
	node := findNodeByID(nodes, path)
	if node == nil {
		t.Fatal("node not found in graph after PATCH")
	}
	if got, _ := node["priority"].(string); got != priority1 {
		t.Errorf("graph node priority: want %q, got %q", priority1, got)
	}

	// 2. Verify on disk — the file must contain `priority: high`.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "priority: "+priority1) {
		t.Errorf("disk file does not contain 'priority: %s':\n%s", priority1, string(raw))
	}

	// 3. Verify other frontmatter fields are unchanged via the API.
	fm := artifactFrontmatterJSON(t, env, path)
	checks := map[string]string{
		"title":   "Round Trip Full",
		"type":    "idea",
		"status":  "draft",
		"lineage": "rtrip-full",
	}
	for field, want := range checks {
		if got, _ := fm[field].(string); got != want {
			t.Errorf("frontmatter %s after PATCH: want %q, got %q", field, want, got)
		}
	}

	// Labels must still be present.
	labels, _ := fm["labels"].([]any)
	if len(labels) != 2 {
		t.Errorf("expected 2 labels after PATCH, got %d", len(labels))
	}
}

// TestPriorityMultipleRapidUpdates PATCHes the priority five times in quick
// succession and verifies the final state is correct in both the API and on disk.
func TestPriorityMultipleRapidUpdates(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rtrip-rapid.md",
			content: makeArtifact("Round Trip Rapid", "idea", "draft", "rtrip-rapid", "", "Rapid update body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/rtrip-rapid.md"

	// Five sequential PATCHes; the final desired value is "low".
	updates := []string{"low", "normal", "medium", "high", "low"}
	for _, prio := range updates {
		resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
			"priority": prio,
		})
		requireStatus(t, resp, 200)
		resp.Body.Close()
	}

	const wantFinal = "low"

	// Verify via the single-artifact GET endpoint.
	fm := artifactFrontmatterJSON(t, env, path)
	if got, _ := fm["priority"].(string); got != wantFinal {
		t.Errorf("final priority via API: want %q, got %q", wantFinal, got)
	}

	// Verify on disk.
	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "priority: "+wantFinal) {
		t.Errorf("disk file does not contain 'priority: %s':\n%s", wantFinal, string(raw))
	}
}

// TestPriorityPatchConcurrentReads issues concurrent GET requests for an
// artifact while a PATCH is in flight, verifying that no request returns an
// error or corrupt/unparseable response.
func TestPriorityPatchConcurrentReads(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rtrip-concurrent.md",
			content: makeArtifactWithPriority("Round Trip Concurrent", "idea", "draft", "rtrip-concurrent", "normal", "Concurrent read body."),
		},
	}

	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	const path = "lifecycle/ideas/rtrip-concurrent.md"
	const numReaders = 8

	type readResult struct {
		err string
	}
	results := make(chan readResult, numReaders)

	var wg sync.WaitGroup
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp, err := http.Get(
				fmt.Sprintf("%s/api/p/testproject/artifacts/%s", env.baseURL, path),
			)
			if err != nil {
				results <- readResult{fmt.Sprintf("reader %d: HTTP error: %v", idx, err)}
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				b, _ := io.ReadAll(resp.Body)
				results <- readResult{fmt.Sprintf("reader %d: status %d: %s", idx, resp.StatusCode, b)}
				return
			}
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				results <- readResult{fmt.Sprintf("reader %d: read body: %v", idx, err)}
				return
			}
			var data map[string]any
			if err := json.Unmarshal(b, &data); err != nil {
				results <- readResult{fmt.Sprintf("reader %d: corrupt JSON: %v (body: %q)", idx, err, b)}
				return
			}
			if _, ok := data["artifact"]; !ok {
				results <- readResult{fmt.Sprintf("reader %d: response missing 'artifact' key", idx)}
				return
			}
			results <- readResult{}
		}(i)
	}

	// Issue PATCH concurrently with the readers.
	patchResp := env.doRequest("PATCH", "/api/p/testproject/artifacts/"+path+"/priority", map[string]any{
		"priority": "high",
	})
	requireStatus(t, patchResp, 200)
	patchResp.Body.Close()

	wg.Wait()
	close(results)

	for r := range results {
		if r.err != "" {
			t.Errorf("concurrent read error: %s", r.err)
		}
	}

	// Final state should be "high" in both API and on disk.
	fm := artifactFrontmatterJSON(t, env, path)
	if got, _ := fm["priority"].(string); got != "high" {
		t.Errorf("final priority after concurrent test: want %q, got %q", "high", got)
	}

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, path))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "priority: high") {
		t.Errorf("disk file does not contain 'priority: high' after concurrent PATCH:\n%s", raw)
	}
}
