---
title: "Backend Plan: Product Owner Superuser Transitions"
type: plan-backend
status: draft
lineage: innovation-maker
parent: lifecycle/defects/product-owner-cannot-transition.md
labels:
    - workflow
    - backend
    - defect-fix
---

# Backend Plan: Product Owner Superuser Transitions

The `product-owner` role is only listed on a subset of transition rules in
`internal/workflow/workflow.go`. The spec §6.2 matrix does not list
`product-owner` on every row, but the product owner expects to override any
transition. This plan adds superuser semantics for `product-owner` so it can
perform any transition without modifying every individual rule.

---

## Milestone 1 — Add `product-owner` bypass in `CanTransition`

### Description
Modify `Engine.CanTransition` to short-circuit and return `true` when the
caller holds the `product-owner` role. This is the minimal change: the
existing rules remain intact for all other roles, and the product-owner gains
universal transition authority.

### Files to change
- `internal/workflow/workflow.go` — update `CanTransition` method

### Implementation detail
At the top of `CanTransition`, iterate `userRoles` and return `true` if any
element equals `"product-owner"`. This keeps the bypass explicit and
auditable — a single `for` loop before the rule scan.

### Acceptance criteria
- [ ] `CanTransition("in-development", "done", ["product-owner"])` returns `true`
- [ ] `CanTransition("in-qa", "approved", ["product-owner"])` returns `true`
- [ ] `CanTransition("draft", "done", ["product-owner"])` returns `true` (skip-ahead)
- [ ] All existing non-product-owner transition rules still work unchanged
- [ ] `go build ./...` and `go vet ./...` pass

---

## Milestone 2 — Update `AllowedTargets` for `product-owner`

### Description
`AllowedTargets` is used by the HTTP handler to return the `allowed_targets`
field in 403 responses and could be used by a future frontend endpoint. When
the caller holds `product-owner`, it should return all known target statuses.

### Files to change
- `internal/workflow/workflow.go` — update `AllowedTargets` method

### Implementation detail
At the top of `AllowedTargets`, check whether `userRoles` contains
`"product-owner"`. If so, collect every distinct `r.to` from `e.rules` that
is reachable from the given `from` status (respecting the empty-from wildcard
rules) and return that set. Alternatively, return the full status vocabulary
since product-owner can transition to anything.

### Acceptance criteria
- [ ] `AllowedTargets("in-development", ["product-owner"])` includes `"done"`, `"in-qa"`, `"abandoned"`, `"rejected"`, `"blocked"` etc.
- [ ] `AllowedTargets("draft", ["backend-developer"])` remains unchanged (no regression)
- [ ] `go build ./...` and `go vet ./...` pass

---

## Milestone 3 — Unit tests for product-owner bypass

### Description
Add a `workflow_test.go` file with table-driven tests covering the
product-owner superuser behaviour as well as regression tests for normal
role-gated transitions.

### Files to change
- `internal/workflow/workflow_test.go` (new file)

### Acceptance criteria
- [ ] Test: product-owner can perform every transition in the default matrix
- [ ] Test: product-owner can perform transitions NOT in the default matrix (e.g. `draft → done`)
- [ ] Test: non-product-owner roles are still denied transitions they don't hold
- [ ] Test: `AllowedTargets` for product-owner returns a superset of targets vs other roles
- [ ] `go test ./internal/workflow/... -short` passes

---

## Cross-links

- [[product-owner-cannot-transition-9-fe]] — frontend should fetch allowed targets from the backend instead of showing a hardcoded list
- [[product-owner-cannot-transition-10-test]] — integration tests covering the HTTP layer
