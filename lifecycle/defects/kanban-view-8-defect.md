---
title: Kanban config API returns empty columns and card_fields due to missing JSON struct tags
type: defect
status: done
lineage: kanban-view
parent: lifecycle/tests/kanban-view-7-test.md
labels:
    - defect
    - backend
assignees:
    - role: backend-developer
      who: agent
release: April2026
---

# Kanban config API returns empty columns and card_fields due to missing JSON struct tags

## Reproduction Steps

1. Write a `lifecycle/config.yaml` that includes a `kanban` section with `columns` and `card_fields`.
2. Authenticate and call `GET /api/p/:project/config/kanban`.
3. Parse the JSON response and inspect `data["kanban"]["columns"]`.

## Expected Behaviour

The response should include `columns`, `uncategorised`, and `card_fields` as top-level keys inside the `kanban` object, e.g.:

```json
{
  "kanban": {
    "columns": [{"name": "Backlog", "statuses": ["draft"]}],
    "uncategorised": true,
    "card_fields": ["title", "type", "priority", "labels"]
  }
}
```

## Actual Behaviour

The `columns` and `card_fields` arrays are empty (length 0) and `uncategorised` is absent. The response looks like:

```json
{
  "kanban": {
    "Columns": [...],
    "Uncategorised": true,
    "CardFields": [...]
  }
}
```

Go's `encoding/json` package uses exported field names (`Columns`, `CardFields`, `Uncategorised`) when no `json:` struct tag is present. Tests and the frontend read lowercase keys and receive nil/empty slices.

## Logs / Output

```
kanban_config_test.go:97: expected 3 columns, got 0
kanban_config_test.go:116: expected uncategorised field in response
kanban_config_test.go:123: expected 4 card_fields, got 0
kanban_config_test.go:176: expected 2 columns, got 0
kanban_config_test.go:245: initial fetch: expected 2 columns, got 0
kanban_validation_test.go:37: expected 2 columns, got 0
kanban_validation_test.go:67: expected 2 columns (including empty one), got 0
kanban_validation_test.go:80: expected 'Empty Column' to be present in response
kanban_validation_test.go:113: expected 3 card_fields (including unknown ones), got 0
kanban_validation_test.go:142: expected 20 columns, got 0
```

Failing tests: `TestKanbanConfig_Full`, `TestKanbanConfig_Minimal`, `TestKanbanConfig_ReloadAfterEdit`, `TestKanbanValidation_DuplicateStatuses`, `TestKanbanValidation_ColumnEmptyStatuses`, `TestKanbanValidation_UnknownCardFields`, `TestKanbanValidation_ManyColumns`.

## Root Cause

`KanbanColumn` and `KanbanConfig` in `internal/config/config.go` (lines 210–219) have only `yaml:` struct tags. Add `json:` tags to match:

```go
type KanbanColumn struct {
    Name     string   `yaml:"name"     json:"name"`
    Statuses []string `yaml:"statuses" json:"statuses"`
}

type KanbanConfig struct {
    Columns       []KanbanColumn `yaml:"columns"              json:"columns"`
    Uncategorised *bool          `yaml:"uncategorised,omitempty" json:"uncategorised,omitempty"`
    CardFields    []string       `yaml:"card_fields,omitempty"   json:"card_fields,omitempty"`
}
```
