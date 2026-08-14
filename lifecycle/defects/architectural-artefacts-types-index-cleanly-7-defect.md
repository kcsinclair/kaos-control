---
title: Pre-seeded architecture, tech-stack, and adr artefacts not indexed on startup scan
type: defect
status: in-development
lineage: architectural-artefacts
parent: lifecycle/tests/architectural-artefacts-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# Pre-seeded architecture, tech-stack, and adr artefacts not indexed on startup scan

Integration test `TestArchitectureTypes_IndexCleanly` fails because the server startup scan (`index.Index.Scan`) only traverses configured project stages (`filepath.Join(lifecycle, stage.Dir)`). Because `architecture` is not listed in `defaultStages` or `lifecycle/config.yaml`, pre-seeded artefacts in `lifecycle/architecture/` (`architectures/tt-arch.md`, `tech-stacks/tt-stack.md`, `decisions/adr-0001-tt.md`) are never indexed at startup and do not appear in `GET /api/p/testproject/artifacts`.

## Reproduction Steps

1. Run the integration test from the repository root:
   ```bash
   go test -tags integration -v ./tests/integration/ -run '^TestArchitectureTypes_IndexCleanly$'
   ```
2. Observe test failure when querying `GET /api/p/testproject/artifacts?limit=0`.

## Expected Behaviour

Pre-seeded artefacts with types `architecture`, `tech-stack`, and `adr` located under `lifecycle/architecture/` should be discovered, parsed, and indexed during server startup scan, appearing in `GET /api/p/testproject/artifacts` with matching `type` fields and zero parse errors.

## Actual Behaviour

The startup scan ignores `lifecycle/architecture/` because it is not a configured stage in `stages:`. As a result, none of the pre-seeded files are indexed at startup and `findArtifactRow` returns `nil` for all three files.

## Logs / Output

```
=== RUN   TestArchitectureTypes_IndexCleanly
2026/08/14 18:55:36 INFO index schema mismatch or missing — rebuilding from disk db=/var/folders/_9/m30sx2q55bx9rf43z8r6mk540000gn/T/TestArchitectureTypes_IndexCleanly1363433893/002/testproject/index.db
2026/08/14 18:55:36 INFO scan complete indexed=0 skipped=0 files=0 duration=0s
2026/08/14 18:55:36 INFO release startup sync: rehydrated project=testproject inserted=0 skipped=0 pruned=0
2026/08/14 18:55:36 INFO kaos-control started addr=127.0.0.1:60714 version=dev
2026/08/14 18:55:36 INFO http method=GET path=/api/health status=200 bytes=28 duration=144.958µs request_id=loki.local/ZZ9oozc7os-000001
2026/08/14 18:55:36 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=42.492542ms request_id=loki.local/ZZ9oozc7os-000002
2026/08/14 18:55:36 INFO http method=GET path=/api/p/testproject/artifacts status=200 bytes=42 duration=439.25µs request_id=loki.local/ZZ9oozc7os-000003
    architecture_types_test.go:85: expected "lifecycle/architecture/architectures/tt-arch.md" in /artifacts response, not found
    architecture_types_test.go:85: expected "lifecycle/architecture/tech-stacks/tt-stack.md" in /artifacts response, not found
    architecture_types_test.go:85: expected "lifecycle/architecture/decisions/adr-0001-tt.md" in /artifacts response, not found
2026/08/14 18:55:36 INFO http method=GET path=/api/p/testproject/parse-errors status=200 bytes=16 duration=167.834µs request_id=loki.local/ZZ9oozc7os-000004
2026/08/14 18:55:36 INFO SCHEDULER: scheduler stopped
--- FAIL: TestArchitectureTypes_IndexCleanly (0.18s)
```
