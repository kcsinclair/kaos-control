---
title: GET /api/p/:project/config/kanban missing requireAuth middleware — returns 200 unauthenticated
type: defect
status: done
lineage: kanban-view
parent: lifecycle/tests/kanban-view-7-test.md
labels:
    - defect
    - backend
    - security
assignees:
    - role: backend-developer
      who: agent
---

# GET /api/p/:project/config/kanban missing requireAuth middleware — returns 200 unauthenticated

## Reproduction Steps

1. Start the server (or use the integration test environment).
2. Without authenticating (no session cookie), send:
   ```
   GET /api/p/testproject/config/kanban
   ```
3. Observe the response.

## Expected Behaviour

An unauthenticated request should receive HTTP 401 with an `unauthorized` error body.

## Actual Behaviour

The endpoint returns HTTP 200 with the kanban config (or `{"kanban":null}` if no config is present). Project configuration is exposed without authentication.

## Logs / Output

```
kanban_routing_test.go:93: expected 401 for unauthenticated request to /config/kanban, got 200: {"kanban":null}
```

Failing test: `TestKanbanRouting_KanbanConfigRequiresAuth`.

## Root Cause

In `internal/http/server.go` (line 162) the `/config/kanban` route is registered without the `requireAuth` middleware:

```go
// current — no auth guard
r.Get("/config/kanban", s.handleGetKanbanConfig)
```

The `requireAuth` middleware already exists (`internal/http/auth.go:79`). The route registration must be wrapped:

```go
r.With(requireAuth).Get("/config/kanban", s.handleGetKanbanConfig)
```

For consistency, `GET /config` (line 160) and `PUT /config` (line 161) should be audited to confirm they are also protected.
