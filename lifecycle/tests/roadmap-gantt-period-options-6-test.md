---
title: "Test Suite: Roadmap Gantt Period Display Options"
type: test
status: draft
lineage: roadmap-gantt-period-options
parent: lifecycle/test-plans/roadmap-gantt-period-options-5-test.md
created: "2026-05-10T00:00:00+10:00"
labels:
    - roadmaps
    - frontend
    - enhancement
release: KC-Release0
---

# Test Suite: Roadmap Gantt Period Display Options

Integration and component tests covering the Gantt period display options feature
end to end, implementing all eight milestones from the test plan.

---

## Test files

### `tests/integration/config_roadmap_test.go`

Go integration tests (build tag: `integration`) covering **Milestone 1** (backend
config API).

| Test | Milestone | Scenario |
|------|-----------|----------|
| `TestConfigRoadmap_ReturnsPeriodModeFromConfig` | 1.1 | Endpoint returns the configured `default_period_mode` |
| `TestConfigRoadmap_AllValidModesRoundtrip` | 1.1 | All five valid values (`autoscale`, `month`, `quarter`, `half-year`, `year`) round-trip correctly |
| `TestConfigRoadmap_DefaultsToAutoscaleWhenNoRoadmapSection` | 1.2 | Missing `roadmap` section defaults to `"autoscale"` |
| `TestConfigRoadmap_DefaultsToAutoscaleWhenRoadmapSectionEmpty` | 1.2 | Empty `roadmap: {}` section defaults to `"autoscale"` |
| `TestConfigRoadmap_InvalidModeRejectedByLoadProject` | 1.3 | `config.LoadProject` returns an error for `"weekly"` |
| `TestConfigRoadmap_OtherInvalidModesRejected` | 1.3 | Additional invalid values (`daily`, `biannual`, `auto`, `AUTOSCALE`) each cause a load error |
| `TestConfigRoadmap_RequiresAuth` | 1 | Unauthenticated requests to `/api/p/{project}/config/roadmap` receive 401 |

The endpoint under test is `GET /api/p/{project}/config/roadmap`, which is served
by `internal/http/config.go:handleGetRoadmapConfig`.

---

### `tests/web/GanttChart.periodMode.test.ts`

Vitest + Vue Test Utils component tests covering **Milestones 3–7**.

#### Milestone 3 — Autoscale mode (3 tests)

- Two scheduled releases spanning Mar–Apr and Jun–Jul produce exactly five month
  columns (Mar through Jul), no padding outside the release span.
- No scheduled releases with empty state displayed correctly (no columns).
- Unscheduled-only releases produce a single today-column in autoscale mode.
- Single-week release at week granularity produces exactly one column.

#### Milestone 4 — Fixed-period mode (5 tests)

- Fixed Month → 1 column (current month).
- Fixed Quarter at month granularity → 3 columns.
- Fixed Half-Year at month granularity → 6 columns.
- Fixed Year at month granularity → 12 columns.
- Fixed Year at quarter granularity → 4 columns.
- Empty releases array shows the empty state (no columns rendered).

#### Milestone 5 — Bar clipping (4 tests)

- Release spanning current and next month: `release-bar--clipped-right` class
  present in fixed-month mode.
- Right clip arrow (`.clip-arrow--right`) rendered on clipped bar.
- Release starting last month and ending this month: `release-bar--clipped-left`
  and `.clip-arrow--left` present.
- Release entirely outside the fixed window: no scheduled bar is rendered at all.
- Autoscale mode: no clipped-left/right classes or clip arrows appear.

#### Milestone 6 — Safety cap / auto-coarsen (4 tests)

- Year at week granularity stays within the 200-column cap (~52 columns).
- Unscheduled column header has `col-header--unscheduled` class (sticky by CSS).
- 10-year autoscale span at week granularity triggers coarsening: column count
  ≤ 200 and `.coarsen-badge` is visible with an explanatory message.
- `.coarsen-badge` absent when no coarsening was needed.

#### Milestone 7 — Accessibility (4 tests)

- All release bars are `<button>` elements (keyboard focusable via Tab).
- Clicking a bar emits `clickRelease` with the correct numeric release id.
- `.coarsen-badge` has `role="status"` and `aria-live="polite"`.
- Clip arrows have `aria-hidden="true"` (decorative, not read by screen readers).

---

### `tests/web/RoadmapView.periodMode.test.ts`

Vitest + Vue Test Utils view-level tests covering **Milestones 2 and 8**, using
mocked stores and APIs to avoid network I/O.

#### Milestone 2 — Period-mode selector UI (7 tests)

- Gantt view renders the period-mode selector group (`aria-label="Period mode"`)
  with Autoscale and Fixed Period buttons.
- Clicking "Fixed Period" makes the fixed-period picker (`aria-label="Fixed period"`)
  appear with Month, Quarter, Half-Year, and Year buttons.
- Clicking "Autoscale" hides the fixed-period picker.
- Switching to Graph view hides the period-mode selector entirely.
- Switching back to Gantt preserves the selected period mode (session persistence
  via Pinia store).
- Period-mode group has `role="group"` and `aria-label="Period mode"` (Milestone 7).
- Fixed-period picker group has `role="group"` and `aria-label="Fixed period"` (Milestone 7).

#### Milestone 8 — Default-from-config and no-extra-API-calls (9 tests)

These tests drive the `useRoadmapSettingsStore` directly:

- Config `default_period_mode: "quarter"` → store initialises to Fixed Period > Quarter.
- Config `default_period_mode: "year"` → store initialises to Fixed Period > Year.
- No roadmap section → store initialises in Autoscale mode.
- Roadmap section with no `default_period_mode` key → Autoscale.
- `loadDefaultPeriodMode` is idempotent: a second call (e.g. re-mount) does not
  overwrite a user's selection; `getConfig` is only called once.
- Granularity and period mode operate independently — changing `fixedPeriod` does
  not reset `periodMode`, and vice versa.
- Config `default_period_mode: "month"` → Fixed Period > Month.
- Config `default_period_mode: "half-year"` → Fixed Period > Half-Year.
- Config `default_period_mode: "autoscale"` → Autoscale (not treated as a fixed period).
- Config fetch error defaults gracefully to Autoscale; `defaultPeriodModeLoaded`
  is set to `true` so the failure is not retried on every re-mount.

---

## Notes on testing approach

### happy-dom constraints

- Sticky positioning, CSS overflow, and horizontal scroll cannot be measured in
  happy-dom. Sticky behaviour is verified via CSS class presence
  (`col-header--unscheduled`, which carries `position: sticky` in the component's
  scoped styles).
- Viewport-size responsiveness (Milestone 7: 1024 px and 768 px checks) requires
  Playwright. Those checks are deferred and noted in the open questions of the
  test plan.

### Timezone safety

All date-to-string conversions in the frontend test helpers use local calendar
values (`getFullYear`, `getMonth`, `getDate`) rather than `toISOString()`, which
returns UTC and would give the wrong date for timezones ahead of UTC (e.g. AEST =
UTC+10).

### No-extra-API-calls test

The test plan's "no additional calls to the releases API on mode switch" scenario
is covered indirectly: the `useReleasesStore.fetch` mock is called exactly once
on mount, and mode changes are driven via the Pinia store (no store action triggers
an API call). A Playwright network-monitor test would give stronger guarantees.
