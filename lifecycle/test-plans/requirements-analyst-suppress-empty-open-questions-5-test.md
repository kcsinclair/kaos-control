---
title: 'Test Plan: Suppress Empty Open Questions Section'
type: plan-test
status: draft
lineage: requirements-analyst-suppress-empty-open-questions
parent: lifecycle/requirements/requirements-analyst-suppress-empty-open-questions-2.md
---

## Overview

Coverage for the backend/config change in
[[requirements-analyst-suppress-empty-open-questions]] backend plan `-3-be` and the
no-regression checks in frontend plan `-4-fe`. Because the behavioural change is in
a prompt template (not deterministically unit-testable), the strategy is:

1. **Assert the config artifact** encodes the new instruction and no longer lists
   `Open Questions` unconditionally — this is the closest deterministic proxy for
   "the agent is told to omit the section".
2. **Lock the detector backstop** with the existing unit tests so a stray
   placeholder still cannot auto-block.
3. **Assert the end-to-end index outcome**: no-section → non-blocking; sentinel-only
   → non-blocking; real-question → blocked with assignee.

Test code lives in Go (`internal/...` unit, `tests/integration/...` integration) and
web (`tests/web/...`); a companion `test` artifact is written to `lifecycle/tests/`.

## Milestone 1 — Config prompt-template assertions

**Description.** Add/extend a unit test that loads `lifecycle/config.yaml` (or the
packaged default) and asserts the analyst prompt templates encode the new
conditional-emission guidance.

**Files to change.**
- `internal/config/config_test.go` (or a new focused test file in `internal/config/`).

**Acceptance criteria.**
- [ ] Test asserts the `requirements-analyst` `analyst` template does **not** contain
      an unconditional `- Open Questions` body-section bullet.
- [ ] Test asserts both `requirements-analyst` and `planning-analyst` templates
      contain the "omit … when there are none" instruction and the forbidden-
      placeholder list (Functional §1, §2, §4, §6).
- [ ] Test asserts the "If you get stuck" escalation block is still present in each
      template that had it (Functional §3).
- [ ] `config.yaml` loads without error (guards against a malformed-YAML edit).

## Milestone 2 — Detector backstop unit tests (retain green)

**Description.** Keep the existing sentinel unit tests green and extend them to cover
every placeholder string the Milestone-1 prompt forbids, ensuring prompt and
detector share one placeholder vocabulary.

**Files to change.**
- `internal/artifact/artifact_test.go`

**Acceptance criteria.**
- [ ] `TestHasOpenQuestions_SentinelIsNotBlocking`,
      `TestHasOpenQuestions_RealQuestionAlongsideSentinelBlocks`,
      `TestHasOpenQuestions_QuestionContainingSentinelWordBlocks` remain green
      (Functional §5; requirement Acceptance Criteria bullet 5).
- [ ] A table-driven case asserts `HasOpenQuestions` returns `false` for each
      forbidden placeholder (`none`, `n/a`, `nil`, `tbd`, `no open questions`,
      `no questions`, bare `-` bullet) as the section's sole content.
- [ ] A case asserts `HasOpenQuestions` returns `true` for a section containing a
      genuine question.

## Milestone 3 — Index-outcome integration tests

**Description.** Assert the end-to-end indexing behaviour that the requirement's
acceptance criteria hinge on, exercising `applyOpenQuestionTransition` via the index.

**Files to change.**
- `tests/integration/open_questions_awaiting_test.go` (extend) or a new
  `tests/integration/*_test.go`.

**Acceptance criteria.**
- [ ] Indexing a requirement body with **no** `## Open Questions` heading leaves
      status `draft` and does not auto-block (requirement Acceptance bullet 2).
- [ ] Indexing a requirement whose only Open Questions content is a sentinel does
      not auto-block (Non-functional §2 backward compatibility).
- [ ] Indexing a requirement with a genuine question auto-blocks and adds a
      `role: product-owner, who: agent` assignee (escalation path preserved).
- [ ] Existing autoblock/awaiting-answers integration tests remain green
      (Non-functional §1; requirement Acceptance bullet 7).

## Milestone 4 — Frontend no-regression tests + test artifact

**Description.** Confirm the web tests covering blocked/open-questions rendering stay
green (per [[requirements-analyst-suppress-empty-open-questions]] frontend plan
`-4-fe`), and write the companion `test` artifact documenting the suite.

**Files to change.**
- `tests/web/artifact-blocked-questions.test.ts` (verify; extend only if a
  no-section rendering case is missing).
- `lifecycle/tests/requirements-analyst-suppress-empty-open-questions-<n>.md`
  (companion `test` artifact; next monotonic lineage index at authoring time).

**Acceptance criteria.**
- [ ] Web test asserts an artifact with no `## Open Questions` section renders with no
      empty placeholder heading and no console error.
- [ ] Existing blocked/open-questions web tests remain green.
- [ ] A `test` artifact is written to `lifecycle/tests/` (frontmatter `type: test`,
      `status: draft`, same `lineage`, `parent` pointing to this test plan)
      summarising the scenarios above and pointing to the specific test files.
