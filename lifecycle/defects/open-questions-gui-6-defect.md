---
created: "2026-07-14T19:34:44+10:00"
title: "Missing integration test file open_questions_parse_test.go"
type: defect
status: done
lineage: open-questions-gui
parent: lifecycle/tests/open-questions-gui-5-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Reproduction Steps

1. Read the test suite specification at [open-questions-gui-5-test.md](file:///Users/keith/Code/kaos-control/lifecycle/tests/open-questions-gui-5-test.md).
2. Locate the file `tests/open_questions_parse_test.go` or `tests/integration/open_questions_parse_test.go` in the workspace.
3. Run the Go integration tests matching this file or functions:
   ```bash
   go test -v -tags=integration ./tests/integration/... -run TestOpenQuestionsParse
   ```

## Expected Behaviour

An integration test file `tests/integration/open_questions_parse_test.go` should exist and implement the Milestone 1 parse/build preview tests:
- Question parsing from markdown with answers pre-populated in blockquotes.
- Handling of malformed, empty, or absent sections.
- Preview endpoint with `complete=false` (idempotent, preserves the rest of the document).
- Preview endpoint with `complete=true` (renames heading to `## Resolved Questions`, errors on incomplete answers).

## Actual Behaviour

The file `tests/integration/open_questions_parse_test.go` does not exist. No integration tests exist for question parsing, preview rendering, or resolution validation.

## Logs / Output

```
$ go test -v -tags=integration ./tests/integration/... -run TestOpenQuestionsParse
testing: warning: no tests to run
PASS
ok  	github.com/kaos-control/kaos-control/tests/integration	0.181s [no tests to run]
```
