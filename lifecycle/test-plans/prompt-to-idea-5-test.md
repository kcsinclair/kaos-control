---
title: "Conversational Idea Capture – Test Plan"
type: plan-test
status: draft
lineage: prompt-to-idea
parent: lifecycle/requirements/prompt-to-idea-2.md
labels:
    - artefacts
    - workflow
    - agent
---

# Conversational Idea Capture – Test Plan

This plan defines the integration tests for the conversational idea capture feature described in [[prompt-to-idea]]. Tests validate the backend conversation endpoint, session lifecycle, slug generation, artifact creation, and coexistence with existing artifact APIs. Tests are implemented in the `tests/` directory and exercise the running server over HTTP.

---

## Milestone 1 – Session Lifecycle Tests

### Description

Test that conversation sessions are created, retrieved, and expire correctly.

### Files to change

- **`tests/idea_chat_session_test.go`** (new) – Integration tests for session creation and expiry.

### Acceptance criteria

- [ ] **Test: new session creation** – `POST /api/p/:project/ideas/converse` with `session_id: null` and a message returns HTTP 200 with a non-empty `session_id` and `status: "conversing"`.
- [ ] **Test: session reuse** – Sending a second message with the returned `session_id` returns HTTP 200 and the same `session_id`.
- [ ] **Test: unknown session** – Sending a message with a fabricated `session_id` returns HTTP 404 with code `session_not_found`.
- [ ] **Test: empty message rejected** – `POST` with `message: ""` returns HTTP 400.
- [ ] **Test: authentication required** – Request without a valid session cookie returns HTTP 401.

---

## Milestone 2 – Conversation Flow Tests

### Description

Test the multi-turn conversation flow: initial message, clarifying questions, and proposal generation.

### Files to change

- **`tests/idea_chat_converse_test.go`** (new) – Integration tests for conversation turns.

### Acceptance criteria

- [ ] **Test: vague input triggers clarification** – Sending a very short message (e.g., "something cool") returns `status: "conversing"` with a non-empty `reply` that is a question (heuristic: ends with `?`).
- [ ] **Test: detailed input produces proposal** – Sending a sufficiently detailed message (50+ words describing a feature) returns `status: "proposed"` with a non-null `preview` containing valid `frontmatter` and `body`.
- [ ] **Test: max 3 clarifications** – Sending 4 consecutive vague messages results in `status: "proposed"` by the 4th response (the agent stops asking and produces the best idea it can).
- [ ] **Test: proposed preview has required fields** – When `status` is `proposed`, `preview.frontmatter` contains `title` (non-empty string), `type: "idea"`, `status: "draft"`, `lineage` (non-empty string matching slug pattern), and `labels` (array of 1–5 strings).
- [ ] **Test: proposed preview body is valid** – `preview.body` starts with a level-1 heading (`# `) and contains at least one paragraph.

---

## Milestone 3 – Proposal Accept / Reject Tests

### Description

Test the confirmation flow: accepting writes the artifact to disk; rejecting discards the session.

### Files to change

- **`tests/idea_chat_confirm_test.go`** (new) – Integration tests for accept/reject actions.

### Acceptance criteria

- [ ] **Test: accept creates artifact** – After receiving a `proposed` response, sending `message: "__accept__"` returns `status: "created"` with a non-null `artifact_path` matching `lifecycle/ideas/<slug>.md`.
- [ ] **Test: artifact file exists on disk** – After accept, the file at `artifact_path` exists, parses as valid markdown with correct frontmatter (`type: idea`, `status: draft`, lineage matches slug).
- [ ] **Test: artifact appears in index** – `GET /api/p/:project/artifacts` with a lineage filter returns the newly created artifact.
- [ ] **Test: session deleted after creation** – Sending another message with the same `session_id` after creation returns HTTP 404.
- [ ] **Test: reject discards session** – After receiving a `proposed` response, sending `message: "__reject__"` returns `status: "conversing"` with `session_id: null`. A subsequent request with the old `session_id` returns HTTP 404.
- [ ] **Test: accept without proposal** – Sending `message: "__accept__"` on a session that is still `conversing` returns HTTP 409 with code `no_proposal`.

---

## Milestone 4 – Slug Generation and Collision Tests

### Description

Test that slugs are valid, derived from content, and that collisions are detected and resolved.

### Files to change

- **`tests/idea_chat_slug_test.go`** (new) – Integration tests for slug handling.

### Acceptance criteria

- [ ] **Test: slug is valid** – The `lineage` field in `preview.frontmatter` matches the regex `^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$`.
- [ ] **Test: slug derived from content** – For a message about "dark mode toggle for settings", the generated slug contains at least one of the key terms (e.g., `dark-mode`, `settings`, `toggle`).
- [ ] **Test: slug collision resolution** – Pre-create a file `lifecycle/ideas/dark-mode.md`, then start a conversation about "dark mode". The resulting `lineage` / slug must differ from `dark-mode` (the agent adjusts to avoid collision).
- [ ] **Test: slug length** – The generated slug is 2–5 hyphen-separated words (split by `-`, count is 2–5 segments after removing numeric suffixes).

---

## Milestone 5 – Label Constraint Tests

### Description

Test that the agent picks labels from the project's existing label vocabulary and does not invent new ones.

### Files to change

- **`tests/idea_chat_labels_test.go`** (new) – Integration tests for label generation.

### Acceptance criteria

- [ ] **Test: labels are from existing set** – Fetch the project's labels via `GET /api/p/:project/labels`, then create an idea via conversation. All labels in `preview.frontmatter.labels` are present in the project's label list.
- [ ] **Test: labels count in range** – `preview.frontmatter.labels` contains between 1 and 5 items.
- [ ] **Test: no duplicate labels** – `preview.frontmatter.labels` contains no duplicates.

---

## Milestone 6 – Coexistence and Regression Tests

### Description

Test that the new feature does not break existing artifact creation or other endpoints.

### Files to change

- **`tests/idea_chat_regression_test.go`** (new) – Regression tests for existing functionality.

### Acceptance criteria

- [ ] **Test: manual artifact creation still works** – `POST /api/p/:project/artifacts` with a valid idea payload succeeds (HTTP 201) and the artifact appears in the index. This is the existing endpoint; it must be unaffected by the new code.
- [ ] **Test: artifact update still works** – `PUT /api/p/:project/artifacts/lifecycle/ideas/<slug>.md` with updated frontmatter and body succeeds (HTTP 200).
- [ ] **Test: agent runs still work** – `POST /api/p/:project/agents/{name}/run` for an existing agent returns HTTP 200/202 (or the appropriate success code) — the new agent config does not interfere with existing agents.
- [ ] **Test: WebSocket events fire** – After accepting an idea via conversation, a WebSocket client connected to `/api/p/:project/ws` receives an `artifact.indexed` event with the new artifact's path.

---

## Milestone 7 – Agent Configuration Tests

### Description

Test that the `idea-capture` agent configuration is correctly loaded and accessible.

### Files to change

- **`tests/idea_chat_config_test.go`** (new) – Integration tests for agent config.

### Acceptance criteria

- [ ] **Test: agent listed in config** – `GET /api/p/:project/agents` returns a list that includes an agent named `idea-capture`.
- [ ] **Test: agent has correct fields** – The `idea-capture` agent entry has `model: sonnet` (or the configured model) and `allowed_write_paths` containing `lifecycle/ideas`.
- [ ] **Test: prompt template exists** – The agent config has a `prompt_templates` entry with an `idea-capture` key that is a non-empty string.

---

## Cross-references

- The [[prompt-to-idea]] backend plan defines the endpoint and session store being tested.
- The [[prompt-to-idea]] frontend plan defines the UI that drives the conversation flow; frontend-specific tests (component tests) are out of scope for this plan but may be added later.
