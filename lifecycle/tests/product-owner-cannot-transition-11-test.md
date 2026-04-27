---
title: "Tests: Product Owner Transition Superuser & Allowed Targets"
type: test
status: draft
lineage: innovation-maker
parent: lifecycle/test-plans/product-owner-cannot-transition-10-test.md
---

# Tests: Product Owner Transition Superuser & Allowed Targets

Integration tests covering the product-owner superuser bypass and the
`GET /allowed-targets` endpoint, as specified in
`lifecycle/test-plans/product-owner-cannot-transition-10-test.md`.

## Test file

`tests/integration/product_owner_transition_test.go`

---

## Scenarios covered

### Milestone 1 — Full lifecycle as product-owner

**`TestProductOwnerFullLifecycle`**

Seeds an idea, a ticket at `draft`, and three approved plans (backend, frontend,
test).  Logs in as `admin@test.local` (holds `product-owner`) and drives the
ticket through every standard state in sequence:

```
draft → clarifying → planning → in-development → in-qa → approved → done
```

Asserts 200 on every step, checks the response `artifact.status` field, and
verifies the final `status: done` is persisted to disk.

The `in-development → in-qa` step is the regression case from the defect: the
product-owner superuser bypass (`workflow.HasProductOwner`) must return `true`
so `CanTransition` short-circuits the role check.  The `planning →
in-development` step also exercises the gate bypass (no gate check when
`HasProductOwner` is true).

---

### Milestone 2 — Skip-ahead transitions

Three separate sub-tests, each with a fresh environment and a single artifact:

| Test | From | To | dev (403) | admin (200) |
|---|---|---|---|---|
| `TestProductOwnerSkipAheadDraftToDone` | draft | done | ✓ | ✓ |
| `TestProductOwnerSkipAheadClarifyingToInQA` | clarifying | in-qa | ✓ | ✓ |
| `TestProductOwnerSkipAheadPlanningToApproved` | planning | approved | ✓ | ✓ |

Each test confirms that `dev@test.local` receives 403 (no rule in the matrix
for that transition and that role) and `admin@test.local` receives 200 via the
superuser bypass.

---

### Milestone 3 — Allowed-targets endpoint

Four tests covering `GET /api/p/:project/artifacts/:path/allowed-targets`.
Response shape: `{"targets": [...]}`.

> **Dependency**: these tests require the `handleAllowedTargets` handler to be
> implemented in `internal/http/transition.go` and registered in
> `internal/http/server.go` under the `GET /artifacts/*` dispatcher (suffix
> check for `/allowed-targets`).  The handler is not yet wired — these tests
> will fail with an unexpected 200 (artifact body) until the route is added.

| Test | User | From | Asserts |
|---|---|---|---|
| `TestAllowedTargetsProductOwnerGetsSuperSet` | admin | in-development | targets includes in-qa, done, approved, rejected, abandoned, blocked |
| `TestAllowedTargetsDevUserSubset` | dev | in-development | includes in-qa, blocked; excludes done, approved |
| `TestAllowedTargetsQAUserDoesNotIncludeInQA` | qa | in-development | does not include in-qa |
| `TestAllowedTargetsUnauthenticatedReturns401` | none | in-development | HTTP 401 |

---

### Milestone 4 — Regression: existing role gates still enforced

**`TestRoleGateRegressionAfterSuperuserBypass`**

Seeds a ticket at `draft` and runs three negative cases to confirm that the
superuser bypass has not relaxed enforcement for other roles:

1. `dev@test.local` → `clarifying` — expects 403 (only product-owner/analyst)
2. `qa@test.local` → `in-development` — expects 403 (only approver)
3. `dev@test.local` → `done` — expects 403 (only approver)

All three assert:
- HTTP 403
- `error.code == "forbidden"`
- `allowed_targets` is non-empty (so callers know what they *can* do)
