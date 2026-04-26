---
title: "Conversational Idea Capture – Integration Tests"
type: test
status: draft
lineage: prompt-to-idea
parent: lifecycle/test-plans/prompt-to-idea-5-test.md
labels:
    - artefacts
    - workflow
    - agent
---

# Conversational Idea Capture – Integration Tests

Integration tests for the conversational idea capture feature (`POST /api/p/:project/ideas/converse`). All tests run against a live server started in-process and exercise the full HTTP stack.

## Test files

| File | Milestone |
|------|-----------|
| `tests/integration/idea_chat_helpers_test.go` | Shared helpers |
| `tests/integration/idea_chat_session_test.go` | M1 – Session lifecycle |
| `tests/integration/idea_chat_converse_test.go` | M2 – Conversation flow |
| `tests/integration/idea_chat_confirm_test.go` | M3 – Accept / reject |
| `tests/integration/idea_chat_slug_test.go` | M4 – Slug generation |
| `tests/integration/idea_chat_labels_test.go` | M5 – Label constraints |
| `tests/integration/idea_chat_regression_test.go` | M6 – Coexistence |
| `tests/integration/idea_chat_config_test.go` | M7 – Agent config |

## Scenarios covered

### M1 – Session lifecycle (`idea_chat_session_test.go`)

- **Unknown session** – fabricated `session_id` returns HTTP 404 / `session_not_found`. No LLM required.
- **Empty message** – `message: ""` returns HTTP 400. No LLM required.
- **Unauthenticated** – request without a session cookie returns HTTP 401. No LLM required.
- **New session creation** – first POST (no `session_id`) returns HTTP 200 with a non-empty `session_id` and `status: "conversing"` or `"proposed"`. Requires `ANTHROPIC_API_KEY`.
- **Session reuse** – second POST with the returned `session_id` produces HTTP 200 and the same `session_id`. Requires `ANTHROPIC_API_KEY`.

### M2 – Conversation flow (`idea_chat_converse_test.go`)

- **Vague input triggers clarification** – short/vague message produces `status: "conversing"` with a reply ending in `?`.
- **Detailed input produces proposal** – 50+ word message drives conversation to `status: "proposed"` with non-null `preview`.
- **Max 3 clarifications** – four consecutive vague messages result in `status: "proposed"` by the fourth response.
- **Proposal frontmatter required fields** – `preview.frontmatter` contains `title` (string), `type: "idea"`, `status: "draft"`, `lineage` (slug pattern), `labels` (array).
- **Proposal body valid** – `preview.body` starts with `# ` and contains at least one paragraph.

All M2 tests require `ANTHROPIC_API_KEY`.

### M3 – Accept / reject (`idea_chat_confirm_test.go`)

- **Accept without proposal** – `__accept__` on a non-proposed session returns HTTP 409 / `no_proposal`. No LLM required.
- **Accept creates artifact** – `__accept__` after a proposal returns `status: "created"` and `artifact_path` matching `lifecycle/ideas/<slug>.md`.
- **Artifact file exists on disk** – artifact file is present with correct frontmatter (`type: idea`, `status: draft`, correct lineage).
- **Artifact appears in index** – `GET /artifacts?lineage=<slug>` returns the new artifact.
- **Session deleted after creation** – subsequent message with old `session_id` returns HTTP 404.
- **Reject discards session** – `__reject__` returns `status: "conversing"`, `session_id: null`; old ID returns HTTP 404.

LLM-dependent tests require `ANTHROPIC_API_KEY`.

### M4 – Slug generation (`idea_chat_slug_test.go`)

- **Slug is valid** – `lineage` in `preview.frontmatter` matches `^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$`.
- **Slug derived from content** – slug for a "dark mode toggle for settings" idea contains at least one of: `dark`, `mode`, `toggle`, `settings`, `theme`.
- **Slug collision resolution** – pre-creating `lifecycle/ideas/dark-mode.md` forces the generated slug to differ from `dark-mode`.
- **Slug length** – slug (excluding trailing numeric disambiguator) has 2–5 hyphen-separated segments.

All M4 tests require `ANTHROPIC_API_KEY`.

### M5 – Label constraints (`idea_chat_labels_test.go`)

- **Labels from existing vocabulary** – all labels in `preview.frontmatter.labels` are present in `GET /api/p/:project/labels`.
- **Labels count in range** – `preview.frontmatter.labels` contains 0–5 items.
- **No duplicate labels** – `preview.frontmatter.labels` has no repeated entries.

All M5 tests seed the project with artifacts carrying known labels, then require `ANTHROPIC_API_KEY`.

### M6 – Coexistence and regression (`idea_chat_regression_test.go`)

- **Manual artifact creation unchanged** – `POST /api/p/:project/artifacts` returns HTTP 201 and file is correctly placed.
- **Artifact update unchanged** – `PUT /api/p/:project/artifacts/*` returns HTTP 200 with updated body.
- **Agent endpoint accessible** – `GET /api/p/:project/agents` returns HTTP 200 with `agents` key (routing unbroken).
- **WebSocket events fire** – after accepting a proposal, a connected WebSocket client receives an `artifact.indexed` event with the new artifact path. Requires `ANTHROPIC_API_KEY`.

### M7 – Agent configuration (`idea_chat_config_test.go`)

Uses `newTestEnvCustomConfig` (in `idea_chat_helpers_test.go`) to start a project that includes the `idea-capture` agent from the start.

- **Agent listed in config** – `GET /api/p/:project/agents` includes an entry named `idea-capture`.
- **Agent has correct fields** – `driver: inline`, `allowed_write_paths` contains `lifecycle/ideas`.
- **Prompt template exists** – `GET /api/p/:project/config` raw YAML contains `prompt_templates:` and `idea-capture:` with non-trivial content.

None of the M7 tests require `ANTHROPIC_API_KEY`.

## Running the tests

```sh
# All integration tests (including LLM-dependent ones):
ANTHROPIC_API_KEY=sk-... go test -tags integration ./tests/integration/ -run 'TestIdeaChat' -v -timeout 5m

# Only non-LLM tests (safe for CI without an API key):
go test -tags integration ./tests/integration/ -run 'TestIdeaChat(UnknownSession|EmptyMessage|AuthRequired|AcceptWithoutProposal|ManualArtifact|ArtifactUpdate|AgentEndpoint|AgentListed|AgentHasCorrect|PromptTemplate)' -v
```
