---
title: "Tests: Auto-block on open questions"
type: plan-test
status: done
lineage: artefact-status-blocked-on-questions
parent: lifecycle/defects/artefact-status-blocked-on-questions.md
---

# Tests: Auto-block on open questions

Integration tests verifying that the backend auto-transitions artifacts to `blocked` when open questions are present, and that the frontend correctly reflects the change. Covers the backend API behaviour from [[artefact-status-blocked-on-questions]] backend plan and the frontend feedback from the frontend plan.

## Milestone 1 — Backend API: auto-block on save with open questions

### Description

Write integration tests against the `PUT /api/p/:project/artifacts/*` endpoint that verify the auto-block logic triggers correctly.

### Files to change

- `tests/api/artifact_blocked_questions_test.go` (new file) — HTTP-level integration tests using the running server.

### Acceptance criteria

- **Test: save with open questions triggers blocked status** — PUT an artifact body containing `## Open Questions\n\n- Why is X?` with `status: draft`. Assert the response has `status: "blocked"` and `assignees` includes `role: product-owner, who: agent`.
- **Test: save without open questions preserves submitted status** — PUT an artifact body without an `## Open Questions` section with `status: draft`. Assert the response has `status: "draft"`.
- **Test: empty open questions section does not trigger block** — PUT an artifact body containing `## Open Questions\n\n` (heading but no content). Assert the response has `status: "draft"` (not blocked).
- **Test: existing blocked status is not auto-unblocked** — PUT an artifact that is currently `blocked` with a body that has no open questions but `status: blocked`. Assert the response preserves `status: "blocked"` (manual unblock required).
- **Test: product-owner assignee is not duplicated** — PUT an artifact that already has a `product-owner` assignee with a body containing open questions. Assert only one `product-owner` assignee in the response.
- All tests pass with `go test ./tests/api/... -run TestBlockedQuestions`.

## Milestone 2 — Backend unit: HasOpenQuestions parser

### Description

Unit tests for the `HasOpenQuestions` helper function covering edge cases in markdown parsing.

### Files to change

- `tests/unit/has_open_questions_test.go` (new file) — or alongside the function in `internal/artifact/artifact_test.go` if test-developer scope permits.

### Acceptance criteria

- **Test: heading with bullet list** — `## Open Questions\n\n- Q1\n- Q2` → `true`.
- **Test: heading with paragraph** — `## Open Questions\n\nSome question here.` → `true`.
- **Test: heading with only whitespace** — `## Open Questions\n\n   \n\n` → `false`.
- **Test: no heading** — body without `## Open Questions` → `false`.
- **Test: heading at wrong level** — `### Open Questions\n\n- Q1` → `false` (must be `##`).
- **Test: heading mid-document** — content before and after the heading, questions present → `true`.
- **Test: heading followed by another heading immediately** — `## Open Questions\n## Next Section` → `false`.
- All tests pass.

## Milestone 3 — Frontend: save response reflects blocked override

### Description

Web integration tests (Vitest + testing-library or Playwright) verifying the frontend handles the backend's status override correctly.

### Files to change

- `tests/web/artifact-blocked-questions.test.ts` (new file) — tests using MSW or direct API mocking.

### Acceptance criteria

- **Test: toast shown when save returns different status** — Mock the PUT response to return `status: "blocked"` when `status: "draft"` was submitted. Assert an info toast with text matching "blocked" and "open questions" appears.
- **Test: no extra toast when status matches** — Mock the PUT response to return `status: "draft"` matching the submission. Assert no additional info toast beyond the standard success toast.
- **Test: blocked banner shown in detail view** — Render `ArtifactDetailView` with an artifact whose status is `blocked` and body contains `## Open Questions\n\n- Q1`. Assert the warning banner is present.
- **Test: blocked banner hidden for non-blocked** — Render with `status: "draft"` and same body. Assert no banner.
- All tests pass with the project's web test runner.
