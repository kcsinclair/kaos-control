---
title: "DELETE /releases/:id returns 500 instead of 404 for non-existent ID"
type: defect
status: draft
lineage: releases-and-roadmaps
parent: lifecycle/tests/releases-and-roadmaps-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# DELETE /releases/:id returns 500 instead of 404 for non-existent ID

## Reproduction Steps

1. Start the server against an empty project.
2. Send `DELETE /api/p/testproject/releases/99999`.
3. Observe the HTTP response status.

## Expected Behaviour

The handler should return `404 Not Found` when the release ID does not exist in the store.

## Actual Behaviour

The handler returns `500 Internal Server Error` with body:

```json
{"error":{"code":"db_error","message":"release 99999 not found in project \"testproject\""}}
```

Same root cause as the PUT case: `Store.Delete` returns an error for a missing ID and the handler does not check whether the error represents "not found" vs. another DB failure.

## Logs / Output

```
releases_test.go:469: expected status 404, got 500: {"error":{"code":"db_error","message":"release 99999 not found in project \"testproject\""}}
--- FAIL: TestReleases_DeleteNotFound (0.10s)
```
