---
title: 'Frontmatter Editor: Add Priority Dropdown and Convert Status to Dropdown'
type: idea
status: done
lineage: frontmatter-editor-dropdowns
priority: normal
labels:
    - enhancement
    - frontend
    - usability
    - vue
release: KC-OG-Sprint
---

# Frontmatter Editor: Add Priority Dropdown and Convert Status to Dropdown

The current frontmatter editor does not expose a `priority` field, making it impossible to set or change priority when editing an artifact directly in the UI. Additionally, the `status` field is a free-text input, which allows invalid values and is slower to use than a constrained list.

The `priority` field should be added to the frontmatter editor as a dropdown populated with the valid priority values (e.g. normal, high). The `status` text input should be replaced with a dropdown pre-populated with the full status vocabulary defined in the spec (`draft`, `clarifying`, `planning`, `in-development`, `in-qa`, `approved`, `rejected`, `abandoned`, `done`).

These changes will reduce invalid frontmatter, improve editing speed, and make the editor more discoverable for users unfamiliar with the allowed vocabulary.
