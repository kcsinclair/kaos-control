---
title: 'Artefacts View: Agent Run Count Column'
type: idea
status: clarifying
lineage: artefacts-agent-run-count-column
created: "2026-05-16T08:50:35+10:00"
priority: normal
labels:
    - artefacts
    - frontend
    - agent
    - enhancement
    - feature
---

# Artefacts View: Agent Run Count Column

The artefacts list view currently has no visibility into how much agent activity has occurred against each artefact. A new "Agent Runs" column should be added to the artefacts table showing the total number of times an agent has been run against each artefact.

This allows users — particularly product owners and QA — to quickly identify artefacts that have had no agent work performed (count = 0) versus those that have been through one or more agent iterations, making it easier to track progress and spot gaps in the workflow.

The count should be derived from the existing agent run history stored by the system and displayed as a simple integer in the new column, sortable so users can surface untouched artefacts quickly.
