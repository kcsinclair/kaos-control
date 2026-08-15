// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Milestone 3 (rice-scoring test plan): index column & grouped sort. Covers
// requirement §24 (back-fill on scan), §12 (N/A-grouped sort), and §25
// (indexed-value sort performance).

// TestRiceScore_BackfilledOnInitialScan verifies that scored idea/defect
// artifacts present on disk before the server starts get rice_score
// back-filled by the initial full scan, and unscored artifacts get a null
// rice_score (requirement §24).
func TestRiceScore_BackfilledOnInitialScan(t *testing.T) {
	reach, impact, confidence, effort := f64p(1000), f64p(2), f64p(80), f64p(4)
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rice-backfill-scored.md",
			content: makeArtifactWithRice("Backfill Scored", "idea", "draft", "rice-backfill-scored", reach, impact, confidence, effort, "Body."),
		},
		{
			relPath: "lifecycle/defects/rice-backfill-unscored.md",
			content: makeArtifact("Backfill Unscored", "defect", "draft", "rice-backfill-unscored", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	items, _ := data["items"].([]any)

	scored := findItemByPath(items, "lifecycle/ideas/rice-backfill-scored.md")
	if scored == nil {
		t.Fatal("expected scored artifact in list")
	}
	want := (1000.0 * 2.0 * (80.0 / 100.0)) / 4.0
	got, present := scored["rice_score"]
	if !present {
		t.Fatal("expected rice_score back-filled for scored artifact after initial scan")
	}
	if got != want {
		t.Errorf("rice_score: want %v, got %v", want, got)
	}

	unscored := findItemByPath(items, "lifecycle/defects/rice-backfill-unscored.md")
	if unscored == nil {
		t.Fatal("expected unscored artifact in list")
	}
	if _, present := unscored["rice_score"]; present {
		t.Errorf("expected rice_score null/absent for unscored artifact, got %v", unscored["rice_score"])
	}
}

// riceSeed describes one seeded artifact for the sort tests: a slug and its
// four RICE components (nil = unset).
type riceSeed struct {
	slug                              string
	reach, impact, confidence, effort *float64
}

func seedRiceArtifacts(t *testing.T, env *testEnv, seeds []riceSeed) {
	t.Helper()
	for _, s := range seeds {
		relPath := fmt.Sprintf("lifecycle/ideas/%s.md", s.slug)
		content := makeArtifactWithRice(s.slug, "idea", "draft", s.slug, s.reach, s.impact, s.confidence, s.effort, "Body.")
		absPath := filepath.Join(env.projectRoot, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s): %v", filepath.Dir(absPath), err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", absPath, err)
		}
		if err := env.proj.Idx.IndexFile(absPath); err != nil {
			t.Fatalf("IndexFile(%s): %v", absPath, err)
		}
	}
}

// TestRiceSort_AscGroupsNullsLast verifies that ?sort=rice:asc returns scored
// rows in ascending numeric order with all null (N/A) rows grouped after them
// (requirement §12).
func TestRiceSort_AscGroupsNullsLast(t *testing.T) {
	env := newTestEnv(t, nil)

	seedRiceArtifacts(t, env, []riceSeed{
		{"rice-sort-asc-high", f64p(1000), f64p(3), f64p(100), f64p(1)}, // 3000
		{"rice-sort-asc-low", f64p(10), f64p(0.25), f64p(50), f64p(2)},  // 0.625
		{"rice-sort-asc-mid", f64p(100), f64p(1), f64p(50), f64p(1)},    // 50
		{"rice-sort-asc-na1", nil, f64p(1), f64p(50), f64p(1)},          // N/A: missing reach
		{"rice-sort-asc-na2", f64p(100), f64p(1), f64p(50), f64p(0)},    // N/A: effort <= 0
	})

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?sort=rice:asc&limit=0", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	items, _ := data["items"].([]any)

	order := extractRiceOrder(items, []string{
		"lifecycle/ideas/rice-sort-asc-high.md",
		"lifecycle/ideas/rice-sort-asc-low.md",
		"lifecycle/ideas/rice-sort-asc-mid.md",
		"lifecycle/ideas/rice-sort-asc-na1.md",
		"lifecycle/ideas/rice-sort-asc-na2.md",
	})

	wantOrder := []string{
		"lifecycle/ideas/rice-sort-asc-low.md",
		"lifecycle/ideas/rice-sort-asc-mid.md",
		"lifecycle/ideas/rice-sort-asc-high.md",
	}
	assertOrderPrefix(t, order, wantOrder)
	assertNullsGroupedLast(t, order, 3)
}

// TestRiceSort_DescGroupsNullsLast verifies that ?sort=rice:desc returns
// scored rows in descending numeric order with all null (N/A) rows still
// grouped after them, in both directions (requirement §12).
func TestRiceSort_DescGroupsNullsLast(t *testing.T) {
	env := newTestEnv(t, nil)

	seedRiceArtifacts(t, env, []riceSeed{
		{"rice-sort-desc-high", f64p(1000), f64p(3), f64p(100), f64p(1)}, // 3000
		{"rice-sort-desc-low", f64p(10), f64p(0.25), f64p(50), f64p(2)},  // 0.625
		{"rice-sort-desc-mid", f64p(100), f64p(1), f64p(50), f64p(1)},    // 50
		{"rice-sort-desc-na1", nil, nil, nil, nil},                       // N/A: all unset
		{"rice-sort-desc-na2", f64p(100), f64p(1), f64p(150), f64p(1)},   // N/A: confidence out of range
	})

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?sort=rice:desc&limit=0", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	items, _ := data["items"].([]any)

	order := extractRiceOrder(items, []string{
		"lifecycle/ideas/rice-sort-desc-high.md",
		"lifecycle/ideas/rice-sort-desc-low.md",
		"lifecycle/ideas/rice-sort-desc-mid.md",
		"lifecycle/ideas/rice-sort-desc-na1.md",
		"lifecycle/ideas/rice-sort-desc-na2.md",
	})

	wantOrder := []string{
		"lifecycle/ideas/rice-sort-desc-high.md",
		"lifecycle/ideas/rice-sort-desc-mid.md",
		"lifecycle/ideas/rice-sort-desc-low.md",
	}
	assertOrderPrefix(t, order, wantOrder)
	assertNullsGroupedLast(t, order, 3)
}

// extractRiceOrder returns the subset of interestingPaths, in the order they
// appear in the full items list, alongside whether each has a rice_score.
func extractRiceOrder(items []any, interestingPaths []string) []string {
	interesting := make(map[string]bool, len(interestingPaths))
	for _, p := range interestingPaths {
		interesting[p] = true
	}
	var order []string
	for _, it := range items {
		item, _ := it.(map[string]any)
		path, _ := item["path"].(string)
		if interesting[path] {
			order = append(order, path)
		}
	}
	return order
}

// assertOrderPrefix asserts that the first len(want) entries of got equal want, in order.
func assertOrderPrefix(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) < len(want) {
		t.Fatalf("expected at least %d relevant items, got %d: %v", len(want), len(got), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("order[%d]: want %q, got %q (full relevant order: %v)", i, w, got[i], got)
		}
	}
}

// assertNullsGroupedLast asserts that all entries after the first
// numScoredExpected are the (unscored) N/A rows in the tail of got, i.e. no
// N/A row appears interleaved before a scored row's position.
func assertNullsGroupedLast(t *testing.T, order []string, numScoredExpected int) {
	t.Helper()
	if len(order) != numScoredExpected+2 {
		t.Fatalf("expected %d relevant items (scored + 2 N/A), got %d: %v", numScoredExpected+2, len(order), order)
	}
	for i := numScoredExpected; i < len(order); i++ {
		if !containsSubstr(order[i], "na") {
			t.Errorf("expected N/A row at position %d, got %q (full order: %v)", i, order[i], order)
		}
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestRiceSort_PerformanceOnLargeList verifies that sorting a large list by
// RICE score responds within a reasonable time budget, consistent with an
// indexed-column sort rather than a per-request reparse of artifact bodies
// (requirement §25).
func TestRiceSort_PerformanceOnLargeList(t *testing.T) {
	const numArtifacts = 500
	env := newTestEnv(t, nil)

	seeds := make([]riceSeed, numArtifacts)
	for i := 0; i < numArtifacts; i++ {
		slug := fmt.Sprintf("rice-perf-%04d", i)
		if i%3 == 0 {
			// Unscored (N/A).
			seeds[i] = riceSeed{slug: slug}
			continue
		}
		reach := f64p(float64(10 + i))
		impact := f64p(1.0)
		confidence := f64p(50.0)
		effort := f64p(1.0)
		seeds[i] = riceSeed{slug: slug, reach: reach, impact: impact, confidence: confidence, effort: effort}
	}
	seedRiceArtifacts(t, env, seeds)

	start := time.Now()
	resp := env.doRequest("GET", "/api/p/testproject/artifacts?sort=rice:desc&limit=0", nil)
	elapsed := time.Since(start)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	if elapsed > 2*time.Second {
		t.Errorf("rice:desc sort over %d artifacts took %v, exceeds 2s budget", numArtifacts, elapsed)
	}
	t.Logf("rice:desc sort over %d artifacts responded in %v", numArtifacts, elapsed)

	items, _ := data["items"].([]any)
	if len(items) != numArtifacts {
		t.Fatalf("expected %d items, got %d", numArtifacts, len(items))
	}

	// Verify sort correctness: scored rows descending, then all N/A rows.
	var lastScore float64 = 1e18
	seenNA := false
	for i, it := range items {
		item, _ := it.(map[string]any)
		score, present := item["rice_score"]
		if !present {
			seenNA = true
			continue
		}
		if seenNA {
			t.Fatalf("scored item at index %d appears after an N/A item — nulls not grouped last", i)
		}
		s, _ := score.(float64)
		if s > lastScore {
			t.Fatalf("rice:desc order violated at index %d: %v > %v", i, s, lastScore)
		}
		lastScore = s
	}
}
