---
title: Rename Analyst Agents for Consistency
type: idea
status: done
lineage: rename-analyst-agents
created: "2026-04-28T16:46:42+10:00"
priority: normal
labels:
    - agent
    - process
---

# Rename Analyst Agents for Consistency

Rename the `analyst-planner` agent to `planning-analyst` and the `analyst-requirements` agent to `requirements-analyst`. This reversal puts the lifecycle phase first and the role second, matching a more intuitive naming convention.

The rename affects agent configuration in `lifecycle/config.yaml`, any prompt templates referencing the agent names, and all places in the codebase (Go and frontend) that hard-code or display these identifiers.

All existing artifacts produced by these agents reference the old agent names in their frontmatter or git history, so the rename should be purely additive in config — no backfill of historical artifacts is required.
