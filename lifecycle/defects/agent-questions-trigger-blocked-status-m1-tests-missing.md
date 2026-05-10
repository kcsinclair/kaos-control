---
title: "Milestone 1 workflow system-role tests not implemented"
type: defect
status: approved
lineage: agent-questions-trigger-blocked-status
parent: lifecycle/tests/agent-questions-trigger-blocked-status-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

## Reproduction Steps

1. Check out the repository at the current HEAD.
2. Run the Milestone 1 workflow unit tests as specified in the test artifact:
   ```
   go test ./internal/workflow/... -run TestSystemRole -v
   ```
3. Observe the output.

## Expected Behaviour

Three test functions described in Milestone 1 of the test artifact should exist in
`internal/workflow/workflow_test.go` and execute:

- `TestSystemRoleCanBlockFromAnyStatus` — iterates all known statuses (`draft`,
  `clarifying`, `planning`, `in-development`, `in-qa`, `approved`, `rejected`,
  `abandoned`, `done`) and asserts `CanTransition(status, "blocked", ["system"])`
  returns `true` for each.
- `TestSystemRoleCanUnblockToDraft` — asserts
  `CanTransition("blocked", "draft", ["system"])` returns `true`.
- `TestSystemRoleCannotDoOtherTransitions` — asserts that disallowed pairs (e.g.
  `draft→clarifying`, `approved→done`) return `false` for the `"system"` actor.

## Actual Behaviour

`go test` reports `no tests to run` for the `TestSystemRole` pattern:

```
testing: warning: no tests to run
PASS
ok  github.com/kaos-control/kaos-control/internal/workflow  0.545s [no tests to run]
```

Inspection of `internal/workflow/workflow_test.go` confirms that none of the three
`TestSystemRole*` functions are present. The file contains only
`TestProductOwnerBypassesAnyTransition`, `TestNonProductOwnerStillRestricted`,
`TestExistingRulesStillApply`, `TestWorkflowPredecessors`, and
`TestAllowedTargetsForProductOwnerCoversAllStatuses`.

The workflow engine itself does implement the `system` role rules
(`internal/workflow/workflow.go` lines 42–43), so this is purely a missing-test gap,
not a backend logic defect.

## Logs / Output

```
$ go test ./internal/workflow/... -run TestSystemRole -v
testing: warning: no tests to run
PASS
ok  	github.com/kaos-control/kaos-control/internal/workflow	0.545s [no tests to run]
```
