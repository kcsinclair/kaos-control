---
title: "Test Fix: SummaryCountsWidget role selector and mock field name"
type: test
status: draft
lineage: stages-distribution-pie-chart
parent: lifecycle/defects/stages-distribution-pie-chart-8.md
created: "2026-05-09T00:00:00+10:00"
---

# Test Fix: SummaryCountsWidget role selector and mock field name

Fixes two bugs in `tests/web/DashboardView.test.ts` that caused four tests in the `SummaryCountsWidget — summary counts after API response` describe block to fail.

## Scenarios covered

### Fix 1 — Role selector corrected (`[role="figure"]` → `.summary-card`)

`SummaryCountCard.vue` assigns `role="link"` to interactive cards (those with a non-null `:to` prop) and `role="figure"` to non-interactive ones. Two of the four cards ("Lifecycle Total" and "Blocked") receive `role="link"`, so querying by `[role="figure"]` found only 2 elements instead of 4. All four tests in the block now use `.summary-card` as the selector, which is present on every card regardless of interactivity.

Affected lines (post-fix): 237, 247, 266, 283.

### Fix 2 — Mock field name corrected (`total` → `total_tickets`)

The backend returns `total_tickets` (per `DashboardStatsRow` in `internal/index/index.go:1630`) and `SummaryCountsWidget` reads `stats.total_tickets`. Both the module-level `vi.mock` default (line 41) and the per-test `mockResolvedValueOnce` (line 254) previously used `total`, causing the component to receive `undefined` for the first card value. Both are now updated to `total_tickets`.

## Test file

`tests/web/DashboardView.test.ts` — `SummaryCountsWidget — summary counts after API response` describe block (lines 230–313).

The four previously-failing tests are:
- `renders four stat cards on mount`
- `shows zero counts while waiting for the API (initial state)`
- `displays counts returned by the API after the response resolves`
- `keeps zero counts when the API call fails (graceful degradation)`
