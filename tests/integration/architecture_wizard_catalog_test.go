// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Test plan: lifecycle/test-plans/onboarding-architecture-selection-5-test.md
// — the wizard/catalog listing endpoint added to unblock FE-plan OQ-6: the
// Browse step needs the full candidate catalog (every architecture + tech-stack
// with title/summary/labels/related_to/pros/cons) before any architecture is
// chosen. Unlike wizard/recommend (needs answers) and wizard/stacks (needs a
// chosen architecture), this takes no inputs and is the single HTTP source of
// pros/cons (parsed from `## Pros`/`## Cons` markdown bodies by LoadCatalog).

import (
	"testing"
)

// archArtifactWithProsCons builds a catalog architecture whose body carries the
// `## Pros`/`## Cons` sections LoadCatalog parses — the fields no other endpoint
// exposes.
func archArtifactWithProsCons() string {
	return "---\n" +
		"title: Modular Monolith\n" +
		"type: architecture\n" +
		"status: draft\n" +
		"lineage: arch-modular-monolith\n" +
		"labels:\n    - architecture\n    - catalog\n    - low-complexity\n" +
		"related_to:\n    - architecture/tech-stacks/go-vue.md\n" +
		"summary: A single deployable app organised into bounded modules.\n" +
		"---\n\n# Modular Monolith\n\nBody.\n\n" +
		"## Pros\n\n- Simple to deploy\n- Easy local dev\n\n" +
		"## Cons\n\n- Scales as one unit\n"
}

func TestWizardCatalog_ReturnsFullCatalogWithProsConsNoInputsRequired(t *testing.T) {
	env := newTestEnv(t, []seedArtifact{
		{relPath: "lifecycle/architecture/architectures/modular-monolith.md", content: archArtifactWithProsCons()},
		{relPath: "lifecycle/architecture/tech-stacks/go-vue.md", content: wizardStackArtifact("Go + Vue", "stack-go-vue",
			[]string{"tech-stack", "catalog", "go", "vue"}, "Go backend + Vue SPA.")},
	})

	// No architecture/answers query params — the whole point of the endpoint.
	resp := env.doRequest("GET", "/api/p/testproject/architecture/wizard/catalog", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	arches, _ := data["architectures"].([]any)
	if len(arches) == 0 {
		t.Fatal("expected the full architecture catalog, got none")
	}
	stacks, _ := data["tech_stacks"].([]any)
	if len(stacks) == 0 {
		t.Fatal("expected the full tech-stack catalog, got none")
	}

	// Find the seeded modular-monolith and confirm its pros/cons came through —
	// the fields no other wizard endpoint exposes (OQ-6).
	var arch map[string]any
	for _, raw := range arches {
		m, _ := raw.(map[string]any)
		if s, _ := m["slug"].(string); s == "modular-monolith" {
			arch = m
			break
		}
	}
	if arch == nil {
		t.Fatal("modular-monolith not present in the returned catalog")
	}
	pros, _ := arch["pros"].([]any)
	cons, _ := arch["cons"].([]any)
	if len(pros) != 2 || len(cons) != 1 {
		t.Errorf("pros/cons not surfaced: pros=%v cons=%v (this is exactly what OQ-6 needed)", arch["pros"], arch["cons"])
	}
}

func TestWizardCatalog_RequiresAuth(t *testing.T) {
	env := newTestEnv(t, wizardCatalogSeeds())
	env.logout()
	resp := env.doRequest("GET", "/api/p/testproject/architecture/wizard/catalog", nil)
	requireStatus(t, resp, 401)
}
