---
title: "useSortableTable perf: text sort for 1,000 rows intermittently exceeds 100 ms under full-suite load"
type: defect
status: done
lineage: sortable-table-columns
parent: lifecycle/tests/sortable-table-columns-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
release: May2026
---

# useSortableTable perf: text sort for 1,000 rows intermittently exceeds 100 ms under full-suite load

## Reproduction Steps

1. Run the full test suite from `tests/web/`:
   ```sh
   cd tests/web && pnpm test
   ```
2. Observe a sporadic failure in `useSortableTable.perf.test.ts`:
   - "useSortableTable — performance: 1,000 rows > text sort completes in under 100 ms"
3. Run the perf test file in isolation:
   ```sh
   cd tests/web && pnpm vitest run useSortableTable.perf
   ```
4. Observe that all 7 tests pass consistently in isolation.

## Expected Behaviour

`text` sort for 1,000 rows should complete in under 100 ms in all execution contexts, including when run alongside the full suite of ~850 other tests.

## Actual Behaviour

When the full suite runs in parallel, the `text sort completes in under 100 ms` test occasionally fails because wall-clock time (measured with `performance.now()`) includes OS/process scheduling latency introduced by concurrent test environments. The same sort completes well within budget when the file is run alone.

This is a flaky test: it passes reliably in isolation but fails intermittently under full-suite parallelism.

## Logs / Output

```
× useSortableTable.perf.test.ts
  > useSortableTable — performance: 1,000 rows
  > text sort completes in under 100 ms

Full suite run:
  Test Files  1 failed | 49 passed (50)
      Tests  3 failed | 849 passed (851)

Isolated run:
  Test Files  1 passed (1)
      Tests  7 passed (7)
   Duration  362ms
```

## Fix Guidance

Options (pick one):

1. **Increase the budget for `text` sort** from 100 ms to a more conservative threshold (e.g. 250 ms) to absorb scheduler jitter. `text` sort uses `localeCompare` which is inherently slower than numeric or date comparison and may vary across environments.
2. **Use `test.concurrent: false`** / run the perf suite in a separate Vitest project or pool so it does not compete for CPU with component-mounting tests.
3. **Replace wall-clock timing with iteration count**: test correctness of sort order and trust existing unit tests for performance; track performance separately via a dedicated benchmark suite.
