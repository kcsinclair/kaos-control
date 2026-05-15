---
title: "Frontend Plan — Make Pipelines Editable"
type: plan-frontend
status: in-development
lineage: pipeline-editing
parent: lifecycle/requirements/pipeline-editing-2.md
created: "2026-05-15T00:00:00+10:00"
labels:
    - frontend
    - devops
release: KC-Release1
assignees:
    - role: frontend-developer
      who: agent
---

# Frontend Plan — Make Pipelines Editable

Implements the UI changes required by the "Make Pipelines Editable" requirement ([[pipeline-editing]]). Depends on the backend endpoints defined in [[pipeline-editing-3-be]].

## Milestone 1 — API client additions

Add `getPipeline` and `updatePipeline` functions to the devops API client.

### Files to change

- `web/src/api/devops.ts` — add two new functions and their response types.

### Implementation details

1. Add `getPipelineDefinition(project: string, slug: string): Promise<string>`:
   - `GET /p/{project}/devops/pipelines/{slug}`
   - Use `api.getText(...)` since the response is raw YAML (Content-Type: text/yaml).
2. Add `updatePipeline(project: string, slug: string, definition: string): Promise<PipelineResponse>`:
   - `PUT /p/{project}/devops/pipelines/{slug}` with JSON body `{ definition }`.
   - Returns `{ slug, name, type, step_count }`.

### Acceptance criteria

- [ ] `getPipelineDefinition` returns the raw YAML string for an existing pipeline.
- [ ] `updatePipeline` sends a PUT request and returns the updated pipeline summary.
- [ ] Both functions propagate `ApiError` on failure (400, 404, 409).

---

## Milestone 2 — Pinia store: update action

Extend the devops store with an action to update a pipeline and refresh the local state.

### Files to change

- `web/src/stores/devops.ts` — add `updatePipeline` action and `anyRunning` getter.

### Implementation details

1. Add computed getter `anyRunning`:
   ```ts
   anyRunning: (state) => {
     for (const [, run] of state.activeRuns) {
       if (run.overallStatus === 'running') return true
     }
     return false
   }
   ```
2. Add action `updatePipeline(project: string, slug: string, definition: string)`:
   - Call `devopsApi.updatePipeline(project, slug, definition)`.
   - On success, find the pipeline in `this.pipelines` by slug and update its `name`, `type`, and `steps` (re-fetch the full list via `fetchPipelines` to get accurate step arrays).
   - Return the response for the dialog to consume.
3. Add WebSocket handler for `pipeline.updated` event:
   - On receipt, call `fetchPipelines(project)` to refresh the pipeline list, ensuring other tabs/clients stay in sync.

### Acceptance criteria

- [ ] `updatePipeline` calls the API and refreshes the local pipeline list.
- [ ] `anyRunning` returns `true` when at least one pipeline's run status is `'running'`.
- [ ] A `pipeline.updated` WebSocket event triggers a pipeline list refresh.

---

## Milestone 3 — Edit button on PipelineCard

Add an "Edit" button to each pipeline card that opens the edit dialog.

### Files to change

- `web/src/components/devops/PipelineCard.vue` — add edit button and emit event.

### Implementation details

1. Add a pencil icon button (lucide-vue-next `Pencil` icon) next to the existing Run button in the card header.
2. The button is visible only when the user has `product-owner` or `devops` role. Use the same role-check pattern already present in DevOpsView for create access.
3. Disable the button with a tooltip ("Editing is disabled while a pipeline is running") when `anyRunning` is `true` in the devops store. This matches resolved question #2 — editing is blocked while *any* pipeline is running.
4. On click, emit `edit` event with the pipeline slug so the parent view can open the edit dialog.

### Acceptance criteria

- [ ] A pencil "Edit" button is visible on pipeline cards for users with `product-owner` or `devops` role.
- [ ] The button is disabled with a tooltip when any pipeline run is active.
- [ ] The button is enabled when no runs are active.
- [ ] Clicking the button emits an `edit` event with the pipeline slug.

---

## Milestone 4 — EditPipelineDialog component

Create the edit dialog that loads the current YAML and allows saving changes.

### Files to change

- `web/src/components/devops/EditPipelineDialog.vue` — new component (mirrors `CreatePipelineDialog.vue` structure).
- `web/src/views/project/DevOpsView.vue` — wire up the dialog: listen for `edit` events, pass slug, manage dialog visibility.

### Implementation details

1. **Props**: `open: boolean`, `project: string`, `slug: string`.
2. **On open** (watch `open` becoming `true`):
   - Fetch raw YAML via `devopsApi.getPipelineDefinition(project, slug)`.
   - Show a loading spinner while fetching.
   - Populate the `YamlEditor` with the fetched content.
   - Store the original definition for comparison.
3. **Slug display**: show the slug as a read-only label at the top of the dialog (not an input field).
4. **Save button state**: disabled until (a) the user has made at least one change (compare current editor value to the stored original) AND (b) client-side YAML validation passes.
5. **Client-side validation** (same rules as create dialog):
   - Valid YAML syntax (`js-yaml.load()`).
   - Required fields present: `name`, `type`, `steps`.
   - At least one step with `name` and `command`.
6. **Step-removal confirmation** (F6):
   - Before submitting, parse both old and new YAML. Count steps in each.
   - If new step count < old step count, show a confirmation prompt: "You are removing N step(s). Save anyway?"
   - Proceed only if the user confirms.
7. **Submission**:
   - Call `devopsStore.updatePipeline(project, slug, definition)`.
   - On success, close the dialog and emit `updated`.
   - On `409` error, show inline message about active runs.
   - On `400` error, show the server validation message inline.
   - On network error, show inline error; keep the dialog open so edits are not lost (NF2).
8. **Cancel button**: closes the dialog, discards unsaved changes.

### Acceptance criteria

- [ ] The edit dialog opens with the current on-disk YAML loaded in the editor.
- [ ] The slug is displayed as a read-only label.
- [ ] Save is disabled until the user modifies the YAML and it passes validation.
- [ ] Removing steps triggers a confirmation prompt before saving.
- [ ] Successful save closes the dialog and updates the pipeline card in place.
- [ ] Server errors (400, 409) and network errors display inline without closing the dialog.
- [ ] Cancel closes the dialog without persisting changes.
- [ ] The dialog layout and styling are consistent with CreatePipelineDialog (NF1).

---

## Milestone 5 — Live UI update after save

Ensure the pipeline card reflects changes immediately after a successful save.

### Files to change

- `web/src/views/project/DevOpsView.vue` — handle dialog `updated` event.
- `web/src/stores/devops.ts` — already handled in Milestone 2 (re-fetch on success).

### Implementation details

1. In `DevOpsView.vue`, listen for the `updated` event from `EditPipelineDialog`.
2. Close the edit dialog and clear the edit-target slug.
3. The store's `updatePipeline` action already refreshes the pipeline list, so the card's `name`, `type`, and step count will reactively update via `pipelinesByType`.
4. No full page reload is required.

### Acceptance criteria

- [ ] After save, the pipeline card updates name, type, and step count without a page reload.
- [ ] The pipeline grid re-sorts correctly if the type was changed (pipeline moves to the correct column).
- [ ] No regressions to the create dialog or run/cancel functionality.
