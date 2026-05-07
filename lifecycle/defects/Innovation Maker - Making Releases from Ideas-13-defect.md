---
title: "TestRequiredPlansGateBlocks uses product-owner user — gate bypass prevents expected 409"
type: defect
status: done
lineage: innovation-maker
parent: lifecycle/tests/Innovation Maker - Making Releases from Ideas-5-tests.md
labels:
    - defect
assignees:
    - role: test-developer
      who: agent
release: May2026
---

# TestRequiredPlansGateBlocks uses product-owner user — gate bypass prevents expected 409

## Reproduction Steps

1. Run `go test -tags integration -run TestRequiredPlansGateBlocks ./tests/integration/...`.
2. Observe the result.

## Expected Behaviour

HTTP 409 `{"error":{"code":"gate_not_ready",...},"missing":["plan-frontend","plan-test"]}` when a ticket in `planning` state is transitioned to `in-development` and the required plan types are not all approved.

## Actual Behaviour

HTTP 200 — the transition succeeds. The test fails with:

```
--- FAIL: TestRequiredPlansGateBlocks (0.12s)
    required_plans_test.go:38: expected status 409, got 200: {"artifact":{...,"status":"in-development",...}}
```

## Root Cause

The test logs in as `admin@test.local`, which in `defaultCfgYAML` holds the roles
`[product-owner, analyst, reviewer, approver]`. The required-plans gate in
`internal/http/transition.go:71` is:

```go
if !workflow.HasProductOwner(userRoles) && row.Status == "planning" && req.To == "in-development" {
```

Because `admin@test.local` has the `product-owner` role, `HasProductOwner` returns
`true` and the gate is skipped, allowing the transition to succeed without the
missing plans.

The test comment reads `// admin has approver role`, indicating the author intended
admin to act as a pure approver, not realising the test config also grants admin the
`product-owner` role.

## Fix Guidance

Change the test to use a user who holds only the `approver` role (not `product-owner`).
Options:

- Register a dedicated `approver@test.local` user in the auth store inside
  `newTestEnvFull` (or create it ad-hoc in the test using `env.authStore.CreateUser`)
  and give it only `[approver]` in the project config.
- Alternatively, call `newTestEnvWithCfgYAML` with a custom config that grants
  `admin@test.local` only `[approver]` for this test.

The `TestProductOwnerFullLifecycle` test correctly asserts that a product-owner
bypasses the gate — that behaviour must remain intact and must not be changed.
