---
title: Promoted architecture copy parent graph edge missing after startup scan
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

# Promoted architecture copy parent graph edge missing after startup scan

Integration test `TestArchitectureLineage_PromotedCopyParentAcceptedWithEdge` fails because pre-seeded files `lifecycle/architecture/postgres-modular-monolith.md` (promoted copy with `parent:` pointing to catalog) and `lifecycle/architecture/architectures/postgres-modular-monolith.md` (catalog target) are not scanned during startup. Because neither node is inserted into the index database, `GET /api/p/testproject/graph` returns an empty edge set instead of resolving the `parent` relationship edge between them.

## Reproduction Steps

1. Run the integration test from the repository root:
   ```bash
   go test -tags integration -v ./tests/integration/ -run '^TestArchitectureLineage_PromotedCopyParentAcceptedWithEdge$'
   ```
2. Observe failure to find the parent edge in `GET /api/p/testproject/graph`.

## Expected Behaviour

Both the catalog source and the promoted copy should be indexed at startup, with no parse errors for the promoted file's catalog `parent:` reference, and `GET /api/p/testproject/graph` should return a `parent`-kind directed edge from `lifecycle/architecture/postgres-modular-monolith.md` to `lifecycle/architecture/architectures/postgres-modular-monolith.md`.

## Actual Behaviour

Neither file is scanned or indexed at startup. As a result, the graph response contains no edges for the promoted artifact.

## Logs / Output

```
=== RUN   TestArchitectureLineage_PromotedCopyParentAcceptedWithEdge
2026/08/14 18:55:47 INFO index schema mismatch or missing — rebuilding from disk db=/var/folders/_9/m30sx2q55bx9rf43z8r6mk540000gn/T/TestArchitectureLineage_PromotedCopyParentAcceptedWithEdge2784641198/002/testproject/index.db
2026/08/14 18:55:47 INFO scan complete indexed=0 skipped=0 files=0 duration=0s
2026/08/14 18:55:47 INFO release startup sync: rehydrated project=testproject inserted=0 skipped=0 pruned=0
2026/08/14 18:55:47 INFO kaos-control started addr=127.0.0.1:60766 version=dev
2026/08/14 18:55:47 INFO http method=GET path=/api/health status=200 bytes=28 duration=44.834µs request_id=loki.local/xv4QSomTdc-000004
2026/08/14 18:55:47 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=38.566791ms request_id=loki.local/xv4QSomTdc-000005
2026/08/14 18:55:47 INFO http method=GET path=/api/p/testproject/parse-errors status=200 bytes=16 duration=238.292µs request_id=loki.local/xv4QSomTdc-000006
2026/08/14 18:55:47 INFO http method=GET path=/api/p/testproject/graph status=200 bytes=28 duration=365.708µs request_id=loki.local/xv4QSomTdc-000007
    architecture_lineage_test.go:76: expected a parent-kind edge from "lifecycle/architecture/postgres-modular-monolith.md" to "lifecycle/architecture/architectures/postgres-modular-monolith.md", not found in graph response
2026/08/14 18:55:47 INFO SCHEDULER: scheduler stopped
--- FAIL: TestArchitectureLineage_PromotedCopyParentAcceptedWithEdge (0.17s)
```
