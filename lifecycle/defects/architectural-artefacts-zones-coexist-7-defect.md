---
created: "2026-08-15T11:32:40+10:00"
title: Catalog and project-own architecture artefacts fail to index on server startup
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

# Catalog and project-own architecture artefacts fail to index on server startup

Integration test `TestArchitectureZones_CoexistAndIndex` fails because pre-seeded files in both the catalog zone (`lifecycle/architecture/architectures/`, `lifecycle/architecture/tech-stacks/`) and the project-own zone (`lifecycle/architecture/architecture-summary.md`, `lifecycle/architecture/decisions/`, `lifecycle/architecture/standards/`) are not discovered during initial server startup scan (`index.Index.Scan`).

## Reproduction Steps

1. Run the integration test from the repository root:
   ```bash
   go test -tags integration -v ./tests/integration/ -run '^TestArchitectureZones_CoexistAndIndex$'
   ```
2. Observe test failure when verifying presence of catalog and project-own artefacts in `GET /api/p/testproject/artifacts?limit=0`.

## Expected Behaviour

All pre-seeded architecture artefacts from both the catalog zone (`zone-arch.md`, `zone-stack.md`) and project-own zone (`architecture-summary.md`, `adr-0001-zone.md`, `zone-standard.md`) should be indexed on startup and returned in the `/artifacts` list, while `lifecycle/architecture/README.md` is ignored per the ignore-readme rule.

## Actual Behaviour

None of the pre-seeded architecture artefacts are indexed on startup because `lifecycle/architecture/` is not walked during `index.Index.Scan`.

## Logs / Output

```
=== RUN   TestArchitectureZones_CoexistAndIndex
2026/08/14 18:55:43 INFO index schema mismatch or missing — rebuilding from disk db=/var/folders/_9/m30sx2q55bx9rf43z8r6mk540000gn/T/TestArchitectureZones_CoexistAndIndex3286005572/002/testproject/index.db
2026/08/14 18:55:43 INFO scan complete indexed=0 skipped=0 files=0 duration=0s
2026/08/14 18:55:43 INFO release startup sync: rehydrated project=testproject inserted=0 skipped=0 pruned=0
2026/08/14 18:55:43 INFO kaos-control started addr=127.0.0.1:60746 version=dev
2026/08/14 18:55:43 INFO http method=GET path=/api/health status=200 bytes=28 duration=131.959µs request_id=loki.local/kvblmw9NPU-000001
2026/08/14 18:55:43 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=37.631458ms request_id=loki.local/kvblmw9NPU-000002
2026/08/14 18:55:43 INFO http method=GET path=/api/p/testproject/artifacts status=200 bytes=42 duration=418.292µs request_id=loki.local/kvblmw9NPU-000003
    architecture_zones_test.go:53: expected "lifecycle/architecture/architectures/zone-arch.md" to be indexed, not found in /artifacts
    architecture_zones_test.go:53: expected "lifecycle/architecture/tech-stacks/zone-stack.md" to be indexed, not found in /artifacts
    architecture_zones_test.go:53: expected "lifecycle/architecture/architecture-summary.md" to be indexed, not found in /artifacts
    architecture_zones_test.go:53: expected "lifecycle/architecture/decisions/adr-0001-zone.md" to be indexed, not found in /artifacts
    architecture_zones_test.go:53: expected "lifecycle/architecture/standards/zone-standard.md" to be indexed, not found in /artifacts
2026/08/14 18:55:43 INFO SCHEDULER: scheduler stopped
--- FAIL: TestArchitectureZones_CoexistAndIndex (0.18s)
```
