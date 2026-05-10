---
title: "Tests: auto-transition artifacts to blocked on open questions"
type: test
status: in-qa
lineage: agent-questions-trigger-blocked-status
parent: lifecycle/test-plans/agent-questions-trigger-blocked-status-5-test.md
---

## Overview

Integration and unit tests implementing the test plan for the auto-block/unblock
feature: automatically transitioning artifacts to `blocked` when a non-empty
`## Open Questions` section is present, and back to `draft` when it is removed.

---

## Test files

| File | Scope |
|------|-------|
| `internal/workflow/workflow_test.go` | Unit — system-role transition permissions (M1) |
| `internal/index/autoblock_test.go` | Unit — `applyOpenQuestionTransition` logic (M2, M3) |
| `tests/integration/autoblock_startup_test.go` | Integration — startup Scan corrects stale status (M5) |
| `tests/integration/autoblock_watcher_test.go` | Integration — watcher auto-block, manual transitions, WS events (M4, M7, M8) |
| `tests/integration/artifact_blocked_questions_test.go` | Integration — API PUT path (M6, extended) |

---

## Milestone 1 — Workflow engine: system role

**File:** `internal/workflow/workflow_test.go`

Added three table-driven test functions:

- **`TestSystemRoleCanBlockFromAnyStatus`** — iterates all known statuses
  (`draft`, `clarifying`, `planning`, `in-development`, `in-qa`, `approved`,
  `rejected`, `abandoned`, `done`) and asserts
  `CanTransition(status, "blocked", ["system"])` returns `true` in each case.

- **`TestSystemRoleCanUnblockToDraft`** — asserts
  `CanTransition("blocked", "draft", ["system"])` returns `true`.

- **`TestSystemRoleCannotDoOtherTransitions`** — table of disallowed pairs
  (e.g. `draft→clarifying`, `approved→done`) asserts each returns `false` for
  the `"system"` actor.

Run:
```
go test ./internal/workflow/... -run TestSystemRole
```

---

## Milestone 2 — Unit tests for `applyOpenQuestionTransition`

**File:** `internal/index/autoblock_test.go` (new)

Uses a temp-directory SQLite index with a function-backed `Transitioner` to
avoid the `index ↔ workflow` circular import. Seven test cases:

1. **`TestAutoBlock_DraftWithOQ`** — draft + OQ → disk `status: blocked`,
   product-owner assignee, 1 event in DB.
2. **`TestAutoBlock_BlockedWithNoOQ`** — blocked + no OQ → disk `status: draft`,
   1 event in DB.
3. **`TestAutoBlock_IdempotentBlockedWithOQ`** — blocked + OQ (already correct)
   → second `IndexFile` call: SHA guard fires, no disk write, no new event.
4. **`TestAutoBlock_IdempotentDraftNoOQ`** — draft + no OQ → second call: SHA
   guard fires, no disk write, no new event.
5. **`TestAutoBlock_WorkflowRejection`** — uses a `rejectAll` transitioner; draft
   + OQ → no disk write, no event (workflow rejection logged, nil returned).
6. **`TestAutoBlock_AssigneeDeduplication`** — artifact already has
   `{role: product-owner, who: agent}`; auto-block does not add a duplicate.
7. **`TestAutoBlock_AtomicWrite`** — after auto-block, no `.tmp` files remain in
   the lifecycle directory.

Run:
```
go test ./internal/index/... -run TestAutoBlock_
```

---

## Milestone 3 — Circular-trigger prevention (SHA-256 guard)

**File:** `internal/index/autoblock_test.go`

- **`TestAutoBlock_NoCircularReindex`** — writes a draft+OQ artifact, calls
  `IndexFile` (triggers auto-block + disk rewrite), calls `IndexFile` a second
  time. Verifies the second call returns via SHA guard: event count unchanged,
  file mtime unchanged.

Run:
```
go test ./internal/index/... -run TestAutoBlock_NoCircularReindex
```

---

## Milestone 4 — Watcher-triggered auto-block

**File:** `tests/integration/autoblock_watcher_test.go` (new)

- **`TestAutoBlock_WatcherTriggersBlock`** — seeds a draft artifact (no OQ),
  rewrites the file on disk with an OQ section, polls
  `GET /api/p/testproject/artifacts/:path` until `status == "blocked"` (timeout
  3 s), asserts `{role: product-owner, who: agent}` in assignees. Then rewrites
  the file without OQ and polls until `status == "draft"`.

Run:
```
go test ./tests/integration/... -tags=integration -run TestAutoBlock_WatcherTriggersBlock
```

---

## Milestone 5 — Startup scan corrects stale status

**File:** `tests/integration/autoblock_startup_test.go` (new)

Two cases, both queried immediately after `newTestEnv` returns (startup Scan
is synchronous):

- **`TestAutoBlock_StartupScanBlocksDraftWithOQ`** — seeds `draft` + OQ;
  startup Scan auto-blocks it. API returns `status: blocked` with PO assignee.
- **`TestAutoBlock_StartupScanUnblocksBlockedWithNoOQ`** — seeds `blocked` + no
  OQ; startup Scan auto-unblocks it. API returns `status: draft`.

Run:
```
go test ./tests/integration/... -tags=integration -run TestAutoBlock_Startup
```

---

## Milestone 6 — API write triggers auto-block/unblock

**File:** `tests/integration/artifact_blocked_questions_test.go` (updated)

New test functions added:

- **`TestBlockedQuestions_PutOQThenRemoveOQ`** — PUT with OQ → `blocked`,
  then PUT without OQ → `draft` (full cycle via API).
- **`TestBlockedQuestions_NoDuplicateEventsOnPut`** — counts `status_changed`
  events before/after a single PUT with OQ; verifies exactly 1 new event.
- **`TestBlockedQuestions_BlockedWithNoOQAutoUnblocks`** — replaces the old
  `TestBlockedQuestions_ExistingBlockedStatusPreserved` test (which assumed
  the old handler-level behaviour). Documents and asserts the new correct
  behaviour: a `blocked` + no-OQ artifact is auto-unblocked to `draft` by both
  the startup Scan and the PUT handler.

Run:
```
go test ./tests/integration/... -tags=integration -run TestBlockedQuestions
```

---

## Milestone 7 — Manual transitions are not broken

**File:** `tests/integration/autoblock_watcher_test.go`

- **`TestAutoBlock_ManualTransitionsUnaffected`** — transitions a no-OQ artifact
  `draft → clarifying → draft` via the transition API; verifies no `status_changed`
  event is emitted by the auto-block path.
- **`TestAutoBlock_ManualBlockRevertsBecauseNoOQ`** — documents the system
  behaviour: manually transitioning a no-OQ artifact to `blocked` is immediately
  overridden by auto-unblock; the response shows `draft`.

Run:
```
go test ./tests/integration/... -tags=integration -run TestAutoBlock_Manual
```

---

## Milestone 8 — WebSocket event verification

**File:** `tests/integration/autoblock_watcher_test.go`

- **`TestAutoBlock_WSEventsOnWatcherBlock`** — registers a hub channel before the
  disk write, triggers auto-block via the watcher, and asserts:
  - `artifact.indexed` with `blocked_reason == "open_questions_detected"` is received.
  - `feed.new` event referencing `open_questions_detected` is received.
  Then triggers auto-unblock and asserts:
  - `artifact.indexed` for the path is received.
  - `feed.new` event referencing `open_questions_resolved` is received.

Run:
```
go test ./tests/integration/... -tags=integration -run TestAutoBlock_WSEventsOnWatcherBlock
```
