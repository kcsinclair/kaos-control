---
title: "Test Plan: Dashboard Clickable Filters"
type: plan-test
status: done
lineage: dashboard-clickable-filters
parent: lifecycle/requirements/dashboard-clickable-filters-2.md
created: "2026-05-09T00:00:00+10:00"
---

# Test Plan: Dashboard Clickable Filters

## Overview

Integration and end-to-end tests for the [[dashboard-clickable-filters]] feature. Tests verify that dashboard click-throughs produce correct URL navigation and that the artifacts list view correctly applies filters from URL query parameters. Tests should run against a live dev server with seeded lifecycle artifacts covering multiple statuses.

## Milestone 1: Test Fixture Setup

**Description:** Create or extend test fixtures with lifecycle artifacts in known statuses to give deterministic counts for dashboard widgets.

**Files to change:**

- `tests/fixtures/` — add markdown artifacts with frontmatter covering at least: 2× `status: draft`, 1× `status: blocked`, 1× `status: in-development`, 1× `status: done`. Each must have valid `title`, `type`, `lineage`, and `status` fields so the indexer picks them up.
- `tests/helpers/` (or equivalent) — if a seed/reset helper exists, extend it to load these fixtures into a test project's `lifecycle/` directory before each test suite.

**Acceptance criteria:**

- [ ] Test fixtures produce a known, deterministic dashboard state (total count, blocked count, status distribution with at least 3 distinct statuses).
- [ ] Fixtures are isolated per test run (no cross-test contamination).

## Milestone 2: ArtifactListView Query Parameter Filter Tests

**Description:** Test that navigating directly to `/p/:project/artifacts?status=<value>` displays filtered results.

**Files to change:**

- `tests/dashboard-clickable-filters.spec.ts` (new file, or extend existing artifact list tests).

**Test cases:**

1. **Direct navigation with status filter** — Navigate to `/p/:project/artifacts?status=blocked`. Assert: the status dropdown shows "blocked", the artifact list contains only artifacts with `status: blocked`, and the count matches the fixture data.
2. **Direct navigation with no filter** — Navigate to `/p/:project/artifacts`. Assert: no dropdown is pre-selected, all artifacts are displayed.
3. **Deep-link bookmark fidelity** — Load `/p/:project/artifacts?status=draft` in a fresh browser context (no prior navigation). Assert: filter is applied correctly on first load.
4. **Unknown status value** — Navigate to `/p/:project/artifacts?status=nonexistent`. Assert: the view loads without error; either shows no results or ignores the unknown filter gracefully.

**Acceptance criteria:**

- [ ] All four test cases pass.
- [ ] Tests run against the live dev server (not mocked).

## Milestone 3: SummaryCountCard Click-Through Tests

**Description:** Test that clicking active Summary Count cards navigates to the correct filtered artifacts list.

**Files to change:**

- `tests/dashboard-clickable-filters.spec.ts` (continued).

**Test cases:**

1. **Lifecycle Total click** — On the dashboard, click the Lifecycle Total card. Assert: browser navigates to `/p/:project/artifacts` with no query parameters. Assert: artifact list shows all artifacts.
2. **Blocked click** — Click the Blocked card. Assert: browser navigates to `/p/:project/artifacts?status=blocked`. Assert: artifact list shows only blocked artifacts and the count matches the dashboard card's displayed count.
3. **In Progress not clickable** — Assert: the In Progress card does not have `cursor: pointer` style and clicking it does not navigate away from the dashboard.
4. **Completed This Week not clickable** — Assert: the Completed This Week card does not have `cursor: pointer` style and clicking it does not navigate away.
5. **Back button** — Click the Blocked card, wait for list view, press browser back. Assert: user returns to the dashboard.

**Acceptance criteria:**

- [ ] All five test cases pass.
- [ ] Click uses router navigation (URL changes without full page reload).

## Milestone 4: StatusDistributionWidget Click-Through Tests

**Description:** Test that clicking pie chart segments navigates to the correct filtered view.

**Files to change:**

- `tests/dashboard-clickable-filters.spec.ts` (continued).

**Test cases:**

1. **Pie segment click** — On the dashboard, click a pie segment representing `status: draft`. Assert: browser navigates to `/p/:project/artifacts?status=draft`. Assert: artifact list shows only draft artifacts.
2. **Different segment** — Click a segment for a different status (e.g., `blocked`). Assert: correct navigation and filtering.
3. **Cursor style** — Hover over a pie segment. Assert: cursor is `pointer`.

**Acceptance criteria:**

- [ ] All three test cases pass.
- [ ] The status value in the URL exactly matches the status key from the chart data (no case mismatch, no display-label leakage).

## Milestone 5: Accessibility Tests

**Description:** Verify keyboard navigation and ARIA attributes for interactive dashboard elements.

**Files to change:**

- `tests/dashboard-clickable-filters.spec.ts` (continued).

**Test cases:**

1. **Keyboard activation — Lifecycle Total** — Tab to the Lifecycle Total card, press Enter. Assert: navigates to artifacts list.
2. **Keyboard activation — Blocked** — Tab to the Blocked card, press Space. Assert: navigates to `/p/:project/artifacts?status=blocked`.
3. **ARIA attributes** — Assert: interactive cards have `role="link"` and `aria-label` containing the card's count and label (e.g., matching pattern `/view \d+ .* artifacts/i`).
4. **Non-interactive cards ARIA** — Assert: In Progress and Completed This Week cards do not have `role="link"`.
5. **Chart container ARIA** — Assert: the StatusDistributionWidget chart container has an `aria-label` that mentions clickability.

**Acceptance criteria:**

- [ ] All five test cases pass.
- [ ] Focus ring is visible on interactive cards (visual check or computed-style assertion).

## Milestone 6: Regression Tests

**Description:** Confirm existing dashboard behaviour is unaffected.

**Files to change:**

- `tests/dashboard-clickable-filters.spec.ts` (continued) or existing regression test files.

**Test cases:**

1. **Dashboard loads without errors** — Navigate to dashboard; assert no console errors and all four widget sections render.
2. **Activity Feed links still work** — Click a feed entry; assert navigation to the correct artifact editor page.
3. **Activity Feed "View all" still works** — Click "View all"; assert navigation to `/p/:project/feed`.
4. **Velocity chart toggle** — Toggle granularity on the velocity chart; assert the chart re-renders without error.
5. **Summary counts update on WebSocket** — Trigger an `artifact.indexed` WebSocket event; assert the summary counts widget refetches and updates its values.

**Acceptance criteria:**

- [ ] All five regression test cases pass.
- [ ] No existing tests broken by the new feature.
