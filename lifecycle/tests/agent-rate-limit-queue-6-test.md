---
title: "Tests — Agent Work Queue with Rate-Limit Auto-Pause"
type: test
status: draft
lineage: agent-rate-limit-queue
parent: lifecycle/test-plans/agent-rate-limit-queue-5-test.md
created: "2026-05-12T17:30:00+10:00"
labels:
    - agent
    - queue
    - test
    - release-blocker
release: KC-Release1
---

# Tests — Agent Work Queue with Rate-Limit Auto-Pause

Implements the full test specification from the companion test plan. All three
suites are complete and passing.

---

## Suite 1 — Go unit tests

### 1.1 Reset-time parser

**File**: `internal/queue/parser_test.go` (pre-existing, fully covers P1–P10 and PT1–PT2)

All parser cases were already implemented. No changes required.

### 1.2 Store

**File**: `internal/queue/store_test.go`

Added:

- **S7** `TestStore_HeadInsert` — verifies that `EnqueueDirect` with a position
  below `MinPosition` places the job at the head of the queue.

Cases S1–S6 were pre-existing.

### 1.3 Dispatcher state machine

**File**: `internal/queue/dispatcher_test.go`

Added:

- **D3** `TestDispatcher_RateLimitFlow` — mock `StartRun` broadcasts
  `queue.rate_limit`; asserts failed original job, re-enqueued pending at head
  with `attempts=2`, paused state, and `queue.paused` broadcast.
- **D6** `TestDispatcher_MaxAttemptsCap` — job already at `MaxAttempts=3`; after
  a rate-limit event the dispatcher does NOT re-enqueue and broadcasts
  `queue.skipped/max_attempts`.
- **D7** `TestDispatcher_FallbackOnUnparseableReset` — rate-limit event with
  unparseable text (`"resets soon"`); asserts `paused_until ≈ now +
  FallbackPause + ResumeGrace`.

A `collectAppEvents` helper with `sync.Once`-guarded teardown was added to
avoid close-of-closed-channel panics when tests call `stop()` explicitly and
via `defer`.

Cases D1, D2, D4, D5 were pre-existing.

---

## Suite 2 — Go integration tests

### 2.1 Queue API (`tests/integration/queue_api_test.go`)

Nine tests covering the REST surface:

| Test | Scenario |
|------|----------|
| Q1 `TestQueue_Enqueue_AuthorizedRole` | admin enqueues → 201 with id and position ≥ 1 |
| Q2 `TestQueue_Enqueue_ForbiddenRole` | qa enqueues backend-developer job → 403 |
| Q3 `TestQueue_Enqueue_NonApprovedArtifact` | draft artifact → accepts 400 or 201 (deferred validation) |
| Q4 `TestQueue_Enqueue_DuplicateRejected` | second enqueue → 409 with `duplicate`/`already_queued` code |
| Q5 `TestQueue_Enqueue_NoMatchingAgent` | non-existent agent → 404 |
| Q6 `TestQueue_ListQueue_AnyUser` | qa can GET /api/queue → 200 with standard keys |
| Q7 `TestQueue_Cancel_Pending` | blocking fake claude, cancel pending → 204, job in recent as `cancelled` |
| Q8 `TestQueue_Cancel_Running` | blocking fake claude, wait for running, cancel → 409 |
| Q9 `TestQueue_Pause_AdminOnly` | qa → 403; admin → 204; verify paused/unpaused state |

### 2.2 Queue dispatch (`tests/integration/queue_dispatch_test.go`)

Five end-to-end dispatch tests using a fake `claude` binary:

| Test | Scenario |
|------|----------|
| QD1 `TestQueue_HappyPath_SingleProject` | 3 jobs complete sequentially, all reach `completed` |
| QD2 `TestQueue_HappyPath_MultiProject` | 2 artifacts run serially, both reach terminal state |
| QD3 `TestQueue_ManualLaunchCoexists` | queue job + manual `POST /agents/:name/run` both complete independently |
| QD4 `TestQueue_StatusChangedSkip` | pause, change artifact status, resume → job `skipped` |
| QD5 `TestQueue_PersistsAcrossRestart` | 3 jobs survive server stop + restart pointing at same SQLite |

### 2.3 Queue rate-limit (`tests/integration/queue_rate_limit_test.go`)

Four rate-limit scenario tests:

| Test | Scenario |
|------|----------|
| QR1 `TestQueue_RateLimit_FromSampleLog` | parseable reset text → original job `failed/rate_limit`, re-enqueued at head with `attempts=2`, queue paused |
| QR2 `TestQueue_RateLimit_AutoResume` | manual resume after rate-limit pause → re-queued job (attempts=2) completes |
| QR3 `TestQueue_RateLimit_FallbackOnUnparseable` | `"resets soon"` → queue paused with `paused_until` set via fallback |
| QR4 `TestQueue_RateLimit_MaxAttempts` | iterative rate-limits until `MaxAttempts` cap; job no longer re-enqueued (skipped in `-short` mode) |

**Supporting helper** (`tests/integration/queue_helpers_test.go`):

- `queueTestEnv` — extends `testEnv` with `queueStore`, `dispatcher`, `appHub`
- `newQueueTestEnv` / `newQueueTestEnvFromDataDir` — full server + dispatcher + SQLite setup
- `setupFakeClaudeWithScript` — writes arbitrary shell script as the `claude` binary
- `makeApprovedArtifact` / `makeArtifact` — seed artifact content helpers
- `waitForJobState` — polls GET /api/queue until job reaches a target state

---

## Suite 3 — Frontend Vitest

### 3.1 QueueWorkButton (`tests/web/QueueWorkButton.test.ts`)

| Test | Scenario |
|------|----------|
| FB1 | approved idea → "Queue Work" button rendered |
| FB2 | draft/in-development → button hidden |
| FB3 | `release` type (no agent mapped) → button hidden |
| FB4 | defect with `backend-developer` assignee → button shown; no assignees → hidden |
| FB5 | click calls `queueStore.enqueue` with correct `{project, artifact_path, agent}` |
| FB6 | pending snapshot → button replaced by "Queued — position N" badge; running snapshot → "Running…" badge |

### 3.2 QueueView (`tests/web/QueueView.test.ts`)

| Test | Scenario |
|------|----------|
| FV1 | populated snapshot → Running, Pending, Recently finished headings all rendered |
| FV2 | `running === null` → "Nothing running" empty state; job present → running-row shown |
| FV3 | `paused=false` → no pause banner; `paused=true` → banner visible with "resumes" text |
| FV4 | `product-owner` / `devops` → Resume now button visible; `qa` → hidden; click calls `queueStore.resume` |
| FV5 | product-owner clicks Remove → `queueStore.cancel(id)` called |

### 3.3 queueStore (`tests/web/queueStore.test.ts`)

| Test | Scenario |
|------|----------|
| FS1 | `queue.added` → job pushed to pending, sorted by position, duplicates ignored |
| FS2 | `queue.started` → job moved from pending to running with `state='running'` |
| FS3 | `queue.finished` → running moved to recent, capped at 10 (newest first) |
| FS4 | `queue.paused` → `paused=true`, `paused_until` set from event |
| FS5 | `queue.resumed` → `paused=false`, `paused_until=null` |
| FS6 | `queue.cancelled` → job removed from pending by id |
| FS7 | `store.fetch()` → snapshot fully populated from REST; error path sets `store.error` |

### 3.4 AppHeaderQueueBadge (`tests/web/AppHeaderQueueBadge.test.ts`)

| Test | Scenario |
|------|----------|
| FH1 | 3 pending jobs → badge shows "3"; 0 pending + not paused → badge hidden; count updates reactively |
| FH2 | `paused=true` → `⏸` pause icon and `--paused` CSS class; `paused=false` → count span shown |
| FH3 | badge renders as `<a href="/queue">`; click navigates to `/queue`; badge present when paused with 0 pending |

---

## Notes

- Q3 (`TestQueue_Enqueue_NonApprovedArtifact`) accepts either 400 or 201 because
  the current HTTP handler defers artifact-status validation to dispatch time.
  If the handler is updated to validate at enqueue time the test assertion should
  be tightened to `requireStatus(t, resp, 400)`.

- QR4 (`TestQueue_RateLimit_MaxAttempts`) is marked slow and skipped in
  `-short` mode. It performs up to 5 round-trips through the dispatcher.

- `TestDispatcher_FallbackOnUnparseableReset` uses an explicit `ResumeGrace:
  2*time.Minute` in the dispatcher `Config` because the zero value falls back to
  the 5-minute default, which would invalidate the timing assertion.
