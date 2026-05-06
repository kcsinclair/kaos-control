---
title: Collapsible Sidebar with Icon-Only Mode
type: requirement
status: blocked
lineage: collapsible-sidebar-icons
created: "2026-04-28"
parent: lifecycle/ideas/collapsible-sidebar-icons.md
labels:
    - frontend
    - feature
    - usability
    - vue
assignees:
    - role: product-owner
      who: agent
---

# Collapsible Sidebar with Icon-Only Mode

## Problem

The left navigation sidebar in the project workspace is fixed at 220 px and always shows full text labels. On smaller screens or when working in the graph view or markdown editor, users cannot reclaim that horizontal space. There is no way to minimise the sidebar without losing navigation access entirely.

## Goals / Non-goals

### Goals

- Let the user collapse the sidebar to a narrow icon-only strip and expand it back to the full label view with a single click.
- Preserve full navigation functionality in the collapsed state.
- Persist the user's preference across page reloads and sessions.
- Add a recognisable icon to every navigation item so collapsed mode is usable.
- Animate the transition smoothly.

### Non-goals

- Completely hiding or auto-hiding the sidebar (it must always remain visible).
- Responsive / mobile-specific breakpoints or gestures (out of scope for this change; may be addressed separately).
- Reordering, adding, or removing navigation items — only the collapse behaviour and icon additions are in scope.
- Keyboard-shortcut toggle (nice-to-have, not required for initial delivery).

## Detailed Requirements

### Functional

1. **Toggle control** — A clickable affordance (chevron / arrow button) is rendered at the bottom of the sidebar. Clicking it toggles between expanded and collapsed states.
2. **Expanded state** — Identical to the current sidebar: 220 px wide, icon + text label per nav item, group headers visible.
3. **Collapsed state** — The sidebar shrinks to a narrow strip (48–56 px). Only icons are shown; text labels and group headers are hidden.
4. **Icons per nav item** — Every top-level and child nav link receives an icon from the `lucide-vue-next` library already in use:
   - Artefacts (group) — `LayoutList`
     - List — `List`
     - Board — `Columns3`
   - Graph — `Network`
   - Agents — `Bot`
   - Parse Errors — `AlertTriangle`
   - Config — `Settings`
5. **Tooltips in collapsed mode** — When collapsed, hovering over an icon shows a tooltip with the item's label. Tooltips must not appear in expanded mode (the label is already visible).
6. **Badge preservation** — The Parse Errors badge count must remain visible in both states. In collapsed mode it is rendered as a small indicator dot or super-positioned number on the icon.
7. **Project header** — In collapsed mode the project name and "Project" label are hidden. Optionally show a single-letter or abbreviated project avatar; otherwise the space is simply omitted.
8. **State persistence** — The expanded/collapsed preference is stored in `localStorage` (key: `sidebar-collapsed`). On mount, the sidebar reads this value and initialises in the correct state.
9. **Transition animation** — The width change uses a CSS transition (200–300 ms ease). Text opacity fades out before width shrinks and fades in after width expands so there is no text-wrapping flash.

### Non-functional

1. **No layout shift** — The main content area must fill the remaining horizontal space fluidly. Existing views (artifact list, graph, editor) must not break or require manual resize handling.
2. **Accessibility** — The toggle button must have `aria-label` ("Collapse sidebar" / "Expand sidebar") and `aria-expanded`. Icon-only links must have `aria-label` matching their text label. Tooltip implementation must be keyboard-accessible (visible on `:focus-visible`).
3. **Performance** — No additional network requests. Icons are tree-shaken from the existing `lucide-vue-next` package. Transition uses CSS `transform` / `width` — avoid JS-driven frame-by-frame animation.
4. **Browser support** — Works in the same set of evergreen browsers already supported by the app.

## Acceptance Criteria

- [ ] Sidebar renders a toggle button; clicking it collapses the sidebar to icon-only mode.
- [ ] Clicking the toggle again expands the sidebar back to full width with labels.
- [ ] Every nav item displays an appropriate icon in both expanded and collapsed states.
- [ ] Tooltips appear on hover/focus for each icon when collapsed; no tooltips in expanded mode.
- [ ] Parse Errors badge is visible in both states.
- [ ] Collapse/expand preference survives a full page reload.
- [ ] Transition between states is animated (no instant jump).
- [ ] Main content area adjusts fluidly — no overflow, scrollbar, or layout-break regressions in artifact list, board, graph, or editor views.
- [ ] Toggle button has correct `aria-expanded` and `aria-label` attributes.
- [ ] Collapsed icon-only links have `aria-label` attributes matching their visible labels.
- [ ] No new runtime warnings or errors in the browser console.

## Open Questions

1. Should the collapsed sidebar support a hover-to-temporarily-expand interaction (i.e., hovering the collapsed strip reveals the full sidebar as an overlay without toggling the persisted state)?

> Yes.

2. Should group children (List, Board under "Artefacts") be shown as a nested flyout in collapsed mode, or should they appear as individual icons in the strip?

> Individual icons, only show clickable things.

3. Is there a preferred icon for the project header area in collapsed mode (e.g., first-letter avatar), or should it simply be hidden?

> Use the Favicon scaled to fit. web/dist/assets/favicon-32x32.png or 16x16
