// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

// Milestone 5 (rice-scoring test plan): cross-tier parity. Covers requirement
// §22 (single source of truth for the formula) and §18 (list editor and
// detail editor save through the same path and yield identical results).

// riceFixtureFile is the shared fixture set of RICE component tuples,
// committed once and intended to be consumed identically by the Go formula
// (internal/artifact.RiceScore, exercised here) and the TS formula
// (web/src/lib/rice.ts riceScore(), exercised by a frontend-developer-owned
// Vitest spec). See tests/fixtures/rice_fixtures.json.
const riceFixtureFile = "../../tests/fixtures/rice_fixtures.json"

type riceFixtureCase struct {
	Name           string   `json:"name"`
	RiceReach      *float64 `json:"rice_reach"`
	RiceImpact     *float64 `json:"rice_impact"`
	RiceConfidence *float64 `json:"rice_confidence"`
	RiceEffort     *float64 `json:"rice_effort"`
	ExpectNA       bool     `json:"expect_na"`
	ExpectedScore  *float64 `json:"expected_score"`
}

type riceFixtureFileShape struct {
	Cases []riceFixtureCase `json:"cases"`
}

func loadRiceFixtures(t *testing.T) []riceFixtureCase {
	t.Helper()
	raw, err := os.ReadFile(riceFixtureFile)
	if err != nil {
		t.Fatalf("reading %s: %v", riceFixtureFile, err)
	}
	var parsed riceFixtureFileShape
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("parsing %s: %v", riceFixtureFile, err)
	}
	if len(parsed.Cases) == 0 {
		t.Fatalf("%s has no fixture cases", riceFixtureFile)
	}
	return parsed.Cases
}

// TestRiceParity_GoFormulaMatchesFixtures verifies that internal/artifact's
// RiceScore produces exactly the fixture-committed expected result for every
// case in the shared fixture set — the Go half of the single-source-of-truth
// check (requirement §22). The TS half is the responsibility of
// web/src/lib/__tests__/rice.spec.ts consuming the same fixture file.
func TestRiceParity_GoFormulaMatchesFixtures(t *testing.T) {
	for _, c := range loadRiceFixtures(t) {
		t.Run(c.Name, func(t *testing.T) {
			fm := artifact.Frontmatter{
				RiceReach:      c.RiceReach,
				RiceImpact:     c.RiceImpact,
				RiceConfidence: c.RiceConfidence,
				RiceEffort:     c.RiceEffort,
			}
			score, ok := artifact.RiceScore(fm)

			if c.ExpectNA {
				if ok {
					t.Errorf("case %q: expected N/A, got score %v", c.Name, score)
				}
				return
			}
			if !ok {
				t.Fatalf("case %q: expected a score, got N/A", c.Name)
			}
			if c.ExpectedScore == nil {
				t.Fatalf("case %q: fixture is malformed — expect_na=false but expected_score is null", c.Name)
			}
			if score != *c.ExpectedScore {
				t.Errorf("case %q: score mismatch — want %v, got %v", c.Name, *c.ExpectedScore, score)
			}
		})
	}
}

// TestRiceParity_ListAndDetailEditorsAgree verifies requirement §18: saving
// the same RICE components via the (conceptual) list-row editor and via the
// detail-view editor — both of which invoke the identical PATCH .../rice
// endpoint — yields byte-identical files and identical rice_score, regardless
// of entry point.
func TestRiceParity_ListAndDetailEditorsAgree(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rice-parity-list.md",
			content: makeArtifact("Rice Parity List Entry", "idea", "draft", "rice-parity-list", "", "Body."),
		},
		{
			relPath: "lifecycle/ideas/rice-parity-detail.md",
			content: makeArtifact("Rice Parity Detail Entry", "idea", "draft", "rice-parity-detail", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	components := map[string]any{
		"rice_reach":      1000,
		"rice_impact":     2,
		"rice_confidence": 80,
		"rice_effort":     4,
	}

	// "List editor" entry point.
	respList := env.doRequest("PATCH", "/api/p/testproject/artifacts/lifecycle/ideas/rice-parity-list.md/rice", components)
	requireStatus(t, respList, http.StatusOK)
	dataList := readJSON(t, respList)

	// "Detail editor" entry point — same endpoint, different artifact,
	// identical component values, simulating the two UI surfaces described in
	// requirement §18 (both save through the shared write path).
	respDetail := env.doRequest("PATCH", "/api/p/testproject/artifacts/lifecycle/ideas/rice-parity-detail.md/rice", components)
	requireStatus(t, respDetail, http.StatusOK)
	dataDetail := readJSON(t, respDetail)

	listArtifact, _ := dataList["artifact"].(map[string]any)
	detailArtifact, _ := dataDetail["artifact"].(map[string]any)
	if listArtifact["rice_score"] != detailArtifact["rice_score"] {
		t.Errorf("rice_score differs by entry point: list=%v, detail=%v", listArtifact["rice_score"], detailArtifact["rice_score"])
	}

	listRaw, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle/ideas/rice-parity-list.md"))
	if err != nil {
		t.Fatal(err)
	}
	detailRaw, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle/ideas/rice-parity-detail.md"))
	if err != nil {
		t.Fatal(err)
	}

	// The two files differ only in title/lineage (seeded differently); strip
	// those lines and compare the RICE lines are byte-identical across both.
	listRiceLines := extractRiceLines(string(listRaw))
	detailRiceLines := extractRiceLines(string(detailRaw))
	if listRiceLines != detailRiceLines {
		t.Errorf("RICE frontmatter lines differ by entry point:\nlist:\n%s\ndetail:\n%s", listRiceLines, detailRiceLines)
	}
}

// extractRiceLines returns just the rice_* frontmatter lines from raw
// artifact content, joined by newline, for cross-file comparison.
func extractRiceLines(raw string) string {
	var out string
	for _, line := range splitLines(raw) {
		for _, prefix := range []string{"rice_reach:", "rice_impact:", "rice_confidence:", "rice_effort:"} {
			trimmed := line
			for len(trimmed) > 0 && (trimmed[0] == ' ' || trimmed[0] == '\t') {
				trimmed = trimmed[1:]
			}
			if len(trimmed) >= len(prefix) && trimmed[:len(prefix)] == prefix {
				out += trimmed + "\n"
			}
		}
	}
	return out
}
