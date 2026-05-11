---
title: "Backend Plan — Role Checks on Mutation Endpoints"
type: plan-backend
status: done
lineage: auth-role-checks-mutations
parent: lifecycle/requirements/auth-role-checks-mutations-2.md
created: "2026-05-11T08:10:00+10:00"
priority: high
labels:
    - security
    - backend
    - release-blocker
release: KC-Release0
---

# Backend Plan — Role Checks on Mutation Endpoints

This plan implements the permission matrix in [[auth-role-checks-mutations-2]]. The approach is mechanical: lift `hasAnyRole` into a shared file, define role constants, and apply the gate uniformly.

## Milestone 1 — Centralise permission constants

### Description

Create `internal/http/permissions.go` to hold the role-name constants, group lists, and a shared `hasAnyRole` helper. The current copy of `hasAnyRole` lives in `internal/http/devops.go:25` — move it here.

### Files to change

- **New** `internal/http/permissions.go`:
  - Move `hasAnyRole` here unchanged.
  - Define exported constants:
    ```go
    const (
        RoleProductOwner      = "product-owner"
        RoleAnalyst           = "analyst"
        RoleBackendDeveloper  = "backend-developer"
        RoleFrontendDeveloper = "frontend-developer"
        RoleTestDeveloper     = "test-developer"
        RoleQA                = "qa"
        RoleReviewer          = "reviewer"
        RoleApprover          = "approver"
        RoleDevops            = "devops"
    )
    ```
  - Define group variables corresponding to the matrix in [[auth-role-checks-mutations-2]]:
    - `RolesArtifactAuthors` — the five authoring roles.
    - `RolesArtifactEditors` — authors plus QA.
    - `RolesAdminOnly` — `[]string{RoleProductOwner}`.
    - `RolesDevopsOrAdmin` — `[]string{RoleProductOwner, RoleDevops}`.
    - `RolesPriorityEditors` — `[]string{RoleProductOwner, RoleAnalyst}`.
  - Add `requireRole(w, r, p, allowed...) bool` helper that:
    1. Reads `user := userFromCtx(r.Context())`; if nil, writes 401 and returns false.
    2. Reads `roles := p.Cfg.RolesFor(user.Email)`; if `!hasAnyRole(roles, allowed...)`, writes 403 with body `apiError("forbidden", "role required: <comma-joined>")` and returns false.
    3. Otherwise returns true.
  - Add `requireAppRole(w, r, allowed...) bool` for app-level endpoints (no project context). Iterates `s.projects`, finds the union of roles for the user, then applies `hasAnyRole`.

- **Edit** `internal/http/devops.go`:
  - Delete the local `hasAnyRole` (lines 24–32).
  - Replace each inline `hasAnyRole(roles, "product-owner", "devops")` block with `if !requireRole(w, r, p, RolesDevopsOrAdmin...) { return }`.

### Acceptance criteria

- `go build ./...` clean.
- All devops integration tests still pass unchanged.
- `go vet ./...` clean.

---

## Milestone 2 — Artifact CRUD role checks

### Files to change

- **Edit** `internal/http/write.go`:
  - `handleCreateArtifact` (line 34): add `if !requireRole(w, r, p, RolesArtifactAuthors...) { return }` after the `projectFromCtx` nil-check.
  - `handleUpdateArtifact` (line 169): add `if !requireRole(w, r, p, RolesArtifactEditors...) { return }`.
  - `handleDeleteArtifact` (line 285): add `if !requireRole(w, r, p, RolesAdminOnly...) { return }`.
  - `handlePatchPriority` (line 430): replace the existing local `userFromCtx == nil → 401` with `if !requireRole(w, r, p, RolesPriorityEditors...) { return }`.

### Acceptance criteria

- New integration tests in `tests/integration/role_enforcement_test.go`:
  - `TestArtifactCreate_RequiresAuthorRole` — `qa@test.local` (qa role only) gets 403; `dev@test.local` (developer roles) gets 201.
  - `TestArtifactUpdate_RequiresEditorRole` — analogous, with `qa` included as allowed.
  - `TestArtifactDelete_RequiresProductOwner` — only `admin@test.local` succeeds.
  - `TestArtifactPriority_RequiresPriorityRole` — only roles in `RolesPriorityEditors` succeed.

---

## Milestone 3 — Scheduler role checks

### Files to change

- **Edit** `internal/http/scheduler.go`:
  - `handleCreateSchedulerJob` (line 102): add `if !requireRole(w, r, p, RolesDevopsOrAdmin...) { return }`.
  - `handleUpdateSchedulerJob` (line 165): same.
  - `handleDeleteSchedulerJob` (line 214): same.

### Acceptance criteria

- `TestSchedulerJobCreate_RequiresDevopsOrAdmin` — `qa@test.local` gets 403; `admin@test.local` gets 201. Validate that an unauthenticated request to `POST /scheduler/jobs` still returns 401 (covered by `requireAuth`).

---

## Milestone 4 — Agent run role check

### Description

`handleStartAgentRun` has a special case: the matrix says "product-owner, plus the agent's own configured role". An agent's role is recorded in `lifecycle/config.yaml` agents section as the `role` field on each agent entry (e.g. the `backend-developer` agent has `role: backend-developer`).

### Files to change

- **Edit** `internal/http/agents.go`:
  - In `handleStartAgentRun` (line 52), after `p.Agents == nil` check, look up the agent by `name` from `p.Cfg.Agents`. If found, derive `allowed := []string{RoleProductOwner, agent.Role}` and call `if !requireRole(w, r, p, allowed...) { return }`.
  - If the agent name isn't configured, return 404 (`apiError("not_found", ...)`) — matches the existing convention used elsewhere in `internal/http/agents.go` and `internal/http/artifacts.go`.

### Acceptance criteria

- `TestAgentRun_AllowsAgentRole` — `dev@test.local` (backend-developer) can launch the `backend-developer` agent (202) but not the `qa` agent (403).
- `TestAgentRun_AllowsProductOwner` — `admin@test.local` can launch any agent.

---

## Milestone 5 — Releases role check

### Files to change

- **Edit** `internal/http/releases.go`:
  - `handleCreateRelease` (line 54): add `if !requireRole(w, r, p, RolesAdminOnly...) { return }`.
  - `handleUpdateRelease` (line 159): same.
  - `handleDeleteRelease` (line 251): same.

### Acceptance criteria

- `TestReleases_RequireProductOwner` — only `admin@test.local` can create/update/delete a release; `dev@test.local` gets 403.

---

## Milestone 6 — Project config role check

### Files to change

- **Edit** `internal/http/config.go`:
  - `handleUpdateConfig` (line 71): add `if !requireRole(w, r, p, RolesAdminOnly...) { return }`.

### Acceptance criteria

- `TestConfigUpdate_RequiresProductOwner` — `admin@test.local` succeeds; `dev@test.local` gets 403. Particularly important because config rewrites could otherwise let a non-admin grant themselves `product-owner`.

---

## Milestone 7 — Ollama instances (app-level)

### Description

Ollama instance routes (`/api/ollama/instances`) are mounted outside the `/api/p/:project/` block, so there's no `projectFromCtx`. Use the new `requireAppRole(w, r, s, RolesDevopsOrAdmin...)` helper.

### Files to change

- **Edit** `internal/http/ollama.go`:
  - `handleCreateOllamaInstance` (line 74): add `if !requireAppRole(w, r, s, RolesDevopsOrAdmin...) { return }` after the `appCfg` nil-check.
  - `handleUpdateOllamaInstance` (line 124): same.
  - `handleDeleteOllamaInstance` (line 171): same.

### Acceptance criteria

- `TestOllamaInstance_RequiresDevopsOrAdmin` — `qa@test.local` gets 403; `admin@test.local` (product-owner in testproject) gets 201.

---

## Milestone 8 — Admin users (post-bootstrap)

### Description

`handleCreateUser` (`internal/http/auth.go:259`) currently allows the first-ever user without auth and requires *any* authenticated user thereafter. Tighten the post-bootstrap path to `product-owner` only.

### Files to change

- **Edit** `internal/http/auth.go`:
  - In `handleCreateUser`, after the `count > 0 && userFromCtx == nil` 401 branch, add the post-bootstrap role check: `if count > 0 && !appUserHasRole(s, user, RoleProductOwner) { writeJSON(w, 403, apiError("forbidden", "product-owner role required")) ; return }`. (`appUserHasRole` is a small inline helper or reuse `requireAppRole` with explicit no-write semantics — see helper design in Milestone 1.)

### Acceptance criteria

- `TestCreateUser_PostBootstrap_RequiresProductOwner` — after the bootstrap user exists, only `admin@test.local` can create a second user.
- `TestCreateUser_Bootstrap_StillAuthless` — the very first user can still be created without auth (existing test `TestBootstrapFirstUser`).

---

## Verification (end-to-end)

1. `make lint` — clean.
2. `make test-unit` — clean.
3. `make test-integration` — clean (the new `role_enforcement_test.go` runs alongside existing tests).
4. Manual smoke: log in as `dev@test.local`, attempt `PUT /api/p/kaos-control/config` — should get 403. Log in as `keith@sinclair.org.au` (product-owner), same call should succeed.

## Risk notes

- **Auto-login user**: `newTestEnvFull` auto-logs in as `admin@test.local`, who holds `product-owner, analyst, reviewer, approver`. This covers most authoring paths. Tests that need a *non-privileged* user must call `env.login("qa@test.local", "qa-pass-123")` explicitly.
- **Frontend impact**: 403 responses are already handled by the SPA's generic error path. The button visibility for restricted actions is a future polish item — not blocked by this work.
- **Backward compatibility**: this is a security tightening, not a behaviour change. Existing valid use cases keep working because `admin@test.local` (and the project owner `keith@sinclair.org.au` in kaos-control) hold `product-owner`.
