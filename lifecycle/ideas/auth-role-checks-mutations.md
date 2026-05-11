---
title: Enforce Role Checks on All Mutation Endpoints
type: idea
status: done
lineage: auth-role-checks-mutations
created: "2026-05-11T08:00:00+10:00"
priority: high
labels:
    - security
    - backend
    - release-blocker
release: KC-Release0
---

# Enforce Role Checks on All Mutation Endpoints

## Problem

The global `requireAuth` middleware introduced by commit `b2921c1` enforces *authentication* on every `/api/*` endpoint, but does not enforce *authorisation*. Today, any user who can log in can mutate any project's state — regardless of their configured role.

Role-based authorisation IS implemented, but only on a subset of handlers:

- `internal/http/devops.go` — checks `product-owner` or `devops` on every mutation
- `internal/http/transition.go` — runs `workflow.CanTransition(roles, ...)` against the rule matrix
- `internal/http/status_check.go` — checks role for QA actions

The following mutation handlers perform **no role check**:

| Endpoint | File:line |
|---|---|
| `POST/PUT/DELETE /api/p/:project/artifacts/*` | `internal/http/write.go:34,169,285` |
| `POST/PUT/DELETE /api/p/:project/scheduler/jobs` | `internal/http/scheduler.go:102,165,214` |
| `POST /api/p/:project/agents/:name/run` | `internal/http/agents.go:52` |
| `POST/PUT/DELETE /api/p/:project/releases` | `internal/http/releases.go:54,159,251` |
| `POST/PUT/DELETE /api/ollama/instances/*` | `internal/http/ollama.go:74,124,171` |
| `PUT /api/p/:project/config` | `internal/http/config.go:71` |

## Impact

Any user with valid credentials can:

1. **Execute arbitrary shell commands** by creating a scheduler job with `target_type: shell` — full privilege escalation.
2. **Modify the project config**, including the `roles` and `users` sections — full privilege escalation by reassigning their own role.
3. **Trigger any agent** against any target path — bypasses the intended agent-launch workflow.
4. **Create / edit / delete any artifact** in any project — bypasses workflow rules and agent `allowed_write_paths`.
5. **Manipulate releases** — affects sprint planning and milestone integrity.
6. **Create / delete Ollama instances** — disrupts other users' integrations.

For any deployment where authenticated users are not uniformly trusted (e.g., a `qa` user who should not be able to alter the project config), this constitutes a hard release blocker.

## Desired outcome

Every mutation endpoint enforces a documented role requirement consistent with the established pattern from `devops.go`. A central permission matrix lives in the codebase so future endpoints stay aligned by default.

## Related

- The `devops`, `transition`, and `status-check` handlers already implement the pattern that should be applied uniformly.
- `internal/http/devops.go:24` defines a reusable `hasAnyRole` helper.
- `config.Project.RolesFor(email)` returns the role list for the authenticated user.
