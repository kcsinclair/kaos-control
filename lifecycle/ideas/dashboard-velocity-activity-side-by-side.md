---
title: 'Dashboard: Completion Velocity and Recent Activity Side by Side'
type: idea
status: clarifying
lineage: dashboard-velocity-activity-side-by-side
created: "2026-05-10T16:06:01+10:00"
priority: normal
labels:
    - frontend
    - enhancement
    - usability
    - vue
    - testing
release: KC-Release0
---

# Dashboard: Completion Velocity and Recent Activity Side by Side

Currently the completion velocity chart and the recent activity panel are stacked vertically on the dashboard, requiring the user to scroll to see both. They should be rendered side by side in a two-column layout so that all key information is visible on a single screen without scrolling.

The Vue dashboard component should be updated to use a responsive two-column grid or flex layout that places these two panels adjacent to each other. Breakpoints should be chosen so the layout degrades gracefully on smaller viewports while prioritising the single-screen experience on typical desktop resolutions.

All affected frontend components must have their corresponding tests updated to reflect the new layout structure, and any existing Playwright or integration tests that assert on dashboard element positions or visibility must be revised to match the new arrangement.
