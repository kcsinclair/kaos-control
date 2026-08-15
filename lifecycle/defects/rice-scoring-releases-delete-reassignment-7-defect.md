---
title: Release delete reassignment verification fails with Scan destination argument mismatch in TestReleaseDelete_WithReassignment
type: defect
status: done
lineage: rice-scoring
parent: lifecycle/tests/rice-scoring-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# Release delete reassignment verification fails with Scan destination argument mismatch in TestReleaseDelete_WithReassignment

Integration test `TestReleaseDelete_WithReassignment` fails when checking reassigned artifacts via `GET /api/p/{project}/releases/{id}/artifacts` because the SQL query in `internal/release/store.go` selects 12 columns while `index.ScanArtifactRows` expects 13 columns (including `rice_score`).

## Reproduction Steps

1. Run the integration test from the repository root:
   ```bash
   go test -tags integration -v ./tests/integration -run '^TestReleaseDelete_WithReassignment$'
   ```
2. The test creates release 1 and release 2, assigns artifacts to release 1, and deletes release 1 with reassignment parameter `reassign_to: 2`.
3. The test queries `GET /api/p/testproject/releases/2/artifacts` to verify the reassigned artifacts.
4. Observe HTTP 500 response from the endpoint.

## Expected Behaviour

`GET /api/p/testproject/releases/2/artifacts` should return HTTP 200 with the list of reassigned artifacts in release 2.

## Actual Behaviour

The endpoint returns HTTP 500 with error:
```json
{"error":{"code":"db_error","message":"sql: expected 12 destination arguments in Scan, not 13"}}
```

Root cause: `internal/release/store.go:380` (`ListArtifactsForRelease`) performs `SELECT path, slug, lineage, idx, stage, type, status, title, frontmatter_json, mtime, created, rel_path FROM artifacts ...` projecting 12 columns, while `index.ScanArtifactRows` in `internal/index/index.go:2556` expects 13 columns (including `rice_score`).

## Logs / Output

```
=== RUN   TestReleaseDelete_WithReassignment
2026/08/15 11:55:08 INFO index schema mismatch or missing — rebuilding from disk db=/var/folders/_9/m30sx2q55bx9rf43z8r6mk540000gn/T/TestReleaseDelete_WithReassignment2644041612/002/testproject/index.db
2026/08/15 11:55:08 INFO scan complete indexed=3 skipped=0 files=3 duration=1ms
2026/08/15 11:55:08 INFO release startup sync: rehydrated project=testproject inserted=0 skipped=0 pruned=0
2026/08/15 11:55:08 INFO kaos-control started addr=127.0.0.1:63222 version=dev
2026/08/15 11:55:08 INFO http method=GET path=/api/health status=200 bytes=28 duration=67.375µs request_id=loki.local/h9FoMkeUsG-003208
2026/08/15 11:55:08 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=32.530708ms request_id=loki.local/h9FoMkeUsG-003209
2026/08/15 11:55:08 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=33.735125ms request_id=loki.local/h9FoMkeUsG-003210
2026/08/15 11:55:08 INFO http method=POST path=/api/p/testproject/releases status=201 bytes=230 duration=1.1125ms request_id=loki.local/h9FoMkeUsG-003211
2026/08/15 11:55:08 INFO http method=POST path=/api/p/testproject/releases status=201 bytes=229 duration=736.375µs request_id=loki.local/h9FoMkeUsG-003212
2026/08/15 11:55:08 INFO http method=DELETE path=/api/p/testproject/releases/1 status=200 bytes=30 duration=10.053583ms request_id=loki.local/h9FoMkeUsG-003213
2026/08/15 11:55:08 INFO http method=GET path=/api/p/testproject/releases/2/artifacts status=500 bytes=97 duration=230.083µs request_id=loki.local/h9FoMkeUsG-003214
2026/08/15 11:55:08 INFO SCHEDULER: scheduler stopped
--- FAIL: TestReleaseDelete_WithReassignment (0.19s)
    releases_delete_test.go:116: expected status 200, got 500: {"error":{"code":"db_error","message":"sql: expected 12 destination arguments in Scan, not 13"}}
```
