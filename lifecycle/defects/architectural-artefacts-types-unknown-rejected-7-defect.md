---
title: Unknown type in architecture artefact not detected during startup scan
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

# Unknown type in architecture artefact not detected during startup scan

Integration test `TestArchitectureTypes_UnknownTypeStillRejected` fails because pre-seeded files in `lifecycle/architecture/` are not scanned during server startup. Consequently, a file with an invalid type (such as `type: bogus` in `lifecycle/architecture/tt-bogus.md`) is never parsed on startup, and no parse error is recorded in `GET /api/p/testproject/parse-errors`.

## Reproduction Steps

1. Run the integration test from the repository root:
   ```bash
   go test -tags integration -v ./tests/integration/ -run '^TestArchitectureTypes_UnknownTypeStillRejected$'
   ```
2. Observe test failure when checking `GET /api/p/testproject/parse-errors`.

## Expected Behaviour

An architecture file containing an unrecognized type (`type: bogus`) present on disk before boot should be parsed during startup scan and record a parse error with message `unknown type "bogus"`.

## Actual Behaviour

Because the startup scan does not traverse `lifecycle/architecture/`, the invalid file is not parsed at startup, leaving the parse error list empty (`[]`).

## Logs / Output

```
=== RUN   TestArchitectureTypes_UnknownTypeStillRejected
2026/08/14 18:55:39 INFO index schema mismatch or missing — rebuilding from disk db=/var/folders/_9/m30sx2q55bx9rf43z8r6mk540000gn/T/TestArchitectureTypes_UnknownTypeStillRejected892309225/002/testproject/index.db
2026/08/14 18:55:39 INFO scan complete indexed=0 skipped=0 files=0 duration=0s
2026/08/14 18:55:39 INFO release startup sync: rehydrated project=testproject inserted=0 skipped=0 pruned=0
2026/08/14 18:55:39 INFO kaos-control started addr=127.0.0.1:60728 version=dev
2026/08/14 18:55:39 INFO http method=GET path=/api/health status=200 bytes=28 duration=126.834µs request_id=loki.local/Mdz1uLoB7a-000001
2026/08/14 18:55:39 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=35.569834ms request_id=loki.local/Mdz1uLoB7a-000002
2026/08/14 18:55:39 INFO http method=GET path=/api/p/testproject/parse-errors status=200 bytes=16 duration=198.459µs request_id=loki.local/Mdz1uLoB7a-000003
    architecture_types_test.go:122: expected a parse error for "lifecycle/architecture/tt-bogus.md" (unknown type), found none
2026/08/14 18:55:39 INFO SCHEDULER: scheduler stopped
--- FAIL: TestArchitectureTypes_UnknownTypeStillRejected (0.16s)
```
