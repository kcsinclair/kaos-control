---
title: Auto Triage New Ideas
type: idea
status: done
lineage: auto-triage-new-ideas
created: "2026-06-05T11:03:08+10:00"
priority: medium
labels:
    - agent
    - agents
    - workflow
    - artifacts
    - process
release: KC-Release3
---

# Auto Triage New Ideas

Implement an automated triage step that detects newly created idea artifacts with status `raw` and runs them through an idea capture agent. The agent enriches the raw input into a well-formed idea while preserving the original brain-dump, then updates the artifact status from `raw` to `draft`.

The resulting artifact structure retains the original content under a `## Raw Idea` heading and appends a new `## Idea` heading containing the LLM-improved, structured version of the idea. This keeps full provenance while surfacing a cleaner representation for downstream lifecycle stages.

The triage step should be triggerable on-demand (e.g. a watcher or API-driven hook detects `status: raw` artifacts) and could run as a dedicated agent role scoped to `lifecycle/ideas/`. Configuration in `lifecycle/config.yaml` would define the agent prompt, allowed write paths, and any gating rules before the artifact advances.
