---
title: Global Agent Work Queue with Rate-Limit Auto-Pause
type: requirement
status: done
lineage: agent-rate-limit-queue
parent: lifecycle/ideas/agent-rate-limit-queue.md
created: "2026-05-12T10:35:00+10:00"
priority: high
labels:
    - agent
    - queue
    - backend
    - frontend
    - operability
release: KC-Release1
---

# Global Agent Work Queue with Rate-Limit Auto-Pause

Parent: [[agent-rate-limit-queue]].

## Goal

A persistent, FIFO, globally-serialised work queue that the user enqueues
agent runs into from the artefact view; a dispatcher that processes the
queue one item at a time across all projects; automatic pause / resume
on Claude rate-limit errors; UI to inspect and control the queue.

## Functional requirements

### Enqueueing

- **FR1 — Queue Work button on artefact view.** Every artefact view that
  represents an artefact with a configured agent (per the existing
  `source_types` mapping) shows a **Queue Work** button. Clicking it
  enqueues `(project, artefact_path, agent_name)` and returns
  immediately. The button is enabled only when the artefact is in status
  `approved` and the artefact's primary type matches at least one
  agent's `source_types`. For defect artefacts, the agent is chosen from
  the first matching `assignees[].role` (mirroring the AgentLaunchModal's
  developer-defect branch).

- **FR2 — Auto-agent selection.** The "right agent" for an artefact is:
  - `idea` → requirements-analyst
  - `ticket` / `requirement` → planning-analyst
  - `plan-backend` → backend-developer
  - `plan-frontend` → frontend-developer
  - `plan-test` → test-developer
  - `test` → qa
  - `defect` → the first agent whose `role` matches any entry in the
    defect's `assignees[].role` list. If no match, the button is hidden.

- **FR3 — Duplicate suppression.** If the same `(project, artefact_path)`
  is already in the queue and is in state `pending` or `running`, the
  Queue Work button is disabled and the artefact view shows a "Queued"
  badge with the position number.

### Persistence

- **FR4 — Survives restart.** The queue is persisted to SQLite at
  `~/.kaos-control/data/queue.db` (app-level, separate from per-project
  index DBs). On server start, items in state `pending` remain pending
  and are processed in original order. Items in state `running` are
  considered orphaned and are moved back to `pending` at the head of the
  queue with a `restart_recovered` flag set to true.

### Dispatching

- **FR5 — One at a time, globally.** The dispatcher runs as a single Go
  goroutine. At any instant, **at most one** queue job is in state
  `running`. The dispatcher does not start a new job until the previous
  one reaches a terminal state (`completed`, `failed`, `skipped`,
  `cancelled`) or until the queue is paused.

- **FR6 — Manual-launch coexistence.** A queued job and a manually-
  launched agent run (via the existing agent panel button) can coexist
  in the same wall-clock window — the queue's serial guarantee covers
  queue-initiated runs only. The per-lineage lock in `internal/lock/`
  continues to enforce that no two agents touch the same lineage
  simultaneously regardless of source.

- **FR7 — Skip-on-status-mismatch.** When the dispatcher picks up the
  head of the queue, it re-checks the artefact's status. If the artefact
  is no longer in `approved`, the job is moved to terminal state
  `skipped` with reason `status_changed_to:<current_status>` and the
  dispatcher continues to the next item. No agent run is started.

### Rate-limit handling

- **FR8 — Detection.** The dispatcher inspects every stream-json event
  emitted by the running `claude` subprocess. Any event with top-level
  field `"error":"rate_limit"` triggers the rate-limit flow.

- **FR9 — Reset-time parsing.** The reset time is extracted from the
  human-readable `content[].text` field of the same event. The parser
  must handle these shapes (case-insensitive):

  | Input fragment | Resolved time |
  |---|---|
  | `resets 8pm (Australia/Brisbane)` | next 20:00 in Australia/Brisbane |
  | `resets 10:30am (America/New_York)` | next 10:30 in America/New_York |
  | `resets 8pm` (no TZ) | next 20:00 in the server's local TZ |
  | `resets at 2026-05-13T08:00:00Z` | parsed ISO 8601 |
  | `retry after 60 seconds` | now + 60s |

  If parsing fails (unknown format), fall back to `now + 1 hour` and log
  the raw text under WARN so the parser can be extended.

- **FR10 — Pause and requeue.** On rate-limit detection:
  1. The currently-running job is marked terminal `failed` with reason
     `rate_limit`, and **also** automatically re-enqueued at the **head**
     of the queue (a single new `pending` row with `attempts=2`).
  2. The dispatcher enters state `paused`. `paused_until` is set to
     `reset_time + grace`, where `grace` is the app-config value
     `queue.resume_grace_minutes` (default `5`).
  3. A WS event `queue.paused` is broadcast with `paused_until` and the
     parsed reset time.

- **FR11 — Auto-resume.** When `now >= paused_until`, the dispatcher
  leaves the `paused` state and resumes from the head of the queue. A
  WS event `queue.resumed` is broadcast.

### Manual control

- **FR12 — Manual pause / resume.** A user with the `product-owner` or
  `devops` role can pause or resume the queue regardless of the
  rate-limit state. Manual pause has no `paused_until`; the queue stays
  paused until resumed by a user.

- **FR13 — Cancel queued item.** A user with the role required to launch
  the corresponding agent can remove a queued item (DELETE). Removing a
  `running` item is **not** allowed via the queue API — the user must
  use the existing kill-run mechanism on the agent panel.

### UI

- **FR14 — Queue view page.** A new top-level route, `/queue`, accessible
  from the app header, lists all queue items grouped by state:
  - **Running** (0 or 1 item) — agent, project, artefact, started-at,
    elapsed time, link to its run log.
  - **Pending** (N items) — in FIFO order, with position number,
    project, artefact, agent, queued-at, "Remove" button per row.
  - **Recently finished** (last 10) — terminal state + reason + when.
  - **Pause state** — if paused, a prominent banner with the reset time
    and "Resume now" button (for privileged users).

- **FR15 — Header badge.** The app header shows a small badge with the
  pending count and a pause icon when the queue is paused. Clicking the
  badge navigates to `/queue`.

- **FR16 — Artefact "Queued" badge.** When an artefact is enqueued, the
  artefact view shows a "Queued — position N" badge near the Queue Work
  button, and the button becomes disabled.

## Non-functional requirements

- **NFR1 — Database isolation.** Queue state lives in its own SQLite
  file (`~/.kaos-control/data/queue.db`). It is NOT a table in the
  per-project index DB, so the queue can reference items across
  projects.

- **NFR2 — WebSocket events.** State changes broadcast on the existing
  hub:
  - `queue.added` — `{ id, project, path, agent, position }`
  - `queue.started` — `{ id, project, path, agent, started_at }`
  - `queue.finished` — `{ id, terminal_state, reason }`
  - `queue.skipped` — `{ id, reason }`
  - `queue.paused` — `{ paused_until, reset_time, raw_text }`
  - `queue.resumed` — `{}`
  - `queue.cancelled` — `{ id }`

- **NFR3 — Configurable.** App config keys:
  ```yaml
  queue:
    enabled: true                  # default true
    resume_grace_minutes: 5        # buffer added after reset_time
    fallback_pause_minutes: 60     # used when parser fails
    max_attempts: 3                # after which a job stays failed
  ```

- **NFR4 — Observability.** Each queue state transition writes a log
  line at INFO with the job id, prior state, new state, and reason.

## Permission model

| Action | Required role |
|---|---|
| `POST /api/queue` (enqueue) | The role required to launch the target agent — same predicate as `handleStartAgentRun` after auth-role-checks Milestone 4. |
| `DELETE /api/queue/{id}` (cancel pending) | Same as enqueue (the user who could enqueue can cancel). Also `product-owner` for any item. |
| `POST /api/queue/pause` | `product-owner` or `devops`. |
| `POST /api/queue/resume` | `product-owner` or `devops`. |
| `GET /api/queue` (read) | Any authenticated user. |

## Acceptance criteria

- **AC1 — Enqueue / dispatch happy path.** Approved artefact + correct
  agent + queue empty → button enqueues → dispatcher starts the agent
  within 1 s → run completes → `queue.finished` broadcast → queue empty.

- **AC2 — Strict serial.** Enqueue three approved artefacts from three
  different projects. Confirm that at no point are two agent runs in
  state `running` simultaneously (verify via WS event log and via the
  agent runs table).

- **AC3 — Rate-limit detection from sample.** Inject the captured
  `error: rate_limit` stream event from
  `runs/22de65f53d82bf14.log` into a test fixture and verify the
  dispatcher transitions to `paused`, captures the reset time
  `next 20:00 Australia/Brisbane`, sets `paused_until` to that + 5 min,
  and re-enqueues the failed job at the head.

- **AC4 — Auto-resume after grace.** Mock the clock; advance to
  `paused_until + 1ms`; verify dispatcher leaves `paused` and starts
  the head job.

- **AC5 — Reset-time parser.** Unit tests cover each row of the FR9
  table plus three malformed inputs that fall back to 1-hour retry.

- **AC6 — Skip-on-status-mismatch.** Enqueue an approved artefact.
  Before the dispatcher picks it up, transition it to `in-development`.
  Verify the dispatcher moves the job to `skipped` with reason
  `status_changed_to:in-development` and continues.

- **AC7 — Persistence.** Enqueue three items, restart the server,
  confirm all three are still pending in FIFO order. Restart while one
  is running and confirm it is moved back to head of pending with
  `restart_recovered=true`.

- **AC8 — Manual pause / resume.** Pause manually; the dispatcher does
  not start a new job even when the queue is non-empty. Resume; head
  starts within 1 s.

- **AC9 — Cancel pending.** A pending item is removed from the queue
  via DELETE. The dispatcher never starts it.

- **AC10 — UI integration.** All four state transitions are reflected
  in the `/queue` page within 500 ms of the WS event without page
  refresh, and the header badge updates similarly.

## Out of scope (for KC-Release1)

- Per-project sub-queues.
- Per-job priority / fairness.
- Auto-retry on non-rate-limit failures.
- Backfilling failed jobs from before the feature shipped.
- A public REST API for external schedulers.

## Open questions

None blocking. Two minor items the implementer can resolve:

1. **Header badge styling** — match the existing run-indicator badge in
   the header rather than introducing a new visual idiom. Frontend
   plan can settle this.

2. **Reset-time parser locale handling.** "8pm" is unambiguous; "8:00"
   could be morning or evening. The example formats use am/pm or
   24-hour with the colon — assume that pattern and reject bare numeric
   times. Document in the parser tests.
