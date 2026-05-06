---
title: "Tests: Status Check Advance Flow"
type: test
status: draft
lineage: status-checker-button-no-status-update
parent: lifecycle/test-plans/status-checker-button-no-status-update-4-test.md
---

# Tests: Status Check Advance Flow

Integration tests verifying the status-check GET and advance POST endpoints —
both the response shapes they return and the on-disk mutations they perform.

## Test file

`tests/integration/status_check_test.go`

## Scenarios covered

### Milestone 1 — Children field shape (GET /status-check)

**`TestStatusCheck_ChildrenFieldShape`**
Seeds a lineage with a parent at `in-development` and a child at `done`.
Calls `GET /status-check?lineage=sc-children` and asserts:
- `stale[0].children` is a non-nil, non-empty array (not bare strings).
- Each element has a non-empty `path` string field.
- Each element has a non-empty `status` string field.
- The requirement child reports `status: "done"`.

### Milestone 2 — Advance endpoint response contract (POST /status-check/advance)

**`TestAdvance_OkAndAdvancedToFields`**
Seeds a lineage with the idea at `in-development` and the requirement at `done`.
Calls `POST /status-check/advance` as admin and asserts:
- `results[0].ok` is `true`.
- `results[0].advanced_to` is non-empty and equals `"done"`.
- The artifact's on-disk frontmatter contains `status: done`.

**`TestAdvance_ResponseContractPermissionDenied`**
Same lineage but advances as the `dev` user (holds `backend-developer` only,
lacks `product-owner`/`analyst`). Asserts:
- `results[0].ok` is `false`.
- `results[0].reason` is non-empty.
- The artifact file is NOT modified (still `status: draft`).

### Milestone 3 — Staleness detection edge cases (GET /status-check)

**`TestStatusCheck_MultiLevelLineageStaleness`**
3-level lineage: idea (`in-development`) → requirement (`done`) → plan (`done`).
Asserts:
- Idea IS stale (its direct child, the requirement, is strictly ahead at `done`).
- Suggested status for the idea is `"done"`.
- Requirement is NOT stale (its child is at the same status `done`, not strictly ahead).

**`TestStatusCheck_MixedProgressSiblings`**
Idea (`in-development`) with two direct children: req-a (`done`) and req-b
(`in-development`). Not ALL non-terminal children are ahead → idea must NOT
appear in the stale list.

**`TestStatusCheck_TerminalChildExcluded_Integration`**
Idea (`in-development`) with two children: req-a (`done`) and req-b (`rejected`).
The rejected child is excluded from the comparison. The remaining non-terminal
child (req-a) is ahead → idea IS stale with `suggested_status: "done"`.
Also asserts that the rejected artifact itself does not appear in the stale list.

### Milestone 4 — E2E round-trip (already covered)

`tests/integration/status_check_e2e_test.go` contains `TestStatusCheckE2E_FullFlow`
which exercises the complete flow: create stale lineage → GET detects staleness
with `can_advance: true` → POST advance → GET confirms lineage is clean → disk
confirms frontmatter updated. This test satisfies the Milestone 4 requirements
from the test plan.

## Supporting struct changes

The `staleEntry` struct was extended with a `Children []staleChild` field
(`json:"children"`) to enable Milestone 1 assertions.

The `advanceResult` struct was updated to match the actual backend response:
`NewStatus string \`json:"new_status"\`` replaced by `Ok bool \`json:"ok"\``
and `AdvancedTo string \`json:"advanced_to,omitempty"\``.
