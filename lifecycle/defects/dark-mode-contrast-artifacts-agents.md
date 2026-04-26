---
title: 'Dark Mode: Unreadable Black Text on Dark Backgrounds and Low-Contrast Stage Pills'
type: defect
status: done
lineage: dark-mode-contrast-artifacts-agents
priority: normal
labels:
    - defect
    - frontend
    - usability
    - vue
---

# Dark Mode: Unreadable Black Text on Dark Backgrounds and Low-Contrast Stage Pills

## Reproduction Steps

1. Enable dark mode in the application.
2. Navigate to the `/artifacts` screen.
3. Observe the page heading.
4. Navigate to the `Agents` screen.
5. Observe the column that identifies which agent is assigned.
6. On either screen, observe the stage pills.

## Expected Behaviour

- All headings, labels, and column text should use a light foreground colour that provides sufficient contrast against the dark background, meeting WCAG AA contrast ratio requirements.
- Stage pills should use clearly readable text and background colour combinations in dark mode so that the stage label is immediately legible.

## Actual Behaviour

- On the `/artifacts` screen, the page heading renders in black, making it effectively invisible against the dark background.
- On the `Agents` screen, the agent name column text renders in black against the dark background, making it unreadable.
- Stage pills lack sufficient contrast in dark mode, making the pill labels difficult to read.
