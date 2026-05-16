---
title: Make Pipelines Editable
type: requirement
status: done
lineage: pipeline-editing
created: "2026-05-15T00:00:00+10:00"
priority: medium
parent: lifecycle/ideas/pipeline-editing.md
labels:
    - feature
    - enhancement
    - frontend
    - backend
    - devops
release: KC-Release1
assignees:
    - role: analyst
      who: agent
    - role: product-owner
      who: agent
---

# Make Pipelines Editable

## Problem

Users can create and run DevOps pipelines, but there is no way to modify an existing pipeline after creation. If a user needs to rename a pipeline, change a step command, adjust a timeout, add or remove steps, or change the pipeline type, they must either edit the YAML file on disk manually or delete and recreate the pipeline through the UI. This breaks the self-service workflow the DevOps view is designed to provide and forces users out of the application for routine maintenance tasks.

## Goals / Non-goals

### Goals

1. Allow users to open an existing pipeline's YAML definition in an in-app editor, modify it, and save changes back to disk.
2. Validate edits client-side and server-side before persisting, providing clear error feedback on invalid YAML or missing required fields.
3. Preserve the existing slug (filename) when editing — only the file contents change.
4. Prevent edits while a pipeline run is active for that pipeline, to avoid mid-run configuration drift.
5. Reflect saved changes immediately in the pipeline card and detail view without requiring a page reload.

### Non-goals

- **Slug/filename renaming**: changing a pipeline's slug requires delete-and-recreate in v1. Rename support is deferred.
- **Version history or diff view**: tracking edit history within the UI is out of scope; git history serves this purpose.
- **Bulk editing**: editing multiple pipelines simultaneously is not required.
- **Step-level drag-and-drop reordering**: a visual step reorder UI is not in scope; users reorder steps by editing the YAML directly.
- **Pipeline duplication/cloning**: cloning an existing pipeline into a new slug is a separate feature.

## Detailed Requirements

### Functional

#### F1 — Edit entry point

- Each pipeline card in the DevOps view must display an "Edit" button (pencil icon), visible only to users with the `product-owner` or `devops` role.
- Clicking the Edit button must open an edit dialog pre-populated with the pipeline's current YAML definition.
- The Edit button must be disabled (with a tooltip explaining why) while a run is active for that pipeline.

#### F2 — Edit dialog

- The edit dialog must reuse the same `YamlEditor` component used by the create dialog, pre-populated with the pipeline's current on-disk YAML content.
- The slug must be displayed as a read-only label (not an editable field) at the top of the dialog.
- The dialog must include "Save" and "Cancel" buttons. "Save" must be disabled until the user has made at least one change and the YAML passes client-side validation.
- Client-side validation must check: valid YAML syntax, presence of required fields (`name`, `type`, `steps`), at least one step with `name` and `command`.

#### F3 — Backend update endpoint

- A new `PUT /api/p/{project}/devops/pipelines/{slug}` endpoint must accept an updated YAML definition and persist it to the existing file at `lifecycle/devops/{slug}.yaml`.
- The endpoint must require the `product-owner` or `devops` role.
- The endpoint must reject the request with `409 Conflict` if the pipeline has an active run.
- The endpoint must validate the YAML with the same rules used by the create endpoint: valid YAML, required fields present, valid step structure, valid timeout durations.
- On validation failure, the endpoint must return `400 Bad Request` with a descriptive error message.
- On success, the endpoint must return the updated pipeline summary (slug, name, type, step_count) with status `200 OK`.

#### F4 — Fetch current definition

- A new `GET /api/p/{project}/devops/pipelines/{slug}` endpoint must return the raw YAML content of the pipeline file, so the frontend can populate the editor with the current on-disk definition.
- If the slug does not exist, the endpoint must return `404 Not Found`.

#### F5 — Live UI update after save

- After a successful save, the edit dialog must close and the pipeline card must update in place to reflect any changed name, type, or step count — without a full page reload.
- The Pinia devops store must be updated with the new pipeline data returned by the PUT response.

#### F6 — Confirmation for destructive edits

- If the user removes one or more steps from the pipeline definition, the Save action must show a confirmation prompt ("You are removing N step(s). Save anyway?") before submitting.
- Renaming existing steps or changing commands does not require confirmation.

### Non-functional

#### NF1 — Consistency

- The edit dialog's layout, styling, and validation behaviour must be consistent with the existing create dialog to maintain a coherent UX.

#### NF2 — Error handling

- Network errors during save must be caught and displayed inline in the dialog (not as a browser alert). The dialog must remain open so the user does not lose their edits.

#### NF3 — Concurrency safety

- The backend must re-check for active runs at write time (not just at request receipt) to prevent a race between a run starting and an edit being saved.

#### NF4 — File integrity

- The backend must write the updated YAML atomically (write to a temp file in the same directory, then rename) to prevent partial writes from corrupting the pipeline file.

## Acceptance Criteria

- [ ] An "Edit" button appears on each pipeline card for users with `product-owner` or `devops` role.
- [ ] The Edit button is disabled with a tooltip while a pipeline run is active.
- [ ] Clicking Edit opens a dialog with the pipeline's current YAML definition pre-loaded in the editor.
- [ ] The slug is shown as a read-only label in the edit dialog.
- [ ] Client-side validation prevents saving invalid YAML or YAML missing required fields.
- [ ] `PUT /api/p/{project}/devops/pipelines/{slug}` persists valid edits to disk atomically.
- [ ] The PUT endpoint returns `409 Conflict` when a run is active.
- [ ] The PUT endpoint returns `400 Bad Request` with a message for invalid YAML.
- [ ] `GET /api/p/{project}/devops/pipelines/{slug}` returns the raw YAML content of the pipeline file.
- [ ] After saving, the pipeline card updates in-place (name, type, step count) without page reload.
- [ ] Removing steps triggers a confirmation prompt before saving.
- [ ] Network errors during save are shown inline; the dialog stays open preserving user edits.
- [ ] No regressions to pipeline creation ([[devops-pipelines]]) or log streaming ([[devops-pipeline-log-streaming]]).
- [ ] Edit and create dialogs are visually and behaviourally consistent.

## Resolved Questions

1. Should the edit dialog support a "diff view" showing changes against the saved version before confirming, or is the raw editor sufficient for v1?

> raw view for v1

2. Should editing be blocked while *any* pipeline is running (global lock), or only while *that specific pipeline* is running? (This requirement assumes per-pipeline locking.)

> yes, disable while any pipeline is running

3. Should the GET endpoint for fetching raw YAML return the file verbatim, or should it strip/normalise whitespace?

> file verbatim
