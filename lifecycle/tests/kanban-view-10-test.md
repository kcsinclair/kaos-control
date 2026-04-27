---
title: Kanban Routing — SPA Catch-All Fix Integration Tests
type: test
status: draft
lineage: kanban-view
parent: lifecycle/defects/kanban-view-9-defect.md
---

# Kanban Routing — SPA Catch-All Fix Integration Tests

Integration tests that verify the fix for the SPA routing 404 regression documented
in `lifecycle/defects/kanban-view-9-defect.md`.  The root cause was that
`newTestEnv` constructed the HTTP server with a zero-value `embed.FS`, so
`handleFrontend` could never open `dist/index.html` and returned 404 on every
SPA route.

## Changes made

### `internal/http/server.go`

- `ServerConfig.Frontend` changed from `embed.FS` to `fs.FS` so any `fs.FS`
  implementation (including `testing/fstest.MapFS`) can be injected.
- `handleFrontend` gains a nil guard: when `Frontend` is nil it returns
  `500 frontend unavailable` immediately rather than panicking or returning a
  misleading 404.

### `tests/integration/helpers_test.go`

- `newTestEnv` refactored into a thin wrapper around the new `newTestEnvFull`
  internal function that accepts an `fs.FS` parameter.
- New exported helper `newTestEnvWithFrontend(t, seeds, frontendFS)` for tests
  that need to exercise SPA catch-all routes.

### `tests/integration/kanban_routing_test.go`

- New helper `stubSPAFrontend()` returns a `fstest.MapFS` with a minimal
  `dist/index.html` containing valid HTML.
- Three tests updated to use `newTestEnvWithFrontend(t, nil, stubSPAFrontend())`
  instead of `newTestEnv(t, nil)`.

## Scenarios covered

| Test | Route | Expected |
|------|-------|----------|
| `TestKanbanRouting_BoardRouteServesSPA` | `GET /p/testproject/artifacts/board` | 200 + HTML body |
| `TestKanbanRouting_ListRouteUnchanged` | `GET /p/testproject/artifacts` | 200 + HTML body |
| `TestKanbanRouting_ArtifactEditorRouteUnchanged` | `GET /p/testproject/artifacts/requirements/kanban-view-3.md` | 200 + HTML body |
| `TestKanbanRouting_KanbanConfigRequiresAuth` | `GET /api/p/testproject/config/kanban` (no session) | 401 |

The fourth test (`KanbanConfigRequiresAuth`) uses the standard `newTestEnv`
helper as it does not need frontend assets.

## Running the tests

```sh
go test ./tests/integration/ -tags integration -run TestKanbanRouting -short -v
```
