---
title: performance.test.ts — correct total_tickets field name in all mocks
type: test
status: in-qa
lineage: dashboard-home-screen
parent: lifecycle/defects/dashboard-home-screen-10-defect.md
---

# performance.test.ts — correct total_tickets field name in all mocks

Fixes the wrong mock field name (`total` → `total_tickets`) in the
five-consecutive-mounts test loop inside `tests/web/performance.test.ts`.

## Scenarios covered

### Fix: five-consecutive-mounts mock field name (line 130)

File: `tests/web/performance.test.ts`

| Test | Description |
|---|---|
| `mounts and renders summary counts within 500 ms` | Top-level module mock already used `total_tickets`; confirmed passing |
| `mounts synchronously before the API resolves` | Synchronous mount test unaffected by field name; confirmed passing |
| `renders within budget across five consecutive mounts` | `mockResolvedValueOnce` in the loop now uses `total_tickets: i` (was `total: i`); all five runs pass without Vue prop-type warnings |

## Change made

`tests/web/performance.test.ts` line 130:

```ts
// Before (broken)
vi.mocked(api.get).mockResolvedValueOnce({ total: i, in_progress: 0, blocked: 0, completed_this_week: 0 })

// After (fixed)
vi.mocked(api.get).mockResolvedValueOnce({ total_tickets: i, in_progress: 0, blocked: 0, completed_this_week: 0 })
```

The `SummaryCountsWidget` interface declares `total_tickets: number`; passing
`total` left `stats.value.total_tickets` as `undefined`, causing a Vue
prop-type warning and preventing two stat cards from rendering with the
expected role.

## Verification

```sh
cd tests/web && npx vitest run performance.test.ts --config vitest.config.ts
# → 3 tests passed, 0 failed
```
