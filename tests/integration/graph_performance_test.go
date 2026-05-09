// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

const perfNumLabels = 50

// buildPerfLabels returns a slice of 50 distinct label names for use in
// performance tests.
func buildPerfLabels() []string {
	labels := make([]string, perfNumLabels)
	for i := range labels {
		labels[i] = fmt.Sprintf("perf-lbl-%02d", i)
	}
	return labels
}

// perfStages is the ordered list of stage directories and their associated
// artifact types used by both performance tests.
var perfStages = []struct {
	dir string
	typ string
}{
	{"lifecycle/ideas", "idea"},
	{"lifecycle/requirements", "ticket"},
	{"lifecycle/backend-plans", "plan-backend"},
	{"lifecycle/frontend-plans", "plan-frontend"},
	{"lifecycle/test-plans", "plan-test"},
}

// TestGraphPerformance500Artifacts generates 500 artifacts with labels drawn
// from a pool of 50 distinct values and verifies that GET /graph responds in
// under 2 seconds and returns all 500 nodes with non-null labels.
func TestGraphPerformance500Artifacts(t *testing.T) {
	const numArtifacts = 500
	allLabels := buildPerfLabels()

	seeds := make([]seedArtifact, 0, numArtifacts)
	for i := 0; i < numArtifacts; i++ {
		stage := perfStages[i%len(perfStages)]
		slug := fmt.Sprintf("perf-a-%04d", i)

		// Assign 0–3 labels deterministically. Using stride 13 (coprime to 50)
		// ensures each artifact's labels are distinct across the pool.
		count := i % 4
		lbls := make([]string, count)
		for j := 0; j < count; j++ {
			lbls[j] = allLabels[(i+j*13)%perfNumLabels]
		}

		seeds = append(seeds, seedArtifact{
			relPath: fmt.Sprintf("%s/%s.md", stage.dir, slug),
			content: makeArtifact(
				fmt.Sprintf("Perf Artifact %d", i),
				stage.typ, "draft",
				slug, "", "Performance test body.",
				lbls...,
			),
		})
	}

	env := newTestEnv(t, seeds)

	start := time.Now()
	resp, err := http.Get(env.baseURL + "/api/p/testproject/graph")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	if elapsed > 2*time.Second {
		t.Errorf("graph API response time %v exceeds 2s limit for %d artifacts", elapsed, numArtifacts)
	}
	t.Logf("graph API for %d artifacts responded in %v", numArtifacts, elapsed)

	nodes := decodeGraphNodes(t, data)
	if len(nodes) != numArtifacts {
		t.Errorf("expected %d nodes in graph response, got %d", numArtifacts, len(nodes))
	}

	// Every node must have a non-null labels array.
	for _, n := range nodes {
		node, _ := n.(map[string]any)
		id, _ := node["id"].(string)
		raw, exists := node["labels"]
		if !exists || raw == nil {
			t.Errorf("node %q has missing or null labels field", id)
		}
	}
}

// TestGraphLabelDensity generates 200 artifacts where each has 3–5 labels
// from a pool of 50 distinct values. Verifies correct per-node label counts
// and that GET /graph responds in under 2 seconds.
func TestGraphLabelDensity(t *testing.T) {
	const numArtifacts = 200
	allLabels := buildPerfLabels()

	seeds := make([]seedArtifact, 0, numArtifacts)
	// wantLabelCounts records how many labels each artifact path should have.
	wantLabelCounts := make(map[string]int, numArtifacts)

	for i := 0; i < numArtifacts; i++ {
		stage := perfStages[i%len(perfStages)]
		slug := fmt.Sprintf("perf-d-%04d", i)
		path := fmt.Sprintf("%s/%s.md", stage.dir, slug)

		// 3–5 labels per artifact, all drawn from the 50-label pool.
		// Stride of 13 (coprime to 50) guarantees distinct labels per artifact.
		count := 3 + (i % 3) // 3, 4, or 5
		lbls := make([]string, count)
		for j := 0; j < count; j++ {
			lbls[j] = allLabels[(i+j*13)%perfNumLabels]
		}

		seeds = append(seeds, seedArtifact{
			relPath: path,
			content: makeArtifact(
				fmt.Sprintf("Density Artifact %d", i),
				stage.typ, "draft",
				slug, "", "Label density test body.",
				lbls...,
			),
		})
		wantLabelCounts[path] = count
	}

	env := newTestEnv(t, seeds)

	start := time.Now()
	resp, err := http.Get(env.baseURL + "/api/p/testproject/graph")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	if elapsed > 2*time.Second {
		t.Errorf("graph API response time %v exceeds 2s limit for %d label-dense artifacts", elapsed, numArtifacts)
	}
	t.Logf("graph API for %d artifacts (3–5 labels each) responded in %v", numArtifacts, elapsed)

	nodes := decodeGraphNodes(t, data)
	if len(nodes) != numArtifacts {
		t.Errorf("expected %d nodes in graph response, got %d", numArtifacts, len(nodes))
	}

	for _, n := range nodes {
		node, _ := n.(map[string]any)
		id, _ := node["id"].(string)

		want, tracked := wantLabelCounts[id]
		if !tracked {
			continue // unexpected node — skip rather than fail
		}

		labels := graphNodeLabels(node)
		if labels == nil {
			t.Errorf("node %q has nil labels", id)
			continue
		}
		if len(labels) != want {
			t.Errorf("node %q label count: want %d, got %d", id, want, len(labels))
		}
	}
}
