---
title: Role Checks on Mutation Endpoints
type: requirement
status: done
lineage: auth-role-checks-mutations
created: "2026-05-11T08:05:00+10:00"
priority: high
parent: lifecycle/ideas/auth-role-checks-mutations.md
labels:
    - security
    - backend
    - release-blocker
release: KC-Release0
---

# Role Checks on Mutation Endpoints

Parent: [[auth-role-checks-mutations]].

## Goal

Every `POST`, `PUT`, `PATCH`, `DELETE` handler in `internal/http/` enforces a role check using the existing `RolesFor` + `hasAnyRole` pattern. The permission matrix below is the source of truth.

## Permission matrix

| Endpoint group | Method | Allowed roles | Rationale |
|---|---|---|---|
| `/api/p/:project/artifacts` create | `POST` | `product-owner`, `analyst`, `backend-developer`, `frontend-developer`, `test-developer` | The five roles that legitimately introduce new lifecycle artifacts. `qa` does not author artifacts (it raises defects via the defect-create path). `reviewer` / `approver` only transition existing artifacts. |
| `/api/p/:project/artifacts` update | `PUT` | `product-owner`, `analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa` | Authoring roles plus `qa` (who edits defect artifacts they raised). |
| `/api/p/:project/artifacts` delete | `DELETE` | `product-owner` | Destructive; restrict to the project superuser. |
| `/api/p/:project/artifacts/*/priority` | `PATCH` | `product-owner`, `analyst` | Priority is a planning signal. |
| `/api/p/:project/artifacts/*/transition` | `POST` | (existing — `workflow.CanTransition`) | No change; already covered by the rule matrix. |
| `/api/p/:project/scheduler/jobs` | `POST` / `PUT` / `DELETE` | `product-owner`, `devops` | Shell-execution capability. Matches the existing devops policy. |
| `/api/p/:project/agents/:name/run` | `POST` | `product-owner`, plus the agent's own configured role (e.g. `backend-developer` for `backend-developer` agent) | Lets the responsible role re-run their own agent; `product-owner` can launch any agent. |
| `/api/p/:project/releases` | `POST` / `PUT` / `DELETE` | `product-owner` | Release-level planning. |
| `/api/ollama/instances` | `POST` / `PUT` / `DELETE` | `product-owner`, `devops` | App-level configuration; matches devops policy. |
| `/api/p/:project/config` | `PUT` | `product-owner` | Config edits can change role assignments — restrict to the superuser. |
| `/api/admin/users` (post-bootstrap) | `POST` | `product-owner` | First-user bootstrap remains auth-less; subsequent calls require product-owner. |

## Functional requirements

1. **Role check uses the existing helpers.** Handlers call `p.Cfg.RolesFor(user.Email)` and pass the result to `hasAnyRole(roles, allowed...)`. No new auth primitives.
2. **Failure response shape.** Unauthorised requests return `403 Forbidden` with `apiError("forbidden", "role required: <role-list>")`. Body matches the existing devops handler shape so the frontend's existing 403 handling works unchanged.
3. **Defensive auth check.** Each handler keeps an early `userFromCtx == nil → 401` check for defence in depth, mirroring the existing devops handlers.
4. **Centralised list.** The permission matrix above lives in a Go file (e.g. `internal/http/permissions.go`) as exported constants/variables so it's testable and discoverable. Handlers reference those constants rather than embedding role lists inline.
5. **App-level endpoints** (`/api/ollama/instances`, `/api/admin/users`) lack a project context for `RolesFor`. For these, the user's role must come from *any* project they participate in — if they have `product-owner` or `devops` in any project, allow. Otherwise 403.

## Non-functional requirements

- No new dependencies.
- No change to the `requireAuth` middleware itself.
- No frontend changes required (the SPA's existing 403 handling is sufficient; future polish to surface role errors is out of scope here).

## Acceptance criteria

- A new integration test file `tests/integration/role_enforcement_test.go` covers each endpoint group with:
  - One "happy path" test where a user with an allowed role succeeds (200/201/204).
  - One "denied" test where a user with the `qa` role (deliberately under-privileged for most groups) receives 403.
- All existing integration tests still pass after the migration (the auto-login in `newTestEnvFull` uses `admin@test.local`, which holds `product-owner, analyst, reviewer, approver` — sufficient for most endpoints).
- `make lint`, `make test-unit`, `make test-integration` all green.

## Out of scope

- Per-stage write permissions for artifact CRUD (e.g. "frontend-developer can only write to `lifecycle/frontend-plans/`"). The role-based gate is sufficient for KC-Release0; finer-grained path policy belongs to a follow-up.
- Auditing / structured access logs.
- Frontend changes to show role-denied messages prominently.

## No questions

None for the backend developer at this stage; the matrix above is the contract.
