//go:build integration

package integration

import (
	"fmt"
	"testing"
)

// TestKanbanValidation_DuplicateStatuses verifies that the kanban config
// endpoint returns 200 even when the same status appears in two different
// columns. The backend does not validate uniqueness — that is a frontend concern.
// Covers Milestone 4, scenario 1.
func TestKanbanValidation_DuplicateStatuses(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	writeProjectConfig(t, env, baseConfigYAML+`
kanban:
  columns:
    - name: Column A
      statuses: [draft, approved]
    - name: Column B
      statuses: [draft, done]
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
}

// TestKanbanValidation_ColumnEmptyStatuses verifies that a column with an empty
// statuses array does not cause a server error.
// Covers Milestone 4, scenario 2.
func TestKanbanValidation_ColumnEmptyStatuses(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	writeProjectConfig(t, env, baseConfigYAML+`
kanban:
  columns:
    - name: Empty Column
      statuses: []
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
		t.Errorf("expected 2 columns (including empty one), got %d", len(columns))
	}

	// Find the empty column and confirm it is present.
	found := false
	for _, c := range columns {
		col, _ := c.(map[string]any)
		if name, _ := col["name"].(string); name == "Empty Column" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Empty Column' to be present in response")
	}
}

// TestKanbanValidation_UnknownCardFields verifies that card_fields entries that
// don't correspond to known artifact fields do not cause a server error.
// Backend does not validate field names.
// Covers Milestone 4, scenario 3.
func TestKanbanValidation_UnknownCardFields(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	writeProjectConfig(t, env, baseConfigYAML+`
kanban:
  columns:
    - name: Backlog
      statuses: [draft]
  card_fields:
    - title
    - nonexistent_field
    - another_unknown_field
`)

	resp := env.doRequest("GET", "/api/p/testproject/config/kanban", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	kanban, ok := data["kanban"].(map[string]any)
	if !ok || kanban == nil {
		t.Fatalf("expected kanban object in response, got %T", data["kanban"])
	}
	cardFields, _ := kanban["card_fields"].([]any)
	if len(cardFields) != 3 {
		t.Errorf("expected 3 card_fields (including unknown ones), got %d", len(cardFields))
	}
}

// TestKanbanValidation_ManyColumns verifies that a kanban config with 20
// columns is returned correctly without a server error.
// Covers Milestone 4, scenario 4.
func TestKanbanValidation_ManyColumns(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Build YAML for 20 columns.
	columnsYAML := "\nkanban:\n  columns:\n"
	for i := 1; i <= 20; i++ {
		columnsYAML += fmt.Sprintf("    - name: Column%02d\n      statuses: [status-%02d]\n", i, i)
	}

	writeProjectConfig(t, env, baseConfigYAML+columnsYAML)

	resp := env.doRequest("GET", "/api/p/testproject/config/kanban", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	kanban, ok := data["kanban"].(map[string]any)
	if !ok || kanban == nil {
		t.Fatalf("expected kanban object in response, got %T", data["kanban"])
	}
	columns, _ := kanban["columns"].([]any)
	if len(columns) != 20 {
		t.Errorf("expected 20 columns, got %d", len(columns))
	}
}
