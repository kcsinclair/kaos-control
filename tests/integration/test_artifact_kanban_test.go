//go:build integration

package integration

import (
	"testing"
)

// ── Milestone 4 — Kanban board test visibility ────────────────────────────
//
// These tests verify that the artifact listing API does not implicitly exclude
// test artifacts from unfiltered results, that the kanban config endpoint
// structure is unaffected by the feature, and that type-specific filters do
// not bleed across types.

// kanbanTestSeeds returns a mix of test, ticket, and idea artifacts for use
// in Kanban visibility tests.
func kanbanTestSeeds() []seedArtifact {
	return []seedArtifact{
		{
			relPath: "lifecycle/tests/kb-test-1.md",
			content: makeArtifact("KB Test 1", "test", "approved", "kb-test-1", "", "Test body."),
		},
		{
			relPath: "lifecycle/tests/kb-test-2.md",
			content: makeArtifact("KB Test 2", "test", "draft", "kb-test-2", "", "Test body."),
		},
		{
			relPath: "lifecycle/requirements/kb-ticket-2.md",
			content: makeArtifact("KB Ticket", "ticket", "approved", "kb-ticket", "", "Ticket body."),
		},
		{
			relPath: "lifecycle/ideas/kb-idea.md",
			content: makeArtifact("KB Idea", "idea", "draft", "kb-idea", "", "Idea body."),
		},
	}
}

// TestKanbanVisibility_UnfilteredIncludesTests verifies that
// GET /artifacts (no type filter) returns test artifacts alongside other types.
// The backend must not apply any implicit exclusion of type=test artifacts.
// Covers test plan Milestone 4, scenario 1.
func TestKanbanVisibility_UnfilteredIncludesTests(t *testing.T) {
	env := newTestEnv(t, kanbanTestSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	// 4 artifacts seeded; all must appear in the unfiltered response.
	if int(total) != 4 {
		t.Errorf("expected total=4 for unfiltered listing, got %d", int(total))
	}
	if len(items) != 4 {
		t.Errorf("expected 4 items for unfiltered listing, got %d", len(items))
	}

	// At least one artifact must be type "test".
	var foundTest bool
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if typ, _ := item["type"].(string); typ == "test" {
			foundTest = true
			break
		}
	}
	if !foundTest {
		t.Error("unfiltered artifact listing does not include any type=test artifacts")
	}
}

// TestKanbanVisibility_ConfigStructureUnchanged verifies that the kanban
// config endpoint returns a well-formed response and is not broken by the
// test artifact feature.  This is a regression guard.
// Covers test plan Milestone 4, scenario 2.
func TestKanbanVisibility_ConfigStructureUnchanged(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, nil, baseConfigYAML+`
kanban:
  columns:
    - name: Backlog
      statuses: [draft]
    - name: Ready
      statuses: [approved]
    - name: Done
      statuses: [done]
  uncategorised: true
  card_fields:
    - title
    - type
    - status
`)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/config/kanban", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	kanban, ok := data["kanban"].(map[string]any)
	if !ok || kanban == nil {
		t.Fatalf("expected kanban object in response, got %T: %v", data["kanban"], data["kanban"])
	}

	columns, _ := kanban["columns"].([]any)
	if len(columns) != 3 {
		t.Errorf("expected 3 kanban columns, got %d", len(columns))
	}

	uncategorised, hasUncategorised := kanban["uncategorised"]
	if !hasUncategorised {
		t.Error("expected uncategorised field in kanban config response")
	}
	if v, _ := uncategorised.(bool); !v {
		t.Errorf("expected uncategorised=true, got %v", uncategorised)
	}

	cardFields, _ := kanban["card_fields"].([]any)
	if len(cardFields) != 3 {
		t.Errorf("expected 3 card_fields, got %d", len(cardFields))
	}
}

// TestKanbanVisibility_TypeFilterExcludesTests verifies that
// GET /artifacts?type=ticket does NOT include test artifacts — the type filter
// applies correctly as an exclusive selector.
// Covers test plan Milestone 4, scenario 3.
func TestKanbanVisibility_TypeFilterExcludesTests(t *testing.T) {
	env := newTestEnv(t, kanbanTestSeeds())
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/artifacts?type=ticket", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	items, _ := data["items"].([]any)
	total, _ := data["total"].(float64)

	// Only 1 ticket artifact was seeded.
	if int(total) != 1 {
		t.Errorf("expected total=1 for type=ticket filter, got %d", int(total))
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item for type=ticket filter, got %d", len(items))
	}

	// Verify no test artifact leaked through.
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if typ, _ := item["type"].(string); typ == "test" {
			t.Errorf("type=ticket filter unexpectedly returned a type=test artifact: %v", item)
		}
		if typ, _ := item["type"].(string); typ != "ticket" {
			t.Errorf("type=ticket filter returned artifact with type %q", typ)
		}
	}
}

// TestKanbanVisibility_AllTestArtifactsInUnfilteredCount verifies that the
// total count in an unfiltered listing equals the sum of individual type-filtered
// counts, confirming that no artifact type is silently excluded.
// This is a regression guard against implicit test exclusion at the query layer.
func TestKanbanVisibility_AllTestArtifactsInUnfilteredCount(t *testing.T) {
	env := newTestEnv(t, kanbanTestSeeds())
	env.login("admin@test.local", "admin-pass-123")

	// Collect totals for each known type.
	types := []string{"test", "ticket", "idea"}
	typeTotal := 0
	for _, typ := range types {
		resp := env.doRequest("GET", "/api/p/testproject/artifacts?type="+typ, nil)
		requireStatus(t, resp, 200)
		d := readJSON(t, resp)
		n, _ := d["total"].(float64)
		typeTotal += int(n)
	}

	// Unfiltered total must equal the sum of per-type totals.
	resp := env.doRequest("GET", "/api/p/testproject/artifacts", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	unfilteredTotal, _ := data["total"].(float64)

	if int(unfilteredTotal) != typeTotal {
		t.Errorf("unfiltered total=%d does not match sum of per-type totals=%d (possible implicit exclusion)",
			int(unfilteredTotal), typeTotal)
	}
}
