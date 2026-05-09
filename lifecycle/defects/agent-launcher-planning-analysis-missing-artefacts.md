---
title: Agent Launcher Panel Does Not Show Available Artefacts for Planning-Analysis
type: defect
status: done
lineage: agent-launcher-planning-analysis-missing-artefacts
created: "2026-04-27T20:05:16+10:00"
priority: normal
labels:
    - defect
    - agent
    - frontend
    - artefacts
release: KC-OG-Sprint
---

# Agent Launcher Panel Does Not Show Available Artefacts for Planning-Analysis

## Reproduction Steps

1. Ensure at least one requirement artifact exists with a status that is ready for planning (e.g. the `kanban-view` requirement is open and eligible for planning-analysis).
2. Open the agent launcher UI.
3. Select the `planning-analysis` agent.
4. Observe the available artefacts panel for that agent.

## Expected Behaviour

The agent launcher panel for the planning-analysis agent should list all requirement artifacts that are in a state eligible for planning (e.g. approved or ready-to-plan status), including the open `kanban-view` requirement.

## Actual Behaviour

No artefacts are shown in the available artefacts panel when the planning-analysis agent is selected. The open `kanban-view` requirement is not visible, preventing the user from launching the agent against it.
