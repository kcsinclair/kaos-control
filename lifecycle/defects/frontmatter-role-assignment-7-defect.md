---
title: "PUT /artifacts accepts assignee with empty `who` field (no 400 returned)"
type: defect
status: done
lineage: frontmatter-role-assignment
parent: lifecycle/tests/frontmatter-role-assignment-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# PUT /artifacts accepts assignee with empty `who` field (no 400 returned)

## Reproduction Steps

1. Start the server with a project config that includes the `qa` role.
2. Authenticate as admin (`POST /api/auth/login`).
3. Seed an artifact at `lifecycle/ideas/assign-empty-who.md` with no assignees.
4. Send `PUT /api/p/testproject/artifacts/lifecycle/ideas/assign-empty-who.md` with body:
   ```json
   {
     "assignees": [{"role": "qa", "who": ""}]
   }
   ```
5. Observe the HTTP response status.

## Expected Behaviour

The server returns `400 Bad Request` because the `who` field is empty. An assignee entry must have both a non-empty `role` and a non-empty `who` value.

## Actual Behaviour

The server returns `200 OK` and persists the assignee with an empty `who` on disk:

```json
{"artifact":{"path":"lifecycle/ideas/assign-empty-who.md","assignees":[{"role":"qa","who":""}],...}}
```

The `who` validation is missing; only `role` emptiness is validated.

## Logs / Output

```
2026/04/28 12:39:12 INFO http method=PUT path=/api/p/testproject/artifacts/lifecycle/ideas/assign-empty-who.md status=200 bytes=413 duration=4.046541ms

    assignee_persistence_test.go:328: expected status 400, got 200: {"artifact":{"path":"lifecycle/ideas/assign-empty-who.md","slug":"assign-empty-who","lineage":"assign-empty-who","index":0,"stage":"ideas","type":"idea","status":"draft","title":"Assign Empty Who","frontmatter":{"title":"Assign Empty Who","type":"idea","status":"draft","lineage":"assign-empty-who","assignees":[{"role":"qa","who":""}]},"mtime":"2026-04-28T12:39:12+10:00","created":"2026-04-28T12:39:12+10:00"}}

--- FAIL: TestPutArtifact_EmptyRoleOrWho (0.12s)
FAIL    github.com/kaos-control/kaos-control/tests/integration  1.530s
```

## Test run summary

| Test | Result |
|---|---|
| TestGetRoles_ReturnsConfiguredRoles | PASS |
| TestGetRoles_EmptyUsers | PASS |
| TestGetRoles_Unauthenticated | PASS |
| TestPutArtifact_AssigneesRoundTrip | PASS |
| TestPutArtifact_RemoveAssignees | PASS |
| TestPutArtifact_MultipleAssignees | PASS |
| TestPutArtifact_InvalidRole | PASS |
| TestPutArtifact_EmptyRoleOrWho | **FAIL** (empty `who` sub-case) |
