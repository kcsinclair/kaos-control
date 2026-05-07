---
title: Lineage Filter with Autocomplete for Artifact List and Board
type: idea
status: abandoned
lineage: lineage-filter-autocomplete
created: "2026-04-29T10:58:00+10:00"
priority: high
labels:
    - feature
    - frontend
    - artefacts
release: May2026
---

# Lineage Filter with Autocomplete for Artifact List and Board

**The free text search feature made this basically redundant**

Add a lineage filter control to both the artifact list view and the board view. The filter should be a text input that supports free-text matching on any part of a lineage slug, allowing users to quickly narrow down artifacts to a specific feature lineage.

As the user begins typing, the input should display an autocomplete dropdown showing matching lineage slugs from the current project. Matches should be substring-based (not prefix-only), so typing any fragment of a lineage slug surfaces relevant suggestions.

Selecting a suggestion or submitting free text should filter the displayed artifacts to only those whose lineage field contains the entered string. The filter should compose cleanly with any other active filters (status, type, etc.).
