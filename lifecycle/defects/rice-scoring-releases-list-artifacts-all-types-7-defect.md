---
title: Release artifacts list query fails with Scan destination argument mismatch in TestReleases_ListArtifactsReturnsAllTypes
type: defect
status: in-development
lineage: rice-scoring
parent: lifecycle/tests/rice-scoring-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# Release artifacts list query fails with Scan destination argument mismatch in TestReleases_ListArtifactsReturnsAllTypes

Integration test `TestReleases_ListArtifactsReturnsAllTypes` fails when calling `GET /api/p/{project}/releases/{id}/artifacts` because the SQL query in `internal/release/store.go` selects 12 columns while `index.ScanArtifactRows` expects 13 columns (including `rice_score`).

## Reproduction Steps

1. Run the integration test from the repository root:
   ```bash
   go test -tags integration -v ./tests/integration -run '^TestReleases_ListArtifactsReturnsAllTypes$'
   ```
2. The test seeds artifacts of various types (ideas, requirements, test plans, defects) assigned to release 1.
3. The test calls `GET /api/p/testproject/releases/1/artifacts` to verify all artifact types are returned.
4. Observe HTTP 500 response from the endpoint.

## Expected Behaviour

`GET /api/p/testproject/releases/1/artifacts` should return HTTP 200 with all artifacts of all types assigned to the release.

## Actual Behaviour

The endpoint returns HTTP 500 with error:
```json
{"error":{"code":"db_error","message":"sql: expected 12 destination arguments in Scan, not 13"}}
```

Root cause: `internal/release/store.go:380` (`ListArtifactsForRelease`) executes a query with a 12-column SELECT list (`path, slug, lineage, idx, stage, type, status, title, frontmatter_json, mtime, created, rel_path`), but passes `rows` to `index.ScanArtifactRows` in `internal/index/index.go:2556`, which scans 13 columns including `rice_score`.

## Logs / Output

```
=== RUN   TestReleases_ListArtifactsReturnsAllTypes
2026/08/15 11:57:16 INFO index schema mismatch or missing — rebuilding from disk db=/var/folders/_9/m30sx2q55bx9rf43z8r6mk540000gn/T/TestReleases_ListArtifactsReturnsAllTypes4037081680/002/testproject/index.db
2026/08/15 11:57:16 INFO scan complete indexed=4 skipped=0 files=4 duration=2ms
2026/08/15 11:57:16 INFO release startup sync: rehydrated project=testproject inserted=0 skipped=0 pruned=0
2026/08/15 11:57:16 INFO kaos-control started addr=127.0.0.1:65358 version=dev
2026/08/15 11:57:16 INFO http method=GET path=/api/health status=200 bytes=28 duration=31.291µs request_id=loki.local/BLCzXhgWUS-000013
2026/08/15 11:57:16 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=31.326333ms request_id=loki.local/BLCzXhgWUS-000014
2026/08/15 11:57:16 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=30.199375ms request_id=loki.local/BLCzXhgWUS-000015
2026/08/15 11:57:16 INFO http method=POST path=/api/p/testproject/releases status=201 bytes=236 duration=917.584µs request_id=loki.local/BLCzXhgWUS-000016
2026/08/15 11:57:16 INFO http method=GET path=/api/p/testproject/releases/1/artifacts status=500 bytes=97 duration=230.25µs request_id=loki.local/BLCzXhgWUS-000017
    releases_test.go:574: expected status 200, got 500: {"error":{"code":"db_error","message":"sql: expected 12 destination arguments in Scan, not 13"}}
2026/08/15 11:57:16 INFO SCHEDULER: scheduler stopped
--- FAIL: TestReleases_ListArtifactsReturnsAllTypes (0.18s)
```
