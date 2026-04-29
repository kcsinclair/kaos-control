---
title: "Frontend Plan — Collapsible Sidebar with Icon-Only Mode"
type: plan-frontend
status: in-development
lineage: collapsible-sidebar-icons
parent: lifecycle/requirements/collapsible-sidebar-icons-2.md
---

# Frontend Plan — Collapsible Sidebar with Icon-Only Mode

## Overview

Modify `AppSidebar.vue` and supporting files to add a toggle that collapses the sidebar to a 48 px icon-only strip, with hover-to-expand overlay, tooltips, icon-per-nav-item, badge preservation, localStorage persistence, and animated transitions. No new dependencies — uses `lucide-vue-next` (already installed) and native CSS transitions.

---

## Milestone 1 — Sidebar Collapse State & Toggle Button

**Description:** Add a reactive `collapsed` state backed by `localStorage`, expose a toggle button, and wire the sidebar width between 220 px (expanded) and 48 px (collapsed).

**Files to change:**
- `web/src/stores/ui.ts` — Add `sidebarCollapsed` ref, `toggleSidebar()` action, and `localStorage` read/write (key: `sidebar-collapsed`).
- `web/src/components/layout/AppSidebar.vue` — Import and consume the store state; add a toggle `<button>` at the bottom of the sidebar with `ChevronLeft` / `ChevronRight` icon; bind `width` via class (`sidebar--collapsed`).
- `web/src/styles/tokens.css` — Add `--sidebar-width-expanded: 220px` and `--sidebar-width-collapsed: 48px` tokens.

**Acceptance criteria:**
- [ ] Clicking the toggle button switches the sidebar between 220 px and 48 px.
- [ ] The collapsed/expanded preference persists across full page reloads.
- [ ] Toggle button renders `ChevronLeft` when expanded, `ChevronRight` when collapsed.
- [ ] Toggle button has `aria-label` ("Collapse sidebar" / "Expand sidebar") and `aria-expanded` reflecting current state.

---

## Milestone 2 — Icons on Every Nav Item

**Description:** Add a Lucide icon to each navigation link. Icons are visible in both expanded and collapsed states; in expanded mode they sit left of the label.

**Files to change:**
- `web/src/components/layout/AppSidebar.vue` — Extend the `NavItem` interface with an `icon` field (Vue component type). Populate `navItems()` with the icons specified in the requirement:
  - List → `List`
  - Board → `Columns3`
  - Graph → `Network`
  - Agents → `Bot`
  - Parse Errors → `AlertTriangle`
  - Config → `Settings`
- Remove group headers in the rendered list (requirement answer: "only show clickable things" — List and Board appear as individual top-level icons, no "Artefacts" group label).

**Acceptance criteria:**
- [ ] Every nav link displays the specified Lucide icon in expanded mode (icon + label).
- [ ] In collapsed mode, only the icon is visible; text label is hidden.
- [ ] "Artefacts" group header is not rendered; List and Board are top-level items.
- [ ] Icons are 18–20 px and visually centred within their nav link.

---

## Milestone 3 — Collapsed State Rendering

**Description:** When collapsed, hide text labels, hide the project header text, and show the favicon in the header area. Ensure layout adjusts correctly.

**Files to change:**
- `web/src/components/layout/AppSidebar.vue` — Conditionally hide `.project-label`, `.project-name`, nav-link text spans, and group labels via a `v-show` or CSS class. Show an `<img>` of the favicon (`/assets/favicon-32x32.png`) in the project header area when collapsed.
- `web/src/views/project/WorkspaceView.vue` — Ensure `.workspace-body` uses flex layout so the main content area expands fluidly when the sidebar shrinks. No hardcoded widths on the main area.

**Acceptance criteria:**
- [ ] In collapsed state, only icons and the favicon are visible — no text.
- [ ] Favicon renders in the project header area at ~24 px, visually centred.
- [ ] Main content area fills remaining width with no overflow or layout break in artifact list, board, graph, or editor views.
- [ ] No horizontal scrollbar appears on the page.

---

## Milestone 4 — Tooltips in Collapsed Mode

**Description:** Show a tooltip with the nav item's label on hover/focus when collapsed. Tooltips must not appear in expanded mode.

**Files to change:**
- `web/src/components/ui/SidebarTooltip.vue` — New component. A lightweight positioned tooltip that renders to the right of the hovered icon. Uses CSS `position: fixed` anchored to the element's bounding rect. Shows on `mouseenter` / `focusin`, hides on `mouseleave` / `focusout`. No external dependencies.
- `web/src/components/layout/AppSidebar.vue` — Wrap each nav link with `<SidebarTooltip>` (or use it as a wrapper), passing the label as content. Disable tooltip rendering when `!collapsed`.

**Acceptance criteria:**
- [ ] Hovering an icon in collapsed mode shows the item's label as a tooltip to the right.
- [ ] Tooltip is keyboard-accessible — appears on `:focus-visible`.
- [ ] No tooltips appear in expanded mode.
- [ ] Tooltip has readable contrast and a subtle background/border.
- [ ] Each icon-only link has `aria-label` matching its text label.

---

## Milestone 5 — Badge Preservation in Collapsed Mode

**Description:** The Parse Errors badge count must remain visible in collapsed mode, rendered as a small super-positioned indicator on the icon.

**Files to change:**
- `web/src/components/layout/AppSidebar.vue` — When collapsed and `parseErrorCount > 0`, render the badge as a small dot or number positioned at the top-right of the `AlertTriangle` icon using `position: absolute` on a wrapper.

**Acceptance criteria:**
- [ ] Parse Errors badge is visible in both expanded and collapsed states.
- [ ] In collapsed mode, the badge renders as a small indicator (dot if count ≤ 9, number if larger) at the top-right of the icon.
- [ ] Badge does not overflow or clip outside the sidebar.

---

## Milestone 6 — Hover-to-Expand Overlay

**Description:** When the sidebar is collapsed, hovering over it temporarily reveals the full-width sidebar as an overlay (without changing the persisted collapsed state).

**Files to change:**
- `web/src/components/layout/AppSidebar.vue` — Add `mouseenter`/`mouseleave` handlers on the sidebar element. When collapsed and hovered, apply a class that expands width to 220 px, positions the sidebar with `position: absolute; z-index` so it overlays the content rather than pushing it. Use a short delay (~200 ms) on `mouseenter` to avoid accidental triggers. On `mouseleave`, revert to collapsed width.

**Acceptance criteria:**
- [ ] Hovering the collapsed sidebar for ~200 ms reveals the full sidebar as an overlay.
- [ ] The overlay shows full labels, icons, and badges (identical to expanded state).
- [ ] Moving the mouse away collapses the overlay back.
- [ ] The persisted `sidebar-collapsed` state does not change during hover-expand.
- [ ] Main content does not shift or reflow during hover-expand (overlay sits on top).

---

## Milestone 7 — Transition Animation

**Description:** Animate the collapse/expand transition with CSS. Text fades before width shrinks; text fades in after width expands.

**Files to change:**
- `web/src/components/layout/AppSidebar.vue` — Add CSS `transition: width 250ms ease` on `.app-sidebar`. Use a separate `transition: opacity 100ms ease` on text elements, sequenced via `transition-delay` so opacity completes before width begins (collapse) or starts after width completes (expand).

**Acceptance criteria:**
- [ ] Collapse: text fades out, then width shrinks — no text-wrapping flash.
- [ ] Expand: width grows, then text fades in — no text-wrapping flash.
- [ ] Total transition feels smooth at ~250 ms.
- [ ] Hover-to-expand overlay also animates smoothly.
- [ ] No JS-driven frame-by-frame animation; pure CSS transitions.

---

## Cross-links

- [[collapsible-sidebar-icons]] backend plan confirms no API changes needed.
- [[collapsible-sidebar-icons]] test plan covers integration tests for toggle, persistence, tooltip, badge, and animation behaviour.
