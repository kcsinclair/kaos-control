// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/architectural-artefacts-5-test.md — Milestone 2
// (FR-1, FR-2, FR-3): the catalog zone (lifecycle/architecture/{architectures,
// tech-stacks}/) and the project-own zone (root, architecture-summary.md,
// decisions/, standards/) coexist, index simultaneously, and catalog source
// bytes are never mutated by project-own activity (promotion in particular).

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	zoneCatalogArchPath  = "lifecycle/architecture/architectures/zone-arch.md"
	zoneCatalogStackPath = "lifecycle/architecture/tech-stacks/zone-stack.md"
	zoneSummaryPath      = "lifecycle/architecture/architecture-summary.md"
	zoneADRPath          = "lifecycle/architecture/decisions/adr-0001-zone.md"
	zoneStandardPath     = "lifecycle/architecture/standards/zone-standard.md"
	zoneReadmePath       = "lifecycle/architecture/README.md"
)

func zoneSeeds() []seedArtifact {
	return []seedArtifact{
		{relPath: zoneReadmePath, content: "# Architecture catalog\n\nCurated candidates.\n"},
		{relPath: zoneCatalogArchPath, content: makeCleanSlugArtifact("Zone Architecture", "architecture", "draft", "Body.")},
		{relPath: zoneCatalogStackPath, content: makeCleanSlugArtifact("Zone Tech Stack", "tech-stack", "draft", "Body.")},
		{relPath: zoneSummaryPath, content: makeCleanSlugArtifact("Architecture Summary", "doc", "draft", "Summary body.")},
		{relPath: zoneADRPath, content: makeCleanSlugArtifact("Zone ADR", "adr", "draft", "Decision body.")},
		{relPath: zoneStandardPath, content: makeCleanSlugArtifact("Zone Standard", "doc", "draft", "Standard body.")},
	}
}

// TestArchitectureZones_CoexistAndIndex asserts that catalog artefacts and
// project-own reference artefacts (summary, decisions, standards) all index
// simultaneously, while the catalog README.md is excluded per the existing
// ignore-readme rule.
func TestArchitectureZones_CoexistAndIndex(t *testing.T) {
	env := newTestEnv(t, zoneSeeds())

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?limit=0", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	present := []string{zoneCatalogArchPath, zoneCatalogStackPath, zoneSummaryPath, zoneADRPath, zoneStandardPath}
	for _, path := range present {
		if findArtifactRow(t, data, path) == nil {
			t.Errorf("expected %q to be indexed, not found in /artifacts", path)
		}
	}

	if row := findArtifactRow(t, data, zoneReadmePath); row != nil {
		t.Errorf("expected catalog README.md to be excluded from indexing (ignore rule), but found it: %v", row)
	}
}

// TestArchitectureZones_PromotionNeverMutatesCatalogSource promotes a catalog
// architecture + tech-stack pair and asserts the catalog source files remain
// byte-identical afterwards (FR-3, FR-6) — project-own activity must never
// write back into the catalog zone.
func TestArchitectureZones_PromotionNeverMutatesCatalogSource(t *testing.T) {
	env := newTestEnv(t, zoneSeeds())

	archAbs := filepath.Join(env.projectRoot, filepath.FromSlash(zoneCatalogArchPath))
	stackAbs := filepath.Join(env.projectRoot, filepath.FromSlash(zoneCatalogStackPath))
	archBefore, err := os.ReadFile(archAbs)
	if err != nil {
		t.Fatal(err)
	}
	stackBefore, err := os.ReadFile(stackAbs)
	if err != nil {
		t.Fatal(err)
	}

	resp := env.doRequest("POST", "/api/p/testproject/architecture/promote", map[string]string{
		"architecture_path": "architectures/zone-arch.md",
		"tech_stack_path":   "tech-stacks/zone-stack.md",
	})
	requireStatus(t, resp, 200)
	resp.Body.Close()

	archAfter, err := os.ReadFile(archAbs)
	if err != nil {
		t.Fatal(err)
	}
	stackAfter, err := os.ReadFile(stackAbs)
	if err != nil {
		t.Fatal(err)
	}

	if string(archBefore) != string(archAfter) {
		t.Errorf("catalog architecture source was modified by promotion:\nbefore:\n%s\nafter:\n%s", archBefore, archAfter)
	}
	if string(stackBefore) != string(stackAfter) {
		t.Errorf("catalog tech-stack source was modified by promotion:\nbefore:\n%s\nafter:\n%s", stackBefore, stackAfter)
	}
}
