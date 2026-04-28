---
title: "Tests: Frontmatter Role-Based Assignment"
type: test
status: draft
lineage: frontmatter-role-assignment
parent: lifecycle/test-plans/frontmatter-role-assignment-5-test.md
---

# Tests: Frontmatter Role-Based Assignment

Integration test coverage for the roles API endpoint and assignee round-trip persistence
added by the frontmatter-role-assignment feature.

## Test files

- `tests/integration/roles_api_test.go` — GET /roles endpoint tests
- `tests/integration/assignee_persistence_test.go` — Assignee round-trip persistence tests

## Scenarios covered

### Roles API (roles_api_test.go)

1. **TestGetRoles_ReturnsConfiguredRoles** — Authenticates as admin, calls
   `GET /api/p/testproject/roles`, and asserts the response contains the full configured
   roles array and the correct user/role bindings (admin@test.local, dev@test.local,
   qa@test.local) sourced from `lifecycle/config.yaml`.

2. **TestGetRoles_EmptyUsers** — Seeds a project config with roles but no `users` section,
   then asserts `GET /roles` returns a populated `roles` array and an empty (non-null)
   `users` array. Uses `newTestEnvWithCfgYAML` to start with a custom project config.

3. **TestGetRoles_Unauthenticated** — Calls `GET /roles` without a session cookie and
   asserts the server returns `401 Unauthorized`.

### Assignee persistence (assignee_persistence_test.go)

4. **TestPutArtifact_AssigneesRoundTrip** — Seeds an artifact with no assignees, then
   PUTs `assignees: [{role: backend-developer, who: agent}]`. Verifies the assignee
   appears in the PUT response JSON, in the on-disk YAML frontmatter (parsed via
   `artifact.Parse`), and in the subsequent GET response.

5. **TestPutArtifact_RemoveAssignees** — Seeds an artifact that already has one assignee,
   then PUTs `assignees: []`. Verifies the PUT response and on-disk frontmatter both
   show zero assignees after the update.

6. **TestPutArtifact_MultipleAssignees** — PUTs two assignees (`backend-developer/agent`
   and `qa/human`). Verifies both are persisted in order on disk and returned by a
   subsequent GET.

7. **TestPutArtifact_InvalidRole** — PUTs `assignees: [{role: nonexistent-role, who: agent}]`.
   Asserts a `400` response with error code `invalid_role` and a message that names the
   invalid role.

8. **TestPutArtifact_EmptyRoleOrWho** — Two sub-cases:
   - `role: ""` with a valid `who` → expects `400`.
   - Valid `role: "qa"` with `who: ""` → expects `400`.

## Notes

- `newTestEnvWithCfgYAML` was added to `helpers_test.go` to support `TestGetRoles_EmptyUsers`,
  which requires a project config without a `users` section. The default test config YAML was
  extracted to the `defaultCfgYAML` constant; `newTestEnvFull` now accepts the config as a
  parameter.
- `TestPutArtifact_EmptyRoleOrWho` case 2 (empty `who`) will fail against the current backend
  if `who` validation is not implemented. This is intentional: the test encodes the specified
  behaviour and will surface the missing validation as a defect.
