---
title: 'Agent Panel: Show Ready Item Count and Running State'
type: idea
status: approved
lineage: agent-panel-status-and-ready-count
created: "2026-05-10T10:31:34+10:00"
priority: normal
labels:
    - agent
    - frontend
    - enhancement
    - usability
    - vue
release: KC-Release0
---

# Agent Panel: Show Ready Item Count and Running State

Each agent card on the Agents screen should display a count of how many artifacts are currently ready for that agent to work on, giving users an at-a-glance view of queue depth and helping them spot when work is waiting without navigating away.

When an agent is actively running, its panel should be visually highlighted — styled with a green background or border consistent with the existing menu bar running-state styling — so the current activity is immediately obvious from across the screen.

This combines operability and usability improvements into a single, coherent agent panel upgrade that surfaces live system state without requiring any additional user interaction.
