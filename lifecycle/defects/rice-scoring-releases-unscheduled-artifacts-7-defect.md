---
title: Release artifacts query fails with Scan destination argument mismatch in TestReleaseUnscheduled_ArtifactAssignment
type: defect
status: draft
lineage: rice-scoring
parent: lifecycle/tests/rice-scoring-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# Release artifacts query fails with Scan destination argument mismatch in TestReleaseUnscheduled_ArtifactAssignment

Integration test `TestReleaseUnscheduled_ArtifactAssignment` fails when retrieving `GET /api/p/{project}/releases/{id}/artifacts` because the SQL query in `internal/release/store.go` selects 12 columns while `index.ScanArtifactRows` expects 13 columns (including `rice_score`).

## Reproduction Steps

1. Run the integration test from the repository root:
   ```bash
   go test -tags integration -v ./tests/integration -run '^TestReleaseUnscheduled_ArtifactAssignment$'
   ```
2. The test creates a release via `POST /api/p/testproject/releases` and assigns artifacts to that release.
3. The test calls `GET /api/p/testproject/releases/1/artifacts` to verify assigned artifacts.
4. Observe HTTP 500 response from the endpoint.

## Expected Behaviour

`GET /api/p/testproject/releases/1/artifacts` should return HTTP 200 with the list of artifacts assigned to release 1.

## Actual Behaviour

The endpoint returns HTTP 500 with error:
```json
{"error":{"code":"db_error","message":"sql: expected 12 destination arguments in Scan, not 13"}}
```

Root cause: `internal/release/store.go:380` (`ListArtifactsForRelease`) performs `SELECT path, slug, lineage, idx, stage, type, status, title, frontmatter_json, mtime, created, rel_path FROM artifacts ...` projecting 12 columns, while `index.ScanArtifactRows` expects 13 columns (including `rice_score`).

## Logs / Output

```
=== RUN   TestReleaseUnscheduled_ArtifactAssignment
2026/08/15 11:57:16 INFO index schema mismatch or missing — rebuilding from disk db=/var/folders/_9/m30sx2q55bx9rf43z8r6mk540000gn/T/TestReleaseUnscheduled_ArtifactAssignment680534024/002/testproject/index.db
2026/08/15 11:57:16 INFO scan complete indexed=2 skipped=0 files=2 duration=1ms
2026/08/15 11:57:16 INFO release startup sync: rehydrated project=testproject inserted=0 skipped=0 pruned=0
2026/08/15 11:57:16 INFO kaos-control started addr=127.0.0.1:65366 version=dev
2026/08/15 11:57:16 INFO http method=GET path=/api/health status=200 bytes=28 duration=27.875µs request_id=loki.local/BLCzXhgWUS-000018
2026/08/15 11:57:16 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=31.344083ms request_id=loki.local/BLCzXhgWUS-000019
2026/08/15 11:57:16 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=29.955416ms request_id=loki.local/BLCzXhgWUS-000020
2026/08/15 11:57:16 INFO http method=POST path=/api/p/testproject/releases status=201 bytes=236 duration=1.036458ms request_id=loki.local/BLCzXhgWUS-000021
2026/08/15 11:57:16 INFO http method=GET path=/api/p/testproject/releases/1/artifacts status=500 bytes=97 duration=278.584µs request_id=loki.local/BLCzXhgWUS-000022
    releases_unscheduled_test.go:117: expected status 200, got 500: {"error":{"code":"db_error","message":"sql: expected 12 destination arguments in Scan, not 13"}}
2026/08/15 11:57:16 INFO SCHEDULER: scheduler stopped
--- FAIL: TestReleaseUnscheduled_ArtifactAssignment (0.17s)
```
