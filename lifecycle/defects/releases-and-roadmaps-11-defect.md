---
title: "DELETE /releases/:id returns 500 instead of 404 for non-existent ID"
type: defect
status: in-development
lineage: releases-and-roadmaps
parent: lifecycle/tests/releases-and-roadmaps-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# DELETE /releases/:id returns 500 instead of 404 for non-existent ID

Defect -8 (same symptom) was incorrectly marked "abandoned" as a duplicate. The DELETE handler has not been fixed: `TestReleases_DeleteNotFound` continues to fail as of 2026-05-07.

## Reproduction Steps

1. Start the server against an empty project.
2. Send `DELETE /api/p/testproject/releases/99999`.
3. Observe the HTTP response status.

## Expected Behaviour

The handler should return `404 Not Found` when the release ID does not exist in the store, matching the spec and the behaviour now implemented for the `PUT` endpoint (fixed in defect -7).

## Actual Behaviour

The handler returns `500 Internal Server Error`:

```json
{"error":{"code":"db_error","message":"release not found"}}
```

`Store.Delete` returns an error when the row is absent, but the DELETE handler does not inspect the error to distinguish "not found" from other database failures, so all errors surface as 500.

## Logs / Output

```
releases_test.go:469: expected status 404, got 500: {"error":{"code":"db_error","message":"release not found"}}
--- FAIL: TestReleases_DeleteNotFound (0.13s)
FAIL    github.com/kaos-control/kaos-control/tests/integration  0.441s
```
