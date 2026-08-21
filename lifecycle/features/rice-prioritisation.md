---
title: RICE prioritisation
type: feature
status: approved
lineage: feature-rice-prioritisation
created: "2026-08-21T15:01:00+10:00"
summary: Reach / Impact / Confidence / Effort fields on any artifact compute a sortable RICE score in the list.
function: Lifecycle & artifacts
labels:
    - feature
    - prioritisation
related_to:
    - lifecycle/ideas/rice-scoring.md
    - lifecycle/requirements/rice-scoring-2.md
---

# RICE prioritisation

Score and sort backlog items without leaving the artifact editor.

## What it does

- **RICE fields on frontmatter.** `rice_reach`, `rice_impact`,
  `rice_confidence`, `rice_effort` are editable on any artifact; `rice_effort`
  accepts decimals (e.g. `0.1` months).
- **Computed score.** The RICE score (`reach × impact × confidence / effort`)
  is computed and stored whenever the inputs change — no manual calculation.
- **Sortable column.** The artifact list exposes RICE score as a sortable
  column alongside status, priority, and release.

Reachable at **Artifacts** (list); edit the RICE fields from the artifact
editor.
