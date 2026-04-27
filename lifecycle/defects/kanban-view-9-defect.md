---
title: SPA routing integration tests fail because test server has no embedded frontend
type: defect
status: in-development
lineage: kanban-view
parent: lifecycle/tests/kanban-view-7-test.md
labels:
    - defect
    - test
assignees:
    - role: test-developer
      who: agent
---

# SPA routing integration tests fail because test server has no embedded frontend

## Reproduction Steps

1. Run the kanban routing integration tests:
   ```sh
   go test ./tests/integration/ -tags integration -run TestKanbanRouting -short -v
   ```
2. Observe that `TestKanbanRouting_BoardRouteServesSPA`, `TestKanbanRouting_ListRouteUnchanged`, and `TestKanbanRouting_ArtifactEditorRouteUnchanged` all fail with 404.

## Expected Behaviour

Each of the three SPA route tests should receive HTTP 200 with an HTML body confirming the catch-all frontend handler is wired up correctly for those paths.

## Actual Behaviour

All three tests receive HTTP 404 with the body `index.html not found`. The chi router correctly dispatches to `handleFrontend`, but that handler cannot serve `index.html` because no frontend assets are embedded in the test server.

## Logs / Output

```
kanban_routing_test.go:36: expected status 200, got 404: index.html not found
kanban_routing_test.go:53: expected status 200, got 404: index.html not found
kanban_routing_test.go:71: expected status 200, got 404: index.html not found
```

## Root Cause

`newTestEnv` in `tests/integration/helpers_test.go` (line 208) constructs the HTTP server without providing a `Frontend` embed.FS:

```go
srv := kaoshttp.New(kaoshttp.ServerConfig{
    Listen: addr,
    Auth:   authStore,
    // Frontend field is zero-value — empty embed.FS
}, ...)
```

`handleFrontend` in `internal/http/server.go` calls `fs.Sub(s.cfg.Frontend, "dist")` on this empty FS. When the fallback to `index.html` is attempted, `serveFSFile` cannot open the file and returns 404.

The fix requires one of:
- Seeding a minimal `web/dist/index.html` stub into the test server's `Frontend` embed.FS, or
- Adding a `newTestEnvWithFrontend` helper variant that injects a minimal `fs.FS` containing a stub `index.html` under `dist/`, and using it in the three routing tests.

The existing approach of asserting SPA responses without a frontend cannot work.
