---
title: "Single-Submit Idea & Defect Capture – Frontend Plan"
type: plan-frontend
status: draft
lineage: prompt-to-idea
parent: lifecycle/requirements/prompt-to-idea-7.md
---

# Single-Submit Idea & Defect Capture – Frontend Plan

This plan implements the Vue 3 / TypeScript frontend for the single-submit "brain dump" idea and defect capture flow described in [[prompt-to-idea]]. The "New Idea" button is re-wired to open a new `BrainDumpModal` instead of the existing `IdeaChatPanel`. The modal provides a large textarea, sends input to the `POST /ideas/generate` endpoint from the backend plan ([[prompt-to-idea]]-be), displays a rendered markdown preview, and lets the user accept, edit, or discard the proposal. `IdeaChatPanel` is retained in the codebase for future use by [[flesh-out-ideas-with-agent]].

The backend endpoint is specified in [[prompt-to-idea]]-be Milestone 3. Integration tests are covered in [[prompt-to-idea]]-test.

---

## Milestone 1 – API Client Function and Types

### Description

Add a new API function to call the `POST /ideas/generate` endpoint and define the corresponding TypeScript types. This follows the same pattern as the existing `converseIdea` function in `web/src/api/ideaChat.ts`.

### Files to change

- **`web/src/api/ideaChat.ts`** — Add `generateIdea(project: string, input: string, type?: 'idea' | 'defect'): Promise<IdeaGenerateResponse>` that calls `POST /api/p/${project}/ideas/generate` with `{ input, type }` using the existing `request()` helper from `api/client.ts`. Returns the parsed `IdeaGenerateResponse`.

- **`web/src/types/api.ts`** — Add the `IdeaGenerateResponse` interface:
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
- [ ] `generateIdea(project, "some text", "defect")` passes the `type` field in the request body.
- [ ] TypeScript compiles without errors (`pnpm exec vue-tsc --noEmit`).

---

## Milestone 2 – Brain Dump Pinia Store

### Description

Create a new Pinia store (`useBrainDumpStore`) that manages the state machine for the single-submit flow: `input → generating → preview → (editing → preview)* → accepted | discarded`. This store is separate from `useIdeaChatStore` (which remains for the conversational flow).

### Files to change

- **`web/src/stores/brainDump.ts`** (new) — Pinia store with:
  - **State**: `input: string`, `artifactType: 'idea' | 'defect'`, `phase: 'input' | 'generating' | 'preview' | 'editing'`, `error: string | null`, `proposal: IdeaGenerateResponse | null`, `editedBody: string | null`.
  - **Getters**: `canSubmit` — true when `input.trim().length > 0` and phase is `input`.
  - **Actions**:
    - `generate(project: string)` — sets phase to `generating`, calls `generateIdea()`, transitions to `preview` on success. On 400 error, captures the error message and transitions back to `input`. On other errors, shows generic message.
    - `acceptProposal(project: string)` — writes the artifact via `POST /api/p/${project}/artifacts` using the proposal data. The `stage` is derived from `target_dir` (strip `lifecycle/` prefix). On success, returns the artifact path.
    - `startEdit()` — transitions to `editing`, copies `proposal.body` into `editedBody`.
    - `applyEdit()` — validates `editedBody` is non-empty, merges into `proposal.body`, transitions back to `preview`.
    - `discard()` — resets all state to initial values.
    - `reset()` — alias for `discard()`.

### Acceptance criteria

- [ ] Store transitions through `input → generating → preview` on successful generate.
- [ ] Store transitions through `preview → editing → preview` when editing.
- [ ] `acceptProposal` calls `POST /artifacts` with correct stage, slug, frontmatter, and body.
- [ ] `discard` returns all state to initial values.
- [ ] API error messages are captured in `error` and phase returns to `input`.
- [ ] TypeScript compiles without errors.

---

## Milestone 3 – BrainDumpModal Component

### Description

Build the `BrainDumpModal` component that replaces `IdeaChatPanel` as the UI for the "New Idea" button. The modal has three visual phases: input, generating (loading), and preview. It follows the same overlay pattern used by `IdeaChatPanel` (Teleport to body, fixed overlay, focus trap) but with a simpler layout centred on a textarea rather than a chat transcript.

### Files to change

- **`web/src/components/idea/BrainDumpModal.vue`** (new) — SFC with:
  - **Props**: `project: string`, `artifactType?: 'idea' | 'defect'` (default `'idea'`).
  - **Emits**: `close`, `created(path: string)`.
  - **Input phase** (FR-6.1, FR-6.2):
    - Teleport to body with fixed overlay (same z-index pattern as `IdeaChatPanel`: `.icp-overlay`).
    - Header: "New Idea" or "New Defect" depending on `artifactType`, with a close button.
    - Large auto-growing `<textarea>` with placeholder *"Describe your idea — paste, ramble, brain dump..."* (or defect-specific placeholder). Minimum 6 rows, `resize: vertical`, `width: 100%`.
    - "Generate" button below the textarea. Disabled when `!canSubmit`.
    - Keyboard: `Ctrl+Enter` / `Cmd+Enter` submits (FR-6.3). `Escape` closes — with a `window.confirm()` discard confirmation if text has been entered (FR-6.3).
    - Textarea supports multi-line paste natively (FR-6.4).
  - **Generating phase** (FR-6.2):
    - Textarea becomes `disabled`. "Generate" button shows a spinner or animated dots.
  - **Preview phase** (FR-5.1, FR-5.2):
    - Rendered markdown preview using the existing `MarkdownPreview` component (`web/src/components/artifact/MarkdownPreview.vue`) which already supports `[[slug]]` wiki-links.
    - Metadata summary above the body: title (as heading), slug shown as `<slug>.md`, labels as styled chips/tags.
    - Three action buttons: **Accept** (primary), **Edit** (secondary), **Discard** (secondary).
    - Accept calls `store.acceptProposal()` and emits `created(path)`.
    - Edit transitions to editing phase: textarea re-appears pre-filled with the generated body and a "Done editing" button that calls `store.applyEdit()`.
    - Discard calls `store.discard()` and emits `close`.
  - **Error display**: Inline error message below the textarea when `store.error` is set.
  - **Accessibility**: focus trap within modal, `aria-modal="true"`, `role="dialog"`, `aria-labelledby` pointing to the header.
  - **Styling**: Reuse CSS custom properties from the existing design system (`--color-*`, `--space-*`, `--radius-*`). Modal width: `max-width: 640px`, centred. Responsive down to 768px viewport (NFR-2).

### Acceptance criteria

- [ ] Modal opens with a large textarea and "Generate" button.
- [ ] `Ctrl+Enter` / `Cmd+Enter` submits the input.
- [ ] `Escape` closes the modal (with confirmation if text entered).
- [ ] During generation, textarea is disabled and button shows loading state.
- [ ] After generation, a rendered markdown preview is shown with metadata summary.
- [ ] Accept writes the artifact and emits `created`.
- [ ] Edit shows textarea pre-filled with generated body; "Done editing" returns to preview.
- [ ] Discard closes the modal.
- [ ] Errors display inline below the textarea.
- [ ] Modal is keyboard-navigable with focus trap.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 4 – Wire New Idea Button to BrainDumpModal

### Description

Replace the `IdeaChatPanel` usage in `ArtifactListView` with the new `BrainDumpModal`. The "New Idea" button opens `BrainDumpModal` instead of `IdeaChatPanel`. `IdeaChatPanel` remains in the codebase (the component file is not deleted) for future use by [[flesh-out-ideas-with-agent]], per Open Question 1.

### Files to change

- **`web/src/views/project/ArtifactListView.vue`**:
  - Replace `import IdeaChatPanel` with `import BrainDumpModal`.
  - Replace `<IdeaChatPanel>` usage (currently at line ~111-118 via Teleport) with `<BrainDumpModal :project="..." :artifact-type="'idea'" @close="showBrainDump = false" @created="onBrainDumpCreated">`.
  - Rename `showIdeaChat` / `openIdeaChat` to `showBrainDump` / `openBrainDump`.
  - Add `onBrainDumpCreated(path)` handler that shows a success toast via the `useUiStore` (which has `addToast` for info/success/error) and optionally navigates to the new artifact via `router.push`.

### Acceptance criteria

- [ ] Clicking "New Idea" opens the `BrainDumpModal`, not `IdeaChatPanel`.
- [ ] On accept, the user sees a success toast and can navigate to the new artifact.
- [ ] On discard, the modal closes cleanly.
- [ ] `IdeaChatPanel.vue` remains in the codebase (not deleted).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 5 – Add "New Defect" Entry Point

### Description

Add a "New Defect" button alongside the "New Idea" button that opens the same `BrainDumpModal` with `artifactType="defect"`. This satisfies the requirement's Open Question 2 (defect brain-dump capture included).

### Files to change

- **`web/src/views/project/ArtifactListView.vue`** — Add a "New Defect" button next to "New Idea" in the header. Both open `BrainDumpModal` but set different `artifactType` props. Use distinct icons: the existing `MessageSquarePlus` (or `Lightbulb`) for ideas and `Bug` for defects (both available from `lucide-vue-next`). Track the selected type in a ref so a single `BrainDumpModal` instance can serve both.

- **`web/src/components/idea/BrainDumpModal.vue`** — Ensure header text and placeholder text adapt based on `artifactType`:
  - Idea: header "New Idea", placeholder "Describe your idea — paste, ramble, brain dump..."
  - Defect: header "New Defect", placeholder "Describe the defect — what happened, what you expected..."

### Acceptance criteria

- [ ] "New Defect" button is visible on the artifact list view.
- [ ] Clicking "New Defect" opens the modal with defect-specific header and placeholder.
- [ ] Generated defect proposal has `type: defect` in frontmatter.
- [ ] On accept, the defect artifact is written to `lifecycle/defects/`.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 6 – Error Handling and Edge Cases

### Description

Handle error states from the generate endpoint gracefully in the UI: short input rejection (400), LLM failures (500), and network errors. Also handle the editing edge case where the user clears the body.

### Files to change

- **`web/src/components/idea/BrainDumpModal.vue`** — Display `store.error` as an inline message below the textarea (styled with `--color-error` or equivalent). After an error, the textarea remains editable so the user can revise and retry without reopening the modal. When in editing phase, validate that `editedBody` is non-empty before allowing "Done editing".

- **`web/src/stores/brainDump.ts`** — Ensure `generate` action catches HTTP errors: for 400 responses, extract and store the `error` field from the response body. For network or 500 errors, set a generic "Something went wrong — please try again." message. In both cases, transition phase back to `input`.

### Acceptance criteria

- [ ] Submitting input that is too short shows the backend's error message below the textarea.
- [ ] Network or server errors show a generic retry message.
- [ ] After an error, the user can edit and resubmit without reopening the modal.
- [ ] Editing the body to empty and clicking "Done editing" shows a validation message.
- [ ] Modal layout works on viewports >= 768px wide (NFR-2).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.
