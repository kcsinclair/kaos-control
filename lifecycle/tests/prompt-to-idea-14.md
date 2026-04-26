---
title: "Single-Submit Idea & Defect Capture – Integration Tests"
type: test
status: draft
lineage: prompt-to-idea
parent: lifecycle/test-plans/prompt-to-idea-13-test.md
---

# Single-Submit Idea & Defect Capture – Integration Tests

Integration tests for the single-submit generate endpoint (`POST /api/p/:project/ideas/generate`). All tests run against a live server started in-process via the existing `testEnv` helper infrastructure.

## Test file

| File | Milestones covered |
|------|--------------------|
| `tests/integration/idea_generate_test.go` | M1–M8 |

## Scenarios covered

### M1 – Happy path (idea) (`TestIdeaGenerate_HappyPath`)

Sends a well-formed idea input and verifies the response contains all required fields: `slug`, `title`, `labels`, `body`, `frontmatter`, `target_dir`. Asserts:

- `slug` matches `^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$`
- `target_dir == "lifecycle/ideas"`
- `frontmatter.type == "idea"`, `frontmatter.status == "draft"`, `frontmatter.lineage == slug`
- `body` contains a level-1 heading (`# `)
- No file is written to disk (preview-only)

Requires `ANTHROPIC_API_KEY`.

### M2 – Input validation (4 tests, no API key required)

- **`TestIdeaGenerate_TooShort`** – single-word input (`"hi"`) → HTTP 400 with `error` field.
- **`TestIdeaGenerate_EmptyInput`** – empty string → HTTP 400.
- **`TestIdeaGenerate_MissingInput`** – request body with no `input` key → HTTP 400.
- **`TestIdeaGenerate_FewWords`** – two-word input (`"fix bug"`) → HTTP 400.

All confirm that validation occurs before any LLM call.

### M3 – Defect mode (`TestIdeaGenerate_DefectMode`)

Posts with `type: "defect"`. Asserts:

- `frontmatter.type == "defect"`
- `target_dir == "lifecycle/defects"`
- Body contains at least one structured defect section (Reproduction Steps, Expected / Actual Behaviour)
- `labels` includes `"defect"`

Requires `ANTHROPIC_API_KEY`.

### M4 – Slug collision detection (`TestIdeaGenerate_SlugCollision`)

Pre-seeds `lifecycle/ideas/dark-mode.md`. Generates an idea with input that strongly implies `dark-mode` as the slug. Asserts:

- Returned slug is not `"dark-mode"` (disambiguated)
- Disambiguated slug still matches the valid slug pattern
- Pre-existing file is untouched on disk

Requires `ANTHROPIC_API_KEY`.

### M5 – Label vocabulary constraint (`TestIdeaGenerate_LabelVocabulary`)

Seeds two artifacts with labels `auth`, `backend`, `api`, `ui`, `frontend`, `usability`. Fetches the vocabulary from `GET /labels`. Generates an idea with input that might suggest novel labels (blockchain, AI). Asserts every returned label is in the known vocabulary.

Requires `ANTHROPIC_API_KEY`.

### M6 – Accept flow end-to-end (`TestIdeaGenerate_AcceptFlow`)

Full cycle: generate proposal → write via `POST /artifacts` → verify file on disk → verify index. Asserts:

- `POST /artifacts` returns HTTP 201 with `path == "lifecycle/ideas/<slug>.md"`
- Written file contains `type: idea`, `status: draft`, `lineage: <slug>`, and the generated body
- `GET /artifacts/<path>` returns HTTP 200 within 3 seconds (watcher has indexed)

Requires `ANTHROPIC_API_KEY`.

### M7 – Non-regression (`TestIdeaConverse_StillWorks`, `TestCreateArtifact_StillWorks`)

- **`TestIdeaConverse_StillWorks`** – `POST /ideas/converse` still returns HTTP 200 with `session_id` and `reply`. Requires `ANTHROPIC_API_KEY`.
- **`TestCreateArtifact_StillWorks`** – `POST /artifacts` with a hand-crafted payload creates the file without using the generate endpoint. No API key required.

### M8 – Unauthenticated access denied (`TestIdeaGenerate_Unauthenticated`)

Raw `http.Post` (no session cookie, no CSRF token) to `/ideas/generate`. Asserts status is 401 or 403 (CSRF middleware fires before auth for unauthenticated requests). No API key required.

## Running the tests

```sh
# All generate tests (including LLM-dependent):
ANTHROPIC_API_KEY=sk-... go test -tags integration ./tests/integration/ -run 'TestIdeaGenerate' -v -timeout 5m

# Non-LLM tests only (safe for CI without an API key):
go test -tags integration ./tests/integration/ -run 'TestIdeaGenerate_(TooShort|EmptyInput|MissingInput|FewWords|Unauthenticated)|TestCreateArtifact_StillWorks' -v

# Non-regression tests:
ANTHROPIC_API_KEY=sk-... go test -tags integration ./tests/integration/ -run 'TestIdea(Converse|CreateArtifact)_StillWorks' -v
```
