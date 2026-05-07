---
title: Collapsible Left Menu with Icon-Only Mode
type: idea
status: done
lineage: collapsible-sidebar-icons
created: "2026-04-28T14:12:22+10:00"
priority: normal
labels:
    - frontend
    - feature
    - usability
    - vue
release: April2026
---

# Collapsible Left Menu with Icon-Only Mode

Add a toggle control to the left navigation menu that allows the user to collapse it from its full label-and-icon view down to a narrow icon-only strip. This gives users more horizontal screen real estate for the main content area, particularly useful when working in the graph view or editor on smaller screens.

The collapsed state should preserve all navigation functionality — each icon remains clickable and ideally shows a tooltip on hover to compensate for the hidden labels. The expanded/collapsed preference should be persisted (localStorage or Pinia store) so it survives page reloads.

A subtle toggle affordance (e.g. a chevron or arrow button at the bottom or edge of the sidebar) should trigger the transition, with a smooth CSS animation between states.
