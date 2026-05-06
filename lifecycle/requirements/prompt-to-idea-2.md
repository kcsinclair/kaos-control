---
title: Conversational Idea Capture Agent
type: requirement
status: done
lineage: prompt-to-idea
parent: lifecycle/ideas/prompt-to-idea.md
labels:
    - artefacts
    - workflow
    - usability
    - agent
assignees:
    - role: product-owner
      who: agent
---

# Conversational Idea Capture Agent

## Problem

Creating an idea artifact today requires the user to manually author a markdown file with correct frontmatter (`title`, `type`, `status`, `lineage`, slug-based filename) and a well-structured body. This imposes friction at the moment of highest creative energy: a user has a vague thought and must context-switch into file mechanics before the thought is captured.

The result is either (a) ideas never get recorded, or (b) they are recorded in low-quality form that later requires significant analyst effort to interpret.

## Goals / Non-goals

### Goals

1. Let a user describe an idea in natural language via a chat-style interface and have an LLM-backed agent produce a complete, valid idea artifact in `lifecycle/ideas/`.
2. The agent derives the slug, filename, frontmatter fields, and structured body automatically — the user never touches YAML or filenames.
3. The agent asks short, focused clarifying questions when the input is too vague to produce a useful idea — but keeps the conversation lightweight (no deep requirements-level interrogation).
4. The resulting artifact is indistinguishable in format and quality from a manually authored idea, so downstream agents (analyst, planner) can consume it without special handling.
5. The feature integrates into the existing web UI so users don't need a separate tool.

### Non-goals

- This agent does **not** produce requirements, plans, or any artifact type other than `idea`.
- It does **not** replace the requirements-analyst agent; it feeds into it.
- It does **not** support batch creation of multiple ideas in a single conversation.
- It does **not** need to work outside the web UI (CLI-only usage is out of scope for v1).

## Detailed Requirements

### Functional

#### FR-1: Chat-based conversation endpoint

The backend exposes a new API endpoint that accepts a user message and returns an agent response, maintaining conversational state for the duration of a single idea-capture session.

- **FR-1.1**: `POST /api/p/:project/ideas/converse` accepts `{ "session_id": string|null, "message": string }`.
- **FR-1.2**: When `session_id` is null, the backend creates a new session and returns a `session_id` in the response.
- **FR-1.3**: The endpoint returns `{ "session_id": string, "reply": string, "status": "conversing" | "proposed" | "created" }`.
- **FR-1.4**: Conversation state is held in-memory (no persistence across server restarts required).

#### FR-2: Slug and filename generation

- **FR-2.1**: The agent derives a slug from the idea content — lowercase, hyphen-separated, 2-5 words, no special characters. Must pass the existing `isValidSlug` check.
- **FR-2.2**: If a file with the derived slug already exists in `lifecycle/ideas/`, the agent must detect the collision and adjust the slug (e.g., append a disambiguating word), informing the user.
- **FR-2.3**: The filename is `lifecycle/ideas/<slug>.md` with no index suffix (originating idea convention).

#### FR-3: Frontmatter generation

The agent produces valid frontmatter with at minimum:

| Field     | Value                              |
|-----------|------------------------------------|
| `title`   | Derived from conversation content  |
| `type`    | `idea`                             |
| `status`  | `draft`                            |
| `lineage` | Same as slug                       |
| `labels`  | 1-5 labels inferred from content   |

- **FR-3.1**: `priority` may optionally be set if the user expresses urgency; otherwise omitted (defaults to `normal`).

#### FR-4: Body generation

- **FR-4.1**: The agent produces a markdown body with a level-1 heading matching the slug (kebab-case, as per existing convention) and 1-3 concise paragraphs capturing the idea.
- **FR-4.2**: The body must be self-contained — a reader unfamiliar with the conversation should understand the idea without seeing the chat transcript.

#### FR-5: Clarifying questions

- **FR-5.1**: If the user's input is too vague to produce a meaningful idea (fewer than ~10 words, or no discernible feature/problem), the agent asks at most 3 clarifying questions across the session.
- **FR-5.2**: Each question must be a single sentence.
- **FR-5.3**: After 3 rounds of clarification, the agent produces the best idea it can from available context rather than continuing to ask.

#### FR-6: Proposal and confirmation flow

- **FR-6.1**: Before writing to disk, the agent returns a `"status": "proposed"` response containing the full artifact content (frontmatter + body) as a preview.
- **FR-6.2**: The user can accept (writes to disk), reject (discards session), or ask for changes (continues conversation).
- **FR-6.3**: On acceptance, the backend writes the file to `lifecycle/ideas/<slug>.md`, triggers the fsnotify watcher (indexing happens automatically), and returns `"status": "created"` with the artifact path.

#### FR-7: Agent configuration

- **FR-7.1**: A new agent entry `idea-capture` is added to `lifecycle/config.yaml` with role `[product-owner]`, scoped `allowed_write_paths: [lifecycle/ideas]`.
- **FR-7.2**: The agent uses the project's configured LLM (model field in config). Default: `sonnet` (fast, low-latency interaction is more important than depth here).

#### FR-8: Web UI integration

- **FR-8.1**: A "New Idea" button is added to the ideas list view or the app header.
- **FR-8.2**: Clicking it opens a chat panel (modal or side-drawer) with a text input and message history.
- **FR-8.3**: The proposed artifact preview renders as formatted markdown so the user can review it before confirming.
- **FR-8.4**: On creation, the UI navigates to the newly created artifact or shows a success notification with a link to it.

### Non-functional

- **NFR-1**: Round-trip latency for each conversational turn should be under 5 seconds (p95) when using a streaming-capable model.
- **NFR-2**: Conversation sessions expire after 30 minutes of inactivity; the backend cleans up in-memory state.
- **NFR-3**: The feature must not break existing idea creation via `POST /api/p/:project/artifacts` — both paths coexist.
- **NFR-4**: The agent prompt must be defined in `lifecycle/config.yaml` under `prompt_templates` so it can be tuned without code changes.

## Acceptance Criteria

- [ ] User can open a chat interface from the web UI and describe an idea in plain language.
- [ ] The agent asks at most 3 short clarifying questions if the input is vague.
- [ ] The agent generates a valid slug, frontmatter, and body without user intervention.
- [ ] The user sees a formatted preview of the proposed artifact before it is written to disk.
- [ ] On confirmation, the artifact is written to `lifecycle/ideas/<slug>.md` with correct frontmatter fields (`title`, `type: idea`, `status: draft`, `lineage`, `labels`).
- [ ] The artifact appears in the index and graph within the normal fsnotify debounce window (~150 ms).
- [ ] Slug collisions are detected and resolved automatically.
- [ ] Existing idea creation via the artifact API (`POST /api/p/:project/artifacts`) continues to work unchanged.
- [ ] Agent configuration lives in `lifecycle/config.yaml` alongside other agents.
- [ ] Conversation sessions are cleaned up after 30 minutes of inactivity.
- [ ] Related lineage: [[prompt-to-idea]]

## Resolved Questions

1. **Streaming responses**: Should the agent stream its replies token-by-token to the UI (via WebSocket or SSE), or is a request/response pattern sufficient for v1?

> request/reponse for v1 please.

2. **Authentication**: The current app supports auth — should idea-capture sessions be tied to the authenticated user, and should the user be recorded in the artifact frontmatter (e.g., `created_by`)?

> idea-capture will be done by an authenticated user.

3. **Label vocabulary**: Should the agent pick labels from a fixed vocabulary (defined in config), or freely infer them? Free inference risks label sprawl; a fixed list risks missing relevant tags.

> The agent should pick from existing labels.

4. **Edit after creation**: If the user wants to tweak the idea immediately after creation, should the chat panel support a follow-up "edit" mode, or should the user be directed to the existing artifact editor?

> existing artifact editor works.
