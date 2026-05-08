---
title: "Tests: Auto-block on open questions"
type: test
status: approved
lineage: artefact-status-blocked-on-questions
parent: lifecycle/test-plans/artefact-status-blocked-on-questions-4-test.md
---

# Tests: Auto-block on open questions

Integration and unit tests verifying the backend auto-block-on-open-questions
feature and the corresponding frontend feedback.

## Test files

- `tests/integration/artifact_blocked_questions_test.go` — HTTP-level integration tests against the PUT endpoint
- `internal/artifact/artifact_test.go` — Unit tests for the `HasOpenQuestions` parser helper
- `tests/web/artifact-blocked-questions.test.ts` — Frontend Vitest tests for the editor save toast and blocked-questions banner

## Milestone 1 — Backend API: auto-block on save (integration tests)

File: `tests/integration/artifact_blocked_questions_test.go`

Run with:

```
go test ./tests/integration/... -tags=integration -run TestBlockedQuestions
```

Scenarios covered:

1. **TestBlockedQuestions_WithOpenQuestionsTriggersBlocked** — PUT a `draft` artifact
   with body `## Open Questions\n\n- Why is X?\n`. Asserts the response has
   `status: "blocked"` and the `assignees` array contains `{role: product-owner, who: agent}`.

2. **TestBlockedQuestions_WithoutOpenQuestionsPreservesStatus** — PUT a `draft`
   artifact with a body that has no `## Open Questions` heading. Asserts the
   response preserves `status: "draft"`.

3. **TestBlockedQuestions_EmptySectionDoesNotBlock** — PUT a `draft` artifact with
   `## Open Questions\n\n   \n\n` (heading present, section body is only whitespace).
   Asserts the response has `status: "draft"` (not blocked).

4. **TestBlockedQuestions_ExistingBlockedStatusPreserved** — PUT a `blocked`
   artifact with no open questions and `status: "blocked"` in the submitted
   frontmatter. Asserts the backend does not auto-unblock: response preserves
   `status: "blocked"`.

5. **TestBlockedQuestions_ProductOwnerAssigneeNotDuplicated** — Artifact is seeded
   with a `product-owner/agent` assignee. PUT with open questions body and the
   same assignee already in the submitted frontmatter. Asserts exactly one
   `product-owner/agent` entry in the response (no duplication).

## Milestone 2 — Backend unit: HasOpenQuestions parser

File: `internal/artifact/artifact_test.go`

Run with:

```
go test ./internal/artifact/... -run TestHasOpenQuestions
```

Scenarios covered:

1. **TestHasOpenQuestions_HeadingWithBulletList** — `## Open Questions` + `- Q1\n- Q2` → `true`
2. **TestHasOpenQuestions_HeadingWithParagraph** — `## Open Questions` + prose paragraph → `true`
3. **TestHasOpenQuestions_HeadingWithOnlyWhitespace** — heading + only blank/whitespace lines → `false`
4. **TestHasOpenQuestions_NoHeading** — body without any `## Open Questions` heading → `false`
5. **TestHasOpenQuestions_HeadingAtWrongLevel** — `### Open Questions` (H3) → `false`
6. **TestHasOpenQuestions_HeadingMidDocument** — heading appears after other content, has questions → `true`
7. **TestHasOpenQuestions_HeadingFollowedImmediatelyByNextHeading** — `## Open Questions\n## Next Section` → `false`

## Milestone 3 — Frontend: save response and blocked banner (Vitest)

File: `tests/web/artifact-blocked-questions.test.ts`

Run with (from `tests/web/`):

```
pnpm vitest run artifact-blocked-questions
```

Scenarios covered (all TDD — tests fail until the frontend feature is implemented):

1. **toast shown when save returns different status** — Mocks `getArtifact` to return
   `status: "blocked"` after save. Asserts an `info`-level toast is present whose
   message contains both "blocked" and "open questions".

2. **no extra toast when status matches** — Mocks `getArtifact` to return `status: "draft"`
   matching the submitted status. Asserts no info toast mentioning "blocked" / "open
   questions" is added.

3. **blocked banner shown in detail view** — Mounts `ArtifactEditorView` (read mode) with
   `status: "blocked"` and a body containing `## Open Questions`. Asserts the
   `.blocked-questions-banner` element is present.

4. **blocked banner hidden for non-blocked** — Same body, `status: "draft"`. Asserts the
   `.blocked-questions-banner` element is absent.

> **Note — ArtifactDetailView:** The frontend plan specifies the banner should live in
> `web/src/views/project/ArtifactDetailView.vue`, which does not yet exist. Tests 3 and 4
> currently target `ArtifactEditorView` in read mode. Once `ArtifactDetailView.vue` is
> created, migrate these tests to import and mount that component instead.
