---
title: "PUT /releases/:id returns 500 instead of 404 for non-existent ID"
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

# PUT /releases/:id returns 500 instead of 404 for non-existent ID

## Reproduction Steps

1. Start the server against an empty project.
2. Send `PUT /api/p/testproject/releases/99999` with a valid JSON body (e.g. `{"name":"x","status":"planned"}`).
3. Observe the HTTP response status.

## Expected Behaviour

The handler should return `404 Not Found` when the release ID does not exist in the store.

## Actual Behaviour

The handler returns `500 Internal Server Error` with body:

```json
{"error":{"code":"db_error","message":"release 99999 not found in project \"testproject\""}}
```

The "not found" error from `Store.Update` is not distinguished from other DB errors, so all errors propagate as 500.

## Logs / Output

```
releases_test.go:398: expected status 404, got 500: {"error":{"code":"db_error","message":"release 99999 not found in project \"testproject\""}}
--- FAIL: TestReleases_UpdateNotFound (0.11s)
```
