//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// baseConfigYAML is a minimal valid project config without a kanban section.
// Tests that need a kanban section append to this.
const baseConfigYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - backend-developer
  - frontend-developer
  - test-developer
  - qa
  - reviewer
  - approver

stages:
  - {name: ideas, dir: ideas}
  - {name: requirements, dir: requirements}
  - {name: backend-plans, dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans, dir: test-plans}
  - {name: tests, dir: tests}
  - {name: prototypes, dir: prototypes}
  - {name: releases, dir: releases}
  - {name: sprints, dir: sprints}
  - {name: defects, dir: defects}

users:
  - email: admin@test.local
    roles: [product-owner, analyst, reviewer, approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
`

// writeProjectConfig replaces lifecycle/config.yaml in the test environment.
func writeProjectConfig(t *testing.T, env *testEnv, yaml string) {
	t.Helper()
	path := filepath.Join(env.projectRoot, "lifecycle", "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatalf("writeProjectConfig: %v", err)
	}
}

// TestKanbanConfig_Full verifies that a config with a complete kanban section
// is returned correctly by GET /api/p/:project/config/kanban.
// Covers Milestone 1, scenario 1.
func TestKanbanConfig_Full(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	writeProjectConfig(t, env, baseConfigYAML+`
kanban:
  columns:
    - name: Backlog
      statuses: [draft]
    - name: Approved
      statuses: [approved]
    - name: Done
      statuses: [done]
  uncategorised: true
  card_fields:
    - title
    - type
    - priority
    - labels
`)

	resp := env.doRequest("GET", "/api/p/testproject/config/kanban", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	kanban, ok := data["kanban"].(map[string]any)
	if !ok || kanban == nil {
		t.Fatalf("expected kanban object in response, got %T: %v", data["kanban"], data["kanban"])
	}

	columns, _ := kanban["columns"].([]any)
	if len(columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(columns))
	}

	// Check first column name and statuses.
	if len(columns) > 0 {
		col, _ := columns[0].(map[string]any)
		if name, _ := col["name"].(string); name != "Backlog" {
			t.Errorf("expected first column name %q, got %q", "Backlog", name)
		}
		statuses, _ := col["statuses"].([]any)
		if len(statuses) != 1 {
			t.Errorf("expected 1 status in Backlog, got %d", len(statuses))
		} else if s, _ := statuses[0].(string); s != "draft" {
			t.Errorf("expected status %q in Backlog, got %q", "draft", s)
		}
	}

	uncategorised, hasUncategorised := kanban["uncategorised"]
	if !hasUncategorised {
		t.Error("expected uncategorised field in response")
	} else if v, _ := uncategorised.(bool); !v {
		t.Errorf("expected uncategorised=true, got %v", uncategorised)
	}

	cardFields, _ := kanban["card_fields"].([]any)
	if len(cardFields) != 4 {
		t.Errorf("expected 4 card_fields, got %d", len(cardFields))
	}
}

// TestKanbanConfig_None verifies that a config without a kanban key returns
// {"kanban": null}.
// Covers Milestone 1, scenario 2.
func TestKanbanConfig_None(t *testing.T) {
	// Default env config has no kanban section.
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/config/kanban", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	// kanban key must be present and null.
	kanbanVal, hasKanban := data["kanban"]
	if !hasKanban {
		t.Error("expected 'kanban' key in response")
	}
	if kanbanVal != nil {
		t.Errorf("expected kanban=null when no kanban section in config, got %v", kanbanVal)
	}
}

// TestKanbanConfig_Minimal verifies that a config with only kanban.columns
// (no uncategorised, no card_fields) is returned correctly.
// Covers Milestone 1, scenario 3.
func TestKanbanConfig_Minimal(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	writeProjectConfig(t, env, baseConfigYAML+`
kanban:
  columns:
    - name: Backlog
      statuses: [draft]
    - name: Done
      statuses: [done]
`)

	resp := env.doRequest("GET", "/api/p/testproject/config/kanban", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	kanban, ok := data["kanban"].(map[string]any)
	if !ok || kanban == nil {
		t.Fatalf("expected kanban object in response, got %T", data["kanban"])
	}

	columns, _ := kanban["columns"].([]any)
	if len(columns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(columns))
	}

	// card_fields should be absent or empty when not configured.
	cardFields, _ := kanban["card_fields"].([]any)
	if len(cardFields) != 0 {
		t.Errorf("expected empty card_fields when not configured, got %v", cardFields)
	}
}

// TestKanbanConfig_EmptyColumns verifies that kanban.columns: [] returns 200
// with an empty columns array.
// Covers Milestone 1, scenario 4.
func TestKanbanConfig_EmptyColumns(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	writeProjectConfig(t, env, baseConfigYAML+`
kanban:
  columns: []
`)

	resp := env.doRequest("GET", "/api/p/testproject/config/kanban", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	kanban, ok := data["kanban"].(map[string]any)
	if !ok || kanban == nil {
		t.Fatalf("expected kanban object in response, got %T", data["kanban"])
	}

	// columns may be null or an empty array — both are valid for an empty list.
	switch v := kanban["columns"].(type) {
	case []any:
		if len(v) != 0 {
			t.Errorf("expected empty columns array, got %d elements", len(v))
		}
	case nil:
		// null is acceptable for an empty YAML list
	default:
		t.Errorf("unexpected type for columns: %T", kanban["columns"])
	}
}

// TestKanbanConfig_ReloadAfterEdit verifies that editing lifecycle/config.yaml
// is reflected in the next request to GET /config/kanban without a server restart.
// Covers Milestone 1, scenario 5.
func TestKanbanConfig_ReloadAfterEdit(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Write initial kanban config with 2 columns.
	writeProjectConfig(t, env, baseConfigYAML+`
kanban:
  columns:
    - name: Backlog
      statuses: [draft]
    - name: Done
      statuses: [done]
`)

	// Fetch initial state — should have 2 columns.
	resp := env.doRequest("GET", "/api/p/testproject/config/kanban", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	kanban, _ := data["kanban"].(map[string]any)
	columns, _ := kanban["columns"].([]any)
	if len(columns) != 2 {
		t.Fatalf("initial fetch: expected 2 columns, got %d", len(columns))
	}

	// Give the watcher a moment to settle (the handler reads from disk anyway).
	time.Sleep(50 * time.Millisecond)

	// Update config to add a third column.
	writeProjectConfig(t, env, baseConfigYAML+`
kanban:
  columns:
    - name: Backlog
      statuses: [draft]
    - name: In Progress
      statuses: [in-development]
    - name: Done
      statuses: [done]
`)

	// Fetch again — new column must appear without server restart.
	resp2 := env.doRequest("GET", "/api/p/testproject/config/kanban", nil)
	requireStatus(t, resp2, 200)
	data2 := readJSON(t, resp2)

	kanban2, _ := data2["kanban"].(map[string]any)
	columns2, _ := kanban2["columns"].([]any)
	if len(columns2) != 3 {
		t.Errorf("after edit: expected 3 columns, got %d", len(columns2))
	}

	// Verify the new column is present.
	found := false
	for _, c := range columns2 {
		col, _ := c.(map[string]any)
		if name, _ := col["name"].(string); name == "In Progress" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'In Progress' column after config reload")
	}
}
