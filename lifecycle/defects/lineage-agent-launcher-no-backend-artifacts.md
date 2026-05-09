---
title: Agent Launcher Panel Shows No Eligible Artifacts for Backend-Developer Despite Approved Artifacts Existing
type: defect
status: done
lineage: lineage-agent-launcher-no-backend-artifacts
created: "2026-04-28T08:47:48+10:00"
priority: normal
labels:
    - defect
    - agent
    - frontend
    - artefacts
release: KC-OG-Sprint
---

# Agent Launcher Panel Shows No Eligible Artifacts for Backend-Developer Despite Approved Artifacts Existing

## Reproduction Steps

1. Open the lineage agent-launcher panel in the UI.
2. Select or view the panel for the `test-developer` agent.
3. Ensure at least one artifact assigned to `test-developer` exists with status `approved`.
4. Observe the list of eligible artifacts displayed in the panel.
5. Example artefact lifecycle/defects/missing-approved-to-done-coverage.md

## Expected Behaviour

Artifacts with status `approved` that are assigned to the `test-developer` agent should be listed as eligible in the agent launcher panel, allowing the user to trigger an agent run against them.

## Actual Behaviour

The agent launcher panel displays "No eligible artifacts for this agent." even when approved artifacts assigned to `test-developer` exist. No artifacts are shown, preventing the user from launching the agent.
