---
title: Move Running Agents Indicator to Menu Bar
type: idea
status: draft
lineage: agents-indicator-in-menu-bar
priority: normal
labels:
    - frontend
    - enhancement
    - usability
    - agent
    - vue
---

# Move Running Agents Indicator to Menu Bar

Currently the 'Agents running' pill is displayed in the bottom-right corner of the screen, where it can be obscured by other UI elements and is easy to miss. Moving this indicator into the menu bar will make it persistently visible regardless of which view or panel the user is in.

The menu bar indicator should display as 'X running agents' (e.g. '2 running agents', '1 running agent') and update in real time as agents start and stop. When no agents are running the indicator should either be hidden or shown in a neutral/inactive state to avoid visual noise.

This change improves at-a-glance awareness of agent activity, which is important for users who trigger agent runs and then navigate away to other parts of the UI — they can now always see agent status without hunting for the pill.
