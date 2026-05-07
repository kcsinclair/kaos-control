---
title: Include Created-By Authorship in Artifact Frontmatter
type: idea
status: done
lineage: frontmatter-created-by
created: "2026-04-28T08:52:45+10:00"
priority: medium
labels:
    - feature
    - artefacts
    - workflow
    - backend
release: May2026
---

# Include Created-By Authorship in Artifact Frontmatter

Every lifecycle artifact (ideas, requirements, plans, defects, etc.) should record who or what created it directly in its YAML frontmatter. This means capturing whether the author was a human or an agent, and which role they were acting in (e.g. `analyst`, `backend-developer`, `qa`, `product-owner`).

A `created_by` block in frontmatter would carry fields such as `kind` (`human` or `agent`), `agent` (the configured agent name, if applicable), and `role`. For human-authored artifacts the value can be derived from the authenticated session; for agent-authored artifacts the agent runner should inject it automatically at write time.

This information enables richer filtering and attribution in the UI (graph, kanban, editor), supports audit trails for compliance and review workflows, and makes it immediately clear in the raw markdown who produced each artifact without consulting git history.
