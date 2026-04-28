---
title: "Test Plan: Frontmatter Role-Based Assignment Control"
type: plan-test
status: done
lineage: frontmatter-role-assignment
parent: lifecycle/requirements/frontmatter-role-assignment-2.md
---

# Test Plan: Frontmatter Role-Based Assignment Control

Integration tests for the roles API endpoint and assignee round-trip persistence added by [[frontmatter-role-assignment]]. These tests follow the existing Go integration test pattern in `tests/integration/` using the `testEnv` helper and the `//go:build integration` build tag.

## Milestone 1 — GET /roles endpoint tests

### Description

Test that the new `GET /api/p/{project}/roles` endpoint returns the correct roles and users from the project's `lifecycle/config.yaml`.

### Files to change

- `tests/integration/roles_api_test.go` — New file. Add build tag `//go:build integration`. Tests:
  1. **TestGetRoles_ReturnsConfiguredRoles** — Seed a test project with a `lifecycle/config.yaml` containing known `roles` and `users` entries. `GET /api/p/{project}/roles` → assert `200`, assert response JSON `roles` array matches the seeded config, assert `users` array contains expected email/role bindings.
  2. **TestGetRoles_EmptyUsers** — Seed a config with roles but no `users` list. Assert `roles` is populated, `users` is an empty array (not null).
  3. **TestGetRoles_Unauthenticated** — If auth is enabled in the test env, assert `GET /roles` without a session returns `401`.

### Acceptance criteria

- [ ] All three tests pass with `go test -tags integration ./tests/integration/ -run TestGetRoles`.
- [ ] Tests use the `testEnv` helper for isolated project setup and teardown.
- [ ] `go vet ./...` passes.

## Milestone 2 — Assignee round-trip persistence tests

### Description

Test that assignees written via `PUT /artifacts/*` are correctly persisted in the file's YAML frontmatter and returned on subsequent `GET`.

### Files to change

- `tests/integration/assignee_persistence_test.go` — New file. Add build tag `//go:build integration`. Tests:
  1. **TestPutArtifact_AssigneesRoundTrip** — Create a seed artifact with no assignees. `PUT` with `assignees: [{role: "backend-developer", who: "agent"}]`. Re-read the file from disk and parse YAML frontmatter → assert `assignees` array matches. Also `GET` the artifact via API → assert response includes the assignees.
  2. **TestPutArtifact_RemoveAssignees** — Seed an artifact that already has assignees. `PUT` with `assignees: []`. Assert the file's frontmatter no longer contains assignee entries.
  3. **TestPutArtifact_MultipleAssignees** — `PUT` with two assignee entries (different roles). Assert both are persisted in order.
  4. **TestPutArtifact_InvalidRole** — `PUT` with `assignees: [{role: "nonexistent-role", who: "agent"}]`. Assert `400` response with an error message naming the invalid role (depends on backend plan Milestone 2).
  5. **TestPutArtifact_EmptyRoleOrWho** — `PUT` with `assignees: [{role: "", who: "agent"}]` and `[{role: "qa", who: ""}]`. Assert `400` for each.

### Acceptance criteria

- [ ] All five tests pass with `go test -tags integration ./tests/integration/ -run TestPutArtifact_Assignee`.
- [ ] Tests verify both the API response and the on-disk file content.
- [ ] Tests use `testEnv` for isolated project setup.
- [ ] `go vet ./...` passes.

## Milestone 3 — Test artifact documentation

### Description

Write a companion test artifact in `lifecycle/tests/` documenting the test coverage for this feature.

### Files to change

- `lifecycle/tests/frontmatter-role-assignment-6-test.md` — New file with frontmatter:
  ```yaml
  title: "Tests: Frontmatter Role-Based Assignment"
  type: test
  status: draft
  lineage: frontmatter-role-assignment
  parent: lifecycle/test-plans/frontmatter-role-assignment-5-test.md
  ```
  Body summarises the scenarios covered (roles endpoint, assignee round-trip, validation) and points to the test files in `tests/integration/`.

### Acceptance criteria

- [ ] The test artifact exists at the correct path with valid frontmatter.
- [ ] The body lists all test scenarios from Milestones 1 and 2.
- [ ] The lineage index (6) is the next unused index in the `frontmatter-role-assignment` lineage.
