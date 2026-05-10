---
title: 'Frontend Plan: Init Bootstrap — Create Pipeline Dialog and YAML Editor'
type: plan-frontend
status: done
lineage: kaos-control-init-bootstrap
parent: lifecycle/requirements/kaos-control-init-bootstrap-2.md
---

# Frontend Plan: Init Bootstrap — Create Pipeline Dialog and YAML Editor

This plan implements the frontend changes for [[kaos-control-init-bootstrap]]: adding a "Create Pipeline" workflow to the DevOps screen with a CodeMirror-based YAML editor, client-side validation, and role-gated visibility.

---

## Milestone 1: Add `@codemirror/lang-yaml` dependency

### Description

Install the CodeMirror YAML language support package so the pipeline editor can provide syntax highlighting and basic error feedback for YAML content.

### Files to change

- `web/package.json` — add `"@codemirror/lang-yaml": "^6.1.1"` (or latest 6.x) to `dependencies`.
- Run `pnpm install` to update `web/pnpm-lock.yaml`.

### Acceptance criteria

- [ ] `@codemirror/lang-yaml` is listed in `package.json` dependencies.
- [ ] `pnpm install` completes without errors.
- [ ] `pnpm run build` succeeds.

---

## Milestone 2: Create `YamlEditor.vue` component

### Description

Create a reusable CodeMirror 6 YAML editor component, modelled on the existing `MarkdownEditor.vue` but using `yaml()` language support instead of `markdown()`. This component will be used by the create-pipeline dialog and can be reused elsewhere (e.g., `ProjectConfigView.vue` could adopt it later).

### Files to change

- `web/src/components/common/YamlEditor.vue` — **new file**. Implementation:
  - Props: `modelValue: string` (v-model compatible), `readonly?: boolean`.
  - Emits: `update:modelValue`.
  - Setup: `EditorView` with `basicSetup`, `yaml()` from `@codemirror/lang-yaml`, `oneDark` theme, line wrapping enabled by default.
  - On external `modelValue` change, update editor content (same pattern as `MarkdownEditor.vue`).
  - Style: consistent with `.editor-container` patterns used elsewhere.

### Acceptance criteria

- [ ] `YamlEditor` renders a CodeMirror editor with YAML syntax highlighting.
- [ ] Typing in the editor emits `update:modelValue` with the current content.
- [ ] External changes to `modelValue` update the editor content.
- [ ] `pnpm run type-check` and `pnpm run build` pass.

---

## Milestone 3: Create `CreatePipelineDialog.vue` component

### Description

Build the dialog component that allows users to author a new pipeline definition. It contains a slug input field and the `YamlEditor` for the pipeline YAML body. On submission it validates the YAML client-side, calls the create API, and emits success.

### Files to change

- `web/src/components/devops/CreatePipelineDialog.vue` — **new file**. Implementation:
  - Props: `open: boolean`, `project: string`.
  - Emits: `close`, `created(pipeline)`.
  - Template:
    - Modal overlay (use existing dialog/modal patterns from the codebase if available, otherwise a simple overlay + panel).
    - Slug input field with pattern validation (`^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$`).
    - `YamlEditor` component for the pipeline definition body, pre-populated with a skeleton:
      ```yaml
      name: My Pipeline
      type: build

      steps:
        - name: Step 1
          description: Describe what this step does
          command: echo "hello"
          timeout: 60s
      ```
    - Error display area for validation/server errors.
    - "Cancel" and "Create" buttons. "Create" is disabled while submitting.
  - Logic:
    - On "Create" click: attempt `yaml.parse()` (using `js-yaml` or a lightweight YAML parser) on the definition text. If parse fails, display error inline and do not submit.
    - On valid YAML: call `createPipeline(project, slug, definition)` from the API layer.
    - On `201` success: emit `created` with the pipeline data and close.
    - On `409`: display "A pipeline with this slug already exists."
    - On `400`: display the server error message.

### Acceptance criteria

- [ ] Dialog opens with a slug input and YAML editor.
- [ ] Submitting invalid YAML shows an inline error and does not call the API.
- [ ] Submitting valid YAML with a unique slug calls the API and closes the dialog on success.
- [ ] Duplicate slug displays an appropriate error message.
- [ ] Cancel closes the dialog without side effects.
- [ ] `pnpm run type-check` and `pnpm run build` pass.

---

## Milestone 4: Add `createPipeline` API function and store action

### Description

Wire up the API layer and Pinia store to support creating pipelines.

### Files to change

- `web/src/api/devops.ts` — add:
  ```typescript
  export interface CreatePipelineRequest {
    slug: string
    definition: string
  }

  export interface CreatePipelineResponse {
    slug: string
    name: string
    type: string
    step_count: number
  }

  export function createPipeline(
    project: string,
    body: CreatePipelineRequest
  ): Promise<CreatePipelineResponse> {
    return api.post<CreatePipelineResponse>(
      `/p/${encodeURIComponent(project)}/devops/pipelines`,
      body
    )
  }
  ```

- `web/src/stores/devops.ts` — add action:
  ```typescript
  async function createPipeline(project: string, slug: string, definition: string): Promise<Pipeline> {
    const res = await devopsApi.createPipeline(project, { slug, definition })
    // Re-fetch pipelines to get the full pipeline object including steps
    await fetchPipelines(project)
    return pipelines.value.find(p => p.slug === res.slug)!
  }
  ```
  Export `createPipeline` from the store return object.

### Acceptance criteria

- [ ] `createPipeline` API function sends `POST` to `/api/p/{project}/devops/pipelines` with JSON body.
- [ ] Store `createPipeline` action calls the API and refreshes the pipeline list on success.
- [ ] `pnpm run type-check` and `pnpm run build` pass.

---

## Milestone 5: Integrate "Create Pipeline" button into DevOpsView

### Description

Add a "Create Pipeline" button to the DevOps screen header and wire it to the `CreatePipelineDialog`. The button and dialog are only rendered for users with the `product-owner` or `devops` role (already gated by `hasAccess`).

### Files to change

- `web/src/views/project/DevOpsView.vue`:
  - Import `CreatePipelineDialog`.
  - Add reactive state: `showCreateDialog: ref(false)`.
  - Add a "Create Pipeline" button in the `.devops-header` div, next to the title. Use a `<button>` styled consistently with other action buttons in the app. Include a `+` icon from `lucide-vue-next` (e.g., `Plus`).
  - Render `<CreatePipelineDialog>` conditionally when `showCreateDialog` is true.
  - On `created` event from the dialog: the store already re-fetched, so pipelines will reactively update in the kanban grid. Close the dialog.
  - On `close` event: set `showCreateDialog = false`.

### Acceptance criteria

- [ ] The DevOps screen shows a "Create Pipeline" button in the header when the user has access.
- [ ] Clicking the button opens the `CreatePipelineDialog`.
- [ ] After successfully creating a pipeline, the dialog closes and the new pipeline appears in the kanban grid without a page refresh.
- [ ] The "test" pipeline type column appears dynamically if the new pipeline has a type not yet in `columnOrder`.
- [ ] `pnpm run type-check` and `pnpm run build` pass.

---

## Milestone 6: Add YAML validation dependency

### Description

Ensure a client-side YAML parser is available for pre-submission validation. If `js-yaml` or equivalent is not already in the project, add it.

### Files to change

- `web/package.json` — check if a YAML parser exists. If not, add `"yaml": "^2.x"` (the `yaml` npm package) or `"js-yaml": "^4.x"` as a dependency.
- `web/src/components/devops/CreatePipelineDialog.vue` — import and use the parser for validation.

### Acceptance criteria

- [ ] Client-side YAML validation catches syntax errors before API submission.
- [ ] The parser is a lightweight, well-maintained package.
- [ ] `pnpm run build` succeeds.

> **Note**: This milestone may be merged with Milestone 3 during implementation if the developer prefers to handle both together. It is separated here for clarity.

---

## Cross-references

- This plan depends on the `POST /api/p/{project}/devops/pipelines` endpoint from [[kaos-control-init-bootstrap-3-be]] Milestone 4.
- Integration tests in [[kaos-control-init-bootstrap-5-test]] will exercise the create-pipeline UI flow end-to-end.
