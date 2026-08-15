---
title: Release artifacts query fails with Scan destination argument mismatch in TestReleases_ListArtifactsForRelease
type: defect
status: draft
lineage: rice-scoring
parent: lifecycle/tests/rice-scoring-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# Release artifacts query fails with Scan destination argument mismatch in TestReleases_ListArtifactsForRelease

Integration test `TestReleases_ListArtifactsForRelease` fails when querying `GET /api/p/{project}/releases/{id}/artifacts` because the SQL query in `internal/release/store.go` selects 12 columns while `index.ScanArtifactRows` expects 13 columns (including `rice_score`).

## Reproduction Steps

1. Run the integration test from the repository root:
   ```bash
   go test -tags integration -v ./tests/integration -run '^TestReleases_ListArtifactsForRelease$'
   ```
2. The test creates a release via `POST /api/p/testproject/releases` and assigns artifacts to that release.
3. The test calls `GET /api/p/testproject/releases/1/artifacts` to verify the list of artifacts in the release.
4. Observe HTTP 500 response from the endpoint.

## Expected Behaviour

`GET /api/p/testproject/releases/1/artifacts` should return HTTP 200 with a JSON array of `ArtifactRow` objects assigned to release 1, including `rice_score`.

## Actual Behaviour

The endpoint returns HTTP 500 with error:
```json
{"error":{"code":"db_error","message":"sql: expected 12 destination arguments in Scan, not 13"}}
```

Root cause: In `internal/release/store.go` (`ListArtifactsForRelease`), the SQL query selects:
```sql
SELECT path, slug, lineage, idx, stage, type, status, title, frontmatter_json, mtime, created, rel_path
FROM artifacts
WHERE json_extract(frontmatter_json, '$.release') = ?
ORDER BY lineage, idx, path
```
which projects 12 columns. However, `index.ScanArtifactRows` in `internal/index/index.go` was updated to scan 13 columns (`&r.Path, &r.Slug, &r.Lineage, &r.Index, &r.Stage, &r.Type, &r.Status, &r.Title, &fmJSON, &mtimeUnix, &createdUnix, &r.RelPath, &riceScore`).

## Logs / Output

```
=== RUN   TestReleases_ListArtifactsForRelease
2026/08/15 11:57:16 INFO index schema mismatch or missing — rebuilding from disk db=/var/folders/_9/m30sx2q55bx9rf43z8r6mk540000gn/T/TestReleases_ListArtifactsForRelease3397029191/002/testproject/index.db
2026/08/15 11:57:16 INFO scan complete indexed=3 skipped=0 files=3 duration=2ms
2026/08/15 11:57:16 INFO release startup sync: rehydrated project=testproject inserted=0 skipped=0 pruned=0
2026/08/15 11:57:16 INFO kaos-control started addr=127.0.0.1:65353 version=dev
2026/08/15 11:57:16 INFO http method=GET path=/api/health status=200 bytes=28 duration=28.542µs request_id=loki.local/BLCzXhgWUS-000008
2026/08/15 11:57:16 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=41.873334ms request_id=loki.local/BLCzXhgWUS-000009
2026/08/15 11:57:16 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=34.406208ms request_id=loki.local/BLCzXhgWUS-000010
2026/08/15 11:57:16 INFO http method=POST path=/api/p/testproject/releases status=201 bytes=236 duration=951.416µs request_id=loki.local/BLCzXhgWUS-000011
2026/08/15 11:57:16 INFO http method=GET path=/api/p/testproject/releases/1/artifacts status=500 bytes=97 duration=312.5µs request_id=loki.local/BLCzXhgWUS-000012
    releases_test.go:500: expected status 200, got 500: {"error":{"code":"db_error","message":"sql: expected 12 destination arguments in Scan, not 13"}}
2026/08/15 11:57:16 INFO SCHEDULER: scheduler stopped
--- FAIL: TestReleases_ListArtifactsForRelease (0.20s)
```
