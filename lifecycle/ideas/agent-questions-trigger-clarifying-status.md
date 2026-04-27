---
title: Agent Questions Trigger Clarifying Status
type: idea
status: draft
lineage: agent-questions-trigger-clarifying-status
created: "2026-04-27T16:34:31+10:00"
priority: normal
labels:
    - agent
    - workflow
    - artefacts
    - process
---

# Agent Questions Trigger Clarifying Status

When an agent writes questions into a plan or requirement artifact — for example, appending a questions section or frontmatter field — the artifact's status should be automatically transitioned to `clarifying`.

This ensures the lifecycle state machine accurately reflects that the artifact is awaiting human input before work can proceed, preventing downstream roles (e.g. developer agents) from picking up an artifact that is not yet fully resolved.

The trigger could be detected by the watcher or indexer when it parses an updated artifact and finds a recognised questions block (e.g. a `## Questions` heading or a `questions:` frontmatter key with non-empty content), applying the `clarifying` status transition if the current status allows it.
