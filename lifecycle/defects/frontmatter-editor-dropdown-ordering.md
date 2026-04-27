---
title: 'Frontmatter Editor: Incorrect Ordering of Priority and Status Dropdowns'
type: defect
status: done
lineage: frontmatter-editor-dropdown-ordering
priority: normal
labels:
    - defect
    - frontend
    - vue
---

# Frontmatter Editor: Incorrect Ordering of Priority and Status Dropdowns

## Reproduction Steps

1. Open the markdown editor for any artifact.
2. Locate the Priority dropdown in the frontmatter editor.
3. Observe the order of options displayed.
4. Locate the Status dropdown in the frontmatter editor.
5. Observe the order of options displayed.

## Expected Behaviour

- The Priority dropdown should list options in this exact order: `normal`, `low`, `medium`, `high`.
- The Status dropdown should list options in alphabetical order (e.g. `abandoned`, `approved`, `clarifying`, `done`, `draft`, `in-development`, `in-qa`, `planning`, `rejected`).

## Actual Behaviour

- The Priority dropdown does not follow the specified order of `normal`, `low`, `medium`, `high`.
- The Status dropdown options are not sorted alphabetically.
