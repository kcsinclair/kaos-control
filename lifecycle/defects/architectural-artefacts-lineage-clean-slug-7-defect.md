---
created: "2026-08-15T11:32:40+10:00"
title: Clean-slug architecture artifact returns 404 after startup scan
type: defect
status: done
lineage: architectural-artefacts
parent: lifecycle/tests/architectural-artefacts-6-test.md
labels:
    - defect
release: KC-Release5
assignees:
    - role: backend-developer
      who: agent
---

# Clean-slug architecture artifact returns 404 after startup scan

Integration test `TestArchitectureLineage_CleanSlugIndexesWithoutError` fails because a pre-seeded clean-slug architecture artifact (`lifecycle/architecture/postgres-modular-monolith.md`) present on disk before startup is not indexed by `index.Index.Scan`. When queried via `GET /api/p/testproject/artifacts/lifecycle/architecture/postgres-modular-monolith.md`, the server returns a 404 Not Found response.

## Reproduction Steps

1. Run the integration test from the repository root:
   ```bash
   go test -tags integration -v ./tests/integration/ -run '^TestArchitectureLineage_CleanSlugIndexesWithoutError$'
   ```
2. Observe 404 response on `GET /api/p/testproject/artifacts/lifecycle/architecture/postgres-modular-monolith.md`.

## Expected Behaviour

The pre-seeded clean-slug architecture file should be indexed during the startup scan with `index == 0` and no parse/lineage errors, returning HTTP 200 with the artifact JSON payload on `GET /api/p/testproject/artifacts/...`.

## Actual Behaviour

The file is not indexed at startup because `lifecycle/architecture/` is omitted from the scan. `GET /api/p/testproject/artifacts/lifecycle/architecture/postgres-modular-monolith.md` returns:
```json
{"error":{"code":"not_found","message":"artifact not found"}}
```

## Logs / Output

```
=== RUN   TestArchitectureLineage_CleanSlugIndexesWithoutError
2026/08/14 18:55:47 INFO index schema mismatch or missing — rebuilding from disk db=/var/folders/_9/m30sx2q55bx9rf43z8r6mk540000gn/T/TestArchitectureLineage_CleanSlugIndexesWithoutError3474942630/002/testproject/index.db
2026/08/14 18:55:47 INFO scan complete indexed=0 skipped=0 files=0 duration=0s
2026/08/14 18:55:47 INFO release startup sync: rehydrated project=testproject inserted=0 skipped=0 pruned=0
2026/08/14 18:55:47 INFO kaos-control started addr=127.0.0.1:60761 version=dev
2026/08/14 18:55:47 INFO http method=GET path=/api/health status=200 bytes=28 duration=149.208µs request_id=loki.local/xv4QSomTdc-000001
2026/08/14 18:55:47 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=41.594417ms request_id=loki.local/xv4QSomTdc-000002
2026/08/14 18:55:47 INFO http method=GET path=/api/p/testproject/artifacts/lifecycle/architecture/postgres-modular-monolith.md status=404 bytes=62 duration=210.459µs request_id=loki.local/xv4QSomTdc-000003
    architecture_lineage_test.go:32: expected status 200, got 404: {"error":{"code":"not_found","message":"artifact not found"}}
2026/08/14 18:55:47 INFO SCHEDULER: scheduler stopped
--- FAIL: TestArchitectureLineage_CleanSlugIndexesWithoutError (0.17s)
```
