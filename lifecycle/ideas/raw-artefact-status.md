---
title: Add 'raw' Artefact Status Before Draft
type: idea
status: done
lineage: raw-artefact-status
created: "2026-05-22T10:12:16+10:00"
priority: normal
labels:
    - artefacts
    - workflow
    - feature
---

# Add 'raw' Artefact Status Before Draft

Introduce a new artefact status `raw` that sits before `draft` in the lifecycle. A `raw` artefact represents something that has been captured (e.g. a brain-dump, a voice note transcription, a quick idea jot) but has not yet been reviewed, structured, or fleshed out into a proper draft.

This distinction is useful because it allows the system to acknowledge and persist an idea immediately upon capture without implying it has received any editorial attention. A `draft` currently carries the implicit meaning that someone has intentionally shaped the content, whereas `raw` signals it is unprocessed input.

The change would require updating the `KnownStatuses` vocabulary in `internal/artifact/artifact.go`, adjusting any workflow state-machine transitions to permit `raw → draft`, and updating UI affordances (filters, graph nodes, status badges) to represent the new status.
