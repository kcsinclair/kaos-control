---
title: "Test Plan — Inline Status Transition Dropdown"
type: plan-test
status: in-development
lineage: artefact-inline-status-change
parent: lifecycle/requirements/artefact-inline-status-change-2.md
---

# Test Plan — Inline Status Transition Dropdown

Integration and end-to-end tests for the inline status transition dropdown feature. Tests verify the full stack: API call → backend transition → WebSocket broadcast → UI update.

Cross-references: [[artefact-inline-status-change]] (backend plan), [[artefact-inline-status-change]] (frontend plan).

---

## Milestone 1 — API integration tests for `allowed-targets`

### Description

Test the `GET /api/p/:project/artifacts/{path}/allowed-targets` endpoint with various user roles and artifact statuses. Verify that the response is role-filtered and that `product-owner` receives the full set.

### Files to change

- `tests/transition_allowed_targets_test.go` — new file (or extend existing transition test file if one exists)

### Acceptance criteria

- [ ] Test: a `draft` artifact returns valid targets (e.g., `clarifying`) for an `analyst` role user.
- [ ] Test: a `draft` artifact returns all reachable targets for a `product-owner` role user.
- [ ] Test: a user with no matching roles receives an empty `targets` array.
- [ ] Test: a non-existent artifact path returns 404.
- [ ] Test: an unauthenticated request returns 401.
- [ ] All tests pass with `go test ./tests/... -run TestAllowedTargets`.

---

## Milestone 2 — API integration tests for transition execution

### Description

Test the `POST /api/p/:project/artifacts/{path}/transition` endpoint for success, role-forbidden, and invalid-target cases.

### Files to change

- `tests/transition_execute_test.go` — new file (or extend existing)

### Acceptance criteria

- [ ] Test: transitioning a `draft` artifact to `clarifying` with an `analyst` user succeeds (200), returns updated `ArtifactRow` with `status: "clarifying"`.
- [ ] Test: transitioning with an unauthorised role returns 403 with `error.code == "forbidden"` and an `allowed_targets` hint.
- [ ] Test: transitioning to an invalid target status (not in the workflow graph) returns 403.
- [ ] Test: transitioning a non-existent artifact returns 404.
- [ ] Test: the artifact file on disk has its `status:` frontmatter field updated after a successful transition.
- [ ] Test: a git commit is created with the expected message format `transition(<lineage>): <from> → <to>`.
- [ ] All tests pass with `go test ./tests/... -run TestTransitionExecute`.

---

## Milestone 3 — WebSocket broadcast tests

### Description

Test that a successful transition broadcasts an `artifact.indexed` WebSocket event with the expected payload shape.

### Files to change

- `tests/transition_ws_test.go` — new file

### Acceptance criteria

- [ ] Test: connect a WebSocket client before transitioning. After `POST .../transition` succeeds, the WS client receives an `artifact.indexed` event with `payload.path`, `payload.action == "transitioned"`, `payload.from`, and `payload.to`.
- [ ] Test: the `feed.new` event is also received with `event_type == "status_transition"`.
- [ ] Tests clean up WebSocket connections after completion.
- [ ] All tests pass with `go test ./tests/... -run TestTransitionWebSocket`.

---

## Milestone 4 — Product-owner override and multi-role tests

### Description

Test the product-owner bypass behaviour and users who hold multiple roles.

### Files to change

- `tests/transition_roles_test.go` — new file

### Acceptance criteria

- [ ] Test: a `product-owner` can transition between any two statuses, including those normally restricted (e.g., `draft` → `approved`).
- [ ] Test: a user with roles `[analyst, backend-developer]` can perform transitions allowed by either role (union behaviour).
- [ ] Test: a user with roles `[analyst, backend-developer]` cannot perform transitions allowed by neither role.
- [ ] Test: `GET .../allowed-targets` for a multi-role user returns the union of targets reachable by any of their roles.
- [ ] All tests pass with `go test ./tests/... -run TestTransitionRoles`.

---

## Milestone 5 — Error resilience and edge case tests

### Description

Test error handling, concurrent transitions, and edge cases.

### Files to change

- `tests/transition_edge_cases_test.go` — new file

### Acceptance criteria

- [ ] Test: two concurrent transition requests on the same artifact — the first succeeds, the second fails with 403 (the `from` status has changed).
- [ ] Test: transitioning an artifact that was deleted between the request and file write returns an appropriate error (404 or 500, not a panic).
- [ ] Test: the `required_plans` gate blocks `planning → in-development` for non-product-owner users when approved plans are missing (409 with `gate_not_ready`).
- [ ] Test: a `product-owner` can bypass the `required_plans` gate.
- [ ] All tests pass with `go test ./tests/... -run TestTransitionEdgeCases`.

---

## Milestone 6 — Companion test artifact

### Description

Write the companion `test` artifact in `lifecycle/tests/` documenting the test suite.

### Files to change

- `lifecycle/tests/artefact-inline-status-change-6-test.md` — new file with `type: test` frontmatter

### Acceptance criteria

- [ ] Artifact has correct frontmatter: `type: test`, `status: draft`, `lineage: artefact-inline-status-change`, `parent: lifecycle/test-plans/artefact-inline-status-change-5-test.md`.
- [ ] Body summarises the scenarios covered and lists the test files in `tests/`.
