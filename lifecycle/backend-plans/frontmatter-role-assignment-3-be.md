---
title: "Backend Plan: Roles & Users API Endpoint"
type: plan-backend
status: done
lineage: frontmatter-role-assignment
parent: lifecycle/requirements/frontmatter-role-assignment-2.md
---

# Backend Plan: Roles & Users API Endpoint

This plan adds a `GET /api/p/{project}/roles` endpoint that returns the project's configured roles and users, enabling the frontend to populate the assignee picker (see [[frontmatter-role-assignment]]).

The existing `PUT /artifacts/*` handler already round-trips the full `assignees` array in frontmatter, so no persistence changes are needed.

## Milestone 1 — Add GET /roles endpoint

### Description

Create a new handler that reads the project's `Cfg.Roles` and `Cfg.Users` from the already-parsed `config.Project` struct and returns them as JSON. Register it in the chi router alongside the existing `/config` routes.

### Files to change

- `internal/http/server.go` — Register `GET /api/p/{project}/roles` route in `buildRouter()`, in the project-scoped group (near the existing `/config` routes around line 159).
- `internal/http/roles.go` — New file. Implement `handleGetRoles(w, r)`:
  1. Extract project from context via `projectFromCtx(r.Context())`.
  2. Build a response struct containing `roles []string` (from `p.Cfg.Roles`) and `users []UserBinding` (from `p.Cfg.Users`, each with `email` and `roles` fields).
  3. Marshal to JSON and write with `200 OK`.

### Response shape

```json
{
  "roles": ["product-owner", "analyst", "backend-developer", ...],
  "users": [
    {"email": "keith@sinclair.org.au", "roles": ["product-owner", "analyst", ...]}
  ]
}
```

### Acceptance criteria

- [ ] `GET /api/p/{project}/roles` returns `200` with the JSON shape above.
- [ ] The `roles` array matches the `roles` list in the project's `lifecycle/config.yaml`.
- [ ] The `users` array matches the `users` list in `lifecycle/config.yaml`, with each entry's `email` and `roles` fields preserved.
- [ ] An unauthenticated request (if auth is enabled) returns `401`.
- [ ] `go build ./...` and `go vet ./...` pass.

## Milestone 2 — Validate assignee roles on artifact save

### Description

Add server-side validation to the existing `PUT /artifacts/*` handler so that any `assignees[].role` value submitted must be present in the project's `Cfg.Roles` list. This prevents the API from persisting invalid roles even if the frontend is bypassed.

### Files to change

- `internal/http/write.go` — In `handleUpdateArtifact()`, after parsing the request body and before writing to disk (around line 212):
  1. If `req.Frontmatter.Assignees` is non-nil, iterate each entry.
  2. For each assignee, check that `assignee.Role` exists in `p.Cfg.Roles`.
  3. If any role is invalid, return `400 Bad Request` with a JSON error body listing the invalid role(s).

### Acceptance criteria

- [ ] Submitting an artifact with `assignees: [{role: "invalid-role", who: "agent"}]` returns `400` with an error message naming the invalid role.
- [ ] Submitting valid roles (e.g. `backend-developer`) succeeds as before.
- [ ] Submitting an artifact with no `assignees` field succeeds as before (no regression).
- [ ] `go build ./...` and `go vet ./...` pass.
