---
title: Mobile Responsiveness — Follow-up Items
type: idea
status: draft
lineage: mobile-responsiveness-followups
created: "2026-06-01T00:00:00+10:00"
priority: medium
labels:
    - frontend
    - mobile
    - ux
assignees:
    - role: product-owner
      who: human
---

# Mobile Responsiveness — Follow-up Items

## Context

A first mobile-responsiveness pass landed across six milestones on
2026-06-01 (M1–M6). The pass established the foundation (breakpoint
tokens at 640 px / 1024 px, a `useViewport()` composable, global
mobile helpers in `main.css`), turned the persistent sidebar into a
drawer with a hamburger toggle, wrapped 11 data tables in
`.table-scroll`, made all 9 known modal overlay/panel pairs go
full-screen below 640 px, reflowed the editor split / Agents-Runs
header / Artifact-List header, and forced the 3D map view into 2D
on mobile with a slide-in filter panel.

This artefact captures the items that were deliberately deferred
or surfaced as gaps during that pass. They split roughly into
"things any mobile user will hit" and "polish".

## High-impact follow-ups

### Swipe gestures for the sidebar drawer
- Swipe-from-left to open, swipe-on-drawer to close. Hammer.js is
  heavyweight; a small pointerdown/pointermove handler in
  `AppSidebar.vue` is enough.
- Acceptance: drawer responds to a single-finger horizontal swipe
  inside the leftmost 20 px of the viewport (open) or anywhere on
  the drawer (close). Reduced-motion users still get the tap path.

### Cards-on-mobile for the run/artifact tables
- `AgentsRunsView` and `ArtifactListView` tables currently
  horizontal-scroll on mobile (M3 fix). For long-form tables that
  works but isn't great — a per-row card layout would be more
  readable.
- Approach: at `≤640px`, hide the `<table>` and render a list of
  per-row card components driven by the same data. Reuse existing
  status-pill / driver-badge components.

### Per-route mobile hardening sweep
- Only ~12% of Vue files had any `@media` queries pre-pass; the
  global helpers cover most of the surface but many feature views
  (TestingBoard, ReleasesView, DevOpsView, KanbanBoard, Roadmap)
  haven't had per-route attention. Walk through every route on a
  375 × 812 viewport and file a defect per cramped/broken layout.

### Kanban board on mobile
- The Kanban (`/p/<project>/artifacts/board`) is multi-column
  horizontal — fine on desktop, painful on phones. Options:
  (a) horizontal scroll the columns, (b) collapse to a status-tabs
  + single-column view on mobile, (c) bottom-sheet column switcher.
  Worth a separate idea.

### Roadmap Gantt on mobile
- `GanttChart.vue` is wide by nature. Decide between horizontal
  scroll (matches tables), zoom-out granularity (already exists
  via auto-coarsen), or a stacked release-list fallback on
  `≤640px`.

## Polish

- **Pull-to-refresh** on list views (`ArtifactListView`, `AgentsRunsView`,
  `QueueView`) so users can re-fetch without finding the kebab.
- **Long-press affordances** for batch-select on tables once card
  layout lands.
- **Better viewport hint for the 3D map**: when forced to 2D on
  mobile, surface a discreet "3D available on a larger screen" pill
  so users don't think the option went missing.
- **Test the dev devops/scheduler routes on mobile** — they're
  table-heavy but rarely used on phones; might be fine to leave
  pinned at "use a real screen" via a banner.
- **CSS audit for `position: fixed` `100vh` on iOS** — Safari's
  dynamic viewport (`100dvh` vs `100vh`) sometimes hides content
  under the address bar. The shell already uses `100vh`; consider
  `100dvh` with a `100vh` fallback for older browsers.
- **Mobile Safari rubber-band scroll inside the drawer** — verify
  the body doesn't scroll under the drawer (use
  `overscroll-behavior: contain` on the drawer if so).

## Testing

- Add Vitest-level snapshot tests for breakpoint-dependent behaviour
  (e.g., `AppHeader` shows the hamburger when `matchMedia` returns
  `<640px`).
- Add a Playwright flow under `tests/e2e/flows/` that drives the
  app at 375 × 812 viewport: open drawer, navigate to a route,
  close drawer, open a modal, dismiss. Covers the M2/M4 plumbing.

## Out of scope (split into separate ideas if pursued)

- Native wrapping via Capacitor / Tauri / PWA installability.
- Offline mode.
- Mobile-first re-skin (the current visual design is desktop-first
  and looks utilitarian on phones — that's a separate visual
  refresh, not a responsiveness gap).
