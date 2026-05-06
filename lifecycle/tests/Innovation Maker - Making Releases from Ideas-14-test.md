---
title: "Fix: TestRequiredPlansGateBlocks — use approver-only user for required-plans gate"
type: test
status: draft
lineage: innovation-maker
parent: lifecycle/defects/Innovation Maker - Making Releases from Ideas-13-defect.md
---

# Fix: TestRequiredPlansGateBlocks — use approver-only user for required-plans gate

Companion artifact for defect
`lifecycle/defects/Innovation Maker - Making Releases from Ideas-13-defect.md`.

## Root cause

`TestRequiredPlansGateBlocks` was calling `newTestEnvFull` (via `newTestEnv`)
which uses `defaultCfgYAML`, where `admin@test.local` holds
`[product-owner, analyst, reviewer, approver]`.  The `product-owner` role causes
`workflow.HasProductOwner` to return `true`, which short-circuits the
required-plans gate in `internal/http/transition.go`, allowing the transition and
producing HTTP 200 instead of the expected 409.

## Fix

Introduced `approverOnlyCfgYAML` — an alternate project config identical to
`defaultCfgYAML` except `admin@test.local` is restricted to `[approver]`.
Both `TestRequiredPlansGateBlocks` and `TestRequiredPlansGateSucceeds` now call
`newTestEnvWithCfgYAML(t, seeds, approverOnlyCfgYAML)` so the gate is evaluated
on plan availability rather than bypassed by the user's role.

## Test file

`tests/integration/required_plans_test.go`

## Scenarios covered

### `TestRequiredPlansGateBlocks`

Seeds an idea, a ticket at `planning`, and **one** approved plan (`plan-backend`
only — `plan-frontend` and `plan-test` are absent).  Logs in as
`admin@test.local` (approver-only config — no `product-owner`).

Asserts:
- `POST .../transition {"to":"in-development"}` → HTTP **409**
- Response body `error.code == "gate_not_ready"`
- `missing` array contains exactly 2 entries: `"plan-frontend"` and `"plan-test"`

### `TestRequiredPlansGateSucceeds`

Seeds an idea, a ticket at `planning`, and **all three** required approved plans
(`plan-backend`, `plan-frontend`, `plan-test`).  Same approver-only config and
user.

Asserts:
- `POST .../transition {"to":"in-development"}` → HTTP **200**
- Response body `artifact.status == "in-development"`

## Unchanged behaviour

`TestProductOwnerFullLifecycle` (in `product_owner_transition_test.go`) continues
to use `defaultCfgYAML`, where `admin@test.local` has `product-owner`, and
asserts that the gate is bypassed for product-owners.  That behaviour is
intentionally preserved.
