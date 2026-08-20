---
created: "2026-07-14T19:34:44+10:00"
title: "Missing integration test file open_questions_awaiting_test.go"
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
2. Locate the file `tests/open_questions_awaiting_test.go` or `tests/integration/open_questions_awaiting_test.go` in the workspace.
3. Run the Go integration tests matching this file or functions:
   ```bash
   go test -v -tags=integration ./tests/integration/... -run TestOpenQuestionsAwaiting
   ```

## Expected Behaviour

An integration test file `tests/integration/open_questions_awaiting_test.go` should exist and verify:
- `GET /api/p/testproject/artifacts?awaiting_answers=true` returns exactly the blocked artifacts with a non-empty `## Open Questions` section.
- Adding `count_only=true` returns `{count:N}` matching that set.
- Resolving the last such artifact drops the count to `0`.
- The count changes are observable via the existing `artifact.indexed` WebSocket event after a write.
- A request to `/open-questions/preview` from a session without the `product-owner` role returns HTTP 403, while a product-owner session succeeds.

## Actual Behaviour

The file `tests/integration/open_questions_awaiting_test.go` does not exist. No integration tests exist for the awaiting questions list/count endpoints or authorization controls.

## Logs / Output

```
$ go test -v -tags=integration ./tests/integration/... -run TestOpenQuestionsAwaiting
testing: warning: no tests to run
PASS
ok  	github.com/kaos-control/kaos-control/tests/integration	0.181s [no tests to run]
```
