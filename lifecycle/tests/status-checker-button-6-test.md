---
title: "Test Suite — Lineage Status Checker"
type: test
status: done
lineage: status-checker-button
parent: lifecycle/test-plans/status-checker-button-5-test.md
---

# Test Suite — Lineage Status Checker

Integration and end-to-end tests covering the lineage status checker feature:
staleness discovery via `GET /status-check`, batch advancement via
`POST /status-check/advance`, permission gating, WebSocket events, and
concurrent-access safety.

## Test files

- `tests/integration/status_check_test.go` — Milestones 2 and 3 (REST API)
- `tests/integration/status_check_e2e_test.go` — Milestone 5 (end-to-end scenarios)

## Out of scope for this suite

- **Milestone 1** (unit tests for `internal/statuscheck`) — written alongside
  the Go package, not in `tests/`.
- **Milestone 4** (Vue component tests) — written under
  `web/src/components/artifact/__tests__/`, not in `tests/`.

---

## Scenarios covered

### Milestone 2 — GET /status-check

| Test | Description |
|------|-------------|
| `TestStatusCheck_SingleLineage` | Single stale parent (idea draft, child req at planning). Lineage filter (`?lineage=`) returns only that lineage. Verifies `current_status`, `suggested_status`, and `lineage` fields. |
| `TestStatusCheck_ProjectWide` | Three lineages seeded: A stale, B stale, C current. Project-wide call returns A and B; C absent. Verifies minimum-child-status selection (`clarifying` wins over `planning`). |
| `TestStatusCheck_NoStaleness` | Parent and child both at `planning`. Expects empty `stale` array (not null). |
| `TestStatusCheck_CanAdvancePermissions` | Dev user (no product-owner/analyst role) sees `can_advance: false` and non-empty `blocked_reason` for a `draft → clarifying` suggestion. Admin sees `can_advance: true`. |
| `TestStatusCheck_Performance` | 1 000 artifacts across 10 lineages. Response must arrive within 500 ms. |

### Milestone 3 — POST /status-check/advance

| Test | Description |
|------|-------------|
| `TestAdvance_Single` | Advances one artifact. Verifies `outcome: "advanced"`, correct `new_status`, disk content updated, index entry updated. |
| `TestAdvance_MultipleSequential` | Advances idea (draft→clarifying) and req (clarifying→planning) in a single POST. Both `outcome: "advanced"`; disk state verified for both. |
| `TestAdvance_PermissionDenied` | Dev user posts an artifact requiring product-owner/analyst. Outcome is `"error"` or `"skipped"` (not `"advanced"`); non-empty `reason`; disk unchanged. |
| `TestAdvance_Idempotent` | Artifact already at the suggested status. Outcome is not `"advanced"`; file mtime unchanged (no disk write). |
| `TestAdvance_WebSocketEvent` | Hub channel registered before advance call. After successful advance, `artifact.indexed` event with correct path is received within 5 s. |
| `TestAdvance_ReEvaluatesAtExecution` | Artifact pre-advanced via `/transition` before the advance call. Endpoint re-evaluates and returns `"skipped"` rather than a conflicting transition. |

### Milestone 5 — End-to-End Scenarios

| Test | Description |
|------|-------------|
| `TestStatusCheckE2E_FullFlow` | Creates stale lineage, discovers via status-check, advances all, re-checks — expects clean result. Verifies disk frontmatter at each step. |
| `TestStatusCheckE2E_ConcurrentAdvance` | Two goroutines advance the same artifact simultaneously. Exactly one `"advanced"` outcome; artifact is at the expected status; neither call errors. |
| `TestStatusCheckE2E_SingleArtifactLineage` | Lineage with only one artifact (no children). Status-check returns empty stale list. |
| `TestStatusCheckE2E_TerminalParentIgnored` | Parent artifact is `rejected`; child is at `planning`. Rejected artifact must not appear in stale results. |

## Testing approach

All tests use the `integration` build tag (`//go:build integration`) and run
inside a `testEnv` that spins up a full HTTP server backed by a temporary
git repository, SQLite index, and auth store. No mocking of the persistence
layer — every assertion against disk state or the index reflects real I/O.

Concurrent tests use `sync.WaitGroup` and a shared `ready` channel to fire
both goroutines at the same moment, exercising the advance endpoint's
optimistic-lock or re-evaluation behaviour under actual race conditions.
