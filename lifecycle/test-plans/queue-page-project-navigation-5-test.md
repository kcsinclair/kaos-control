---
title: "Queue Page Project Navigation — Test Plan"
type: plan-test
status: approved
lineage: queue-page-project-navigation
parent: lifecycle/requirements/queue-page-project-navigation-2.md
created: "2026-05-13"
---

# Queue Page Project Navigation — Test Plan

This plan covers unit tests for new frontend logic, component tests for the sidebar and modified queue components, and end-to-end tests for the full filtering and navigation workflow.

## Milestone 1 — Unit Tests for Sidebar Job Count Logic

### Description

Test the reactive computation that counts running + pending jobs per project from the queue snapshot. This logic drives the badge counts in the sidebar and must handle edge cases: empty queues, projects with no jobs, a running job belonging to a different project than any pending job.

### Files to create/change

- `tests/web/components/queue/QueueSidebar.spec.ts`

### Test Cases

1. **No jobs** — all project badges show 0.
2. **One pending job** — badge for that job's project shows 1, others show 0.
3. **Running + pending for same project** — badge shows combined count.
4. **Running for project A, pending for project B** — each badge shows 1.
5. **Multiple pending for one project** — badge shows correct aggregate count.
6. **Recent (finished) jobs are NOT counted** — badge excludes completed/failed/skipped/cancelled jobs.

### Acceptance Criteria

- [ ] All six test cases pass.
- [ ] Tests use mocked store state, no API calls.

## Milestone 2 — Component Tests for QueueSidebar

### Description

Mount `QueueSidebar` with a mocked project store and queue store. Verify rendering, selection behaviour, collapse toggle, and ARIA attributes.

### Files to create/change

- `tests/web/components/queue/QueueSidebar.spec.ts` (same file as Milestone 1, additional describe block)

### Test Cases

1. **Renders project list** — sidebar contains an item for each project from the store, plus "All Projects".
2. **Default selection** — "All Projects" is selected on mount (has `aria-current`).
3. **Click project** — emits selection event / updates reactive state; clicked item gains `aria-current`, previous loses it.
4. **Click "All Projects"** — clears selection back to default.
5. **Collapse toggle** — clicking the toggle hides project names; clicking again restores them.
6. **Keyboard navigation** — Tab moves focus between items; Enter/Space triggers selection.
7. **ARIA attributes** — `<nav>` has `role="navigation"` and `aria-label`; toggle button has `aria-expanded`.

### Acceptance Criteria

- [ ] All seven test cases pass.
- [ ] Component renders correctly with zero projects (empty sidebar, only "All Projects").

## Milestone 3 — Component Tests for Filtered Queue Components

### Description

Test that `QueueRunningPanel`, `QueuePendingTable`, and `QueueRecentTable` correctly apply the project filter prop/injection.

### Files to create/change

- `tests/web/components/queue/QueueRunningPanel.spec.ts` (extend existing or create)
- `tests/web/components/queue/QueuePendingTable.spec.ts`
- `tests/web/components/queue/QueueRecentTable.spec.ts`

### Test Cases

**QueueRunningPanel:**
1. **No filter** — shows running job regardless of project.
2. **Filter matches running job** — shows the running job.
3. **Filter does not match running job** — shows "Nothing running".
4. **No running job + filter active** — shows "Nothing running".

**QueuePendingTable:**
5. **No filter** — shows all pending jobs.
6. **Filter active** — shows only jobs where `job.project === filter`.
7. **Filter active, no matching jobs** — shows project-specific empty state message.

**QueueRecentTable:**
8. **No filter** — shows all recent jobs.
9. **Filter active** — shows only matching recent jobs.
10. **Filter active, no matching jobs** — shows project-specific empty state message.

### Acceptance Criteria

- [ ] All ten test cases pass.
- [ ] Filtering is purely client-side (no additional API calls when filter changes).

## Milestone 4 — Component Tests for Clickable Project Links

### Description

Verify that project names in all three queue components render as `<RouterLink>` elements with correct `to` attributes.

### Files to create/change

- `tests/web/components/queue/QueueRunningPanel.spec.ts`
- `tests/web/components/queue/QueuePendingTable.spec.ts`
- `tests/web/components/queue/QueueRecentTable.spec.ts`

### Test Cases

1. **Running panel link** — project name renders as `<RouterLink to="/p/my-project/dashboard">`.
2. **Pending table link** — project name renders as `<RouterLink to="/p/my-project/agents">`.
3. **Recent table link** — project name renders as `<RouterLink to="/p/my-project/agents">`.
4. **Link text** — link text matches `job.project` value exactly.
5. **Multiple jobs** — each row has its own correctly-targeted link.

### Acceptance Criteria

- [ ] All five test cases pass.
- [ ] Links use `<RouterLink>` (not `<a>` tags with href) for SPA navigation.

## Milestone 5 — Integration / E2E Tests for URL Sync and Full Flow

### Description

End-to-end tests that verify the full user flow: loading the queue page, selecting a project in the sidebar, observing filtered results, verifying URL query parameter sync, and navigating via project links.

### Files to create/change

- `tests/web/views/QueueView.spec.ts` (or E2E test file depending on test framework)

### Test Cases

1. **Mount with no query param** — sidebar shows "All Projects" selected, all queue sections show unfiltered data.
2. **Mount with `?project=valid-project`** — sidebar pre-selects that project, queue sections show only that project's jobs.
3. **Mount with `?project=nonexistent`** — falls back to "All Projects" without error, URL is cleaned up.
4. **Select project in sidebar** — URL updates to `?project=<name>` without full navigation (uses `router.replace`).
5. **Select "All Projects"** — `?project=` query param is removed from URL.
6. **Concurrent loading** — both project list and queue snapshot begin loading on mount; sidebar does not block queue content.
7. **Click project link in running panel** — navigates to `/p/:project/dashboard`.
8. **Click project link in pending table** — navigates to `/p/:project/agents`.
9. **Real-time update while filtered** — when a WebSocket event adds a job for a different project, it does not appear in the filtered view; switching to "All Projects" reveals it.

### Acceptance Criteria

- [ ] All nine test cases pass.
- [ ] Tests cover the golden path (select → filter → navigate) and edge cases (invalid param, empty queue, real-time updates).
- [ ] No regressions in existing queue test coverage.

## Cross-references

- [[queue-page-project-navigation]] backend plan — Milestone 1 (project field consistency) should be verified before running integration tests.
- [[queue-page-project-navigation]] frontend plan — test cases map directly to the acceptance criteria in each frontend milestone.
