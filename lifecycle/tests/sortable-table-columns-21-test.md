---
title: "useSortableTable perf: text sort flakiness fix — isolated pool + raised threshold"
type: test
status: in-qa
lineage: sortable-table-columns
parent: lifecycle/defects/sortable-table-columns-19-defect.md
---

# useSortableTable perf: text sort flakiness fix — isolated pool + raised threshold

Addresses the intermittent failure of the `text sort completes in under 100 ms`
test in `useSortableTable.perf.test.ts` when run as part of the full test suite.
The root cause was OS/process scheduler jitter inflating wall-clock timings
measured by `performance.now()` when the perf file competed for CPU with
component-mounting tests in the parallel worker-thread pool.

## Files modified

| File | Change |
|------|--------|
| `tests/web/vitest.config.ts` | Added `poolMatchGlobs` to route `*.perf.test.ts` and `*.perf.spec.ts` to the `forks` pool |
| `tests/web/useSortableTable.perf.test.ts` | Raised `text` sort budget from 100 ms to 250 ms; updated test name and header comment |

## Fix strategy

Two complementary changes were applied (options 1 + 2 from the defect's fix
guidance):

**Isolated pool (option 2):** `vitest.config.ts` now uses `poolMatchGlobs` to
run all `*.perf.test.ts` / `*.perf.spec.ts` files inside dedicated forked
processes (`pool: 'forks'`).  A forked process has its own V8 heap and is
scheduled by the OS independently of the worker threads that mount Vue
components, eliminating the primary source of timing interference.

**Raised text-sort budget (option 1):** `localeCompare`-based `text` sort is
inherently slower than numeric or ISO-date comparison.  The 100 ms threshold
was too tight for some CI environments even without concurrency pressure.  The
budget for the `text` sort at 1,000 rows is now 250 ms, which still catches
genuine algorithmic regressions while absorbing realistic environment variation.
The `string`, `date`, and `number` sorts retain their original 100 ms budgets.

## Scenarios covered

### `tests/web/useSortableTable.perf.test.ts`

| Dataset | Sort type | Budget | Status |
|---------|-----------|--------|--------|
| 1,000 rows | string | < 100 ms | unchanged |
| 1,000 rows | date | < 100 ms | unchanged |
| 1,000 rows | number | < 100 ms | unchanged |
| 1,000 rows | text | < 250 ms | raised from 100 ms |
| 5,000 rows | string | < 500 ms | unchanged |
| 5,000 rows | date | < 500 ms | unchanged |
| 5,000 rows | number | < 500 ms | unchanged |

### `tests/web/vitest.config.ts`

Pool isolation rule: `['**/*.perf.test.ts', 'forks']` and
`['**/*.perf.spec.ts', 'forks']` ensure all performance test files run in
isolated forked processes.  `singleFork: false` means each perf file gets its
own fork (no cross-file CPU contention within the perf group either).
