---
title: "Test Plan — Collapsible Sidebar with Icon-Only Mode"
type: plan-test
status: done
lineage: collapsible-sidebar-icons
parent: lifecycle/requirements/collapsible-sidebar-icons-2.md
---

# Test Plan — Collapsible Sidebar with Icon-Only Mode

## Overview

Verify the collapsible sidebar feature end-to-end, covering toggle behaviour, icon rendering, tooltip accessibility, badge preservation, localStorage persistence, hover-to-expand overlay, animation smoothness, and layout integrity across all workspace views.

Tests will be written as integration tests in the `tests/` directory, targeting the running application in a browser via the existing test framework.

---

## Milestone 1 — Toggle Behaviour

**Description:** Verify the sidebar toggle button collapses and expands the sidebar correctly.

**Files to change:**
- `tests/sidebar_collapse_test.go` (or equivalent test file matching project conventions) — New test file for sidebar collapse feature.
- `lifecycle/tests/collapsible-sidebar-icons-test.md` — Test artifact describing coverage.

**Acceptance criteria:**
- [ ] Test clicks the toggle button and asserts the sidebar width changes from 220 px to 48 px.
- [ ] Test clicks the toggle again and asserts the sidebar returns to 220 px.
- [ ] Test asserts the toggle button icon changes between `ChevronLeft` and `ChevronRight`.
- [ ] Test asserts `aria-expanded` attribute toggles between `"true"` and `"false"`.
- [ ] Test asserts `aria-label` updates ("Collapse sidebar" ↔ "Expand sidebar").

---

## Milestone 2 — Icon Rendering

**Description:** Verify every navigation item displays the correct icon in both states.

**Files to change:**
- `tests/sidebar_collapse_test.go` — Add icon rendering tests.

**Acceptance criteria:**
- [ ] Test asserts each nav item (List, Board, Graph, Agents, Parse Errors, Config) renders an SVG icon element.
- [ ] In expanded mode, both icon and text label are visible for each nav item.
- [ ] In collapsed mode, icon is visible and text label is hidden (not in DOM or `display: none` / `opacity: 0`).
- [ ] The "Artefacts" group header is not rendered — List and Board are top-level.

---

## Milestone 3 — Tooltip Behaviour

**Description:** Verify tooltips appear in collapsed mode and not in expanded mode.

**Files to change:**
- `tests/sidebar_collapse_test.go` — Add tooltip tests.

**Acceptance criteria:**
- [ ] Test collapses sidebar, hovers over a nav icon, and asserts a tooltip element appears with the correct label text.
- [ ] Test moves hover away and asserts the tooltip disappears.
- [ ] Test expands sidebar, hovers over a nav item, and asserts no tooltip appears.
- [ ] Test verifies tooltip is keyboard-accessible: focusing an icon via Tab shows the tooltip.
- [ ] Test asserts each collapsed nav link has an `aria-label` attribute matching its visible label.

---

## Milestone 4 — Badge Preservation

**Description:** Verify the Parse Errors badge is visible in both collapsed and expanded states.

**Files to change:**
- `tests/sidebar_collapse_test.go` — Add badge tests.

**Acceptance criteria:**
- [ ] Test navigates to a project with parse errors, asserts badge count is visible in expanded mode.
- [ ] Test collapses sidebar, asserts badge indicator is still visible on the Parse Errors icon.
- [ ] Badge does not overflow or clip outside the sidebar boundary.

---

## Milestone 5 — State Persistence

**Description:** Verify the collapsed/expanded preference survives page reloads.

**Files to change:**
- `tests/sidebar_collapse_test.go` — Add persistence tests.

**Acceptance criteria:**
- [ ] Test collapses sidebar, reloads the page, and asserts the sidebar is still collapsed.
- [ ] Test expands sidebar, reloads the page, and asserts the sidebar is still expanded.
- [ ] Test asserts `localStorage` key `sidebar-collapsed` is set to the correct value after toggling.

---

## Milestone 6 — Hover-to-Expand Overlay

**Description:** Verify the hover-to-expand overlay behaviour in collapsed mode.

**Files to change:**
- `tests/sidebar_collapse_test.go` — Add hover-expand tests.

**Acceptance criteria:**
- [ ] Test collapses sidebar, hovers over the sidebar area, and asserts it expands to full width as an overlay after ~200 ms.
- [ ] Test asserts the overlay shows full labels and icons.
- [ ] Test moves hover away and asserts the sidebar returns to collapsed state.
- [ ] Test asserts main content does not shift during hover-expand (overlay, not push).
- [ ] Test asserts the `localStorage` value does not change during hover-expand.

---

## Milestone 7 — Layout Integrity Across Views

**Description:** Verify that collapsing the sidebar does not break layout in any workspace view.

**Files to change:**
- `tests/sidebar_collapse_test.go` — Add layout regression tests.

**Acceptance criteria:**
- [ ] Test collapses sidebar and navigates to each view (artifact list, board, graph, editor, agents, parse errors, config).
- [ ] For each view: asserts no horizontal overflow, no scrollbar regression, and the main content area fills the remaining width.
- [ ] Test expands sidebar and repeats the same checks.
- [ ] No new console errors or warnings are emitted during navigation in either state.

---

## Milestone 8 — Animation Quality

**Description:** Verify the collapse/expand transition is animated (not an instant jump).

**Files to change:**
- `tests/sidebar_collapse_test.go` — Add animation verification test.

**Acceptance criteria:**
- [ ] Test asserts the sidebar element has a CSS `transition` property that includes `width`.
- [ ] Test captures sidebar width at multiple points during the transition and asserts it is neither the start nor end value (i.e., an intermediate width exists).
- [ ] No text-wrapping flash during transition — text opacity transitions before/after width.

---

## Cross-links

- [[collapsible-sidebar-icons]] frontend plan defines the implementation these tests verify.
- [[collapsible-sidebar-icons]] backend plan confirms no API changes to test.
