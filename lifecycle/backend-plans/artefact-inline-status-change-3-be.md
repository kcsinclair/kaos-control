---
title: "Backend Plan — Inline Status Transition Dropdown"
type: plan-backend
status: approved
lineage: artefact-inline-status-change
parent: lifecycle/requirements/artefact-inline-status-change-2.md
---

# Backend Plan — Inline Status Transition Dropdown

This plan covers backend changes required to support the inline status transition dropdown on the artefact detail view. The existing `POST /api/p/:project/artifacts/*/transition` and `GET .../allowed-targets` endpoints already satisfy the core API contract. The work here is limited to exposing the allowed-targets API function to the frontend (it exists in the backend but has no corresponding frontend client function) and ensuring the WebSocket `artifact.indexed` payload carries all fields needed for live badge updates.

Cross-references: [[artefact-inline-status-change]] (frontend plan for dropdown UI), [[artefact-inline-status-change]] (test plan for integration tests).

---

## Milestone 1 — Verify and document existing endpoint contracts

### Description

Confirm that the existing `GET /api/p/:project/artifacts/{path}/allowed-targets` and `POST /api/p/:project/artifacts/{path}/transition` endpoints meet every functional requirement in the spec. No code changes are expected; this milestone is a review gate.

### Files to review

- `internal/http/transition.go` — `handleAllowedTargets` (line ~22) and `handleTransitionArtifact` (line ~49)
- `internal/workflow/workflow.go` — `AllowedTargets()`, `CanTransition()`, product-owner bypass
- `internal/http/router.go` — route registration for the two endpoints

### Acceptance criteria

- [ ] `GET .../allowed-targets` returns `{ "targets": [...] }` filtered by the authenticated user's roles.
- [ ] `product-owner` role receives the full set of reachable target statuses (bypass behaviour confirmed).
- [ ] `POST .../transition` with `{ "to": "<status>" }` patches the file, re-indexes, commits, and broadcasts the WS event.
- [ ] A 403 is returned with `{ "error": { "code": "forbidden", ... }, "allowed_targets": [...] }` when the user lacks the required role.
- [ ] A 404 is returned when the artifact path does not exist.
- [ ] No code changes are committed for this milestone — it is a verification-only gate.

---

## Milestone 2 — Ensure `artifact.indexed` WebSocket payload includes `status`

### Description

The `artifact.indexed` event broadcast after a transition already includes `from` and `to` fields (see `internal/http/transition.go:172`). Verify that the payload is sufficient for the frontend to update the badge without a re-fetch. The payload should include: `path`, `action` ("transitioned"), `from`, `to`.

If any fields are missing or the shape is inconsistent across the three broadcast sites (`applyTransition`, `autoBlock`, `autoUnblock`), normalise them.

### Files to change

- `internal/http/transition.go` — `applyTransition` WS broadcast (only if payload adjustment is needed)
- `internal/index/autoblock.go` — `autoBlock` / `autoUnblock` WS broadcasts (only if payload adjustment is needed)

### Acceptance criteria

- [ ] All `artifact.indexed` events include at minimum `{ "path": string, "action": string }`.
- [ ] Events emitted by `applyTransition` include `"from"` and `"to"` string fields.
- [ ] Events emitted by `autoBlock` include `"to": "blocked"`.
- [ ] Events emitted by `autoUnblock` include `"from": "blocked"` and the resulting target status.
- [ ] `go build ./...` and `go vet ./...` pass.
