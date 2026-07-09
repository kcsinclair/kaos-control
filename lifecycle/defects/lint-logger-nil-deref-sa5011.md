---
title: 'staticcheck SA5011: possible nil pointer dereference in backfillRecord'
type: defect
status: done
lineage: devops-pipeline-run-history
parent: internal/devops/logger.go
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

## Resolution (2026-07-09)

`first` and `last` are set together in the scan loop, so the `first == nil`
guard already guarantees `last != nil`. Removed the redundant `if last != nil`
checks (which is what made staticcheck flag the `endedAt := last.Time` deref as
nilable) and added a comment noting the invariant. staticcheck clean; devops
tests pass; `make lint` green.

# staticcheck SA5011: possible nil pointer dereference in backfillRecord

## Reproduction Steps

1. Run `make lint`
2. Observe `go vet ./...` (staticcheck SA5011) fails on `internal/devops/logger.go`

## Expected Behaviour

`make lint` should pass with no static-analysis warnings.

## Actual Behaviour

`backfillRecord` in `internal/devops/logger.go` dereferences `last.Time` at
line 199 (`endedAt := last.Time`) before ever checking `last` for nil, then
guards a later use of `last` with `last != nil` at line 209. staticcheck
flags this as a possible nil dereference because the later check implies
`last` can be nil.

In practice `last` cannot be nil at line 199 given `first != nil` was
already checked (the scan loop sets `last` on every successful iteration
that also sets `first`), but the code doesn't make that invariant visible
to the analyzer. Restructure so the nil-safety is explicit at the point of
use — e.g. drop the redundant `first == nil` early-return in favor of
checking `last == nil` directly, or hoist a single guard before both uses.

## Logs / Output

```
go vet ./...
internal/devops/logger.go:199:18: possible nil pointer dereference (SA5011)
	internal/devops/logger.go:209:5: this check suggests that the pointer can be nil
make[1]: *** [lint] Error 1
```
