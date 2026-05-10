---
title: "Test Plan: Roadmap Gantt Period Display Options"
type: plan-test
status: approved
lineage: roadmap-gantt-period-options
parent: lifecycle/requirements/roadmap-gantt-period-options-2.md
created: "2026-05-10T00:00:00+10:00"
labels:
    - roadmaps
    - frontend
    - enhancement
release: KC-Release0
assignees:
    - role: test-developer
      who: agent
---

# Test Plan: Roadmap Gantt Period Display Options

Integration tests verifying the Gantt period display options feature end to end.
Tests run against a live kaos-control server using Playwright (or the existing
integration test harness in `tests/`). The test suite covers the acceptance
criteria from the requirement and validates interactions between the backend
config and frontend behaviour.

Related plans: [[roadmap-gantt-period-options]] (backend plan for config;
frontend plan for UI logic).

---

## Milestone 1: Backend config tests

### Description

Test that the project configuration correctly parses, validates, and serves
the `roadmap.default_period_mode` setting. These are Go unit/integration tests
in `tests/` (or additions to `internal/config/config_test.go` if the test-
developer agent has write access to `web/src/` and `tests/`).

### Files to change

- `tests/config_roadmap_test.go` (new) — or extend existing config test files.

### Acceptance criteria

- [ ] Test: `GET /api/p/{project}/config` returns
      `roadmap.default_period_mode` matching the value in `lifecycle/config.yaml`.
- [ ] Test: when `lifecycle/config.yaml` has no `roadmap` section, the endpoint
      returns `"autoscale"` as the default.
- [ ] Test: an invalid `default_period_mode` value (e.g., `"weekly"`) causes
      a config load error (unit test or startup failure test).

---

## Milestone 2: Period-mode selector UI tests

### Description

Integration tests verifying the period-mode selector control appears in the
toolbar, responds to clicks, and conditionally shows the fixed-period picker.

### Files to change

- `tests/roadmap_gantt_period_test.go` (new) or equivalent Playwright test file.

### Acceptance criteria

- [ ] Test: navigate to Roadmap Gantt view; verify the period-mode selector
      (Autoscale / Fixed Period) is visible.
- [ ] Test: click "Fixed Period"; verify the secondary period picker (Month,
      Quarter, Half-Year, Year) appears.
- [ ] Test: click "Autoscale"; verify the secondary period picker is hidden.
- [ ] Test: switch to Graph view; verify the period-mode selector is hidden.
- [ ] Test: switch back to Gantt view; verify the period-mode selection is
      preserved (session persistence).

---

## Milestone 3: Autoscale mode tests

### Description

Verify that autoscale mode computes the time axis correctly for various
release configurations.

### Files to change

- `tests/roadmap_gantt_period_test.go` — add autoscale test cases.

### Acceptance criteria

- [ ] Test: create two releases spanning Mar–Apr and Jun–Jul; select Autoscale
      with month granularity; verify the time axis starts at Mar and ends at Jul
      — no columns before Mar or after Jul.
- [ ] Test: delete all scheduled releases; verify a single column containing
      today's date is displayed.
- [ ] Test: with one release spanning a single week, autoscale at week
      granularity shows exactly the columns needed — no padding.

---

## Milestone 4: Fixed-period mode tests

### Description

Verify that each fixed-period option anchors the time axis correctly to the
current calendar period.

### Files to change

- `tests/roadmap_gantt_period_test.go` — add fixed-period test cases.

### Acceptance criteria

- [ ] Test: select Fixed Period > Month; verify the first column starts at the
      1st of the current month and the last column ends at the last day of the
      current month.
- [ ] Test: select Fixed Period > Quarter; verify the axis spans the current
      calendar quarter (e.g., Apr–Jun for Q2).
- [ ] Test: select Fixed Period > Half-Year; verify the axis spans Jan–Jun or
      Jul–Dec depending on the current date.
- [ ] Test: select Fixed Period > Year; verify the axis spans Jan 1 to Dec 31
      of the current year.

---

## Milestone 5: Bar clipping tests

### Description

Verify that release bars extending beyond the fixed-period window are clipped
rather than hidden, and that visual clip indicators appear.

### Files to change

- `tests/roadmap_gantt_period_test.go` — add bar clipping test cases.

### Acceptance criteria

- [ ] Test: create a release spanning two months; select Fixed Period > Month
      (current month covers only part of the release); verify the bar is visible
      and clipped at the window boundary.
- [ ] Test: verify a clip indicator (arrow/chevron) is rendered on the clipped
      edge of the bar.
- [ ] Test: a release entirely outside the fixed-period window does not render
      an empty row.
- [ ] Test: in autoscale mode, no clip indicators appear (bars always fit the
      axis).

---

## Milestone 6: Horizontal scrolling and safety-cap tests

### Description

Verify horizontal scrolling behaviour and the 200-column safety cap with
auto-coarsening.

### Files to change

- `tests/roadmap_gantt_period_test.go` — add scrolling and safety-cap test
  cases.

### Acceptance criteria

- [ ] Test: select Fixed Period > Year with Week granularity (~52 columns);
      verify the chart is horizontally scrollable and the left label column
      remains sticky (fixed position during scroll).
- [ ] Test: create a scenario that would exceed 200 columns (e.g., Year period
      with some hypothetical sub-week granularity, or manipulate conditions);
      verify granularity is auto-coarsened and a visual indicator is displayed.
- [ ] Test: the existing sticky unscheduled column on the right remains
      functional during horizontal scroll.

---

## Milestone 7: Accessibility and responsiveness tests

### Description

Verify keyboard navigation, ARIA attributes, and responsive layout of the new
toolbar controls.

### Files to change

- `tests/roadmap_gantt_period_test.go` — add accessibility test cases.

### Acceptance criteria

- [ ] Test: Tab through the period-mode selector and fixed-period picker;
      verify all buttons are focusable and activatable with Enter/Space.
- [ ] Test: verify `role="group"` and `aria-label` attributes on the new
      control groups.
- [ ] Test: at viewport width 1024 px, verify the toolbar does not wrap or
      overflow.
- [ ] Test: at viewport width 768 px, verify controls stack gracefully
      (no content cut off).

---

## Milestone 8: Default-from-config and no-extra-API-calls tests

### Description

Verify the integration between the backend config default and the frontend
initial state, and confirm that switching modes does not trigger additional
release-data API requests.

### Files to change

- `tests/roadmap_gantt_period_test.go` — add config-default and network test
  cases.

### Acceptance criteria

- [ ] Test: set `roadmap.default_period_mode: "quarter"` in config; load the
      Roadmap page; verify the Gantt initialises in Fixed Period > Quarter mode.
- [ ] Test: with no `roadmap` config, load the Roadmap page; verify the Gantt
      initialises in Autoscale mode.
- [ ] Test: switch between Autoscale and Fixed Period modes; monitor network
      requests; verify no additional calls to the releases API endpoint are made.
- [ ] Test: granularity and period mode operate independently — changing one
      does not reset the other.

---

## Companion test artifact

After all milestones pass, create a companion artifact in `lifecycle/tests/`
documenting what this test suite covers:

- **File**: `lifecycle/tests/roadmap-gantt-period-options-6-test.md`
- **Frontmatter**: `type: test`, `status: draft`,
  `lineage: roadmap-gantt-period-options`,
  `parent: lifecycle/test-plans/roadmap-gantt-period-options-5-test.md`
- **Body**: summary of scenarios covered, pointing to the test files in `tests/`.
