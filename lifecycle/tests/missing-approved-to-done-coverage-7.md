---
title: Integration tests — approved → done transition coverage
type: test
status: draft
lineage: innovation-maker
parent: lifecycle/defects/missing-approved-to-done-coverage.md
---

## Overview

This artifact documents the integration tests added to cover the
`approved → done` workflow transition and the complete terminal lifecycle
path, addressing the regression gap identified in the defect
`lifecycle/defects/missing-approved-to-done-coverage.md`.

Tests live in `tests/integration/approved_to_done_test.go` and run with:

```
go test -v -tags=integration -run 'TestApprovedToDone|TestFullLifecycle' ./tests/integration/
```

## Scenarios Covered

### `TestApprovedToDoneByApprover`

Verifies role enforcement on the `approved → done` transition:

- A user holding only the `qa` role gets **403 forbidden**.
- A user holding only developer roles (`backend-developer`, `frontend-developer`,
  `test-developer`) gets **403 forbidden**.
- A user holding the `approver` role (admin: `[product-owner, analyst, reviewer,
  approver]`) gets **200** and the artifact status becomes `done`.
- The `status: done` value is written to disk (file content verified).

### `TestFullLifecyclePlanningToDone`

Exercises the complete terminal path end-to-end:

```
planning → in-development → in-qa → approved → done
```

Setup seeds a ticket in `planning` state with all three required plan types
(`plan-backend`, `plan-frontend`, `plan-test`) already in `approved` status so
the planning gate is satisfied.

Each step is performed by a user holding the required role:

| Transition             | User               | Role(s) used                                     |
|------------------------|--------------------|--------------------------------------------------|
| planning → in-development | admin@test.local | approver                                        |
| in-development → in-qa   | dev@test.local   | backend-developer / test-developer              |
| in-qa → approved         | qa@test.local    | qa                                              |
| approved → done          | admin@test.local | approver                                        |

After all transitions:

- Response body reports `status: done`.
- `status: done` is present in the artifact file on disk.
- `git.Log` for the artifact returns at least 2 commits (initial seed + at
  least one recorded transition).

## Test File

| File | Scenarios |
|------|-----------|
| `tests/integration/approved_to_done_test.go` | Role gate on `approved → done`; full `planning → done` chain |
