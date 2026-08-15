---
title: Test Suite — Suppress Empty Open Questions Section
type: test
status: blocked
lineage: requirements-analyst-suppress-empty-open-questions
parent: lifecycle/test-plans/requirements-analyst-suppress-empty-open-questions-5-test.md
assignees:
    - role: product-owner
      who: agent
---

# Test Suite — Suppress Empty Open Questions Section

Coverage for the index-outcome (Milestone 3) and frontend no-regression
(Milestone 4) portions of the test plan. The behavioural change itself is a
prompt-template edit, not deterministically unit-testable; this suite locks
in the end-to-end outcomes the requirement's acceptance criteria hinge on:
the `HasOpenQuestions` detector backstop and the `applyOpenQuestionTransition`
autoblock coupling behave correctly for the "no section", "sentinel-only",
and "genuine question" cases, and the artifact renderer needs no change to
handle a body with no `## Open Questions` heading.

## Test files

- `tests/integration/open_questions_index_outcome_test.go` — Milestone 3
  (index-outcome integration tests)
- `tests/web/artifact-blocked-questions.test.ts` — Milestone 4 (extended with
  one new test; the four pre-existing blocked/open-questions tests were
  verified to remain green, no `web/src/**` change required)

## Scenarios covered

### Milestone 3 — Index-outcome integration tests

| Test | Description |
|------|-------------|
| `TestOpenQuestionsIndexOutcome_NoSectionStaysNonBlocking` | Seeds a `draft` requirement with no `## Open Questions` heading anywhere in the body. After indexing, status remains `draft` and no `product-owner` assignee is added. |
| `TestOpenQuestionsIndexOutcome_SentinelOnlyStaysNonBlocking` | Table-driven over every currently recognised sentinel (`None`, `N/A`, `na`, `nil`, `TBD`, `no open questions`, `no questions`) as the section's sole content: a freshly authored `draft` artifact stays `draft`. A separate sub-test seeds a previously-`blocked` artifact whose section has been reduced to a sentinel and confirms it auto-unblocks to `draft` (Non-functional §2 backward compatibility). |
| `TestOpenQuestionsIndexOutcome_GenuineQuestionAutoBlocksWithAssignee` | Seeds a `draft` requirement with a real question under `## Open Questions`. After indexing, status auto-blocks to `blocked` and a `{role: product-owner, who: agent}` assignee is added — the escalation path preserved. |

All three tests exercise `applyOpenQuestionTransition` via the same startup
scan path used to index seeded fixtures (`internal/index/index.go`), not via
the `PUT`/`preview` endpoints already covered by
`TestOpenQuestionsResolve_EndToEnd` in `open_questions_resolve_e2e_test.go`.
The full `TestOpenQuestions*` suite (existing awaiting-answers, parse, and
resolve-e2e tests) was re-run alongside the new tests and remains green.

### Milestone 4 — Frontend no-regression

| Test | Description |
|------|-------------|
| `ArtifactEditorView — artifact with no Open Questions section at all › renders with no empty Open Questions heading and no console error` (new) | Mounts `ArtifactEditorView` with a `draft` artifact whose body has no `## Open Questions` heading. Asserts `.blocked-questions-banner` is absent, no leftover "Open Questions" text is rendered, and `console.error` is never called. |

The four pre-existing tests in this file (info-toast-on-override,
no-extra-toast-when-status-matches, banner-shown-when-blocked,
banner-hidden-when-not-blocked) were run unchanged and remain green,
confirming the escalation-path UI (blocked badge + banner) is unaffected.

## Out of scope for this suite (blocked — see Open Questions below)

- **Milestone 1** (`internal/config/config_test.go` — prompt-template
  assertions against `lifecycle/config.yaml`)
- **Milestone 2** (`internal/artifact/artifact_test.go` — table-driven
  sentinel-vocabulary lock-in extending the existing
  `TestHasOpenQuestions_*` tests)

## Testing approach

Milestone 3 tests use the `integration` build tag (`//go:build integration`)
and run inside the shared `testEnv` (real HTTP server, temporary git
repository, SQLite index, auth store) already used throughout
`tests/integration/`. Milestone 4 tests use Vitest + `@vue/test-utils`
against the real `ArtifactEditorView` component with its heavy children
stubbed, matching the existing file's conventions.

## Open Questions

- **Milestones 1 and 2 target files outside the test-developer role's write
  scope.** The test plan's own header states test code for this lineage
  "lives in Go (`internal/...` unit, `tests/integration/...` integration)",
  but `internal/config/config_test.go` and `internal/artifact/artifact_test.go`
  are not under `tests/` or `lifecycle/tests/` — they're Go unit tests
  co-located with the packages under `internal/`, which the test-developer
  role's `allowed_write_paths` in `lifecycle/config.yaml` does not include
  (`tests`, `web/src`, `lifecycle/tests`, `lifecycle/test-plans`,
  `lifecycle/architecture/decisions`). This is the same category of
  role/write-scope mismatch the backend plan
  (`requirements-analyst-suppress-empty-open-questions-3-be.md`) hit for
  Milestone 1's `lifecycle/config.yaml` edit, and that plan is still
  `status: blocked` on it, unresolved.
- **Even if in scope, Milestones 1 and 2 are not yet implementable.**
  `lifecycle/config.yaml`'s `requirements-analyst` prompt still lists
  `Open Questions` unconditionally under "Body sections" (no "omit when
  empty" instruction exists yet), so a Milestone-1 config-content assertion
  would currently fail. Separately, `internal/artifact/artifact.go`'s
  `isOpenQuestionSentinel` still returns `true` (real question) for a bare
  `-` bullet rather than treating it as a sentinel — the exact gap the
  backend plan's own Milestone 2 flagged as needing a fix — so a
  Milestone-2 table-driven case covering that placeholder would also
  currently fail.
- Should `internal/config` and `internal/artifact` unit-test coverage for
  this lineage be added by the backend-developer role instead (once the
  backend plan's blocker is resolved and Milestone 1/2 land), or should
  test-developer's `allowed_write_paths` be extended to include `internal`
  for cases like this? Milestones 3 and 4 above do not depend on this
  answer — they test already-implemented detector/autoblock behaviour and
  already-`done` frontend behaviour — so they are delivered now regardless.
