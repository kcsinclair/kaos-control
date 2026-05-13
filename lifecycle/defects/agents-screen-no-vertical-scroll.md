---
title: 'Agents Screen: No Vertical Scroll When Content Exceeds Viewport'
type: defect
status: draft
lineage: agents-screen-no-vertical-scroll
created: "2026-05-13T11:51:09+10:00"
priority: normal
labels:
    - defect
    - frontend
    - usability
    - vue
---

# Agents Screen: No Vertical Scroll When Content Exceeds Viewport

## Reproduction Steps

1. Navigate to the Agents screen in the application.
2. Ensure there are enough agents listed to exceed the visible screen height.
3. Observe the page layout.

## Expected Behaviour

When the list of agents exceeds the available viewport height, a vertical scrollbar should appear (or the container should be scrollable), allowing the user to scroll down and view all agents.

## Actual Behaviour

No vertical scrollbar is present on the Agents screen. Content that extends beyond the bottom of the viewport is inaccessible — the user cannot scroll down to see or interact with agents below the fold.
