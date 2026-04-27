---
title: "Frontend Plan: Filter Transition Dialog by Allowed Targets"
type: plan-frontend
status: draft
lineage: innovation-maker
parent: lifecycle/defects/product-owner-cannot-transition.md
labels:
    - frontend
    - defect-fix
---

# Frontend Plan: Filter Transition Dialog by Allowed Targets

The `TransitionDialog.vue` component currently shows a hardcoded list of all
statuses (minus the current one). When a user selects a status they are not
authorised for, the backend returns 403 and the user sees a cryptic error. The
dialog should only present transitions the user is actually allowed to make.

---

## Milestone 1 — Add `getAllowedTargets` API call

### Description
Create a new API function that calls the backend to retrieve the list of
statuses the current user can transition a given artifact to. The backend
already returns `allowed_targets` in the 403 error response body, but a
dedicated endpoint is cleaner. Until such an endpoint exists, derive allowed
targets client-side by attempting a lightweight approach: add a new
`GET /api/p/:project/artifacts/:path/allowed-targets` endpoint on the backend,
or reuse the existing 403 response shape.

**Preferred approach**: add a thin backend endpoint (see
[[product-owner-cannot-transition-8-be]]) that wraps `AllowedTargets`. If
that endpoint is not yet available, fall back to showing all statuses and
handling the 403 gracefully (current behaviour, improved error message).

### Files to change
- `web/src/api/artifacts.ts` — add `getAllowedTargets(project, path)` function

### Acceptance criteria
- [ ] New function `getAllowedTargets` exists and returns `string[]`
- [ ] Function calls `GET /api/p/{project}/artifacts/{path}/allowed-targets`
- [ ] TypeScript compiles without errors (`pnpm exec vue-tsc --noEmit`)

---

## Milestone 2 — Add backend `GET /allowed-targets` endpoint

### Description
Wire up a lightweight handler that returns the list of statuses the
authenticated user may transition a given artifact to. This keeps the
frontend from guessing.

### Files to change
- `internal/http/transition.go` — add `handleAllowedTargets` handler
- `internal/http/routes.go` (or wherever routes are registered) — register `GET .../allowed-targets`

### Acceptance criteria
- [ ] `GET /api/p/:project/artifacts/*path/allowed-targets` returns `{"targets": ["clarifying", "abandoned", ...]}`
- [ ] Unauthenticated requests get 401
- [ ] Response for product-owner includes all reachable statuses (per [[product-owner-cannot-transition-8-be]])
- [ ] `go build ./...` and `go vet ./...` pass

---

## Milestone 3 — Update `TransitionDialog.vue` to use allowed targets

### Description
Replace the hardcoded `STATUSES` array with a dynamic list fetched from the
backend on dialog open. Show a loading state while fetching. If the list is
empty, display an informative message ("No transitions available for your
role").

### Files to change
- `web/src/components/artifact/TransitionDialog.vue`

### Implementation detail
- On mount (or when props change), call `getAllowedTargets`.
- Replace the static `STATUSES` const with a reactive `ref<string[]>([])`.
- Add a `loading` ref; show a spinner or "Loading…" while the fetch is in
  flight.
- If the returned list is empty, show "No transitions available" and disable
  the confirm button.

### Acceptance criteria
- [ ] Dialog only shows statuses the current user is authorised to transition to
- [ ] Product-owner sees all possible target statuses
- [ ] A developer role sees only the transitions they are authorised for
- [ ] Loading state is displayed while fetching
- [ ] Empty state ("No transitions available") renders when the list is empty
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass

---

## Milestone 4 — Improve error message on transition failure

### Description
Even with filtered targets, the backend may still reject a transition (e.g.
plan gate not satisfied). Improve the error message shown in the dialog to
display the backend's `message` field rather than the generic "Transition
failed" string.

### Files to change
- `web/src/components/artifact/TransitionDialog.vue` — update `catch` block
- `web/src/api/artifacts.ts` — ensure `transitionArtifact` propagates the
  backend error message

### Acceptance criteria
- [ ] On 403, the dialog shows the backend's human-readable message (e.g. "required plans are not yet approved")
- [ ] On 409 (gate_not_ready), the dialog shows the missing plan types
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass

---

## Cross-links

- [[product-owner-cannot-transition-8-be]] — backend bypass logic and `AllowedTargets` method
- [[product-owner-cannot-transition-10-test]] — integration tests for the new endpoint and dialog behaviour
