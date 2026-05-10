---
title: "Tests: system-role workflow transition permissions (M1)"
type: test
status: draft
lineage: agent-questions-trigger-blocked-status
parent: lifecycle/defects/agent-questions-trigger-blocked-status-m1-tests-missing.md
---

## Overview

Implements the three missing Milestone 1 unit tests that verify the `system`
actor's transition permissions inside the workflow engine.  These tests were
absent (defect `agent-questions-trigger-blocked-status-m1-tests-missing.md`)
and are now added to the existing test file.

---

## Test file

| File | Scope |
|------|-------|
| `internal/workflow/workflow_test.go` | Unit — system-role transition permissions (M1) |

---

## Scenarios covered

### `TestSystemRoleCanBlockFromAnyStatus`

Iterates every non-blocked status in `KnownStatuses`
(`draft`, `clarifying`, `planning`, `in-development`, `in-qa`, `approved`,
`rejected`, `abandoned`, `done`) and asserts that
`CanTransition(status, "blocked", ["system"], "")` returns `true` in every
case.  Exercises the `{from: "", to: "blocked", roles: ["system"]}` rule in
`defaultRules`.

### `TestSystemRoleCanUnblockToDraft`

Asserts `CanTransition("blocked", "draft", ["system"], "")` returns `true`.
Exercises the `{from: "blocked", to: "draft", roles: ["system"]}` rule.

### `TestSystemRoleCannotDoOtherTransitions`

Table-driven test covering nine disallowed pairs
(`draft→clarifying`, `clarifying→planning`, `planning→in-development`,
`in-development→in-qa`, `in-qa→approved`, `approved→done`,
`draft→rejected`, `draft→abandoned`, `draft→done`).
Each asserts `CanTransition(from, to, ["system"], "")` returns `false`,
confirming the `system` actor has no additional privileges beyond the two
auto-block/unblock transitions.

---

## How to run

```sh
go test ./internal/workflow/... -run TestSystemRole -v
```

Expected output: all three functions `PASS`.
