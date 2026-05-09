---
title: Release Description Field
type: idea
status: draft
lineage: release-description-field
created: "2026-05-10T09:47:16+10:00"
priority: normal
labels:
    - releases
    - feature
    - enhancement
---

# Release Description Field

Release artifacts should support an optional `description` field in their frontmatter and/or body that allows authors to capture a release goal, theme, or high-level notes. This gives teams a lightweight way to articulate intent — e.g. "Focus on auth hardening and onboarding improvements" — directly on the release artifact rather than relying on commit messages or external docs.

The description should be surfaced in the UI wherever releases are displayed, such as the release detail view and any roadmap or sprint panels that reference releases. This helps contributors and reviewers quickly understand the purpose of a release without having to inspect its linked tickets.

On the backend, the artifact parser should recognise the `description` frontmatter key and index it alongside existing fields so it is queryable and available via the REST API.
