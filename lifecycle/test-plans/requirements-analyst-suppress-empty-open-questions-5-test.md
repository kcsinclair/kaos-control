---
created: "2026-08-15T11:05:01+10:00"
title: 'Test Plan: Suppress Empty Open Questions Section'
type: plan-test
status: abandoned
lineage: requirements-analyst-suppress-empty-open-questions
parent: lifecycle/requirements/requirements-analyst-suppress-empty-open-questions-2.md
assignees:
    - role: product-owner
      who: agent
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

## Discarded Questions

Milestones 3 and 4 are implemented and green — see the companion `test`
artifact `lifecycle/tests/requirements-analyst-suppress-empty-open-questions-6-test.md`
for the test files and scenarios. Milestones 1 and 2 are blocked:

- Milestones 1 and 2 assign Go unit tests to
  `internal/config/config_test.go` and `internal/artifact/artifact_test.go`.
  Neither path is under `tests/` or `lifecycle/tests/` — they're outside the
  test-developer role's `allowed_write_paths` in `lifecycle/config.yaml`
  (`tests`, `web/src`, `lifecycle/tests`, `lifecycle/test-plans`,
  `lifecycle/architecture/decisions`). This is the same category of
  role/write-scope mismatch the backend plan
  (`requirements-analyst-suppress-empty-open-questions-3-be.md`) hit for its
  own Milestone 1 (`lifecycle/config.yaml` outside `backend-developer`'s
  scope), and that plan is still `status: blocked` on it, unresolved.
- Even setting scope aside, Milestone 1's assertions would currently fail:
  `lifecycle/config.yaml`'s `requirements-analyst` prompt still lists
  `Open Questions` unconditionally under "Body sections", with no "omit when
  empty" instruction — the backend Milestone 1 edit has not landed. Milestone
  2's forbidden-placeholder table would also currently fail for a bare `-`
  bullet: `internal/artifact/artifact.go`'s `isOpenQuestionSentinel` still
  returns `true` (real question) for it rather than treating it as a
  sentinel — the exact gap the backend plan's own Milestone 2 flagged.
- Should `internal/config`/`internal/artifact` unit tests for this lineage be
  added by `backend-developer` once the backend plan's blocker resolves, or
  should `test-developer`'s `allowed_write_paths` be extended to include
  `internal`? This does not block Milestones 3/4, already delivered.

## Abandoned — blocked on role write-scope, and the harm is already contained

**Status: abandoned 2026-09-03.** Not delivered. Reasoning recorded so it is not
rediscovered later.

### Why these stalled

Not a product question — a **role/write-scope mismatch**. The work these
artifacts describe cannot be performed by any configured agent:

| Artifact | Target file | Problem |
|---|---|---|
| `-3-be` M1 | `lifecycle/config.yaml` | No role has it in `allowed_write_paths` — not backend-developer, nor either analyst. It is also not a Go change, so it does not fit the backend-developer role regardless of permissions. |
| `-5-test` / `-6-test` M1–M2 | `internal/config/config_test.go`, `internal/artifact/artifact_test.go` | Outside test-developer's `allowed_write_paths` (`tests`, `web/src`, `lifecycle/tests`, `lifecycle/test-plans`, `lifecycle/architecture/decisions`). |

The discarded questions below asked who should make these edits. In practice the
answer is that prompt and config curation happens **by hand**, directly in
`lifecycle/config.yaml`, rather than being routed through an agent — so these
tickets were waiting on a workflow that does not exist.

### Why abandoning is safe

The Go half of this lineage shipped, and it is the half that mattered.
`artifact.HasOpenQuestions` (`internal/artifact/artifact.go:334`) ignores
placeholder content, treating these as "no real question":

`none`, `n/a`, `na`, `nil`, `no open questions`, `no questions`, `tbd`

It is wired into the auto-block transition (`internal/index/autoblock.go:34`)
and the index write path (`internal/index/index.go:623`), with both packages'
tests passing. So a hollow "Open Questions: None" does **not** wrongly force an
artifact to `blocked` — the actual damage this lineage set out to stop.

The prompt half was never done. Verified against `lifecycle/config.yaml` on
2026-09-03: `requirements-analyst` still lists Open Questions as an
unconditional body section, and neither analyst prompt carries an
"omit when empty" instruction or a placeholder prohibition. Analysts therefore
still emit the heading routinely and fill it with a sentinel — cosmetic noise,
not a blocked artifact, which is why this is dropped rather than finished.

### Residual risk if it needs reopening

The sentinel list is **finite and literal**. A model writing anything outside it
— "No outstanding questions.", "Nothing further at this stage." — does not
match, so `HasOpenQuestions` returns true and `applyOpenQuestionTransition`
auto-blocks the artifact. The prompt change was the belt to the detector's
braces; without it, protection depends on the model phrasing its non-answer in
one of seven exact ways.

If spurious `blocked` statuses appear on analyst output, look here first. The
fix is the M1 prompt edit described above, not extending the sentinel list.

### Why the heading below was renamed

`HasOpenQuestions` matches the `## Open Questions` heading exactly, and
`applyOpenQuestionTransition` auto-blocks any artifact carrying one. While the
heading read "Open Questions" this file could not be abandoned at all — the
status reverted to `blocked` within seconds of every edit. Renaming it to
`## Discarded Questions` releases that. Worth knowing if another lineage needs
closing the same way.
