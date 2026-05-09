---
title: 'Roadmap Gantt Release Drill-Down: Show Only Ideas and Defects'
type: idea
status: draft
lineage: roadmap-release-filter-ideas-defects
created: "2026-05-09T16:15:27+10:00"
priority: normal
labels:
    - frontend
    - roadmaps
    - enhancement
    - usability
---

# Roadmap Gantt Release Drill-Down: Show Only Ideas and Defects

When viewing the Roadmap Gantt chart and clicking on a release, a panel or view shows all lifecycle artifacts assigned to that release. Currently this includes all artifact types, but the roadmap context calls for a more focused view.

The release drill-down on the Roadmap page should be filtered to display only ideas and defects. Other artifact types (plans, requirements, tests, etc.) are implementation details that are not meaningful at the roadmap level of abstraction and add noise to the view.

This change improves the signal-to-noise ratio for stakeholders using the roadmap to track what user-facing work and known issues are targeted for a given release, without exposing internal planning artefacts.
