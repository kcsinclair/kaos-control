---
title: 'go vet lint failure: unused seedRunRecord/writeMinimalLogFile test helpers'
type: defect
status: done
lineage: devops-pipeline-run-history
parent: internal/devops/run_record_test.go
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Resolution (2026-07-09)

Removed the unused `seedRunRecord` and `writeMinimalLogFile` helpers from
`internal/devops/run_record_test.go`, and dropped the now-unused
`encoding/json` import. Package tests pass; staticcheck clean; `make lint` green.

# go vet lint failure: unused seedRunRecord/writeMinimalLogFile test helpers

## Reproduction Steps

1. Run `make lint`
2. Observe `go vet ./...` (staticcheck U1000) fails on `internal/devops/run_record_test.go`

## Expected Behaviour

`make lint` should pass with no unused-code warnings.

## Actual Behaviour

`staticcheck` reports two unused test helper functions in
`run_record_test.go`, not called from any test in the package:

```
internal/devops/run_record_test.go:341:6: func seedRunRecord is unused (U1000)
internal/devops/run_record_test.go:350:6: func writeMinimalLogFile is unused (U1000)
```

Either delete these helpers, or use them from the backfill/run-record test
cases they were apparently written to support.

## Logs / Output

```
go vet ./...
internal/devops/run_record_test.go:341:6: func seedRunRecord is unused (U1000)
internal/devops/run_record_test.go:350:6: func writeMinimalLogFile is unused (U1000)
make[1]: *** [lint] Error 1
```
