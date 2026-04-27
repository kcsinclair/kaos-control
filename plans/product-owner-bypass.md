# Product Owner Transition Bypass

## Context

The `product-owner` role is treated like any other role by the workflow engine, and the default rule matrix only grants product-owner a small set of transitions:

- `draft ↔ clarifying`
- `* → abandoned`
- `blocked → draft`

This blocks the product-owner from doing maintenance / recovery work — e.g. setting an artifact's status directly to `done` to fix data drift, or pushing through a stuck artifact while plans are being smoothed out.

**Reproducer** — [`lifecycle/defects/product-owner-cannot-transition.md`](lifecycle/defects/product-owner-cannot-transition.md). Even with all four privileged roles assigned (`product-owner, analyst, reviewer, approver`), the user got `role(s) ... cannot transition "in-progress" → "done"` because no rule covers that edge for any of those roles.

The intent: **product-owner is the project superuser. Treat them as exempt from the role matrix and the required-plans gate.**

(Side note: `in-progress` appears in the FE `TransitionDialog.vue` status list but is **not** in backend `KnownStatuses` — that inconsistency is how the artifact ended up in an unreachable state. Out of scope for this plan; track separately.)

## Approach

Single short-circuit at the workflow engine entry, plus a sibling bypass for the gate check in the HTTP handler. No data model or config changes.

### 1. `internal/workflow/workflow.go`

Add a small helper and short-circuit both public methods:

```go
func hasProductOwner(roles []string) bool {
    for _, r := range roles {
        if r == "product-owner" {
            return true
        }
    }
    return false
}
```

In `CanTransition` (line 62), return `true` immediately if the user has product-owner. Place the check at the top, before the rule loop.

In `AllowedTargets` (line 83), if the user has product-owner, return every distinct `to` value across all rules (which covers the full known-status set). Otherwise the existing logic.

### 2. `internal/http/transition.go`

The handler currently enforces a separate plan-readiness gate at lines 70–82 (`row.Status == "planning" && req.To == "in-development"` → `workflow.GateReady`). Wrap it in the same product-owner bypass:

```go
if !hasProductOwner(userRoles) && row.Status == "planning" && req.To == "in-development" {
    // existing GateReady logic
}
```

Reuse the helper from the workflow package by exporting it (`workflow.HasProductOwner`) or just inlining the loop in `transition.go` — exporting is cleaner.

## Files to modify

| File | Change |
|---|---|
| `internal/workflow/workflow.go` | `HasProductOwner` helper (exported); short-circuit `CanTransition` and `AllowedTargets` for product-owner |
| `internal/http/transition.go` | Skip `GateReady` for product-owner via `workflow.HasProductOwner(userRoles)` |

## Verification

1. `go build ./...` — clean.
2. `go vet ./...` — clean.
3. Add unit test in `internal/workflow/workflow_test.go` (create if missing): assert `Engine.CanTransition("in-progress", "done", []string{"product-owner"}) == true` and `Engine.CanTransition("in-progress", "done", []string{"reviewer"}) == false`.
4. Manual end-to-end:
   - `make build && make run`
   - Log in as `keith@sinclair.org.au` (configured product-owner in `lifecycle/config.yaml`).
   - Open the defect `lifecycle/defects/product-owner-cannot-transition.md`.
   - Click **Change Status** → choose any target (e.g. `done`) → Confirm. Should succeed; status persists to disk; commit is created.
   - Mark the defect's own status `done` and verify the artifact list reflects it.
5. Negative check: temporarily edit the user binding to remove `product-owner` (keep `reviewer`), restart, attempt the same transition — expect 403.
