---
title: No integration test covers approved → done transition
type: defect
status: approved
lineage: innovation-maker
parent: lifecycle/tests/Innovation Maker - Making Releases from Ideas-5-tests.md
labels:
    - defect
assignees:
    - role: test-developer
      who: agent
---

## Reproduction Steps

1. Run the workflow integration tests:
   ```
   go test -v -tags=integration -run 'TestTransition' ./tests/integration/
   ```
2. Observe that `TestTransitionChainDraftToDone` does **not** exercise the
   `in-development → in-qa → approved → done` path. It terminates at
   `rejected` instead, with the comment:
   > "Note: planning → in-development requires approved plans (tested separately)"
3. Search for any test covering `approved → done` by the `product-owner`
   (or `approver`) role — none exists.

## Expected Behaviour

At least one integration test should exercise the complete terminal path:

```
planning → in-development → in-qa → approved → done
```

with a user holding the `approver` role (e.g., `admin@test.local` who has
`[product-owner, analyst, reviewer, approver]`), confirming that the
`approved → done` transition succeeds and the status is persisted to disk.

This directly covers the scenario from defect
`lifecycle/defects/product-owner-cannot-transition.md`.

## Actual Behaviour

No integration test covers `approved → done`. A regression in the
`workflow.Engine.CanTransition` logic for this specific edge (or a future
change to `defaultRules`) would not be caught by the test suite.

## Logs / Output

```
=== RUN   TestTransitionChainDraftToDone
transitions executed: draft→clarifying, clarifying→planning, planning→rejected
--- PASS: TestTransitionChainDraftToDone (0.18s)

$ go test -v -tags=integration -run 'TestProductOwner|TestApprovedToDone|TestFullLifecycle' ./tests/integration/
testing: warning: no tests to run
PASS
ok      github.com/kaos-control/kaos-control/tests/integration  0.380s [no tests to run]
```

No test matching the `approved → done` scenario exists.
