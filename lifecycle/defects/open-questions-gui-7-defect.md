---
title: "Missing integration test file open_questions_resolve_e2e_test.go"
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
2. Locate the file `tests/open_questions_resolve_e2e_test.go` or `tests/integration/open_questions_resolve_e2e_test.go` in the workspace.
3. Run the Go integration tests matching this file or functions:
   ```bash
   go test -v -tags=integration ./tests/integration/... -run TestOpenQuestionsResolve
   ```

## Expected Behaviour

An integration test file `tests/integration/open_questions_resolve_e2e_test.go` should exist and test the end-to-end resolution flow:
- Creating/PUTting an artifact whose body has a non-empty `## Open Questions` section results in `status: blocked` with a `product-owner` assignee.
- A partial resolve (preview `complete=false` -> PUT body) leaves the artifact `blocked` and the heading `## Open Questions`.
- A complete resolve (preview `complete=true` -> PUT body) renames the heading to `## Resolved Questions` and the artifact auto-transitions to `draft`.
- The captured PUT request body contains only `frontmatter` and `body` and no `status` mutation authored by the client.
- Re-saving the same completed answers is idempotent.
- Re-opening by moving an item back under a non-empty `## Open Questions` heading and PUTting re-blocks the artifact.

## Actual Behaviour

The file `tests/integration/open_questions_resolve_e2e_test.go` does not exist. No integration tests exist for the end-to-end resolution/unblocking flow.

## Logs / Output

```
$ go test -v -tags=integration ./tests/integration/... -run TestOpenQuestionsResolve
testing: warning: no tests to run
PASS
ok  	github.com/kaos-control/kaos-control/tests/integration	0.181s [no tests to run]
```
