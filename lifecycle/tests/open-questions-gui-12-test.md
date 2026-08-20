---
created: "2026-07-14T19:34:44+10:00"
title: "Test Suite: Open-Questions End-to-End Resolve → Unblock"
type: test
status: draft
lineage: open-questions-gui
parent: lifecycle/defects/open-questions-gui-7-defect.md
---

# Test Suite Coverage

Fixes [[open-questions-gui]] defect `open-questions-gui-7-defect`: the missing
integration test file for Milestone 2 of
[[open-questions-gui]]-5-test (`tests/open_questions_resolve_e2e_test.go`).

## Test File

- `tests/open_questions_resolve_e2e_test.go`

## Scenarios Covered

A single end-to-end test, `TestOpenQuestionsResolve_EndToEnd`, drives one
artifact through the full resolve → unblock lifecycle via the real HTTP API,
using `t.Run` subtests so each stage builds on the on-disk state left by the
previous one:

1. **`CreateWithQuestionsBlocksAndAssignsProductOwner`** — PUTting a body with
   a non-empty `## Open Questions` section auto-blocks the artifact
   (`status: blocked`) and injects a `{role: product-owner, who: agent}`
   assignee.
2. **`PartialResolveStaysBlocked`** — `POST .../open-questions/preview` with
   `complete=false` answering one of two questions, then PUTting the returned
   body: the artifact stays `blocked` and the on-disk body still carries the
   `## Open Questions` heading (not renamed).
3. **`CompleteResolveRenamesHeadingAndUnblocks`** — `POST .../open-questions/preview`
   with `complete=true` answering the remaining question, then PUTting the
   returned body: the heading is renamed to `## Resolved Questions` and the
   artifact auto-transitions to `draft`.
4. **`PutPayloadCarriesNoClientAuthoredStatusMutation`** — re-marshals the
   completing PUT payload and asserts it carries exactly two top-level keys
   (`frontmatter`, `body`), that the submitted `frontmatter.status` is the
   artifact's pre-resolve status (`blocked`, unchanged by the client), and
   that the server-side status is nonetheless `draft` — proving the
   draft transition is computed by `applyOpenQuestionTransition`
   (`internal/index/autoblock.go`), never authored by the client.
5. **`ResavingCompletedAnswersIsIdempotent`** — PUTs the identical completed
   body a second time and asserts both the status (`draft`) and the on-disk
   body are unchanged.
6. **`ReopeningReblocksArtifact`** — renames the heading back from
   `## Resolved Questions` to `## Open Questions` (answers, and therefore
   section content, remain non-empty) and PUTs: the artifact re-blocks
   (`status: blocked`) with the product-owner assignee restored.

Uses the existing `testEnv` integration harness
(`tests/integration/helpers_test.go`): auto-login as admin, `makeArtifact`
seeding, and the plain `GET /artifacts/*path` endpoint (not
`/open-questions`, whose `heading` field is a hardcoded constant) to read
back the on-disk body for heading assertions.

Run with:

```sh
go test -v -tags=integration ./tests/integration/... -run TestOpenQuestionsResolve
```
