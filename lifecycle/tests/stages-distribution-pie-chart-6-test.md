---
title: "Test Suite: Stages Distribution Pie Chart"
type: test
status: done
lineage: stages-distribution-pie-chart
parent: lifecycle/test-plans/stages-distribution-pie-chart-5-test.md
created: "2026-05-09T00:00:00+10:00"
---

# Test Suite: Stages Distribution Pie Chart

## What this covers

Integration and unit tests for the `stages-distribution-pie-chart` feature,
implementing all five milestones from the test plan.

## Test files

### Milestone 1 — Backend API integration tests

**File:** `tests/integration/dashboard_stage_distribution_test.go`

Seven test cases exercising `GET /api/p/:project/dashboard/stage-distribution`:

| Test function | Scenario |
|---|---|
| `TestStageDistribution_Empty` | Empty project returns `{"distribution":[]}` (non-null array) |
| `TestStageDistribution_HappyPath` | Multi-stage project returns correct stage names and counts |
| `TestStageDistribution_TrackedTypesFiltering` | `Dashboard.TrackedTypes` subset excludes untracked types |
| `TestStageDistribution_ExcludesDoneAndAbandoned` | `done` / `abandoned` artifacts are absent from response |
| `TestStageDistribution_MixedStatuses` | Only non-done/non-abandoned artifacts are counted per stage |
| `TestStageDistribution_SingleStage` | All artifacts in one stage → exactly one response entry |
| `TestStageDistribution_AlphabeticalOrdering` | Distribution array is sorted by stage name ascending |

Build tag: `//go:build integration`. Run with `go test -tags integration ./tests/integration/...`.

### Milestone 2 — Index unit tests

**File:** `internal/index/stage_distribution_test.go`

Four unit tests for `(*Index).StageDistribution` in package `index`:

| Test function | Scenario |
|---|---|
| `TestStageDistribution_CorrectGrouping` | Artifacts in multiple stages are grouped and counted correctly |
| `TestStageDistribution_EmptyDatabase` | Empty index returns non-nil `[]StageCount{}` |
| `TestStageDistribution_TrackedTypesDefault` | `nil` / `[]string{}` tracked types fall back to `["ticket"]` |
| `TestStageDistribution_StatusExclusion` | `done` and `abandoned` artifacts are excluded |

Run with `go test ./internal/index/...`.

### Milestone 3 — Frontend widget component tests

**File:** `tests/web/StagesDistributionWidget.test.ts`

Seventeen test cases covering the `StagesDistributionWidget.vue` component:

- **TC1** — Renders `[role="img"]` chart container when distribution is non-empty
- **TC2** — Shows "No artifacts yet" empty state for `distribution: []`
- **TC3** — Shows "No artifacts yet" when all stage counts are zero
- **TC4** — echarts click event calls `router.push({ name: 'artifacts', params: { project }, query: { stage } })`; exact stage name preserved; no navigation on undefined segment
- **TC5** — Chart container has `role="img"`; `aria-label` includes stage names, counts, and mentions clickability; empty state has no `[role="img"]` element
- **TC6** — Changing the `project` prop triggers a re-fetch using the new project name in the URL
- **TC7** — API error causes graceful degradation to the empty state without throwing
- **General** — Widget title is "Stages Distribution"; API called with correct project-scoped URL

Uses `happy-dom` + `@vue/test-utils` + Vitest. echarts is mocked to capture the click handler. No real network calls.

Run with `pnpm --filter=tests/web test` or `vitest run` from the `tests/web/` directory.

### Milestone 4 — Widget registration tests

**File:** `tests/web/widgetRegistry.test.ts` (added describe block)

Six test cases in the `widgetRegistry — stages-distribution registration (Milestone 4)` describe block:

- **TC1** — `stages-distribution` is registered at `slot: 'chart'`, `order: 1`
- **TC2** — Chart-slot ordering is `status-distribution(0)` → `stages-distribution(1)` → `velocity-chart(2)`
- **TC3** — Registering `stages-distribution` twice yields exactly one entry (first wins)
- **TC3b** — First registration's slot/order/component are preserved on duplicate
- Ordering invariants: `status-distribution` stays at 0, `velocity-chart` stays at 2

### Milestone 5 — End-to-end dashboard integration

**File:** `tests/web/DashboardView.test.ts` (added describe block)

Six test cases in the `DashboardGrid — Milestone 5: StagesDistributionWidget integration` describe block:

- **TC1** — Widget title "Stages Distribution" is visible when registered in the dashboard
- **TC1b** — Widget renders inside the `section[aria-label="Charts"]` DOM section
- **TC6** — `status-distribution`, `stages-distribution`, and `velocity-chart` all render together without regressions
- **TC6b** — Chart-slot widgets appear in correct DOM order (status → stages → velocity)
- **TC5** — Bookmarkable URL: DashboardGrid passes the correct `project` prop down to the widget

**Deferred to Playwright E2E:**
- M5-TC2 click-through URL verification (covered in widget isolation in Milestone 3 TC4)
- M5-TC4 back-navigation restoring the dashboard (requires real browser history)

## Known limitations / deferred coverage

- Viewport layout assertions (two-column vs. single-column grid) require Playwright — deferred as per prior project decision.
- Full click-through → filtered list → back navigation flow requires a real browser; deferred to E2E Playwright suite.
- ResizeObserver behaviour is not tested (happy-dom does not emulate ResizeObserver resize events triggering `chart.resize()`).
