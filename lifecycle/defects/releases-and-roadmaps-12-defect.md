---
title: "DELETE /releases/:id returns 500 instead of 404 for non-existent release"
type: defect
status: done
lineage: releases-and-roadmaps
parent: lifecycle/tests/releases-and-roadmaps-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
release: KC-Feature-Sprint
---

# DELETE /releases/:id returns 500 instead of 404 for non-existent release

## Reproduction Steps

1. Start the server against an empty test project.
2. Send `DELETE /api/p/testproject/releases/99999` (an ID that does not exist).
3. Observe the HTTP response status code.

## Expected Behaviour

The handler should return `404 Not Found` with a structured error body when the release ID does not exist in the store, consistent with how `GET /releases/:id` handles the same case.

## Actual Behaviour

The handler returns `500 Internal Server Error` with body:

```json
{"error":{"code":"db_error","message":"release not found"}}
```

The `Store.Delete` implementation surfaces a not-found condition as a generic `db_error` and the HTTP handler does not distinguish it from other database errors, so it falls through to the 500 path.

## Logs / Output

```
--- FAIL: TestReleases_DeleteNotFound (0.13s)
    releases_test.go:469: expected status 404, got 500: {"error":{"code":"db_error","message":"release not found"}}
FAIL    github.com/kaos-control/kaos-control/tests/integration  0.440s
```

Full test run:

```
DELETE /api/p/testproject/releases/99999 status=500 bytes=60
```
