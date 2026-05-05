---
title: 'Test plan: auto-transition artifacts to blocked on open questions'
type: plan-test
status: in-development
lineage: agent-questions-trigger-blocked-status
parent: lifecycle/requirements/agent-questions-trigger-blocked-status-2.md
---

## Overview

This plan covers unit and integration tests for the auto-block/unblock feature described in [[agent-questions-trigger-blocked-status]]. Tests validate every acceptance criterion from the requirement, plus edge cases around idempotency, circular triggers, and workflow rejection.

Existing tests in `internal/artifact/artifact_test.go` (HasOpenQuestions) and `tests/integration/artifact_blocked_questions_test.go` provide a foundation. New tests extend coverage to the indexer-level auto-transition logic introduced by the backend plan.

---

## Milestone 1 — Unit tests for the system role in the workflow engine

**Description:** Add tests to `internal/workflow/workflow_test.go` that verify the `system` role can perform `any → blocked` and `blocked → draft` transitions but cannot perform other transitions.

**Files to change:**

- `internal/workflow/workflow_test.go` — add test cases:
  - `TestSystemRoleCanBlockFromAnyStatus` — iterate all non-terminal statuses, assert `CanTransition(status, "blocked", ["system"])` returns `true`.
  - `TestSystemRoleCanUnblockToDraft` — assert `CanTransition("blocked", "draft", ["system"])` returns `true`.
  - `TestSystemRoleCannotDoOtherTransitions` — assert `CanTransition("draft", "clarifying", ["system"])` returns `false` (and a selection of other disallowed pairs).

**Acceptance criteria:**

- [ ] All three test cases pass.
- [ ] Tests are table-driven where applicable.

---

## Milestone 2 — Unit tests for `applyOpenQuestionTransition`

**Description:** Add unit tests for the new `applyOpenQuestionTransition` function in `internal/index/`. These test the core detect-validate-rewrite-upsert logic in isolation using a temp directory and in-memory SQLite index.

**Files to change:**

- `internal/index/autoblock_test.go` (new) — test cases:
  1. **Auto-block:** Artifact with `## Open Questions` content and status `draft` → status becomes `blocked` on disk, assignees include `{role: product-owner, who: agent}`, event inserted.
  2. **Auto-unblock:** Artifact with status `blocked` and no open questions → status becomes `draft` on disk, event inserted.
  3. **Idempotent no-op (blocked):** Artifact already `blocked` with open questions → no disk write, no event.
  4. **Idempotent no-op (not blocked):** Artifact with status `draft` and no open questions → no disk write, no event.
  5. **Workflow rejection:** Artifact with status `done` and open questions → `done → blocked` rejected by engine, no disk write, warning logged.
  6. **Assignee deduplication:** Artifact already has `{role: product-owner, who: agent}` in assignees → auto-block does not add a duplicate.
  7. **Atomic write:** Verify no `.tmp` files are left behind after the operation completes.

**Acceptance criteria:**

- [ ] All seven test cases pass.
- [ ] Each test verifies both the on-disk file content and the SQLite index state.
- [ ] Event insertion is verified by querying the events table.

---

## Milestone 3 — Unit test for circular-trigger prevention (SHA-256 guard)

**Description:** Verify that after `applyOpenQuestionTransition` rewrites a file, a subsequent `IndexFile` call on the same file detects the matching SHA-256 and skips re-processing.

**Files to change:**

- `internal/index/autoblock_test.go` — add test case:
  - `TestAutoBlock_NoCircularReindex` — write an artifact with open questions, call `IndexFile` (triggers auto-block + rewrite), call `IndexFile` again on the same path, assert that no second event is inserted and no second disk write occurs.

**Acceptance criteria:**

- [ ] The second `IndexFile` call returns without inserting a duplicate event.
- [ ] The file mtime/content is unchanged after the second call.

---

## Milestone 4 — Integration test: watcher-triggered auto-block

**Description:** End-to-end test that writes a file with `## Open Questions` to disk, waits for the watcher to pick it up, and verifies the artifact is auto-blocked via the API.

**Files to change:**

- `tests/integration/autoblock_watcher_test.go` (new) — test case:
  1. Seed a `draft` artifact without open questions.
  2. Rewrite the file on disk to add `## Open Questions\n- Why?`.
  3. Poll `GET /api/p/:project/artifacts/:path` until status is `blocked` (timeout 2s).
  4. Assert status is `blocked`, assignees include `{role: product-owner, who: agent}`.
  5. Rewrite the file to remove the open questions section.
  6. Poll until status is `draft`.
  7. Assert status is `draft`.

**Acceptance criteria:**

- [ ] The test passes reliably (not flaky — use polling with timeout, not sleep).
- [ ] Uses `newTestEnv` helper with seed artifacts.
- [ ] Tagged with `//go:build integration`.

---

## Milestone 5 — Integration test: startup scan corrects stale status

**Description:** Verify that when the server starts, artifacts with stale status are corrected.

**Files to change:**

- `tests/integration/autoblock_startup_test.go` (new) — test cases:
  1. **Should-be-blocked:** Create a `draft` artifact with `## Open Questions` content on disk. Start the server (via `newTestEnv`). Query the API and assert status is `blocked`.
  2. **Should-be-unblocked:** Create a `blocked` artifact without open questions on disk. Start the server. Query the API and assert status is `draft`.

**Acceptance criteria:**

- [ ] Both cases pass.
- [ ] The startup scan handles these before the watcher is fully operational (tests query immediately after `newTestEnv` returns).

---

## Milestone 6 — Integration test: API write still triggers auto-block

**Description:** Verify that the existing API PUT path still results in auto-block after the handler-level logic is removed (backend plan milestone 5).

**Files to change:**

- `tests/integration/artifact_blocked_questions_test.go` — update or extend existing tests to verify:
  1. PUT an artifact with `## Open Questions` content → response shows `blocked` status.
  2. PUT the same artifact with the questions removed → response shows `draft` status.
  3. No duplicate events in the feed for a single PUT.

**Acceptance criteria:**

- [ ] Existing tests continue to pass after the handler refactor.
- [ ] The auto-block path via the indexer produces the same observable API behaviour as the old handler path.

---

## Milestone 7 — Integration test: manual transitions are not broken

**Description:** Verify that the auto-block feature does not interfere with manual status transitions via the API.

**Files to change:**

- `tests/integration/autoblock_watcher_test.go` (or a new file) — test case:
  1. Create a `draft` artifact without open questions.
  2. Manually transition to `blocked` via `POST /artifacts/:path/transition` as product-owner.
  3. Assert status is `blocked`.
  4. Manually transition back to `draft`.
  5. Assert status is `draft` and no auto-unblock interference (the artifact has no open questions, so auto-unblock should be a no-op in step 3, but verify it doesn't fire a conflicting event).

**Acceptance criteria:**

- [ ] Manual block/unblock works independently of the auto-block logic.
- [ ] No spurious events from the auto-block path during manual transitions.

---

## Milestone 8 — WebSocket event verification

**Description:** Verify that `artifact.indexed` and `feed.new` WebSocket events are broadcast for every automatic status change.

**Files to change:**

- `tests/integration/autoblock_watcher_test.go` — extend the watcher test (milestone 4) to:
  1. Register a WebSocket listener on the hub before the file write.
  2. After auto-block, assert that an `artifact.indexed` event was received with `blocked_reason`.
  3. After auto-unblock, assert that an `artifact.indexed` event was received.
  4. Assert a `feed.new` event was received for each transition.

**Acceptance criteria:**

- [ ] Both `artifact.indexed` and `feed.new` events are captured for auto-block.
- [ ] Both events are captured for auto-unblock.
- [ ] No events are emitted for idempotent re-indexes.
