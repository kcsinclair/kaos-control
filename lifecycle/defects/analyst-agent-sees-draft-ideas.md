---
title: Requirements Analyst Agent Surfaces Draft Ideas in Agent Launcher Panel
type: defect
status: done
lineage: analyst-agent-sees-draft-ideas
created: "2026-04-28T16:39:28+10:00"
priority: normal
labels:
    - defect
    - agent
    - workflow
release: KC-Feature-Sprint
---

# Requirements Analyst Agent Surfaces Draft Ideas in Agent Launcher Panel

## Reproduction Steps

1. Open the agent launcher panel in the UI.
2. Select or view the requirements analyst agent.
3. Observe the list of ideas presented as candidates for the agent to process.

## Expected Behaviour

The requirements analyst agent should only be offered ideas with a status of `approved`. Ideas in any other status (e.g. `draft`, `clarifying`, `rejected`, `abandoned`) must not appear in the agent launcher panel for this agent.

## Actual Behaviour

Ideas with a status of `draft` are visible to the requirements analyst agent in the agent launcher panel, allowing the agent to be invoked against ideas that have not yet been approved for requirements work.
