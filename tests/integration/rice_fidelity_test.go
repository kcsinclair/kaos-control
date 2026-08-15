// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Milestone 2 (rice-scoring test plan): frontmatter round-trip & backward
// compatibility. Covers requirement §21 (no migration), §4 (unset vs zero),
// §23 (persistence fidelity), and §20 (clearing removes the YAML line).

// makeArtifactWithRice builds a markdown artifact string with an explicit
// subset of RICE component fields. Pass a nil pointer for a component to
// leave it absent from frontmatter entirely (distinct from a present zero).
func makeArtifactWithRice(title, typ, status, lineage string, reach, impact, confidence, effort *float64, body string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("title: " + title + "\n")
	sb.WriteString("type: " + typ + "\n")
	sb.WriteString("status: " + status + "\n")
	sb.WriteString("lineage: " + lineage + "\n")
	if reach != nil {
		fmt.Fprintf(&sb, "rice_reach: %v\n", *reach)
	}
	if impact != nil {
		fmt.Fprintf(&sb, "rice_impact: %v\n", *impact)
	}
	if confidence != nil {
		fmt.Fprintf(&sb, "rice_confidence: %v\n", *confidence)
	}
	if effort != nil {
		fmt.Fprintf(&sb, "rice_effort: %v\n", *effort)
	}
	sb.WriteString("---\n\n")
	sb.WriteString(body + "\n")
	return sb.String()
}

func f64p(v float64) *float64 { return &v }

// findItemByPath locates an item in a GET /artifacts "items" list by its
// "path" field. Returns nil if not found.
func findItemByPath(items []any, path string) map[string]any {
	for _, it := range items {
		item, _ := it.(map[string]any)
		if p, _ := item["path"].(string); p == path {
			return item
		}
	}
	return nil
}

// TestRiceFidelity_NoFieldsUnaffected verifies a pre-existing idea/defect
// with no RICE fields parses, indexes, and exposes rice_score absent/null,
// and the file on disk is untouched by indexing (requirement §21).
func TestRiceFidelity_NoFieldsUnaffected(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rice-no-fields.md",
			content: makeArtifact("No Rice Fields", "idea", "draft", "rice-no-fields", "", "Body."),
		},
		{
			relPath: "lifecycle/defects/rice-no-fields-defect.md",
			content: makeArtifact("No Rice Fields Defect", "defect", "draft", "rice-no-fields-defect", "", "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	before, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle/ideas/rice-no-fields.md"))
	if err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?sort=rice:asc", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	items, _ := data["items"].([]any)

	for _, path := range []string{"lifecycle/ideas/rice-no-fields.md", "lifecycle/defects/rice-no-fields-defect.md"} {
		row := findItemByPath(items, path)
		if row == nil {
			t.Fatalf("expected item %q in list response", path)
		}
		if _, present := row["rice_score"]; present {
			t.Errorf("%s: expected rice_score absent from response, got %v", path, row["rice_score"])
		}
		fm, _ := row["frontmatter"].(map[string]any)
		for _, k := range []string{"rice_reach", "rice_impact", "rice_confidence", "rice_effort"} {
			if _, present := fm[k]; present {
				t.Errorf("%s: expected frontmatter.%s absent, got %v", path, k, fm[k])
			}
		}
	}

	after, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle/ideas/rice-no-fields.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("indexing an artifact with no RICE fields modified it on disk:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// TestRiceFidelity_PresentZeroIsNotUnset verifies (at the API boundary) that a
// present rice_reach: 0 is distinguishable from an unset field: it appears in
// the response frontmatter and the score is computed as 0, not N/A
// (requirement §4, §8 acceptance: "reach = 0 (others valid) -> score 0").
func TestRiceFidelity_PresentZeroIsNotUnset(t *testing.T) {
	reach, impact, confidence, effort := f64p(0), f64p(1), f64p(50), f64p(2)
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rice-zero-reach.md",
			content: makeArtifactWithRice("Zero Reach", "idea", "draft", "rice-zero-reach", reach, impact, confidence, effort, "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/artifacts/lifecycle/ideas/rice-zero-reach.md", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	artifactObj, _ := data["artifact"].(map[string]any)
	fm, _ := artifactObj["frontmatter"].(map[string]any)

	reachVal, present := fm["rice_reach"]
	if !present {
		t.Fatal("expected rice_reach present in frontmatter (a present 0 must not be dropped)")
	}
	if reachVal != float64(0) {
		t.Errorf("rice_reach: want 0, got %v", reachVal)
	}

	riceScore, present := artifactObj["rice_score"]
	if !present {
		t.Fatal("expected rice_score present when all four components are valid, even though reach=0")
	}
	if riceScore != float64(0) {
		t.Errorf("rice_score: want 0 (not N/A) when reach=0, got %v", riceScore)
	}
}

// TestRicePatchByteForByte verifies that after PATCH .../rice the on-disk
// file is identical to the original except for the RICE lines: all other
// frontmatter fields, their ordering, and the body are unchanged
// (requirement §23).
func TestRicePatchByteForByte(t *testing.T) {
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rice-byte-preserve.md",
			content: makeArtifact("Byte Preserve", "idea", "draft", "rice-byte-preserve", "", "Body text.\n\nSecond paragraph.", "alpha", "beta"),
		},
	}
	env := newTestEnv(t, seeds)

	absPath := filepath.Join(env.projectRoot, "lifecycle/ideas/rice-byte-preserve.md")
	before, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/lifecycle/ideas/rice-byte-preserve.md/rice", map[string]any{
		"rice_reach":      100,
		"rice_impact":     0.5,
		"rice_confidence": 80,
		"rice_effort":     2,
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	after, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatal(err)
	}

	beforeLines := strings.Split(string(before), "\n")
	afterLines := strings.Split(string(after), "\n")

	riceLinePrefix := []string{"rice_reach:", "rice_impact:", "rice_confidence:", "rice_effort:"}
	isRiceLine := func(l string) bool {
		for _, p := range riceLinePrefix {
			if strings.HasPrefix(strings.TrimSpace(l), p) {
				return true
			}
		}
		return false
	}

	// Filter out RICE lines from "after" and compare what remains, in order,
	// against "before" (which has none).
	var afterNonRice []string
	for _, l := range afterLines {
		if !isRiceLine(l) {
			afterNonRice = append(afterNonRice, l)
		}
	}
	if strings.Join(beforeLines, "\n") != strings.Join(afterNonRice, "\n") {
		t.Errorf("non-RICE content changed by PATCH .../rice:\nbefore:\n%s\nafter (RICE lines stripped):\n%s",
			strings.Join(beforeLines, "\n"), strings.Join(afterNonRice, "\n"))
	}

	// The four RICE lines must actually have been added.
	var addedRiceLines int
	for _, l := range afterLines {
		if isRiceLine(l) {
			addedRiceLines++
		}
	}
	if addedRiceLines != 4 {
		t.Errorf("expected 4 RICE lines added, found %d", addedRiceLines)
	}
}

// TestRicePatchClearRemovesLine verifies that clearing a component (sending
// JSON null) removes the YAML line entirely rather than leaving a 0 or ""
// residue (requirement §20).
func TestRicePatchClearRemovesLine(t *testing.T) {
	reach, impact, confidence, effort := f64p(100), f64p(0.5), f64p(80), f64p(2)
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/rice-clear.md",
			content: makeArtifactWithRice("Clear Field", "idea", "draft", "rice-clear", reach, impact, confidence, effort, "Body."),
		},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("PATCH", "/api/p/testproject/artifacts/lifecycle/ideas/rice-clear.md/rice", map[string]any{
		"rice_reach": nil,
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	raw, err := os.ReadFile(filepath.Join(env.projectRoot, "lifecycle/ideas/rice-clear.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if strings.Contains(content, "rice_reach:") {
		t.Errorf("expected rice_reach line removed entirely, found it in:\n%s", content)
	}
	for _, residue := range []string{"rice_reach: 0", "rice_reach: \"\"", "rice_reach: null"} {
		if strings.Contains(content, residue) {
			t.Errorf("expected no residue %q, found it in:\n%s", residue, content)
		}
	}

	// The remaining three components are still present.
	for _, k := range []string{"rice_impact:", "rice_confidence:", "rice_effort:"} {
		if !strings.Contains(content, k) {
			t.Errorf("expected %s to remain present after clearing only rice_reach, file:\n%s", k, content)
		}
	}

	// With only three of four components, the API must report N/A (absent rice_score).
	resp2 := env.doRequest("GET", "/api/p/testproject/artifacts/lifecycle/ideas/rice-clear.md", nil)
	requireStatus(t, resp2, 200)
	data := readJSON(t, resp2)
	artifactObj, _ := data["artifact"].(map[string]any)
	if _, present := artifactObj["rice_score"]; present {
		t.Errorf("expected rice_score absent (N/A) after clearing one of four components, got %v", artifactObj["rice_score"])
	}
}
