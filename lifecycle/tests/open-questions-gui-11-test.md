---
title: "Test Suite: Open-Questions Parse & Build-Preview"
type: test
status: draft
lineage: open-questions-gui
parent: lifecycle/defects/open-questions-gui-6-defect.md
---

# Test Suite Coverage

Fixes [[open-questions-gui]] defect `open-questions-gui-6-defect`: the missing
integration test file for Milestone 1 of
[[open-questions-gui]]-5-test (`tests/open_questions_parse_test.go`).

## Test File

- `tests/integration/open_questions_parse_test.go`

## Scenarios Covered

1. **`TestOpenQuestionsParse_PrepopulatedAnswersPreserved`** — seeds an
   artifact with two top-level list items under `## Open Questions`, the
   first already answered with a trailing blockquote; asserts
   `GET .../artifacts/*path/open-questions` returns both questions in order,
   preserving the existing answer on the first and leaving the second's
   answer empty.
2. **`TestOpenQuestionsParse_MalformedOrEmptySectionReturnsEmptyList`** —
   asserts the parse endpoint returns an empty (non-null) `questions` array,
   not an error, both for a `## Open Questions` heading with no top-level
   list items (malformed/empty) and for an artifact body with no such
   heading at all (NFR6 graceful parsing).
3. **`TestOpenQuestionsParse_PreviewPartialIsIdempotentAndPreservesRest`** —
   calls `POST .../open-questions/preview` with `complete=false`, asserting
   the returned body keeps prose before and the `## ` section after Open
   Questions byte-for-byte intact, writes the supplied answer as a
   blockquote, leaves the heading as `## Open Questions`, and produces
   byte-identical output when the identical request is repeated.
4. **`TestOpenQuestionsParse_PreviewCompleteRenamesHeadingAndErrorsOnIncomplete`**
   — calls preview with `complete=true`: asserts a request that leaves one
   question unanswered returns HTTP 422 and leaves the on-disk artifact
   untouched (confirmed via a follow-up GET), then asserts a request
   answering every question succeeds, renames the heading to
   `## Resolved Questions`, and writes every answer as a blockquote.

All scenarios use the existing `testEnv` integration harness
(`tests/integration/helpers_test.go`): auto-login as admin, `makeArtifact`
seeding, and the `doRequest`/`readJSON`/`requireStatus` pattern already used
by `tests/integration/open_questions_config_test.go` and
`tests/integration/open_questions_awaiting_test.go`.

Run with:

```sh
go test -v -tags=integration ./tests/integration/... -run TestOpenQuestionsParse
```
