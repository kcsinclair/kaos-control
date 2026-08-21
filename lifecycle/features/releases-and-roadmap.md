---
title: Releases & roadmap
type: feature
status: approved
lineage: feature-releases-and-roadmap
created: "2026-08-21T15:06:00+10:00"
summary: SQLite-backed release records that artifacts assign into, with rename propagation and reassignment on delete.
function: Releases & roadmap
labels:
    - feature
    - releases
    - roadmap
related_to:
    - lifecycle/ideas/releases-and-roadmaps.md
    - lifecycle/requirements/releases-and-roadmaps-2.md
    - lifecycle/ideas/release-artefacts.md
    - lifecycle/requirements/release-artefacts-9.md
---

# Releases & roadmap

Group artifacts into releases and see them laid out on a roadmap.

## What it does

- **First-class release records.** Stored in SQLite (not as markdown
  artifacts), with name, status, optional `start_date` / `end_date`.
- **Artifact assignment.** Artifacts carry an optional `release:`
  frontmatter field. Assigned artifacts show in the release detail view and
  on the roadmap graph as children of their release.
- **Release CRUD via REST + WS broadcast.** Create / update / delete / list
  with `release.created` / `.updated` / `.deleted` events.
- **Rename propagation.** Renaming a release auto-rewrites every assigned
  artifact's `release:` frontmatter and commits the change.
- **Reassign on delete.** Optional `?reassign_to=<id>` on delete moves the
  doomed release's artifacts onto another release.

Reachable at **Releases** and **Roadmap**; API under `/releases`. See also
[[graph-visualisation]] for the roadmap graph/Gantt views.
