---
title: "Single-Submit Idea Capture – Test Plan"
type: plan-test
status: rejected
lineage: prompt-to-idea
parent: lifecycle/requirements/prompt-to-idea-7.md
labels:
    - agent
    - workflow
    - usability
    - enhancement
---

# Single-Submit Idea Capture – Test Plan

This plan defines integration tests for the single-submit "brain dump" idea and defect capture feature described in [[prompt-to-idea]]. Tests validate the backend endpoint (`POST /ideas/generate`), the artifact write flow via `POST /artifacts`, slug collision handling, input validation, agent config resolution, and non-regression of existing endpoints. The backend implementation is specified in [[prompt-to-idea]]-be and the frontend in [[prompt-to-idea]]-fe.

Tests live in the `tests/` directory. A companion `lifecycle/tests/` artifact documents what is covered.

---

## Milestone 1 – Generate Endpoint: Happy Path (Idea)

### Description

Integration test that starts the server, authenticates, and sends a valid idea input to `POST /api/p/:project/ideas/generate`. Validates that the response contains a well-formed proposal with slug, title, labels, body, and frontmatter — all matching the schema specified in FR-1.3.

### Files to change

- **`tests/idea_generate_test.go`** (new) – Test function `TestIdeaGenerate_HappyPath`:
  1. Start the test server with a project that has existing idea artifacts (for label vocabulary).
  2. Authenticate as a valid user.
  3. `POST /api/p/test-project/ideas/generate` with `{ "input": "We should add a dark mode toggle to the settings page so users can switch between light and dark themes based on their preference" }`.
  4. Assert 200 response.
  5. Assert response JSON has non-empty `slug`, `title`, `labels`, `body`, `frontmatter`, and `target_dir`.
  6. Assert `frontmatter.type == "idea"`, `frontmatter.status == "draft"`, `frontmatter.lineage == slug`.
  7. Assert `slug` matches `^[a-z0-9][a-z0-9\-]*[a-z0-9]$`.
  8. Assert `target_dir == "lifecycle/ideas"`.
  9. Assert the body contains a level-1 heading.
  10. Assert no artifact file was written to disk (generate is preview-only).

### Acceptance criteria

- [ ] Test sends a single request and receives a complete proposal.
- [ ] All response fields are validated against the schema.
- [ ] Slug format is validated by regex.
- [ ] Frontmatter type, status, and lineage are correct.
- [ ] No file is written to disk by the generate endpoint.
- [ ] Test passes with `go test ./tests/ -run TestIdeaGenerate_HappyPath`.

---

## Milestone 2 – Generate Endpoint: Input Validation

### Description

Integration tests for input rejection (FR-1.5). The endpoint must return a 400 error with a user-facing message when input is too short or unintelligible.

### Files to change

- **`tests/idea_generate_test.go`** – Add test functions:
  - `TestIdeaGenerate_TooShort`: POST with `{ "input": "hi" }` → 400 with error code and message.
  - `TestIdeaGenerate_EmptyInput`: POST with `{ "input": "" }` → 400.
  - `TestIdeaGenerate_MissingInput`: POST with `{}` → 400.
  - `TestIdeaGenerate_FewWords`: POST with `{ "input": "fix bug" }` → 400 (under 5-word threshold).

### Acceptance criteria

- [ ] Each short/empty input returns HTTP 400.
- [ ] The response body contains an `error` field with a user-facing message (not a stack trace).
- [ ] The error message is suitable for display in the UI (FR-1.5: "asking for more detail").
- [ ] Tests pass with `go test ./tests/ -run TestIdeaGenerate_Too`.

---

## Milestone 3 – Generate Endpoint: Defect Mode

### Description

Integration test for defect brain-dump capture. Validates that passing `type: "defect"` produces a defect-shaped proposal.

### Files to change

- **`tests/idea_generate_test.go`** – Add `TestIdeaGenerate_DefectMode`:
  1. POST with `{ "input": "When I click the save button on the artifact editor, the page refreshes and all unsaved changes are lost. Expected: changes are saved without page refresh.", "type": "defect" }`.
  2. Assert 200.
  3. Assert `frontmatter.type == "defect"`.
  4. Assert `target_dir == "lifecycle/defects"`.
  5. Assert body contains defect-specific sections (Reproduction Steps, Expected Behaviour, or Actual Behaviour).
  6. Assert `labels` includes `"defect"`.

### Acceptance criteria

- [ ] Defect generation returns correct frontmatter type and target directory.
- [ ] Body contains structured defect sections.
- [ ] Labels include `"defect"`.
- [ ] Test passes with `go test ./tests/ -run TestIdeaGenerate_DefectMode`.

---

## Milestone 4 – Slug Collision Detection

### Description

Integration test for slug collision resolution (FR-2.2). Creates a fixture idea file on disk with a known slug, then generates an idea whose content would likely produce the same slug, and verifies the returned slug has a numeric suffix.

### Files to change

- **`tests/idea_generate_test.go`** – Add `TestIdeaGenerate_SlugCollision`:
  1. Write a fixture file `lifecycle/ideas/dark-mode.md` to the test project directory.
  2. Ensure the index picks it up (or re-index).
  3. POST generate with input that strongly suggests "dark-mode" as the slug.
  4. Assert the returned slug is not `"dark-mode"` — it should be `"dark-mode-2"` or similar.
  5. Clean up the fixture file.

### Acceptance criteria

- [ ] When a file with the proposed slug already exists, the returned slug has a numeric disambiguator.
- [ ] The disambiguated slug still matches the valid slug regex.
- [ ] Test passes with `go test ./tests/ -run TestIdeaGenerate_SlugCollision`.

---

## Milestone 5 – Label Vocabulary Constraint

### Description

Integration test for FR-1.4: labels must be selected from the existing label corpus, not freely invented. The test pre-populates the project with artifacts that have specific labels, then verifies that the generate endpoint only returns labels from that set.

### Files to change

- **`tests/idea_generate_test.go`** – Add `TestIdeaGenerate_LabelVocabulary`:
  1. Ensure the test project has artifacts with a known set of labels (e.g., `["agent", "workflow", "usability"]`).
  2. POST generate with input that might suggest novel labels.
  3. Assert every label in the response is a member of the known label set.

### Acceptance criteria

- [ ] No label in the response is outside the existing vocabulary.
- [ ] At least one label is returned (the LLM selects relevant ones).
- [ ] Test passes with `go test ./tests/ -run TestIdeaGenerate_LabelVocabulary`.

---

## Milestone 6 – Accept Flow: End-to-End Write

### Description

Integration test for the full accept flow: generate a proposal, then write it to disk via `POST /artifacts`, and verify the file exists with correct content. This tests the interaction between [[prompt-to-idea]]-be Milestone 3 (generate) and the existing `handleCreateArtifact` endpoint.

### Files to change

- **`tests/idea_generate_test.go`** – Add `TestIdeaGenerate_AcceptFlow`:
  1. POST `/ideas/generate` to get a proposal.
  2. POST `/artifacts` with `{ stage: "ideas", slug: proposal.slug, frontmatter: proposal.frontmatter, body: proposal.body }`.
  3. Assert 201 response.
  4. Read the file from disk at `lifecycle/ideas/<slug>.md`.
  5. Assert the file contains the expected frontmatter (title, type: idea, status: draft, lineage).
  6. Assert the file contains the generated body.
  7. GET `/artifacts/lifecycle/ideas/<slug>.md` and verify the index has the artifact.
  8. Clean up the written file.

### Acceptance criteria

- [ ] The full generate → accept → write → verify cycle works end-to-end.
- [ ] The written file has correct YAML frontmatter and markdown body.
- [ ] The artifact appears in the index after creation.
- [ ] Test passes with `go test ./tests/ -run TestIdeaGenerate_AcceptFlow`.

---

## Milestone 7 – Non-Regression: Existing Endpoints

### Description

Verify that the existing endpoints are unaffected by the new code (NFR-3, NFR-4).

### Files to change

- **`tests/idea_generate_test.go`** – Add:
  - `TestIdeaConverse_StillWorks`: POST `/ideas/converse` with a new session and a message, verify 200 response with `session_id` and `reply` — the conversational flow is intact.
  - `TestCreateArtifact_StillWorks`: POST `/artifacts` with a manual idea payload (no generate step), verify 201 and file written — direct artifact creation is unbroken.

### Acceptance criteria

- [ ] `/ideas/converse` returns a valid conversational response.
- [ ] `POST /artifacts` creates an artifact directly without using the generate endpoint.
- [ ] Neither endpoint returns errors or changed response shapes.
- [ ] Tests pass with `go test ./tests/ -run TestIdea(Converse|Create)_StillWorks`.

---

## Milestone 8 – Test Artifact Documentation

### Description

Write a companion test artifact in `lifecycle/tests/` documenting what the test suite covers.

### Files to change

- **`lifecycle/tests/prompt-to-idea-11.md`** (new) – Frontmatter:
  ```yaml
  title: "Single-Submit Idea Capture – Integration Tests"
  type: test
  status: draft
  lineage: prompt-to-idea
  parent: lifecycle/test-plans/prompt-to-idea-10-test.md
  ```
  Body summarises the scenarios covered (happy path, input validation, defect mode, slug collision, label vocabulary, accept flow, non-regression) and points to the test file `tests/idea_generate_test.go`.

### Acceptance criteria

- [ ] Artifact exists at `lifecycle/tests/prompt-to-idea-11.md` with correct frontmatter.
- [ ] Body lists all test scenarios and references the test file.
- [ ] Lineage index (11) is the next monotonic value after the test plan (10).
