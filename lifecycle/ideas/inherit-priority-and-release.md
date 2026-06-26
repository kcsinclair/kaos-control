---
title: Inherit Priority and Release Through Lineage
type: idea
status: clarifying
lineage: inherit-priority-and-release
created: "2026-05-10T10:12:41+10:00"
priority: high
labels:
    - feature
    - workflow
    - artefacts
release: KC-Release4
parent: lifecycle/ideas/devops-pipeline-run-history.md
---

# Inherit Priority and Release Through Lineage

When an artifact is created from a parent in the lineage chain, its `priority` and `release` frontmatter fields should be automatically inherited from the parent rather than left blank or requiring manual entry. For example, if an idea is marked `priority: high` and `release: v1.2`, the resulting requirements, plans, and test artifacts should carry those same values by default.

This inheritance should apply at artifact creation time — whether triggered by an agent or a manual workflow transition — and should be visible in the editor UI with a clear indication that the value was inherited. Users should still be able to override the inherited value on any individual artifact without affecting siblings or the parent.

Propagating these fields consistently reduces the risk of mismatched metadata across a lineage and makes release planning and prioritisation views more reliable, since all artifacts belonging to an idea will naturally cluster together under the correct priority and release milestone.
