---
title: Agent Launcher Panels on Agents Screen
type: idea
status: clarifying
lineage: agent-launcher-panels
created: "2026-04-27T15:25:56+10:00"
priority: normal
labels:
    - feature
    - frontend
    - agent
    - vue
---

# Agent Launcher Panels on Agents Screen

Above the agent runs list on the Agents screen, display a row of small panels — one per configured agent — showing the agent's name, role, and model. This gives users a quick overview of what agents are available without having to dig into configuration files.

Each panel is clickable and opens a launch flow: the user is presented with a filtered list of artifacts that are in an approved or ready state for that agent's stage in the lifecycle. The user selects an artifact and confirms to start the agent run. The idea-capture agent panel is rendered but not clickable, since it is driven externally rather than manually triggered.

This improves discoverability and reduces friction for kicking off agent work — currently users have no in-UI way to see which agents exist or to initiate a run against a specific artifact.
