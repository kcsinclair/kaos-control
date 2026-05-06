---
title: Single-Submit Idea Capture (Brain Dump Mode)
type: requirement
status: done
lineage: prompt-to-idea
parent: lifecycle/ideas/idea-capture-vs-interactive.md
labels:
    - agent
    - workflow
    - usability
    - enhancement
assignees:
    - role: product-owner
      who: agent
---

# Single-Submit Idea Capture (Brain Dump Mode)

## Problem

The current "New Idea" flow opens a chat panel (`IdeaChatPanel`) that uses a multi-turn conversational agent. While the conversational approach can extract detail, it imposes friction at the moment of capture: the user must wait for agent replies, answer clarifying questions, and navigate a propose-then-confirm loop before an idea is persisted.

Most idea capture happens in a "brain dump" moment — the user already has the thought formed (or has text copied from elsewhere) and wants to record it immediately. Forcing a conversation at this stage discourages capture. Interactive refinement is a separate concern, addressed by [[flesh-out-ideas-with-agent]] after the idea already exists.

## Goals / Non-goals

### Goals

1. Replace the chat-based New Idea modal with a single large text area where the user writes or pastes their idea in free-form text.
2. On submission, the text is sent to the LLM in a single request; the LLM derives all frontmatter (`title`, `slug`, `lineage`, `labels`, `status`, `type`) and produces a well-structured markdown body — no conversational turns.
3. The user sees a preview of the generated artifact and can accept, edit, or discard it — one round-trip, not a multi-turn conversation.
4. Keep the UI path to a persisted idea as short as possible: open modal → paste/type → submit → preview → accept.
5. The existing `idea-capture` inline agent configuration in `lifecycle/config.yaml` is updated (not replaced) to support this single-shot mode.

### Non-goals

- This requirement does **not** cover interactive brainstorming or idea fleshing-out — that is [[flesh-out-ideas-with-agent]].
- The conversational converse endpoint (`POST /api/p/:project/ideas/converse`) may be retained for future use by the flesh-out flow but is no longer the primary capture path.
- Batch idea creation (multiple ideas from one paste) is out of scope.
- CLI-based idea capture is out of scope.

## Detailed Requirements

### Functional

#### FR-1: Single-submit endpoint

- **FR-1.1**: The backend exposes `POST /api/p/:project/ideas/generate` accepting `{ "input": string }`.
- **FR-1.2**: The endpoint sends the input text to the configured LLM with a system prompt that instructs it to produce a structured JSON response containing `slug`, `title`, `labels`, and `body`.
- **FR-1.3**: The endpoint returns `{ "slug": string, "title": string, "labels": string[], "body": string, "frontmatter": object }` — the complete proposed artifact, but not yet written to disk.
- **FR-1.4**: The LLM prompt must select labels only from the set of labels already in use across existing idea artifacts (no free invention).
- **FR-1.5**: If the input is too short or unintelligible to produce a meaningful idea (e.g., fewer than 5 words with no discernible intent), the endpoint returns an error with a user-facing message asking for more detail — it does **not** enter a conversation.

#### FR-2: Slug and filename generation

- **FR-2.1**: The LLM derives a slug from the content: lowercase, hyphen-separated, 2–5 words, matching `^[a-z0-9][a-z0-9\-]*[a-z0-9]$`.
- **FR-2.2**: The backend validates the slug against existing files in `lifecycle/ideas/`. On collision, the backend appends a disambiguating numeric suffix (e.g., `my-idea-2`) and includes the adjusted slug in the response.
- **FR-2.3**: The filename is `lifecycle/ideas/<slug>.md` with no lineage index suffix (originating idea convention).

#### FR-3: Frontmatter generation

The generated artifact must contain valid frontmatter:

| Field      | Value                                           |
|------------|-------------------------------------------------|
| `title`    | Derived from content by the LLM                 |
| `type`     | `idea`                                          |
| `status`   | `draft`                                         |
| `lineage`  | Same as slug                                    |
| `labels`   | 1–5 labels selected from existing label corpus  |
| `priority` | `normal` (unless the user's text signals urgency)|

#### FR-4: Body generation

- **FR-4.1**: The LLM produces a markdown body with a level-1 heading matching the title and 1–3 paragraphs that capture the idea.
- **FR-4.2**: The body must be self-contained — a reader unfamiliar with the raw input should understand the idea.
- **FR-4.3**: If the user's input references other ideas or artifacts by name, the body should include `[[slug]]` links where identifiable.

#### FR-5: Preview and confirmation

- **FR-5.1**: After the LLM responds, the UI shows a rendered markdown preview of the full artifact (frontmatter summary + body).
- **FR-5.2**: The user can **Accept** (writes to disk), **Discard** (closes modal, nothing persisted), or **Edit** (opens the text area again with the generated body pre-filled so the user can tweak before re-submitting or directly accepting the edited version).
- **FR-5.3**: On accept, the backend writes the file via `POST /api/p/:project/artifacts` (existing endpoint) and the UI navigates to the new artifact or shows a success notification with a link.

#### FR-6: UI — New Idea modal

- **FR-6.1**: The "New Idea" button opens a modal (or full-width drawer) containing a large, auto-growing `<textarea>` with placeholder text like *"Describe your idea — paste, ramble, brain dump…"*.
- **FR-6.2**: A single "Generate" button submits the text. While the LLM is processing, the button shows a loading state and the textarea is disabled.
- **FR-6.3**: The modal must be keyboard-navigable: `Ctrl+Enter` / `Cmd+Enter` submits, `Escape` closes (with discard confirmation if text has been entered).
- **FR-6.4**: The textarea must support paste of multi-line text and preserve formatting.

#### FR-7: Agent configuration update

- **FR-7.1**: The existing `idea-capture` agent entry in `lifecycle/config.yaml` is updated with a `prompt_templates.idea-generate` template for the single-shot mode.
- **FR-7.2**: The prompt instructs the LLM to return structured JSON (same schema as the current `idea-capture` `propose` action) in a single response, with no clarifying questions.

### Non-functional

- **NFR-1**: End-to-end latency from submit to preview must be under 8 seconds (p95). The single LLM call should be faster than multiple conversational round-trips.
- **NFR-2**: The modal must work on viewports ≥ 768px wide (tablet and above).
- **NFR-3**: The feature must not break existing artifact creation via `POST /api/p/:project/artifacts`.
- **NFR-4**: The existing conversational endpoint (`/ideas/converse`) remains functional (no removal) for potential future use by [[flesh-out-ideas-with-agent]].

## Acceptance Criteria

- [ ] The "New Idea" button opens a modal with a large text area, not a chat panel.
- [ ] The user can type or paste free-form text and submit with a single button click or `Ctrl+Enter`.
- [ ] A single LLM call generates slug, title, labels, and structured body — no multi-turn conversation.
- [ ] Labels are selected from the existing label corpus, not freely invented.
- [ ] The user sees a rendered markdown preview of the proposed artifact before it is persisted.
- [ ] The user can accept, discard, or edit-then-accept the proposal.
- [ ] On accept, the artifact is written to `lifecycle/ideas/<slug>.md` with correct frontmatter (`title`, `type: idea`, `status: draft`, `lineage`, `labels`).
- [ ] Slug collisions are detected and resolved with a numeric suffix.
- [ ] Input that is too short or unintelligible returns a user-friendly error, not a broken artifact.
- [ ] The existing `POST /api/p/:project/artifacts` and `/ideas/converse` endpoints continue to work unchanged.
- [ ] The `idea-capture` agent config in `lifecycle/config.yaml` includes the new `idea-generate` prompt template.
- [ ] Related lineage: [[prompt-to-idea]]

## Resolved Questions

1. **Discard the chat panel entirely?** Should `IdeaChatPanel` be removed from the codebase now, or retained as the future UI for [[flesh-out-ideas-with-agent]]? The current recommendation is to retain it but disconnect it from the "New Idea" button.

> Keep it for use later.  New Idea button should use new method.

2. **Defect capture reuse**: The parent idea mentions this pattern should also work for new defects (`"This new idea assistance is great, I think it should be new idea or new defect"`). Should defect brain-dump capture be included in this requirement or tracked as a separate idea?

> yes, please include, good to get it included now.
