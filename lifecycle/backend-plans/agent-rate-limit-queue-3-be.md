---
title: "Backend Plan — Agent Work Queue with Rate-Limit Auto-Pause"
type: plan-backend
status: in-development
lineage: agent-rate-limit-queue
parent: lifecycle/requirements/agent-rate-limit-queue-2.md
created: "2026-05-12T10:40:00+10:00"
priority: high
labels:
    - agent
    - queue
    - backend
    - release-blocker
release: KC-Release1
---

# Backend Plan — Agent Work Queue with Rate-Limit Auto-Pause

Implements [[agent-rate-limit-queue-2]]. New package `internal/queue/`
holding the store, the dispatcher, and the rate-limit parser; HTTP
handlers wired into the existing chi router; WS events on the existing
hub.

## Package structure

```
internal/queue/
├── store.go         SQLite-backed queue Store (CRUD)
├── store_test.go    Store unit tests
├── dispatcher.go    Single-goroutine worker loop
├── dispatcher_test.go
├── parser.go        Rate-limit reset-time parser
├── parser_test.go   Format-coverage unit tests
└── types.go         Job, JobState, JobReason
```

`Store` and `Dispatcher` are wired up in `cmd/kaos-control/main.go`
alongside the existing scheduler. The dispatcher gets a reference to
`*agent.Manager` to start runs, a `*hub.Hub` to broadcast events, and
a `time.Now` injection point for tests.

---

## Milestone 1 — Queue persistence (Store)

### Description

A SQLite-backed queue table living in `~/.kaos-control/data/queue.db`.
Schema:

```sql
CREATE TABLE jobs (
  id            TEXT PRIMARY KEY,         -- 16-hex random
  project       TEXT NOT NULL,
  artifact_path TEXT NOT NULL,
  agent_name    TEXT NOT NULL,
  state         TEXT NOT NULL CHECK(state IN ('pending','running','completed','failed','skipped','cancelled')),
  reason        TEXT,
  attempts      INTEGER NOT NULL DEFAULT 1,
  enqueued_at   INTEGER NOT NULL,         -- unix seconds
  started_at    INTEGER,
  finished_at   INTEGER,
  position      INTEGER NOT NULL,         -- FIFO ordering; lower = earlier
  enqueued_by   TEXT NOT NULL             -- user email
);

CREATE INDEX idx_jobs_state_position ON jobs(state, position);
CREATE INDEX idx_jobs_project_path ON jobs(project, artifact_path);

CREATE TABLE queue_state (
  k TEXT PRIMARY KEY,                     -- 'paused', 'paused_until', 'pause_reason'
  v TEXT NOT NULL
);
```

`position` is monotonic-increasing on enqueue; on dispatch we
`SELECT … ORDER BY position ASC LIMIT 1`.

### Files to change

- **New** `internal/queue/types.go` — defines `Job`, `JobState`, `JobReason` enums.
- **New** `internal/queue/store.go`:
  - `func Open(path string) (*Store, error)` — open DB, apply schema.
  - `Enqueue(j Job) error`
  - `Dequeue() (*Job, error)` — picks head of pending FIFO, marks running.
  - `MarkTerminal(id string, state JobState, reason string) error`
  - `ListByState(states ...JobState) ([]*Job, error)`
  - `GetByID(id string) (*Job, error)`
  - `FindActiveByPath(project, path string) (*Job, error)` — for FR3 duplicate suppression.
  - `Cancel(id string) error` — only pending; running rejected with ErrCannotCancelRunning.
  - `RecoverOrphans() error` — at startup: any `running` rows become `pending` with `attempts++`.
  - `GetPauseState() (paused bool, until time.Time, reason string, _ error)`
  - `SetPauseState(paused bool, until time.Time, reason string) error`

### Acceptance criteria

- `go build ./...` clean; `go vet` clean.
- Unit tests in `store_test.go` cover: enqueue/dequeue order, terminal
  transitions, duplicate detection, orphan recovery on reopen, pause
  state round-trip.

---

## Milestone 2 — Rate-limit reset-time parser

### Description

Pure function `parser.ParseResetTime(text string, now time.Time) (time.Time, bool)`
that handles the FR9 formats. The "bool" return signals successful parse;
on false the dispatcher falls back to `now + fallback_pause_minutes`.

### Files to change

- **New** `internal/queue/parser.go`:
  ```go
  // ParseResetTime extracts a reset time from a Claude rate-limit text
  // payload. Returns (resetTime, true) on success or (time.Time{}, false)
  // on unrecognised format.
  func ParseResetTime(text string, now time.Time) (time.Time, bool)
  ```
  Internal patterns to try in order:
  1. ISO 8601 timestamp via `time.Parse(time.RFC3339, ...)`.
  2. `retry after N seconds` / `retry after N minutes` → `now + N`.
  3. `resets HH:MMam/pm (TZ)` or `resets HHam/pm (TZ)` — extract hour
     (and optional minute), parse TZ via `time.LoadLocation`, compute
     next occurrence of that wall time.
  4. `resets HH:MMam/pm` / `resets HHam/pm` — same but assume the
     server's local TZ.

  Helper `nextOccurrence(hour, minute int, loc *time.Location, now time.Time) time.Time`
  returns today's occurrence if it's in the future, otherwise tomorrow's.

### Acceptance criteria

- Unit tests in `parser_test.go` covering every row of the FR9 table
  PLUS the following malformed inputs returning `false`:
  - `"resets soon"` (no time)
  - `"resets 25pm (Australia/Brisbane)"` (invalid hour)
  - `"resets at TZ/Made-Up"` (invalid timezone)
  - `""` (empty)
- Bonus: a property test that for any valid hour/TZ pair, the parsed
  result is strictly after `now`.

---

## Milestone 3 — Dispatcher loop

### Description

A single goroutine spawned at server start. Pseudocode:

```go
for {
    select {
    case <-ctx.Done():
        return
    case <-tick.C:               // every 1s
        if dispatcher.paused() { continue }
        job, err := store.Dequeue()
        if err != nil || job == nil { continue }
        broadcastQueueStarted(job)
        run := agentMgr.StartRun(...)
        // wait for run completion via channel from supervisor
        result := <-runDone
        switch result.kind {
        case "completed":
            store.MarkTerminal(job.id, Completed, "")
        case "failed_rate_limit":
            handleRateLimit(job, result)   // M4 / M5
        case "failed":
            store.MarkTerminal(job.id, Failed, result.reason)
        }
        broadcastQueueFinished(job)
    }
}
```

The FR7 skip-on-status-mismatch check happens inside Dequeue's caller:
between Dequeue and StartRun, re-read the artefact from the index and
verify status == "approved"; if not, MarkTerminal(skipped, "status_changed_to:<s>")
and `continue`.

### Files to change

- **New** `internal/queue/dispatcher.go`:
  - `type Dispatcher struct { … }`
  - `func New(store *Store, agentMgr *agent.Manager, hub *hub.Hub, cfg Config) *Dispatcher`
  - `func (d *Dispatcher) Start(ctx context.Context)` — spawns the goroutine.
  - `func (d *Dispatcher) Pause(reason string)`
  - `func (d *Dispatcher) Resume()` — manual; clears `paused_until`.
  - `func (d *Dispatcher) handleRateLimit(job *Job, rawText string)` — see M4.

- **Edit** `cmd/kaos-control/main.go`:
  - Open the queue store after the auth store.
  - Construct the dispatcher and call `dispatcher.Start(ctx)` after the
    HTTP server is listening.
  - Inject the dispatcher into the HTTP `Server` via a new
    `ServerConfig.Queue *queue.Dispatcher` field.

### Acceptance criteria

- Integration test that enqueues 3 jobs across 3 projects, with the
  agent supervisor mocked to "complete immediately"; verifies all 3
  run sequentially (their `started_at` timestamps don't overlap) and
  reach `completed` state.
- FR6 manual-launch coexistence: integration test launches one queue
  job and one manual run targeting different lineages, both run to
  completion without the queue waiting for the manual one.

---

## Milestone 4 — Rate-limit detection in stream

### Description

The agent supervisor at `internal/agent/agent.go` reads stream-json
events line-by-line. Add a hook for the dispatcher to inspect each
event for `"error":"rate_limit"`. Cleanest split:

- The supervisor's read loop emits a typed event to a per-run channel.
- A new event variant `StreamEvent{Type: "rate_limit", RawText: "<content text>"}`
  is published when an event with `"error":"rate_limit"` is seen.
- The dispatcher, which already waits on the run-done channel from M3,
  also subscribes to this stream-event channel; on `rate_limit` it
  records the event and lets the run finish (the run will exit shortly
  after), then calls `handleRateLimit`.

### Files to change

- **Edit** `internal/agent/agent.go`:
  - In the stream reader (around line 189), after JSON-unmarshalling
    each event, check for top-level `error == "rate_limit"`. Extract
    the human-readable text from `message.content[0].text`. Send
    `StreamEvent{Type: "rate_limit", RawText: text}` on a channel
    that's already plumbed up to the caller, or add one if not.
  - Surface the channel via the `RunHandle` returned by `StartRun`.

- **Edit** `internal/queue/dispatcher.go`:
  - In the run-execution path, also select on the stream-event channel.
    On `rate_limit`, store the raw text on the job; when the run
    terminates, route to `handleRateLimit` instead of the normal
    `failed` branch.

### `handleRateLimit` implementation

```go
func (d *Dispatcher) handleRateLimit(job *Job, rawText string) {
    resetTime, ok := ParseResetTime(rawText, time.Now())
    if !ok {
        resetTime = time.Now().Add(d.cfg.FallbackPause)
    }
    pausedUntil := resetTime.Add(d.cfg.ResumeGrace)

    // 1. Mark current job failed.
    d.store.MarkTerminal(job.ID, Failed, "rate_limit")

    // 2. Re-enqueue at head (smallest position).
    requeue := *job
    requeue.ID = newID()
    requeue.State = Pending
    requeue.Attempts = job.Attempts + 1
    requeue.Position = d.store.MinPosition() - 1   // head of queue
    requeue.EnqueuedAt = time.Now().Unix()
    if requeue.Attempts > d.cfg.MaxAttempts {
        // Don't re-enqueue past max-attempts.
        d.broadcast("queue.skipped", map[string]any{"id": job.ID, "reason": "max_attempts"})
        return
    }
    d.store.Enqueue(requeue)

    // 3. Pause.
    d.store.SetPauseState(true, pausedUntil, "rate_limit")
    d.setPausedUntil(pausedUntil)
    d.broadcast("queue.paused", map[string]any{
        "paused_until": pausedUntil.Format(time.RFC3339),
        "reset_time":   resetTime.Format(time.RFC3339),
        "raw_text":     rawText,
    })
}
```

### Acceptance criteria

- Unit test that feeds the captured stream event from
  `~/.kaos-control/data/kaos-control/runs/22de65f53d82bf14.log` into the
  parser path and asserts `paused_until` equals
  `next 20:00 Australia/Brisbane + 5 minutes`.
- Integration test with a fake claude binary that emits the rate-limit
  event, asserts the queue transitions to paused, the failed job is
  re-enqueued at the head with `attempts=2`, and `queue.paused` is
  broadcast.

---

## Milestone 5 — Pause / resume + auto-resume

### Description

The dispatcher tick must check `paused_until` before dequeueing. If
`now >= paused_until`, transition out of paused (clear the flag, set
`paused_until = zero`), broadcast `queue.resumed`, and proceed.

Manual `Pause(reason)` sets `paused = true, paused_until = zero` —
the auto-resume condition never fires, so it stays paused until
`Resume()` is called.

### Files to change

- **Edit** `internal/queue/dispatcher.go` — `paused()` predicate, tick
  loop branch.

### Acceptance criteria

- Auto-resume after grace: with a mocked clock, advance to
  `paused_until + 1ms`; verify the dispatcher leaves paused and starts
  the head job.
- Manual pause/resume: verify a manually paused queue stays paused
  forever and resumes only on explicit `Resume()`.

---

## Milestone 6 — HTTP API

### Description

Wire the new endpoints into the existing chi router. All routes live
under `/api/queue` (app-level, not project-scoped) since the queue
spans projects.

### Endpoints

| Method | Path | Handler | Roles |
|---|---|---|---|
| `POST` | `/api/queue` | `handleEnqueue` | role required for the target agent |
| `GET` | `/api/queue` | `handleListQueue` | any authenticated user |
| `DELETE` | `/api/queue/{id}` | `handleCancelQueue` | enqueuer or product-owner |
| `POST` | `/api/queue/pause` | `handlePauseQueue` | product-owner or devops |
| `POST` | `/api/queue/resume` | `handleResumeQueue` | product-owner or devops |

Request / response shapes:

```jsonc
// POST /api/queue
// Request:
{ "project": "kaos-control", "artifact_path": "lifecycle/ideas/foo.md", "agent": "requirements-analyst" }
// Response 201:
{ "id": "ab12cd34ef56gh78", "position": 3 }

// GET /api/queue
// Response 200:
{
  "running":  null | Job,
  "pending":  [Job, …],
  "recent":   [Job, …],          // last 10 terminal
  "paused":   true|false,
  "paused_until": "RFC3339|null",
  "pause_reason": "rate_limit|manual|null"
}
```

### Files to change

- **New** `internal/http/queue.go` with the five handlers and a
  `Server.handleEnqueue` that reuses the existing agent-role predicate
  from `agents.go:handleStartAgentRun`.
- **Edit** `internal/http/server.go` to mount the routes.
- **Edit** `internal/http/permissions.go` (introduced by
  `auth-role-checks-mutations-3-be`) — no new constants needed; the
  enqueue handler computes its allowed-role list from the agent
  config the same way `handleStartAgentRun` does.

### Acceptance criteria

- Integration tests in `tests/integration/queue_api_test.go`:
  - `TestEnqueue_Authorized` — admin can enqueue any agent.
  - `TestEnqueue_Forbidden` — qa cannot enqueue a backend-developer run.
  - `TestEnqueue_DuplicateRejected` — same artefact twice → 409.
  - `TestCancelQueue_PendingOnly` — running job → 409.
  - `TestPauseResume_AdminOnly` — qa gets 403; admin gets 204.

---

## Milestone 7 — WebSocket events

### Description

Broadcast on the existing per-project hub for project-scoped events
AND on a new app-level hub channel for queue-level events (pause
state, list snapshots). The frontend subscribes to both.

### Files to change

- **Edit** `internal/hub/` — confirm app-level broadcast is supported
  (it should be; otherwise add a global channel).
- **Edit** `internal/queue/dispatcher.go` — broadcast helpers for each
  state transition listed in NFR2.

### Acceptance criteria

- Integration test connects a WS, enqueues a job, asserts the order
  of events: `queue.added → queue.started → queue.finished`.

---

## Verification (end-to-end)

1. `make lint` clean.
2. `make test-unit` clean.
3. `make test-integration` clean, including new queue test files.
4. Manual smoke:
   - Configure two projects.
   - Enqueue one artefact in each.
   - Verify second job waits for first.
   - Inject a fake rate-limit event via a test pipe; verify queue
     pauses, banner shows reset time, head job is re-enqueued.
   - Advance the clock; verify auto-resume.

## Risk notes

- **Concurrent enqueue.** If two users click Queue Work for the same
  artefact within milliseconds, the `FindActiveByPath` check + Enqueue
  must be transactional. Implement as a single `INSERT … WHERE NOT
  EXISTS (SELECT 1 FROM jobs WHERE project=? AND artifact_path=? AND
  state IN ('pending','running'))` and rely on the affected-rows count.

- **Clock skew.** `paused_until` is server-time, not wall-clock-from-
  the-error-message-source. Tests that mock the clock use a
  `clockFn func() time.Time` injection point on the Dispatcher.

- **Parser drift.** Anthropic may change the rate-limit message
  wording without notice. The `fallback_pause_minutes` knob keeps us
  safe; the WARN log of the raw text gives us a paper trail to extend
  the parser. Document this in the parser test file's header comment.
