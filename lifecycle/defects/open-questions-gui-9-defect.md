---
title: "Missing integration test file open_questions_routing_test.go"
type: defect
status: approved
lineage: open-questions-gui
parent: lifecycle/tests/open-questions-gui-5-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Reproduction Steps

1. Read the test suite specification at [open-questions-gui-5-test.md](file:///Users/keith/Code/kaos-control/lifecycle/tests/open-questions-gui-5-test.md).
2. Locate the file `tests/open_questions_routing_test.go` or `tests/integration/open_questions_routing_test.go` in the workspace.
3. Run the Go integration tests matching this file or functions:
   ```bash
   go test -v -tags=integration ./tests/integration/... -run TestOpenQuestionsRouting
   ```

## Expected Behaviour

An integration test file `tests/integration/open_questions_routing_test.go` should exist and verify post-resolution routing rules:
- For a `type: requirement` artifact, after a complete resolve the status is `draft` (awaiting approval) and no approval/transition has occurred without an explicit approve action.
- For a developer-raised artifact (`plan-*` containing open questions), after resolve no requeue/run is started automatically; an explicit requeue call targets the originating developer role and starts a run.
- Neither the approve nor the requeue side effect is observable until the corresponding deliberate API call is made.

## Actual Behaviour

The file `tests/integration/open_questions_routing_test.go` does not exist. No integration tests exist for post-resolution routing behavior.

## Logs / Output

```
$ go test -v -tags=integration ./tests/integration/... -run TestOpenQuestionsRouting
testing: warning: no tests to run
PASS
ok  	github.com/kaos-control/kaos-control/tests/integration	0.181s [no tests to run]
```
