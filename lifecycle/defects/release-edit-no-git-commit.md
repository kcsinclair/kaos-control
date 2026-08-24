---
title: Release Artifact Not Automatically Committed to Git After Edit
type: defect
status: approved
lineage: release-edit-no-git-commit
created: "2026-08-24T11:41:56+10:00"
priority: normal
labels:
    - defect
    - releases
    - git
    - editor
    - backend
release: KC-Release6
assignees:
    - role: backend-developer
      who: agent
---

# Release Artifact Not Automatically Committed to Git After Edit

## Reproduction Steps

1. Open the kaos-control UI and navigate to a release artifact.
2. Click to edit the release (e.g. update content, status, or metadata).
3. Save the changes via the editor.
4. Inspect the git repository state (e.g. `git status` or `git log`).

## Expected Behaviour

After saving edits to a release artifact, the updated file should be automatically committed to the git repository, consistent with how other artifact writes are handled (or as expected by the lifecycle tooling).

## Actual Behaviour

The release artifact is updated on disk but no git commit is created. The change remains as an uncommitted modification in the working tree.
