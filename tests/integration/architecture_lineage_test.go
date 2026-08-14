// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/architectural-artefacts-5-test.md — Milestone 5
// (FR-19, FR-20, NFR-2): clean-slug (no -N) files under lifecycle/architecture/
// index without lineage-validation errors, a promoted copy's parent: pointing
// at a catalog entry is accepted, the relaxation is path-scoped (a regression
// guard for lifecycle/requirements/ etc.), and all three index paths (startup
// scan, live watch, API write) surface architecture artefacts.

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestArchitectureLineage_CleanSlugIndexesWithoutError seeds a clean-slug
// architecture file present before boot (startup scan) and asserts it's
// retrievable with no validation/parse error and Index == 0.
func TestArchitectureLineage_CleanSlugIndexesWithoutError(t *testing.T) {
	path := "lifecycle/architecture/postgres-modular-monolith.md"
	seeds := []seedArtifact{
		{relPath: path, content: makeCleanSlugArtifact("Postgres Modular Monolith", "architecture", "draft", "Body.")},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	row, _ := data["artifact"].(map[string]any)
	if row == nil {
		t.Fatal("expected an \"artifact\" object in the response")
	}
	if idx, _ := row["index"].(float64); idx != 0 {
		t.Errorf("index = %v, want 0", row["index"])
	}

	perrResp := env.doRequest("GET", "/api/p/testproject/parse-errors", nil)
	requireStatus(t, perrResp, 200)
	perrs := parseErrorPaths(t, readJSON(t, perrResp))
	if msg, ok := perrs[path]; ok {
		t.Errorf("unexpected parse error for clean-slug architecture file: %s", msg)
	}
}

// TestArchitectureLineage_PromotedCopyParentAcceptedWithEdge seeds a promoted
// copy whose parent: points at a catalog entry and asserts it is accepted
// (no parse error) and that a "parent" graph edge resolves to the catalog
// target.
func TestArchitectureLineage_PromotedCopyParentAcceptedWithEdge(t *testing.T) {
	catalogPath := "lifecycle/architecture/architectures/postgres-modular-monolith.md"
	promotedPath := "lifecycle/architecture/postgres-modular-monolith.md"
	seeds := []seedArtifact{
		{relPath: catalogPath, content: makeCleanSlugArtifact("Postgres Modular Monolith", "architecture", "draft", "Catalog body.")},
		{
			relPath: promotedPath,
			content: "---\ntitle: Postgres Modular Monolith\ntype: architecture\nstatus: draft\nparent: " +
				catalogPath + "\n---\n\nPromoted body.\n",
		},
	}
	env := newTestEnv(t, seeds)

	perrResp := env.doRequest("GET", "/api/p/testproject/parse-errors", nil)
	requireStatus(t, perrResp, 200)
	perrs := parseErrorPaths(t, readJSON(t, perrResp))
	if msg, ok := perrs[promotedPath]; ok {
		t.Errorf("unexpected parse error for promoted copy with catalog parent: %s", msg)
	}

	graphData := graphResponseForProject(t, env)
	edges := decodeGraphEdges(t, graphData)
	assertParentEdge(t, edges, promotedPath, catalogPath)
}

// TestArchitectureLineage_RelaxationIsPathScoped is the regression guard: a
// file under lifecycle/requirements/ with no lineage: must still report the
// missing-lineage validation error — the relaxation must not leak beyond
// lifecycle/architecture/.
func TestArchitectureLineage_RelaxationIsPathScoped(t *testing.T) {
	path := "lifecycle/requirements/no-lineage.md"
	seeds := []seedArtifact{
		{relPath: path, content: makeCleanSlugArtifact("No Lineage", "requirement", "draft", "Body.")},
	}
	env := newTestEnv(t, seeds)

	resp := env.doRequest("GET", "/api/p/testproject/parse-errors", nil)
	requireStatus(t, resp, 200)
	perrs := parseErrorPaths(t, readJSON(t, resp))
	msg, ok := perrs[path]
	if !ok {
		t.Fatalf("expected a missing-lineage parse error for %q outside lifecycle/architecture/, found none", path)
	}
	if want := "missing required field: lineage"; msg != want {
		t.Errorf("parse error message = %q, want %q", msg, want)
	}
}

// TestArchitectureLineage_LiveWatchIndexesNewFile covers the live-watch index
// path (NFR-2): a file written to lifecycle/architecture/ after boot triggers
// re-indexing and becomes retrievable via the API.
func TestArchitectureLineage_LiveWatchIndexesNewFile(t *testing.T) {
	env := newTestEnv(t, nil)

	path := "lifecycle/architecture/live-watch-arch.md"
	absPath := filepath.Join(env.projectRoot, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	content := makeCleanSlugArtifact("Live Watch Architecture", "architecture", "draft", "Body.")
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	var found bool
	for time.Now().Before(deadline) {
		resp := env.doRequest("GET", "/api/p/testproject/artifacts?limit=0", nil)
		if resp.StatusCode == 200 {
			data := readJSON(t, resp)
			if findArtifactRow(t, data, path) != nil {
				found = true
				break
			}
		} else {
			resp.Body.Close()
		}
		time.Sleep(30 * time.Millisecond)
	}
	if !found {
		t.Fatalf("expected %q to be indexed by the live watcher within 2s", path)
	}
}
