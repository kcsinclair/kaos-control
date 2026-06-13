---
title: Mobile-Responsive UI
type: idea
status: done
lineage: mobile-responsiveness
created: "2026-06-01T00:00:00+10:00"
priority: medium
labels:
    - frontend
    - mobile
    - ux
    - feature
release: KC-Release3
---

# Mobile-Responsive UI

> Backfilled idea — the M1–M6 responsiveness pass shipped before a formal idea
> artifact was raised. Outstanding polish lives in [[mobile-responsiveness-followups]].

## Idea

Make the Vue SPA usable on phones. Baseline before the work: only ~12% of Vue
files had any `@media` rules, the sidebar was a fixed 220 px column stealing 58%
of a 375 px viewport, 11 data tables had no overflow handling, 15+ modals were
centred desktop dialogs, and the 3D map was effectively unusable on phones.

## What shipped (Milestones 1–6)

- **M1 — Responsive foundation:** `--bp-mobile: 640px` / `--bp-tablet: 1024px`
  tokens, a `useViewport()` composable, and global mobile helpers (touch-target
  floor, iOS-Safari font-size bump, `.table-scroll` utility).
- **M2 — App shell:** sidebar becomes an overlay drawer below 640 px with a
  header hamburger; ESC, route-change, and backdrop-tap dismiss.
- **M3 — Tables:** all 11 data tables wrapped in `.table-scroll`.
- **M4 — Modals:** go full-screen below 640 px.
- **M5 — Reflow:** editor split stacks vertically; AgentsRuns and ArtifactList
  headers wrap.
- **M6 — Map:** 3D map forced to 2D on mobile; the 200 px filter rail becomes a
  slide-in panel with a toggle.

## References

- PROJECT_PLAN rolling log — 2026-06-01 (M1–M6 pass; commits `c94c31ed`,
  `929579f2`, `4f3dfbf4`, `d206f34f`, `48db3dff`, `96c44386`). See
  [plans/PROJECT_PLAN.md](../../plans/PROJECT_PLAN.md).
- Release notes: [RELEASE_NOTES-0.1.3.md](../../RELEASE_NOTES-0.1.3.md) —
  "Mobile-responsive UI".
- Follow-up items: [[mobile-responsiveness-followups]].
