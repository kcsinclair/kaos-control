---
title: "Tests: Dashboard Clickable Filters"
type: test
status: approved
lineage: dashboard-clickable-filters
parent: lifecycle/test-plans/dashboard-clickable-filters-5-test.md
created: "2026-05-09T00:00:00+10:00"
---

# Tests: Dashboard Clickable Filters

Companion artifact documenting the test suite built for the
[[dashboard-clickable-filters]] feature.

## Test Files

### Integration tests (Go)

`tests/integration/dashboard_clickable_filters_test.go`

Go integration tests that start a full HTTP server with a seeded
lifecycle project and drive the REST API directly.

### Web component tests (Vitest / Vue Test Utils)

`tests/web/dashboard-clickable-filters.test.ts`

Component-level tests for `SummaryCountCard`, `SummaryCountsWidget`,
and `StatusDistributionWidget` running under happy-dom.

### Config change

`tests/web/vitest.config.ts` — added canonical aliases for
`echarts/core`, `echarts/charts`, `echarts/components`, and
`echarts/renderers` so that `vi.mock('echarts/core')` intercepts the
same module path that `StatusDistributionWidget.vue` imports. Follows
the same pattern used for cytoscape.

## Scenarios Covered

### Milestone 1 — Test fixture setup

`dashboardClickableFiltersSeeds()` helper in the Go test file seeds a
deterministic project state:
- 2× `status: draft` ideas
- 1× `status: blocked` ticket (with `## Open Questions` to prevent
  autoblock transition)
- 1× `status: in-development` ticket
- 1× `status: done` ticket
- 1× `status: planning` ticket

Each test runs in an isolated `t.TempDir()` — no cross-test contamination.

### Milestone 2 — ArtifactListView query parameter filter tests

All four test cases from the plan implemented as Go integration tests:

| Test | Plan reference |
|---|---|
| `TestDCF_DirectNavigationWithStatusFilter` | M2-TC1 |
| `TestDCF_DirectNavigationNoFilter` | M2-TC2 |
| `TestDCF_DeepLinkBookmarkFidelity` | M2-TC3 |
| `TestDCF_UnknownStatusGracefulDegradation` | M2-TC4 |

All tests run against the live dev server (no mocking).

### Milestone 3 — SummaryCountCard click-through tests

Implemented as Vitest component tests against `SummaryCountCard.vue` and
`SummaryCountsWidget.vue`:

| Test | Plan reference |
|---|---|
| Lifecycle Total card `role="link"` | M3-TC1 |
| Blocked card navigates to `?status=blocked` | M3-TC2 |
| In Progress card `role="figure"`, no navigation | M3-TC3 |
| Completed This Week `role="figure"`, no navigation | M3-TC4 |
| Enter key activates Lifecycle Total card | M3-TC5 / M5-TC1 |

**M3-TC5 (back button)** requires real browser history; deferred to a
Playwright E2E suite.

The `.summary-card--interactive` CSS class is asserted as a proxy for
`cursor: pointer` since happy-dom does not evaluate CSS.

### Milestone 4 — StatusDistributionWidget click-through tests

| Test | Plan reference |
|---|---|
| Pie segment click → `?status=draft` | M4-TC1 |
| Different segment → `?status=blocked` | M4-TC2 |
| Status key in URL is exact (no case mismatch) | M4-TC3 |
| Empty distribution → no chart rendered | — |

The echarts click handler is captured during `chart.on('click', ...)` in
the mocked `init` call and fired manually in tests.

Cursor `pointer` style (M4-TC3) is asserted via the `cursor: 'pointer'`
option in the echarts series config rather than a CSS evaluation.

### Milestone 5 — Accessibility tests

| Test | Plan reference |
|---|---|
| Interactive card `role="link"`, `tabindex="0"` | M5-TC3 |
| Non-interactive cards do not have `role="link"` | M5-TC4 |
| Chart container `role="img"`, aria-label mentions "click" | M5-TC5 |
| `aria-label` matches `/view \d+ .* artifacts/i` | M5-TC3 |
| Enter key activates Lifecycle Total | M5-TC1 |
| Space key activates Blocked card | M5-TC2 |
| `.summary-card--interactive` class present (focus ring proxy) | M5-TC5 |

### Milestone 6 — Regression tests

| Test | Plan reference | Layer |
|---|---|---|
| `TestDCF_Regression_DashboardEndpointsLoad` | M6-TC1 | Go |
| `TestDCF_Regression_FeedEndpointReachable` | M6-TC3 | Go |
| `TestDCF_Regression_FeedActivityLinksHaveValidPaths` | M6-TC2 | Go |
| `TestDCF_Regression_StatsUpdateAfterReindex` | M6-TC5 | Go |
| WS handler registered for `artifact.indexed` | M6-TC5 | Vitest |
| Stats refetch fires after WS handler invoked | M6-TC5 | Vitest |

**M6-TC4 (velocity chart granularity toggle)** requires a real canvas
renderer; deferred to the Playwright E2E suite.

### Additional invariant tests (Go)

Two extra tests verify key cross-layer contracts:

- `TestDCF_PieSegmentClickNavigationContract` — for every status in the
  distribution response, the count matches the artifact list filter count.
  Breaks if pie chart data and artifact index diverge.
- `TestDCF_BlockedCardClickContract` — the stats `blocked` count is ≥ the
  exact-match filter count (stats includes clarifying; filter is exact).

## Deferred to Playwright E2E

- M3-TC5: Browser back button after Blocked card click
- M5-TC1 visual: Focus ring appearance on interactive cards
- M6-TC4: Velocity chart granularity toggle (canvas rendering)
