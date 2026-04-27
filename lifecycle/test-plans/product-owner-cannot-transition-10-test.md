---
title: "Test Plan: Product Owner Transition Superuser & Allowed Targets"
type: plan-test
status: done
lineage: innovation-maker
parent: lifecycle/defects/product-owner-cannot-transition.md
labels:
    - testing
    - defect-fix
---

# Test Plan: Product Owner Transition Superuser & Allowed Targets

Integration tests covering the product-owner superuser bypass
([[product-owner-cannot-transition-8-be]]) and the new allowed-targets
endpoint ([[product-owner-cannot-transition-9-fe]]).

---

## Milestone 1 — Product owner can perform every standard transition

### Description
Add a test that walks a single artifact through the full lifecycle
(`draft → clarifying → planning → in-development → in-qa → approved → done`)
using only the product-owner (admin) user. Currently this fails at
`in-development → in-qa` because `product-owner` is not in that rule's role
list.

### Files to change
- `tests/integration/workflow_test.go` (or a new `product_owner_transition_test.go`)

### Test steps
1. Seed an idea + requirement + three approved plans (to satisfy the gate).
2. Login as admin (`admin@test.local` — holds `product-owner`).
3. POST transition for each step: `draft → clarifying → planning → in-development → in-qa → approved → done`.
4. Assert 200 on every step.
5. Verify final on-disk status is `done`.

### Acceptance criteria
- [ ] Test passes with the backend fix from [[product-owner-cannot-transition-8-be]] applied
- [ ] Test fails on the current (unfixed) codebase, confirming the defect
- [ ] Uses the existing `newTestEnv` / `seedArtifact` helpers

---

## Milestone 2 — Product owner can skip-ahead transitions

### Description
Test non-standard transitions that only a superuser should be able to perform
(e.g. `draft → done`, `clarifying → in-qa`). These don't appear in the
default rule matrix but should succeed for `product-owner`.

### Files to change
- `tests/integration/product_owner_transition_test.go`

### Test steps
1. Seed an artifact at `draft`.
2. Login as admin.
3. POST transition to `done`.
4. Assert 200.
5. Repeat for `clarifying → in-qa` and `planning → approved`.

### Acceptance criteria
- [ ] All skip-ahead transitions return 200 for product-owner
- [ ] Same transitions return 403 for a non-product-owner user (e.g. `dev@test.local`)

---

## Milestone 3 — Allowed-targets endpoint tests

### Description
Test the `GET /api/p/:project/artifacts/:path/allowed-targets` endpoint
added in [[product-owner-cannot-transition-9-fe]] Milestone 2.

### Files to change
- `tests/integration/product_owner_transition_test.go`

### Test steps
1. Seed an artifact at `in-development`.
2. Login as admin (product-owner) → GET allowed-targets → assert it includes
   `in-qa`, `done`, `approved`, `rejected`, `abandoned`, `blocked`.
3. Login as dev (backend-developer) → GET allowed-targets → assert it includes
   `in-qa` and `blocked` but NOT `done` or `approved`.
4. Login as qa → GET allowed-targets → assert it does NOT include `in-qa`.

### Acceptance criteria
- [ ] Product-owner gets a superset of every other role's targets
- [ ] Non-product-owner roles get only their authorised targets
- [ ] Unauthenticated request returns 401
- [ ] Response shape is `{"targets": [...]}`

---

## Milestone 4 — Regression: existing role gates still enforced

### Description
Ensure the superuser bypass hasn't weakened enforcement for other roles.
Re-run key negative cases.

### Files to change
- `tests/integration/product_owner_transition_test.go`

### Test steps
1. Seed artifact at `draft`.
2. Login as dev (backend-developer) → POST transition to `clarifying` → assert 403.
3. Login as qa → POST transition to `in-development` → assert 403.
4. Login as dev → POST transition to `done` → assert 403.

### Acceptance criteria
- [ ] All four negative cases return 403
- [ ] Error response includes `"code": "forbidden"` and non-empty `allowed_targets`
- [ ] Existing tests `TestTransitionWithRoleGate` and `TestApprovedToDoneByApprover` still pass

---

## Cross-links

- [[product-owner-cannot-transition-8-be]] — backend superuser logic under test
- [[product-owner-cannot-transition-9-fe]] — allowed-targets endpoint under test
