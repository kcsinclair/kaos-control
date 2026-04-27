---
title: Frontmatter Created Date for Artifacts
type: idea
status: draft
lineage: frontmatter-created-date
priority: normal
labels:
    - artefacts
    - backend
    - feature
---

# Frontmatter Created Date for Artifacts

Every artifact created in the lifecycle system should include a `created` date field in its YAML frontmatter, recording the date and time the file was first generated. This provides a reliable, human-readable timestamp that is independent of git history or filesystem metadata.

The created date should be set automatically at artifact creation time (via the API or agent tooling) and never modified after the fact. It should follow ISO 8601 format (e.g. `2026-04-27T10:00 +10:00`) for consistency and easy sorting.

The SQLite index should surface this field so the UI and agents can filter, sort, and display artifacts by creation date without parsing git history.

The date should be displayed in the web view, with a tooltip hover to display the date and time.

Modified date should be displayed in the same way.
