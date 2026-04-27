---
title: "Kanban View — Backend Plan"
type: plan-backend
status: approved
lineage: kanban-view
parent: requirements/kanban-view-3.md
---

# Kanban View — Backend Plan

This plan covers the backend changes required to expose kanban board configuration to the frontend. The requirement ([[kanban-view]]) explicitly states the board is assembled client-side from the existing artifact list endpoint, so no new artifact query or grouping APIs are needed. The backend work is limited to parsing and serving the `kanban` config block.

## Milestone 1 — Add Kanban types to project config parser

### Description

Extend `internal/config/config.go` to define Go structs for the `kanban` section of `lifecycle/config.yaml` and include the field on the `Project` struct so it is parsed automatically by `LoadProject`.

### Files to change

- `internal/config/config.go` — Add `KanbanConfig`, `KanbanColumn`, and a `Kanban *KanbanConfig` field on `Project`.

### Implementation detail

```go
// KanbanColumn is one column definition in the kanban board.
type KanbanColumn struct {
    Name     string   `yaml:"name"`
    Statuses []string `yaml:"statuses"`
}

// KanbanConfig is the optional kanban board configuration.
type KanbanConfig struct {
    Columns       []KanbanColumn `yaml:"columns"`
    Uncategorised *bool          `yaml:"uncategorised,omitempty"` // default true
    CardFields    []string       `yaml:"card_fields,omitempty"`
}
```

The `Project` struct gains:

```go
Kanban *KanbanConfig `yaml:"kanban,omitempty"`
```

`Uncategorised` uses `*bool` so the default-true semantics can be applied at read time: if nil, treat as true.

### Acceptance criteria

- [ ] `LoadProject` on a config with a `kanban:` section populates `Project.Kanban` with columns, statuses, uncategorised flag, and card_fields.
- [ ] `LoadProject` on a config without `kanban:` leaves `Project.Kanban` as nil.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 2 — Expose parsed kanban config via the config API

### Description

The existing `GET /api/p/:project/config` endpoint returns the raw YAML text of `lifecycle/config.yaml`. The frontend needs structured kanban data. Add a dedicated `GET /api/p/:project/config/kanban` endpoint that returns the parsed `KanbanConfig` as JSON, or a `null` body when no kanban section is configured. This avoids forcing the frontend to parse raw YAML.

### Files to change

- `internal/http/config.go` — Add `handleGetKanbanConfig` handler.
- `internal/http/server.go` — Register `r.Get("/config/kanban", s.handleGetKanbanConfig)` inside the project sub-router.

### Implementation detail

The handler loads the project config (via `config.LoadProject` or the already-loaded project on the request context), serialises `Project.Kanban` to JSON, and returns it. If `Project.Kanban` is nil, return `{"kanban": null}`. If present, return:

```json
{
  "kanban": {
    "columns": [
      {"name": "Backlog", "statuses": ["draft"]},
      {"name": "Done", "statuses": ["done"]}
    ],
    "uncategorised": true,
    "card_fields": ["title", "type", "priority", "labels", "age"]
  }
}
```

The handler should reload the config from disk each time (config is not cached in the project runtime today) so that edits to `config.yaml` via the Config editor are reflected immediately.

### Acceptance criteria

- [ ] `GET /api/p/:project/config/kanban` returns 200 with the parsed kanban block when present.
- [ ] `GET /api/p/:project/config/kanban` returns `{"kanban": null}` when no `kanban` key exists.
- [ ] The response `Content-Type` is `application/json`.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 3 — Unit tests for kanban config parsing

### Description

Add unit tests to `internal/config/config_test.go` covering the new kanban parsing paths.

### Files to change

- `internal/config/config_test.go` — Add test cases.

### Test cases

1. **Full kanban config** — YAML with columns, uncategorised, and card_fields parses correctly.
2. **Minimal kanban config** — YAML with only `columns` (no `uncategorised`, no `card_fields`) parses; `Uncategorised` defaults to true semantics.
3. **No kanban key** — `Project.Kanban` is nil.
4. **Empty columns list** — Parses without error; `Columns` is an empty slice.

### Acceptance criteria

- [ ] All four test cases pass via `go test ./internal/config/ -run TestKanban`.
- [ ] No existing tests are broken.
