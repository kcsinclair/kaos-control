---
title: "Single-Submit Idea & Defect Capture – Test Plan"
type: plan-test
status: draft
lineage: prompt-to-idea
parent: lifecycle/requirements/prompt-to-idea-7.md
---

# Single-Submit Idea & Defect Capture – Test Plan

This plan defines integration tests for the single-submit "brain dump" idea and defect capture feature described in [[prompt-to-idea]]. Tests validate the backend endpoint (`POST /ideas/generate`), the artifact write flow via `POST /artifacts`, slug collision handling, input validation, label vocabulary constraints, and non-regression of existing endpoints. The backend implementation is specified in [[prompt-to-idea]]-be and the frontend in [[prompt-to-idea]]-fe.

Tests live in the `tests/integration/` directory using the existing `testEnv` helper infrastructure (`helpers_test.go`) — `newTestEnv()` for server setup, `doRequest()` for authenticated HTTP calls, `readJSON()` for response parsing, and `requireStatus()` for status assertions. Tests requiring a live LLM use the `skipIfNoAPIKey()` guard.

A companion `lifecycle/tests/` artifact documents what is covered.

---

## Milestone 1 – Generate Endpoint: Happy Path (Idea)

### Description

Integration test that starts the server, authenticates, and sends valid idea input to `POST /api/p/:project/ideas/generate`. Validates that the response contains a well-formed proposal with slug, title, labels, body, frontmatter, and target_dir — all matching the schema specified in FR-1.3. Confirms no file is written to disk (generate is preview-only).

### Files to change

- **`tests/integration/idea_generate_test.go`** (new) — Test function `TestIdeaGenerate_HappyPath`:
  1. `skipIfNoAPIKey(t)` — skip if no LLM available.
  2. `newTestEnv(t)` — starts server with a project that has existing idea artifacts (for label vocabulary).
  3. `env.login(t)` — authenticate.
  4. `env.doRequest("POST", "/api/p/test-project/ideas/generate", map{"input": "We should add a dark mode toggle to the settings page so users can switch between light and dark themes based on their preference"})`.
  5. `requireStatus(t, resp, 200)`.
  6. `readJSON(t, resp)` — parse response.
  7. Assert non-empty `slug`, `title`, `labels`, `body`, `frontmatter`, `target_dir`.
  8. Assert `frontmatter.type == "idea"`, `frontmatter.status == "draft"`, `frontmatter.lineage == slug`.
  9. Assert `slug` matches `^[a-z0-9][a-z0-9\-]*[a-z0-9]$`.
  10. Assert `target_dir == "lifecycle/ideas"`.
  11. Assert body contains a level-1 heading (`# `).
  12. Assert no file was written to disk at `lifecycle/ideas/<slug>.md`.

### Acceptance criteria

- [ ] Test sends a single request and receives a complete proposal.
- [ ] All response fields are validated against the schema.
- [ ] Slug format is validated by regex.
- [ ] Frontmatter type, status, and lineage are correct.
- [ ] No file is written to disk by the generate endpoint.
- [ ] Test passes with `go test ./tests/integration/ -tags=integration -run TestIdeaGenerate_HappyPath`.

---

## Milestone 2 – Generate Endpoint: Input Validation

### Description

Integration tests for input rejection (FR-1.5). The endpoint must return 400 with a user-facing message when input is too short, empty, or missing. These tests do not require an LLM — the validation happens before the LLM call.

### Files to change

- **`tests/integration/idea_generate_test.go`** — Add test functions:
  - `TestIdeaGenerate_TooShort`: POST with `{ "input": "hi" }` → `requireStatus(t, resp, 400)`. Assert response body contains `"error"` field with a user-facing message.
  - `TestIdeaGenerate_EmptyInput`: POST with `{ "input": "" }` → 400.
  - `TestIdeaGenerate_MissingInput`: POST with `{}` → 400.
  - `TestIdeaGenerate_FewWords`: POST with `{ "input": "fix bug" }` → 400.

### Acceptance criteria

- [ ] Each short/empty/missing input returns HTTP 400.
- [ ] The response body contains an `error` field with a user-facing message (not a stack trace).
- [ ] The error message is suitable for UI display (FR-1.5).
- [ ] Tests do not require `ANTHROPIC_API_KEY` (no LLM call is made for validation failures).
- [ ] Tests pass with `go test ./tests/integration/ -tags=integration -run TestIdeaGenerate_Too`.

---

## Milestone 3 – Generate Endpoint: Defect Mode

### Description

Integration test for defect brain-dump capture. Validates that passing `type: "defect"` produces a defect-shaped proposal with correct frontmatter, target directory, and structured body sections.

### Files to change

- **`tests/integration/idea_generate_test.go`** — Add `TestIdeaGenerate_DefectMode`:
  1. `skipIfNoAPIKey(t)`.
  2. POST with `{ "input": "When I click the save button on the artifact editor the page refreshes and all unsaved changes are lost. Expected: changes are saved without page refresh.", "type": "defect" }`.
  3. `requireStatus(t, resp, 200)`.
  4. Assert `frontmatter.type == "defect"`.
  5. Assert `target_dir == "lifecycle/defects"`.
  6. Assert body contains at least one of: "Reproduction Steps", "Expected Behaviour", "Actual Behaviour".
  7. Assert `labels` includes `"defect"`.

### Acceptance criteria

- [ ] Defect generation returns correct frontmatter type and target directory.
- [ ] Body contains structured defect sections.
- [ ] Labels include `"defect"`.
- [ ] Test passes with `go test ./tests/integration/ -tags=integration -run TestIdeaGenerate_DefectMode`.

---

## Milestone 4 – Slug Collision Detection

### Description

Integration test for slug collision resolution (FR-2.2). Creates a fixture idea file on disk with a known slug, then generates an idea whose content would likely produce the same slug, and verifies the returned slug has a numeric suffix.

### Files to change

- **`tests/integration/idea_generate_test.go`** — Add `TestIdeaGenerate_SlugCollision`:
  1. `skipIfNoAPIKey(t)`.
  2. Write a fixture file `lifecycle/ideas/dark-mode.md` with valid frontmatter to the test project directory.
  3. Wait briefly for the index to pick it up (or trigger a manual re-index if the test helper supports it).
  4. POST generate with input that strongly suggests "dark-mode" as the slug (e.g., "We need a dark mode feature for the settings page to toggle between light and dark themes").
  5. Assert the returned slug is not `"dark-mode"` — it should be `"dark-mode-2"` or similar.
  6. Assert the disambiguated slug still matches the valid slug regex.
  7. Fixture file cleanup is handled by `testEnv` temp directory teardown.

### Acceptance criteria

- [ ] When a file with the proposed slug already exists, the returned slug has a numeric disambiguator.
- [ ] The disambiguated slug still matches `^[a-z0-9][a-z0-9\-]*[a-z0-9]$`.
- [ ] Test passes with `go test ./tests/integration/ -tags=integration -run TestIdeaGenerate_SlugCollision`.

---

## Milestone 5 – Label Vocabulary Constraint

### Description

Integration test for FR-1.4: labels must be selected from the existing label corpus, not freely invented. The test project is pre-populated with artifacts that have specific labels, then verifies that the generate endpoint only returns labels from that set.

### Files to change

- **`tests/integration/idea_generate_test.go`** — Add `TestIdeaGenerate_LabelVocabulary`:
  1. `skipIfNoAPIKey(t)`.
  2. Determine the known label set from the test project's seed artifacts (the `newTestEnv` seeds should produce a known label corpus).
  3. POST generate with input that might suggest novel labels (e.g., mentioning "blockchain" or "AI" — terms unlikely to be in the existing vocabulary).
  4. Assert every label in the response is a member of the known label set.
  5. Assert at least one label is returned (the LLM selects relevant ones).

### Acceptance criteria

- [ ] No label in the response is outside the existing vocabulary.
- [ ] At least one label is returned.
- [ ] Test passes with `go test ./tests/integration/ -tags=integration -run TestIdeaGenerate_LabelVocabulary`.

---

## Milestone 6 – Accept Flow: End-to-End Write

### Description

Integration test for the full accept flow: generate a proposal, then write it to disk via the existing `POST /api/p/:project/artifacts` endpoint, and verify the file exists with correct content. This tests the interaction between [[prompt-to-idea]]-be (generate) and the existing artifact creation handler.

### Files to change

- **`tests/integration/idea_generate_test.go`** — Add `TestIdeaGenerate_AcceptFlow`:
  1. `skipIfNoAPIKey(t)`.
  2. POST `/ideas/generate` to get a proposal.
  3. POST `/artifacts` with `{ "stage": "ideas", "slug": proposal.slug, "frontmatter": proposal.frontmatter, "body": proposal.body }`.
  4. `requireStatus(t, resp, 201)`.
  5. Read the file from disk at `lifecycle/ideas/<slug>.md`.
  6. Assert the file contains the expected frontmatter fields (`title`, `type: idea`, `status: draft`, `lineage`).
  7. Assert the file contains the generated body.
  8. GET `/api/p/test-project/artifacts/lifecycle/ideas/<slug>.md` and verify the index has the artifact.

### Acceptance criteria

- [ ] The full generate → accept → write → verify cycle works end-to-end.
- [ ] The written file has correct YAML frontmatter and markdown body.
- [ ] The artifact appears in the index after creation.
- [ ] Test passes with `go test ./tests/integration/ -tags=integration -run TestIdeaGenerate_AcceptFlow`.

---

## Milestone 7 – Non-Regression: Existing Endpoints

### Description

Verify that the existing endpoints are unaffected by the new code (NFR-3, NFR-4). These tests confirm that adding the generate endpoint does not break the conversational flow or direct artifact creation.

### Files to change

- **`tests/integration/idea_generate_test.go`** — Add:
  - `TestIdeaConverse_StillWorks`: `skipIfNoAPIKey(t)`. POST `/ideas/converse` with `{ "session_id": null, "message": "I have an idea for improving search" }`. `requireStatus(t, resp, 200)`. Assert response contains `session_id` and `reply` fields — the conversational flow is intact.
  - `TestCreateArtifact_StillWorks`: POST `/artifacts` with a manually constructed idea payload (no generate step): `{ "stage": "ideas", "slug": "manual-test-idea", "frontmatter": { "title": "Manual Test", "type": "idea", "status": "draft", "lineage": "manual-test-idea" }, "body": "# Manual Test\n\nTest body." }`. `requireStatus(t, resp, 201)`. Assert file exists on disk.

### Acceptance criteria

- [ ] `/ideas/converse` returns a valid conversational response with session_id and reply.
- [ ] `POST /artifacts` creates an artifact directly without using the generate endpoint.
- [ ] Neither endpoint returns errors or changed response shapes.
- [ ] Tests pass with `go test ./tests/integration/ -tags=integration -run "TestIdea(Converse|CreateArtifact)_StillWorks"`.

---

## Milestone 8 – Unauthenticated Access Denied

### Description

Verify that the generate endpoint requires authentication, consistent with the other idea endpoints.

### Files to change

- **`tests/integration/idea_generate_test.go`** — Add `TestIdeaGenerate_Unauthenticated`:
  1. `newTestEnv(t)` — start server but do NOT call `env.login()`.
  2. Make a raw `http.Post` (no session cookies) to `/api/p/test-project/ideas/generate` with valid input.
  3. Assert response status is 401.

### Acceptance criteria

- [ ] Unauthenticated POST to `/ideas/generate` returns 401.
- [ ] Test does not require `ANTHROPIC_API_KEY`.
- [ ] Test passes with `go test ./tests/integration/ -tags=integration -run TestIdeaGenerate_Unauthenticated`.

---

## Milestone 9 – Test Artifact Documentation

### Description

Write a companion test artifact in `lifecycle/tests/` documenting what the integration test suite covers, per the test-developer agent convention.

### Files to change

- **`lifecycle/tests/prompt-to-idea-14.md`** (new) — Frontmatter:
  ```yaml
  title: "Single-Submit Idea & Defect Capture – Integration Tests"
  type: test
  status: draft
  lineage: prompt-to-idea
  parent: lifecycle/test-plans/prompt-to-idea-13-test.md
  ```
  Body summarises the scenarios covered (happy path idea, input validation, defect mode, slug collision, label vocabulary, accept flow, non-regression, unauthenticated access) and references the test file `tests/integration/idea_generate_test.go`.

### Acceptance criteria

- [ ] Artifact exists at `lifecycle/tests/prompt-to-idea-14.md` with correct frontmatter.
- [ ] Body lists all test scenarios and references the test file.
- [ ] Lineage index (14) is the next monotonic value after the test plan (13).
