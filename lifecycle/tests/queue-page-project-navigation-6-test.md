---
title: "Tests — Queue Page Project Navigation"
type: test
status: approved
lineage: queue-page-project-navigation
parent: lifecycle/test-plans/queue-page-project-navigation-5-test.md
created: "2026-05-13T00:00:00+10:00"
---

# Tests — Queue Page Project Navigation

Implements the full test specification from the companion test plan. Tests cover
the QueueSidebar job-count logic, sidebar component rendering, project-filter
behaviour in all three queue panels, RouterLink targets, and the QueueView URL
sync and full navigation flow.

---

## Suite overview

All tests live in the `tests/web/` directory and run with Vitest + @vue/test-utils
under happy-dom. No Go integration tests were required — the feature is purely
frontend.

---

## Milestone 1 — Job Count Logic (`QueueSidebar.test.ts`)

**File**: `tests/web/QueueSidebar.test.ts` — describe block
"QueueSidebar — Milestone 1: job count logic"

| Case | ID  | Scenario |
|------|-----|----------|
| M1-1 | No jobs | All project badges show 0 |
| M1-2 | One pending job | Badge for that project shows 1, others 0 |
| M1-3 | Running + pending same project | Badge shows combined count (2) |
| M1-4 | Running for A, pending for B | Each project shows 1 |
| M1-5 | Multiple pending for one project | Badge shows correct aggregate |
| M1-6 | Recent (finished) jobs excluded | Completed/failed/skipped/cancelled not counted |

The `jobCounts` computed is tested indirectly through the rendered `.item-badge`
text values. The "All Projects" button's badge reflects `totalCount` (sum of all
per-project counts).

**Status**: All 6 cases pass.

---

## Milestone 2 — QueueSidebar Component Tests (`QueueSidebar.test.ts`)

**File**: `tests/web/QueueSidebar.test.ts` — describe block
"QueueSidebar — Milestone 2: rendering and behaviour"

| Case | ID    | Scenario |
|------|-------|----------|
| M2-1 | Render | Item per project + "All Projects" |
| M2-2 | Default | "All Projects" has `aria-current="page"` on mount |
| M2-3 | Click project | Clicked item gains `aria-current`; previous loses it |
| M2-4 | Click All | Clears selection back to "All Projects" |
| M2-5 | Collapse | Toggle hides nav (`v-if`); click again restores it |
| M2-6 | Keyboard | Items are `<button>` (natively tabbable); click fires selection |
| M2-7 | ARIA | `<nav>` has `role="navigation"` + `aria-label`; toggle has `aria-expanded` |

Additional cases cover zero-project state, `select`/`update:modelValue` event
emissions, reactive badge updates, and `fetchProjects()` invocation on mount.

`window.matchMedia` is stubbed globally (happy-dom provides none) to prevent
crashes during QueueSidebar setup.

**Status**: All 14 cases pass.

---

## Milestone 3 — Filtered Queue Components

### QueueRunningPanel (`tests/web/QueueRunningPanel.filter.test.ts`)

**File**: `tests/web/QueueRunningPanel.filter.test.ts` — describe block
"QueueRunningPanel — Milestone 3: project filter"

| Case | Scenario |
|------|----------|
| M3-1 | No filter → shows running job regardless of project |
| M3-2 | Filter matches → shows the job |
| M3-3 | Filter does not match → "Nothing running" empty state |
| M3-4 | No running job + filter active → "Nothing running" |
| Extra | Prop change triggers no API call (client-side only) |

**Status**: All 5 cases pass.

### QueuePendingTable (`tests/web/QueuePendingTable.filter.test.ts`)

| Case | Scenario |
|------|----------|
| M3-5 | No filter → shows all pending jobs |
| M3-6 | Filter active → only matching jobs shown |
| M3-7 | Filter active, no match → project-specific empty message |
| Extra | Generic empty message when unfiltered queue is empty |
| Extra | Prop change triggers no API call |

**Status**: All 5 cases pass.

### QueueRecentTable (`tests/web/QueueRecentTable.filter.test.ts`)

| Case | Scenario |
|------|----------|
| M3-8  | No filter → shows all recent jobs |
| M3-9  | Filter active → only matching recent jobs shown |
| M3-10 | Filter active, no match → project-specific empty message |
| Extra | Generic "No recent jobs" when unfiltered list is empty |
| Extra | Prop change triggers no API call |

**Status**: All 5 cases pass.

---

## Milestone 4 — Clickable Project Links

Covered in the same three filter test files, each with a dedicated describe block
"Milestone 4: project links".

| Component | Target path |
|-----------|-------------|
| QueueRunningPanel | `/p/:project/dashboard` |
| QueuePendingTable | `/p/:project/agents` |
| QueueRecentTable  | `/p/:project/agents` |

Additional checks: link text equals `job.project` exactly; multiple jobs each get
their own link; href is a relative path (RouterLink, not raw `<a>`).

**Status**: All link cases pass for all three components.

---

## Milestone 5 — URL Sync and Full Flow (`QueueView.projectNav.test.ts`)

**File**: `tests/web/QueueView.projectNav.test.ts` — describe block
"QueueView — Milestone 5: project navigation and URL sync"

| Case  | Scenario | Status |
|-------|----------|--------|
| M5-1  | No query param → "All Projects" active, unfiltered | ✓ pass |
| M5-2  | `?project=valid` → sidebar pre-selects, queue filtered | ✓ pass |
| M5-3  | `?project=nonexistent` → falls back to All Projects | ✓ pass |
| M5-3† | URL cleaned up after invalid param | ✗ **fail (gap)** |
| M5-4  | Sidebar click → URL updated to `?project=<name>` | ✓ pass |
| M5-4  | URL update uses `router.replace` (no new history entry) | ✓ pass |
| M5-5  | "All Projects" click → `?project=` removed from URL | ✓ pass |
| M5-6  | Both `queueStore.fetch` and `fetchProjects` called on mount | ✓ pass |
| M5-6  | Queue content visible while projects still loading | ✓ pass |
| M5-7  | Running panel project link → `/p/:project/dashboard` | ✓ pass |
| M5-8  | Pending table project link → `/p/:project/agents` | ✓ pass |
| M5-9  | New WS job for different project hidden while filtered | ✓ pass |
| M5-9  | Switching to All Projects reveals the new job | ✓ pass |
| –     | No regressions: Running/Pending/Recently finished sections | ✓ pass |

**†** The M5-3 URL-cleanup assertion (`expect(query.project).toBeUndefined()`)
currently fails. The implementation falls back to "All Projects" correctly but
does not call `router.replace({})` to strip the invalid `?project=` param.
This test documents the gap and will pass once the feature is implemented.

---

## Implementation notes

- `window.matchMedia` is stubbed in `beforeAll` for all test files that mount
  `QueueSidebar` or `QueueView` (which embeds it). happy-dom does not provide
  this API and QueueSidebar calls it synchronously at component setup.
- Stores are mocked with module-level reactive `ref`s (same pattern as
  `QueueView.test.ts`) so tests can mutate snapshot state to simulate WS events
  without touching the real Pinia store.
- Both `QueueView` and `QueueSidebar` call `projectStore.fetchProjects()` on
  mount; M5-6 asserts `toHaveBeenCalled()` (not `toHaveBeenCalledOnce()`) to
  accommodate both calls through the shared mock.
