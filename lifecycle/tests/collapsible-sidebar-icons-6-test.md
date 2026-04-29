---
title: Collapsible Sidebar Icons — Integration Tests
type: test
status: draft
lineage: collapsible-sidebar-icons
parent: lifecycle/test-plans/collapsible-sidebar-icons-5-test.md
---

# Collapsible Sidebar Icons — Integration Tests

Tests for the collapsible sidebar feature, covering toggle behaviour, icon
rendering, tooltip accessibility, badge preservation, localStorage persistence,
hover-to-expand overlay, layout integrity across views, and CSS animation classes.

All test files live under `tests/web/` and run with Vitest + `@vue/test-utils`
+ happy-dom.

## Test infrastructure

| File | Purpose |
|------|---------|
| `tests/web/AppSidebar.test.ts` | All sidebar collapse integration tests (59 tests) |
| `tests/web/vitest.config.ts` | Vitest + Vue plugin + `@` path alias to `web/src/` |

Run the suite:
```sh
cd tests/web && pnpm install && pnpm test
```

### Mocks

| Module | Mock behaviour |
|--------|---------------|
| `@/api/client` | `api.get` returns `{ errors: [] }` by default; individual tests override with parse errors |
| `@/api/ws` | `getProjectWs` returns a stub with a no-op `onType` function |
| `@/stores/project` | `useProjectStore` returns `{ current: { name: 'Test Project' } }` |

---

## Milestone 1 — Toggle Behaviour

**File:** `tests/web/AppSidebar.test.ts` — describe block `AppSidebar — Milestone 1: toggle behaviour`

### Scenarios covered

| Scenario | Assertion |
|----------|-----------|
| Default state | Sidebar starts expanded (no `sidebar--collapsed` class) |
| Collapse on click | Clicking toggle adds `sidebar--collapsed` class |
| Expand on second click | Second click removes `sidebar--collapsed` class |
| ChevronLeft shown when expanded | Toggle button contains an SVG; `uiStore.sidebarCollapsed` is `false` |
| ChevronRight shown when collapsed | Toggle button contains an SVG; `uiStore.sidebarCollapsed` is `true` |
| `aria-expanded="true"` when expanded | Toggle button attribute reflects open state |
| `aria-expanded="false"` when collapsed | Toggle button attribute reflects collapsed state |
| `aria-label="Collapse sidebar"` when expanded | Descriptive label present |
| `aria-label="Expand sidebar"` when collapsed | Descriptive label present |
| aria-label updates after click | Clicking toggles the label text |

---

## Milestone 2 — Icon Rendering

**File:** `tests/web/AppSidebar.test.ts` — describe block `AppSidebar — Milestone 2: icon rendering`

### Scenarios covered

| Scenario | Assertion |
|----------|-----------|
| SVG icons in expanded mode | Each of the 6 nav items has an `<svg>` element |
| SVG icons in collapsed mode | SVG icons present in all 6 nav items when collapsed |
| Nav label text in expanded mode | All 6 labels (List, Board, Graph, Agents, Parse Errors, Config) visible |
| Nav labels hidden via CSS when collapsed | `.sidebar--collapsed` class present; `.nav-label` elements remain in DOM (hidden by CSS, not `v-if`) |
| No "Artefacts" group header | `wrapper.text()` does not contain "Artefacts" |
| All 6 nav items rendered | Each expected label appears in the rendered text |

---

## Milestone 3 — Tooltip Behaviour

**File:** `tests/web/AppSidebar.test.ts` — describe block `AppSidebar — Milestone 3: tooltip behaviour`

### Scenarios covered

| Scenario | Assertion |
|----------|-----------|
| Collapsed links have `aria-label` | Each `.nav-link` has a truthy `aria-label` when collapsed |
| Expanded links lack `aria-label` | Each `.nav-link` has no `aria-label` when expanded |
| Tooltip on `mouseenter` (collapsed) | `.sidebar-tooltip` appears in `document.body` |
| Tooltip removed on `mouseleave` | `.sidebar-tooltip` absent after `mouseleave` |
| No tooltip on `mouseenter` (expanded) | `.sidebar-tooltip` absent when sidebar is expanded |
| Tooltip on `focusin` (keyboard, collapsed) | `.sidebar-tooltip` appears in `document.body` |
| Tooltip removed on `focusout` | `.sidebar-tooltip` absent after `focusout` |
| `aria-label` values match nav labels | Each collapsed link's `aria-label` equals its display text |

---

## Milestone 4 — Badge Preservation

**File:** `tests/web/AppSidebar.test.ts` — describe block `AppSidebar — Milestone 4: badge preservation`

### Scenarios covered

| Scenario | Assertion |
|----------|-----------|
| Expanded badge visible | `.badge` element present when parse errors > 0 and sidebar is expanded |
| Collapsed badge-dot visible | `.badge-dot` element present when parse errors > 0 and sidebar is collapsed |
| Badge-dot aria-label | `aria-label` on `.badge-dot` includes the error count |
| No badge when no errors | Neither `.badge` nor `.badge-dot` rendered when error count is 0 |
| Badge type swaps on toggle | Collapsing switches from `.badge` to `.badge-dot`; expanding reverses it |

---

## Milestone 5 — State Persistence

**File:** `tests/web/AppSidebar.test.ts` — describe block `AppSidebar — Milestone 5: localStorage persistence`

### Scenarios covered

| Scenario | Assertion |
|----------|-----------|
| Collapse writes `true` | `localStorage.getItem('sidebar-collapsed')` equals `'true'` after collapse |
| Expand writes `false` | `localStorage.getItem('sidebar-collapsed')` equals `'false'` after expand |
| Initialises collapsed from storage | Mounts with `sidebar-collapsed=true` → `sidebar--collapsed` class present |
| Initialises expanded from storage | Mounts with `sidebar-collapsed=false` → no `sidebar--collapsed` class |
| Two toggles return to `false` | localStorage reflects final state after two clicks |
| Fresh mount reads persisted state | Unmounting and remounting reads collapsed state from localStorage |

---

## Milestone 6 — Hover-to-Expand Overlay

**File:** `tests/web/AppSidebar.test.ts` — describe block `AppSidebar — Milestone 6: hover-to-expand overlay`

Uses `vi.useFakeTimers()` to control the 200 ms delay.

### Scenarios covered

| Scenario | Assertion |
|----------|-----------|
| Overlay appears after 200 ms | `sidebar--overlay` class added after `vi.advanceTimersByTime(200)` |
| No overlay before 200 ms | Class absent at 150 ms |
| `mouseleave` removes overlay | `sidebar--overlay` class removed after leaving |
| Collapsed class absent during overlay | Template binding `collapsed && !hoverExpanded` is false during hover-expand |
| localStorage unchanged during hover | `sidebar-collapsed` remains `'true'` during hover-expand |
| Hover does nothing when already expanded | `sidebar--overlay` not added when sidebar is not collapsed |
| Labels visible during hover-expand | `.nav-label` elements rendered while overlay is active |

---

## Milestone 7 — Layout Integrity Across Views

**File:** `tests/web/AppSidebar.test.ts` — describe block `AppSidebar — Milestone 7: layout integrity`

Iterates over all six project routes: `/artifacts`, `/artifacts/board`, `/graph`, `/agents`, `/parse-errors`, `/config`.

### Scenarios covered

| Scenario | Assertion |
|----------|-----------|
| Collapsed on each route | `sidebar--collapsed` class present on `nav.app-sidebar` |
| Expanded on each route | No `sidebar--collapsed` class |
| 6 nav links on every route | `wrapper.findAll('.nav-link').length === 6` |

---

## Milestone 8 — Animation Quality

**File:** `tests/web/AppSidebar.test.ts` — describe block `AppSidebar — Milestone 8: animation / CSS transition`

happy-dom does not compute CSS; tests verify the sequenced animation-direction
classes that carry the transition rules.

### Scenarios covered

| Scenario | Assertion |
|----------|-----------|
| Transition CSS or class present | Inline `transition` style contains `"width"`, or `sidebar--collapsing` class is applied after toggling |
| `sidebar--collapsing` on collapse | Class added immediately when sidebar is collapsed |
| `sidebar--expanding` on expand | Class added immediately when sidebar is expanded |
| Mutual exclusion | `sidebar--collapsing` and `sidebar--expanding` are never both present simultaneously |
