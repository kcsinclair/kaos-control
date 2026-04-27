---
title: Artifact Agent Run History
type: idea
status: draft
lineage: artifact-agent-run-history
created: "2026-04-28T09:49:31+10:00"
priority: normal
labels:
    - agent
    - artefacts
    - feature
    - frontend
    - backend
---

# Artifact Agent Run History

relates-to: [[ideas/improved-agent-handling]] 
relates-to: [[ideas/agent-completion-status-detail]]

When viewing an artifact in the detail panel, a list of agent runs associated with that artifact should be displayed, showing each run's ID and the date it was executed. This information is already available in the existing database and does not require new data collection.

Clicking a run ID should open a modal displaying the full details of that agent run, giving users visibility into what work was performed against the artifact and when. This supports auditability and traceability throughout the lifecycle.

This idea is related to broader agent visibility themes explored in other ideas in the backlog. Implementation will require a backend query to join agent runs to artifacts by lineage or file path, and a frontend component to render the run list and detail modal within the existing artifact detail panel.
