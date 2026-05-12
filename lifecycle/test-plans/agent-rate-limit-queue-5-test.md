---
title: "Test Plan — Agent Work Queue with Rate-Limit Auto-Pause"
type: plan-test
status: done
lineage: agent-rate-limit-queue
parent: lifecycle/requirements/agent-rate-limit-queue-2.md
created: "2026-05-12T10:50:00+10:00"
priority: high
labels:
    - agent
    - queue
    - test
    - release-blocker
release: KC-Release1
---

# Test Plan — Agent Work Queue with Rate-Limit Auto-Pause

Covers the acceptance criteria in [[agent-rate-limit-queue-2]]. Three
suites:

1. Go unit tests for the parser and Store.
2. Go integration tests for the dispatcher + HTTP API + WS broadcast.
3. Vitest unit / component tests for the SPA pieces.

A trailing manual smoke script covers the bits that can't be
mechanised cheaply.

---

## Suite 1 — Go unit tests

### 1.1 Reset-time parser (`internal/queue/parser_test.go`)

| # | Input | Expected |
|---|---|---|
| P1 | `"You're out of extra usage · resets 8pm (Australia/Brisbane)"` | next 20:00 Australia/Brisbane |
| P2 | `"resets 10:30am (America/New_York)"` | next 10:30 America/New_York |
| P3 | `"resets 8pm"` (no TZ) | next 20:00 in server's local TZ |
| P4 | `"resets at 2026-05-13T08:00:00Z"` | exact 2026-05-13T08:00:00Z |
| P5 | `"retry after 60 seconds"` | now + 60s |
| P6 | `"retry after 5 minutes"` | now + 5 min |
| P7 | `""` | `(zero, false)` |
| P8 | `"resets soon"` | `(zero, false)` |
| P9 | `"resets 25pm (Australia/Brisbane)"` | `(zero, false)` |
| P10 | `"resets 9am (TZ/Made-Up)"` | `(zero, false)` |

Plus property tests:

- **PT1** — for any (hour ∈ [0,23], minute ∈ {0,30}, loc ∈ supported TZs),
  `ParseResetTime(formatted, now)` returns a time strictly after `now`.
- **PT2** — `next 20:00 in TZ X` always falls on either today or tomorrow
  in TZ X (never further).

### 1.2 Store (`internal/queue/store_test.go`)

| # | Test | Description |
|---|---|---|
| S1 | `TestStore_EnqueueDequeueFIFO` | Enqueue three jobs, dequeue three times, assert order matches enqueue order. |
| S2 | `TestStore_MarkTerminal` | Dequeue, mark completed, GetByID returns state=completed, finished_at set. |
| S3 | `TestStore_DuplicateRejected` | `FindActiveByPath` returns the active job; transactional insert prevents two pending rows for the same `(project, path)`. |
| S4 | `TestStore_OrphanRecoveryOnReopen` | Insert a `running` row, close+reopen store, `RecoverOrphans()` moves it back to pending at the head with `attempts++`. |
| S5 | `TestStore_PauseStateRoundTrip` | `SetPauseState(true, t, "rate_limit")` followed by `GetPauseState()` returns the same values. |
| S6 | `TestStore_CancelPendingOnly` | Cancel on a `pending` job succeeds; cancel on a `running` job returns `ErrCannotCancelRunning`. |
| S7 | `TestStore_HeadInsert` | Re-enqueue at head sets a position smaller than the current min. |

### 1.3 Dispatcher state machine (`internal/queue/dispatcher_test.go`)

Uses a mock `agent.Manager` and a mock clock injected via `clockFn`.

| # | Test | Description |
|---|---|---|
| D1 | `TestDispatcher_SerialExecution` | Enqueue 3 jobs, drive the tick loop, assert exactly one is in `running` at any tick. |
| D2 | `TestDispatcher_SkipOnStatusMismatch` | Enqueue an approved artefact, then change its status to `in-development` between enqueue and tick. The tick marks the job `skipped` with reason `status_changed_to:in-development` and proceeds. |
| D3 | `TestDispatcher_RateLimitFlow` | Mock manager emits `StreamEvent{Type: "rate_limit", RawText: "<sample>"}`; dispatcher: (a) marks the running job `failed/rate_limit`, (b) re-enqueues at head with `attempts=2`, (c) calls `SetPauseState(true, …)`, (d) broadcasts `queue.paused`. |
| D4 | `TestDispatcher_AutoResume` | Set `paused_until` to clock+1s; advance clock; tick clears the paused state, broadcasts `queue.resumed`, dequeues the head. |
| D5 | `TestDispatcher_ManualPauseStaysPaused` | Pause manually; advance clock by 10h; assert dispatcher does not dequeue. |
| D6 | `TestDispatcher_MaxAttemptsCap` | Force a third rate-limit on a job already at `attempts=3`; dispatcher does NOT re-enqueue; broadcasts `queue.skipped/max_attempts`. |
| D7 | `TestDispatcher_FallbackOnUnparseableReset` | Mock manager emits a rate-limit event with `"resets soon"` text; dispatcher pauses for `cfg.FallbackPause`. |

---

## Suite 2 — Go integration tests (`tests/integration/`)

Each test uses `newTestEnvWithCfgYAML` to spin a full server with a
configured agent. The integration tests use a fake `claude` binary
(via the existing `setupFakeClaude` helper extended where needed) so
runs complete in milliseconds.

### 2.1 `queue_api_test.go`

| # | Test | Description |
|---|---|---|
| Q1 | `TestQueue_Enqueue_AuthorizedRole` | admin enqueues backend-developer job → 201, position 1. |
| Q2 | `TestQueue_Enqueue_ForbiddenRole` | qa attempts to enqueue backend-developer job → 403. |
| Q3 | `TestQueue_Enqueue_NonApprovedArtifact` | Enqueue an artefact in `draft` → 400 with `not_approved`. |
| Q4 | `TestQueue_Enqueue_DuplicateRejected` | Enqueue same path twice → 409 `already_queued`. |
| Q5 | `TestQueue_Enqueue_NoMatchingAgent` | Enqueue an artefact of type `release` (no source-type match) → 400 `no_agent_for_type`. |
| Q6 | `TestQueue_ListQueue_AnyUser` | qa can GET `/queue` even though they can't enqueue → 200. |
| Q7 | `TestQueue_Cancel_Pending` | Enqueue, cancel before dispatcher picks up → 204, job moves to `cancelled`. |
| Q8 | `TestQueue_Cancel_Running` | Cancel a running job → 409 `cannot_cancel_running`. |
| Q9 | `TestQueue_Pause_AdminOnly` | qa attempts pause → 403; admin pauses → 204. |

### 2.2 `queue_dispatch_test.go`

Full enqueue → dispatch → completion cycle with the fake claude binary.

| # | Test | Description |
|---|---|---|
| QD1 | `TestQueue_HappyPath_SingleProject` | Enqueue 3 jobs in one project; assert all three run sequentially and reach `completed`; assert WS events `added → started → finished` for each. |
| QD2 | `TestQueue_HappyPath_MultiProject` | Enqueue 1 job in each of 2 projects; assert serial execution across projects (no overlap in `started_at` / `finished_at` intervals). |
| QD3 | `TestQueue_ManualLaunchCoexists` | Enqueue 1 queue job and start 1 manual agent run via the existing `POST /agents/:name/run` against a different lineage; assert both run to completion without the queue waiting for the manual one. |
| QD4 | `TestQueue_StatusChangedSkip` | Enqueue an approved artefact. Before dispatch, PUT a status change to `in-development`. Assert the dispatcher emits `queue.skipped` with reason `status_changed_to:in-development` and proceeds. |
| QD5 | `TestQueue_PersistsAcrossRestart` | Enqueue 3 jobs, stop the server, restart, assert all 3 still pending in original order. Bonus: stop the server *while one is running*, restart, assert the running job is back at the head with `attempts=2` and `restart_recovered=true`. |

### 2.3 `queue_rate_limit_test.go`

| # | Test | Description |
|---|---|---|
| QR1 | `TestQueue_RateLimit_FromSampleLog` | Replay the captured stream-json event from `runs/22de65f53d82bf14.log` into the fake claude binary's stdout. Assert: (a) the running job is marked `failed/rate_limit`, (b) a fresh `pending` row exists with the same `(project, path, agent)` at position smaller than any other pending row, with `attempts=2`, (c) `queue_state.paused = true`, `paused_until` equals `next 20:00 Australia/Brisbane + 5 min`, (d) `queue.paused` WS event was broadcast with the parsed reset_time. |
| QR2 | `TestQueue_RateLimit_AutoResume` | Same setup as QR1; advance the dispatcher's injected clock to `paused_until + 1ms`; assert `queue.resumed` WS event and the head job starts. |
| QR3 | `TestQueue_RateLimit_FallbackOnUnparseable` | Inject a rate-limit event with `"resets soon"` text; assert `paused_until = now + cfg.FallbackPause` and a WARN log line containing the raw text. |
| QR4 | `TestQueue_RateLimit_MaxAttempts` | Force 3 consecutive rate-limit failures on the same artefact; assert the 4th does NOT re-enqueue, and a `queue.skipped/max_attempts` event is broadcast. |

---

## Suite 3 — Frontend (Vitest)

### 3.1 `tests/web/QueueWorkButton.test.ts`

| # | Test | Description |
|---|---|---|
| FB1 | `renders when artifact is approved and agent matches` | An idea in `approved` shows the button with `agent=requirements-analyst`. |
| FB2 | `hides when status is not approved` | Same artefact in `draft` → no button. |
| FB3 | `hides when no agent matches the type` | An artefact of type `release` → no button. |
| FB4 | `defect falls back to assignee role` | A defect with `assignees: [{ role: backend-developer }]` shows the button targeting backend-developer. |
| FB5 | `click calls queueStore.enqueue` | Click → `enqueue` called with correct args; button enters loading state. |
| FB6 | `replaces button with "Queued — position N" badge` | When store snapshot lists this artefact pending at position 2, the button is replaced. |

### 3.2 `tests/web/QueueView.test.ts`

| # | Test | Description |
|---|---|---|
| FV1 | `renders running + pending + recent sections` | All three tables populated from a mocked snapshot. |
| FV2 | `empty running shows empty state` | `running === null` → "Nothing running" placeholder. |
| FV3 | `pause banner only when paused` | `paused === false` → banner hidden; `true` → banner visible with reset time. |
| FV4 | `Resume now visible only for product-owner / devops` | Mocked auth store with role qa → button hidden; role product-owner → visible. |
| FV5 | `Remove on a pending row calls queueStore.cancel` | Click → `cancel(id)` called. |

### 3.3 `tests/web/queueStore.test.ts`

| # | Test | Description |
|---|---|---|
| FS1 | `queue.added pushes to pending` | WS event → `snapshot.pending` grows by 1, position respected. |
| FS2 | `queue.started moves to running` | WS event → moves the matching pending item to `running`. |
| FS3 | `queue.finished moves to recent` | Moves running to recent, capped at 10. |
| FS4 | `queue.paused sets paused state` | `paused = true, paused_until = <RFC3339>`. |
| FS5 | `queue.resumed clears paused state` | `paused = false, paused_until = null`. |
| FS6 | `queue.cancelled removes from pending` | Match by id, drop from pending. |
| FS7 | `initial fetch sets full snapshot` | `fetch()` populates state from REST. |

### 3.4 `tests/web/AppHeaderQueueBadge.test.ts`

| # | Test | Description |
|---|---|---|
| FH1 | `renders pending count` | 3 pending → badge shows "3". |
| FH2 | `paused state shows pause icon` | `paused === true` → pause icon overlay rendered. |
| FH3 | `click navigates to /queue` | Mock router; click → `router.push({ name: 'queue' })`. |

---

## Manual smoke (not automated)

Run after all three suites pass.

1. `make all && make run` against the kaos-control project itself.
2. Log in as `keith@sinclair.org.au` (product-owner).
3. Approve two ideas. Click "Queue Work" on both.
4. Open `/queue` — expect: running = first idea, pending = second idea
   at position 1.
5. Wait for both runs to complete. Expect both in "Recently finished".
6. Hit the manual pause button. Approve a third idea. Click Queue
   Work. Expect: pending = 1, dispatcher does NOT start it.
7. Click Resume. Expect run starts within 1 s.
8. Hard reload the browser. Verify the header badge and `/queue` view
   reflect the live state without a manual refresh of any component.

### Optional adversarial smoke

If you have access to an account that's at or near the Claude weekly
cap:

- Trigger a real rate-limit by queuing a large-context job. Verify
  the UI banner appears with the actual reset time. Confirm the
  failed job is re-enqueued at the head. Wait or fast-forward time
  to verify auto-resume.

If you don't have that access, the QR1–QR4 tests in Suite 2 fully
cover this path with a replay of the captured sample event.
