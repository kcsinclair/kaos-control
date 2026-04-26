---
title: "Single-Submit Idea Capture – Frontend Plan"
type: plan-frontend
status: rejected
lineage: prompt-to-idea
parent: lifecycle/requirements/prompt-to-idea-7.md
labels:
    - usability
    - workflow
    - enhancement
---

# Single-Submit Idea Capture – Frontend Plan

This plan implements the Vue 3 / TypeScript frontend for the single-submit "brain dump" idea and defect capture flow described in [[prompt-to-idea]]. The "New Idea" button is re-wired to open a new `BrainDumpModal` instead of the existing `IdeaChatPanel`. The modal provides a large textarea, sends input to the new `POST /ideas/generate` endpoint from the backend plan ([[prompt-to-idea]]-be), displays a rendered markdown preview, and lets the user accept, edit, or discard the proposal. `IdeaChatPanel` is retained in the codebase for future use by [[flesh-out-ideas-with-agent]].

The backend endpoint is specified in [[prompt-to-idea]]-be Milestone 3. Integration tests are covered in [[prompt-to-idea]]-test.

---

## Milestone 1 – API Client Function

### Description

Add a new API function to call the `POST /ideas/generate` endpoint and define the corresponding TypeScript types.

### Files to change

- **`web/src/api/ideaChat.ts`** – Add `generateIdea(project, input, type?)` function that calls `POST /api/p/:project/ideas/generate` with `{ input, type }`. Returns `IdeaGenerateResponse`.

- **`web/src/types/api.ts`** – Add `IdeaGenerateResponse` interface:
  ```ts
  export interface IdeaGenerateResponse {
    slug: string
    title: string
    labels: string[]
    body: string
    frontmatter: Record<string, unknown>
    target_dir: string
  }
  ```

### Acceptance criteria

- [ ] `generateIdea(project, "some text")` calls the correct endpoint and returns typed data.
- [ ] `generateIdea(project, "some text", "defect")` passes the type parameter.
- [ ] TypeScript compiles without errors (`pnpm exec vue-tsc --noEmit`).

---

## Milestone 2 – Brain Dump Store

### Description

Create a new Pinia store (`useBrainDumpStore`) that manages the state for the single-submit flow: idle → generating → preview → accepted/discarded. This store is separate from `useIdeaChatStore` (which remains for the conversational flow).

### Files to change

- **`web/src/stores/brainDump.ts`** (new) – Pinia store with:
  - State: `input: string`, `artifactType: 'idea' | 'defect'`, `phase: 'input' | 'generating' | 'preview' | 'editing'`, `loading: boolean`, `error: string | null`, `proposal: IdeaGenerateResponse | null`, `editedBody: string | null`.
  - Actions:
    - `generate(project: string)` — calls `generateIdea`, transitions to `preview` on success, sets `error` on failure.
    - `acceptProposal(project: string)` — writes the artifact via `POST /api/p/:project/artifacts` using the proposal's data (slug, frontmatter, body). The `stage` field is derived from `target_dir` (e.g., `"ideas"` or `"defects"`). Transitions to accepted.
    - `startEdit()` — transitions to `editing`, copies `proposal.body` into `editedBody`.
    - `applyEdit()` — transitions back to `preview` with the edited body merged into the proposal.
    - `discard()` — resets all state.
    - `reset()` — resets all state.
  - Getters: `canSubmit` (input trimmed length > 0 and not loading).

### Acceptance criteria

- [ ] Store transitions through `input → generating → preview` on successful generate.
- [ ] Store transitions through `preview → editing → preview` when editing.
- [ ] `acceptProposal` calls `POST /artifacts` with correct stage, slug, frontmatter, and body.
- [ ] `discard` and `reset` return all state to initial values.
- [ ] Error from the API is captured in `error` and phase returns to `input`.
- [ ] TypeScript compiles without errors.

---

## Milestone 3 – BrainDumpModal Component

### Description

Build the new `BrainDumpModal` component that replaces `IdeaChatPanel` as the UI for the "New Idea" button. The modal has three phases: input, generating (loading), and preview.

### Files to change

- **`web/src/components/idea/BrainDumpModal.vue`** (new) – SFC with:
  - **Props**: `project: string`, `artifactType?: 'idea' | 'defect'` (default `'idea'`).
  - **Emits**: `close`, `created(path: string)`.
  - **Input phase** (FR-6.1, FR-6.2):
    - Full-width modal overlay (same pattern as `IdeaChatPanel` overlay: `.icp-overlay` / `aria-modal`).
    - Header: "New Idea" or "New Defect" depending on `artifactType`.
    - Large auto-growing `<textarea>` with placeholder *"Describe your idea — paste, ramble, brain dump…"* (FR-6.1). Minimum 6 rows, auto-grows to fit content.
    - "Generate" button below the textarea. Disabled when input is empty or loading.
    - `Ctrl+Enter` / `Cmd+Enter` submits (FR-6.3). `Escape` closes with discard confirmation if text has been entered (FR-6.3).
    - Textarea supports multi-line paste (FR-6.4).
  - **Generating phase** (FR-6.2):
    - Textarea becomes disabled, "Generate" button shows spinner/loading state.
    - A brief "Generating…" indicator is shown.
  - **Preview phase** (FR-5.1, FR-5.2):
    - Rendered markdown preview of the generated body using `markdown-it` (same as `IdeaChatPanel`'s preview).
    - Metadata summary above the body: title, slug (`<slug>.md`), lineage, labels as chips.
    - Three action buttons: **Accept** (primary), **Edit** (ghost), **Discard** (ghost).
    - Accept writes the artifact and emits `created`.
    - Edit transitions to editing phase: textarea re-appears pre-filled with the generated body, with a "Done editing" button that returns to preview with the updated body.
    - Discard closes the modal.
  - **Accessibility**: focus trap within modal, `aria-modal="true"`, `role="dialog"`, `aria-labelledby`.
  - **Styling**: reuse CSS custom properties from the existing design system (`--color-*`, `--space-*`, `--radius-*`, `--text-*`, etc.). The modal should be wider than the chat panel — `max-width: 640px` — to accommodate the larger textarea.

### Acceptance criteria

- [ ] Modal opens with a large textarea and "Generate" button.
- [ ] `Ctrl+Enter` / `Cmd+Enter` submits the input.
- [ ] `Escape` closes the modal (with confirmation if text entered).
- [ ] During generation, textarea is disabled and button shows loading state.
- [ ] After generation, a rendered markdown preview is shown with metadata summary.
- [ ] Accept button writes the artifact and emits `created`.
- [ ] Edit button shows textarea pre-filled with generated body; "Done editing" returns to preview.
- [ ] Discard button closes the modal.
- [ ] Modal is keyboard-navigable with focus trap.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 4 – Wire New Idea Button to BrainDumpModal

### Description

Replace the `IdeaChatPanel` usage in `ArtifactListView` with the new `BrainDumpModal`. The "New Idea" button opens `BrainDumpModal` instead of `IdeaChatPanel`. `IdeaChatPanel` import is removed from this view but the component file is retained for future use.

### Files to change

- **`web/src/views/project/ArtifactListView.vue`**:
  - Replace `import IdeaChatPanel` with `import BrainDumpModal`.
  - Replace `<IdeaChatPanel>` usage with `<BrainDumpModal :project="..." @close="..." @created="...">`.
  - The `openIdeaChat` function and `showIdeaChat` ref are renamed to `openBrainDump` / `showBrainDump` for clarity.
  - On `created`, navigate to the new artifact or show a success toast with a link (FR-5.3).

### Acceptance criteria

- [ ] Clicking "New Idea" opens the `BrainDumpModal`, not `IdeaChatPanel`.
- [ ] On accept, the user sees a success notification and can navigate to the new artifact.
- [ ] On discard, the modal closes cleanly.
- [ ] `IdeaChatPanel.vue` remains in the codebase (not deleted).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 5 – Add "New Defect" Entry Point

### Description

Add a "New Defect" button or menu option that opens the same `BrainDumpModal` with `artifactType="defect"`. This can be placed in the artifact list view alongside the "New Idea" button, or as a dropdown option from the same button.

### Files to change

- **`web/src/views/project/ArtifactListView.vue`** – Add a "New Defect" button next to "New Idea". Both open `BrainDumpModal` but with different `artifactType` props. The buttons should be visually distinguishable (e.g., different icons: `Lightbulb` for ideas, `Bug` for defects).

- **`web/src/components/idea/BrainDumpModal.vue`** – Ensure the header text, placeholder, and prompt template key adapt based on `artifactType`. For defects, placeholder might read *"Describe the defect — what happened, what you expected…"*.

### Acceptance criteria

- [ ] "New Defect" button is visible on the artifact list view.
- [ ] Clicking "New Defect" opens the modal with defect-specific header and placeholder.
- [ ] Generated defect proposal has `type: defect` in frontmatter.
- [ ] On accept, the defect artifact is written to `lifecycle/defects/`.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 6 – Error Handling and Edge Cases

### Description

Handle error states from the generate endpoint gracefully in the UI: short input rejection (400), LLM failures (500), and network errors. Also handle the edge case where the user edits the body to empty.

### Files to change

- **`web/src/components/idea/BrainDumpModal.vue`** – Display error messages inline below the textarea when the API returns an error. The error message from a 400 (input too short) should be shown as-is. For 500 or network errors, show a generic "Something went wrong — please try again." message. After an error, the textarea remains editable so the user can revise and retry.

- **`web/src/stores/brainDump.ts`** – Ensure `generate` catches errors, stores the message, and transitions phase back to `input`.

### Acceptance criteria

- [ ] Submitting fewer than 5 words shows the backend's error message below the textarea.
- [ ] Network errors show a generic retry message.
- [ ] After an error, the user can edit and resubmit without reopening the modal.
- [ ] Editing the body to empty and clicking "Done editing" shows a validation message.
- [ ] Works on viewports ≥ 768px wide (NFR-2).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.
