---
title: "Conversational Idea Capture – Frontend Plan"
type: plan-frontend
status: in-development
lineage: prompt-to-idea
parent: lifecycle/requirements/prompt-to-idea-2.md
labels:
    - artefacts
    - workflow
    - usability
---

# Conversational Idea Capture – Frontend Plan

This plan implements the web UI components for the conversational idea capture feature described in [[prompt-to-idea]]. The frontend provides a chat panel (modal dialog), integrates with the backend conversation endpoint defined in the [[prompt-to-idea]] backend plan, and handles the proposal preview/confirmation flow.

---

## Milestone 1 – API Client for Idea Conversation

### Description

Add an API module that wraps the `POST /api/p/:project/ideas/converse` endpoint. This provides a typed interface for the rest of the frontend to use.

### Files to change

- **`web/src/api/ideaChat.ts`** (new) – Export `converseIdea(project: string, sessionId: string | null, message: string): Promise<IdeaConverseResponse>`.
- **`web/src/types/api.ts`** (or equivalent types file) – Add `IdeaConverseResponse` type and extend `WsEventType` if new event types are needed.

### Acceptance criteria

- [ ] `IdeaConverseResponse` type matches the backend response: `{ session_id: string; reply: string; status: 'conversing' | 'proposed' | 'created'; preview: { frontmatter: Record<string, any>; body: string } | null; artifact_path: string | null }`.
- [ ] `converseIdea` calls `api.post(`/p/${project}/ideas/converse`, { session_id, message })`.
- [ ] Errors are propagated as `ApiError` (existing pattern).
- [ ] `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 2 – Idea Chat Store (Pinia)

### Description

Create a Pinia store that manages the state of an active idea-capture conversation: session ID, message list, loading state, conversation status, and the proposed artifact preview.

### Files to change

- **`web/src/stores/ideaChat.ts`** (new) – Define and export `useIdeaChatStore`.

### Acceptance criteria

- [ ] State: `sessionId: string | null`, `messages: Array<{ role: 'user' | 'assistant'; content: string }>`, `status: 'idle' | 'conversing' | 'proposed' | 'created'`, `loading: boolean`, `preview: { frontmatter: Record<string, any>; body: string } | null`, `createdPath: string | null`.
- [ ] Action `sendMessage(project: string, text: string)`: appends user message to `messages`, sets `loading: true`, calls `converseIdea`, appends assistant reply to `messages`, updates `status`, `preview`, `sessionId`. Sets `loading: false` on completion or error.
- [ ] Action `acceptProposal(project: string)`: calls `converseIdea` with message `"__accept__"`, updates `status` to `created`, stores `createdPath`.
- [ ] Action `rejectProposal(project: string)`: calls `converseIdea` with message `"__reject__"`, resets store to idle state.
- [ ] Action `reset()`: clears all state back to initial values.
- [ ] Error handling: on API error, sets `loading: false` and surfaces the error message via the `useUiStore().error()` toast.
- [ ] `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 3 – Chat Panel Component

### Description

Build the `IdeaChatPanel.vue` component — a modal dialog containing a scrollable message history, a text input, and action buttons. This is the primary user-facing component for the feature.

### Files to change

- **`web/src/components/idea/IdeaChatPanel.vue`** (new) – SFC with `<script setup lang="ts">`, `<template>`, `<style scoped>`.

### Acceptance criteria

- [ ] The component renders as a fixed-position modal overlay (matching existing dialog patterns: `position: fixed; inset: 0; background: rgba(0,0,0,0.45)`), with a centered panel (max-width `560px`, max-height `80vh`).
- [ ] Header shows "New Idea" title and a close button (lucide `X` icon).
- [ ] Message area is a scrollable container. User messages are right-aligned with accent background; assistant messages are left-aligned with surface background. Messages render as plain text (no markdown in the chat itself).
- [ ] The message area auto-scrolls to the bottom when new messages are added.
- [ ] Text input area at the bottom: a `<textarea>` (auto-growing, max 4 rows) with a send button (lucide `SendHorizontal` icon). Enter sends (Shift+Enter for newline). Disabled while `loading` is true.
- [ ] A loading indicator (three-dot animation or spinner) appears in the message area while waiting for the agent's reply.
- [ ] When `status` is `proposed`, the input area is replaced by a preview section and three action buttons: "Accept", "Edit", and "Discard".
- [ ] The preview section renders the proposed artifact's markdown body using `markdown-it` (existing dependency) inside a bordered preview card.
- [ ] "Edit" returns the conversation to `conversing` status by sending a user message like "I'd like to make some changes" and re-enabling the text input.
- [ ] "Accept" calls `acceptProposal`. On success, shows a success toast and either navigates to the new artifact or shows a link.
- [ ] "Discard" calls `rejectProposal`, shows an info toast, and closes the panel.
- [ ] Pressing Escape or clicking the overlay closes the panel (emits `close` event). If a conversation is in progress (`status !== 'idle'`), shows a brief confirmation ("Discard this conversation?") before closing.
- [ ] The component uses CSS custom properties from `tokens.css` for all colours, spacing, and radii.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 4 – Integration into Artifact List View

### Description

Add a "New Idea" button to the `ArtifactListView` that opens the chat panel. Wire up navigation on successful idea creation.

### Files to change

- **`web/src/views/project/ArtifactListView.vue`** – Add a "New Idea" button in the header/toolbar area. Import and conditionally render `IdeaChatPanel`. Handle the `close` event and post-creation navigation.
- **`web/src/components/idea/IdeaChatPanel.vue`** – Emit `created(path: string)` event on successful creation so the parent can navigate.

### Acceptance criteria

- [ ] A "New Idea" button (lucide `Sparkles` or `MessageSquarePlus` icon + text) is visible in the `ArtifactListView` toolbar, next to the existing filter controls.
- [ ] Clicking the button sets a local `showIdeaChat` ref to `true` and renders `<IdeaChatPanel>` via `<Teleport to="body">`.
- [ ] On `created(path)` event, the view navigates to `/p/:project/artifacts/<path>` using `router.push`.
- [ ] On `close` event, the `IdeaChatPanel` is unmounted and `ideaChatStore.reset()` is called.
- [ ] The button is visually consistent with existing toolbar actions (same padding, radius, font).
- [ ] The artifact list refreshes (via existing `artifact.indexed` WebSocket listener) after creation — no manual refresh needed.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 5 – Proposal Preview Polish

### Description

Refine the proposal preview rendering to show both the frontmatter metadata (as a summary table) and the markdown body, so the user has full visibility before accepting.

### Files to change

- **`web/src/components/idea/IdeaChatPanel.vue`** – Enhance the preview section.

### Acceptance criteria

- [ ] The preview card has two sections: a metadata summary and a body preview.
- [ ] Metadata summary shows: title, slug (as filename `<slug>.md`), labels (as chips/badges), and lineage.
- [ ] Body preview renders the markdown via `markdown-it` with the same styling used in `ArtifactEditorView`'s preview pane.
- [ ] The preview card has a subtle border and distinct background (`--color-surface-raised` or similar) to visually separate it from chat messages.
- [ ] The preview card is scrollable if the content exceeds the available space.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 6 – Accessibility and Keyboard Navigation

### Description

Ensure the chat panel meets basic accessibility standards: focus management, ARIA attributes, and keyboard operability.

### Files to change

- **`web/src/components/idea/IdeaChatPanel.vue`** – Add ARIA attributes and focus management.

### Acceptance criteria

- [ ] The modal has `role="dialog"` and `aria-modal="true"`.
- [ ] The modal has `aria-labelledby` pointing to the header text.
- [ ] Focus is trapped within the modal while open (Tab cycles through interactive elements within the panel).
- [ ] On open, focus moves to the text input.
- [ ] On close, focus returns to the "New Idea" button.
- [ ] Screen reader announces new assistant messages via `aria-live="polite"` region.
- [ ] All interactive elements (buttons, textarea) have visible focus indicators using the existing `--color-accent` outline style.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Cross-references

- The [[prompt-to-idea]] backend plan defines the `POST /api/p/:project/ideas/converse` endpoint consumed by this UI.
- The [[prompt-to-idea]] test plan covers end-to-end tests for the chat flow, including proposal preview and artifact creation.
