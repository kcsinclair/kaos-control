---
title: "Backend Plan: Project Feed"
type: plan-backend
status: draft
lineage: project-feed
parent: lifecycle/requirements/project-feed-2.md
created: "2026-04-29"
---

# Backend Plan: Project Feed

relates-to: [[project-feed]]

## Overview

Add an `events` table to the SQLite index, record lifecycle events at every existing `hub.Broadcast` call site, expose a paginated REST endpoint `GET /api/p/{project}/feed`, and prune stale events on startup. The [[project-feed-4-fe]] frontend plan depends on the REST endpoint and the WebSocket events delivered here.

---

## Milestone 1 — Create the `events` table

### Description

Add `ensureEventsTable()` following the same pattern as `ensureAgentRunsTable()` (line 1078 of `internal/index/index.go`). The table is created outside the versioned schema so it survives rebuilds. Create indices on `(timestamp DESC)` and `(event_type)` per NFR-1.

### Files to change

- `internal/index/index.go`
  - Add `ensureEventsTable()` after `ensureAgentRunsTable()` (line ~1098):
    ```sql
    CREATE TABLE IF NOT EXISTS events (
        id              INTEGER PRIMARY KEY AUTOINCREMENT,
        event_type      TEXT NOT NULL,
        timestamp       INTEGER NOT NULL,
        actor           TEXT NOT NULL,
        artifact_path   TEXT,
        run_id          TEXT,
        summary         TEXT NOT NULL,
        payload_json    TEXT
    );
    CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp DESC);
    CREATE INDEX IF NOT EXISTS idx_events_event_type ON events(event_type);
    ```
  - Call `ensureEventsTable()` from `Open()` (after line 80, the `ensureAgentRunsTable` call).

### Acceptance criteria

- The `events` table exists after startup and survives `dropAndRecreate` schema rebuilds.
- Both indices appear in `sqlite_master`.
- `go build ./...` and `go vet ./...` pass.

---

## Milestone 2 — Event row type and insert/query methods

### Description

Define the `EventRow` struct and add `InsertEvent`, `ListEvents`, and `PruneEvents` methods to the `Index` type.

### Files to change

- `internal/index/index.go`
  - Add `EventRow` struct:
    ```go
    type EventRow struct {
        ID           int64   `json:"id"`
        EventType    string  `json:"event_type"`
        Timestamp    int64   `json:"timestamp"`
        Actor        string  `json:"actor"`
        ArtifactPath *string `json:"artifact_path,omitempty"`
        RunID        *string `json:"run_id,omitempty"`
        Summary      string  `json:"summary"`
        PayloadJSON  *string `json:"payload_json,omitempty"`
    }
    ```
  - `InsertEvent(e *EventRow) error` — INSERT into events; set `e.ID` from `LastInsertId`.
  - `ListEvents(limit int, beforeID int64, types []string) ([]*EventRow, error)` — SELECT in reverse-chronological order with:
    - `WHERE id < ?` when `beforeID > 0` (cursor pagination).
    - `WHERE event_type IN (...)` when `types` is non-empty.
    - `LIMIT ?` (caller-supplied, capped at 200).
    - Returns the rows plus supports the caller deriving `next_cursor` from the last row's ID.
  - `PruneEvents(maxAgeDays int, maxCount int) error` — DELETE events older than `maxAgeDays` days OR exceeding `maxCount` total (keeping newest), whichever is more aggressive. Run as two DELETE statements in a transaction.

### Acceptance criteria

- `InsertEvent` writes a row and populates `e.ID`.
- `ListEvents(50, 0, nil)` returns newest-first, up to 50 rows.
- `ListEvents(50, 42, []string{"transition"})` returns only `transition` events with `id < 42`.
- `PruneEvents(30, 5000)` deletes rows older than 30 days AND trims to 5000 most recent.
- No new dependencies introduced.
- `go build ./...` and `go vet ./...` pass.

---

## Milestone 3 — Event recording at broadcast call sites

### Description

At every existing `hub.Broadcast` call that matches the FR-1 taxonomy, insert an event row into the `events` table. Insertion happens in the same goroutine as the broadcast — no new goroutines per event. The index is accessible from every call site via `p.Idx` (HTTP handlers) or `m.idx` (agent manager).

### Files to change

- `internal/http/transition.go` (line ~130) — after the `artifact.indexed` broadcast for transitions, insert a **status_transition** event. Actor = authenticated user email. Summary = `"{title}" transitioned from {old} → {new}"`.

- `internal/http/write.go` — at each broadcast call:
  - Line ~141 (create): insert **artifact_created** event. Actor = user email. Summary = `"Created {type} "{title}""`.
  - Lines ~262, ~481 (update / priority patch): no feed event — updates are too noisy.
  - Line ~308 (delete): no feed event — deletions are rare and not in FR-1.

- `internal/http/idea_chat.go` (line ~180) — insert **artifact_created** event for idea artifacts created via chat.

- `internal/agent/agent.go`:
  - Line ~431 (`agent.started` broadcast): insert **agent_started** event. Actor = agent name. Summary = `"Agent {name} started on {target_path}"`. Set `run_id`.
  - Line ~545 (`agent.finished`/`agent.failed` broadcast): insert **agent_finished** or **agent_failed** event. Include produced artifacts count in summary.
  - Line ~518 (`git.committed` broadcast): insert **git_committed** event. Summary = `"Agent {name} committed {n} file(s)"`. Include short SHA if available.

- `internal/watcher/watcher.go` (line ~151) — the watcher broadcasts `file.changed`, not `artifact.indexed`. Per FR-1 and the open-questions resolution, we do NOT record file.changed events. However, the watcher's `IndexFile` call (line ~144) may detect new artifacts (type=defect). To capture **defect_raised** events: add a check after `IndexFile` succeeds — query the freshly indexed row, and if `type == "defect"` and the path was not previously indexed, insert a **defect_raised** event. Actor = "system". This requires passing the index result back from `IndexFile` or doing a follow-up `Get`. Alternatively, detect new defects in the HTTP create path (write.go line ~141) since defects created by the QA agent still go through the watcher. The simpler approach: handle defect detection in the watcher by checking whether the path existed before `IndexFile`.

**Recommended approach for the watcher**: Before calling `idx.IndexFile`, call `idx.Get(relPath)` to check existence. After `IndexFile` succeeds, if the path was new and the row's type is `defect`, insert a **defect_raised** event.

### Acceptance criteria

- Status transitions produce `status_transition` events with old/new status, actor, and artifact path.
- New artifacts produce `artifact_created` events.
- Agent start/finish/fail produce corresponding events with run_id.
- Git commits produce `git_committed` events.
- New defect artifacts produce `defect_raised` events.
- Broadcast latency is not measurably increased (insert is synchronous but SQLite WAL mode keeps it fast).
- `go build ./...` and `go vet ./...` pass.

---

## Milestone 4 — Event pruning on startup

### Description

Call `PruneEvents` during project `Open()` to enforce retention. Read `feed.retention_days` (default 30) and `feed.max_events` (default 5000) from the project config.

### Files to change

- `internal/config/project.go` — add `Feed` struct to `Project` config:
  ```go
  Feed struct {
      RetentionDays int `yaml:"retention_days"`
      MaxEvents     int `yaml:"max_events"`
  } `yaml:"feed"`
  ```
  Default `RetentionDays` to 30 and `MaxEvents` to 5000 if zero after load.

- `internal/project/project.go` — in `Open()`, after the index scan (line ~83 area), call:
  ```go
  _ = idx.PruneEvents(cfg.Feed.RetentionDays, cfg.Feed.MaxEvents)
  ```

### Acceptance criteria

- Events older than configured retention are deleted on startup.
- Events exceeding the max count are pruned on startup (keeping newest).
- Missing or zero config values default to 30 days / 5000 events.
- `go build ./...` and `go vet ./...` pass.

---

## Milestone 5 — REST endpoint `GET /api/p/{project}/feed`

### Description

Add the feed endpoint to the project-scoped router, returning paginated events in reverse-chronological order.

### Files to change

- `internal/http/server.go` — register the route inside the `/p/{project}` block (after line ~155):
  ```go
  r.Get("/feed", s.handleGetFeed)
  ```

- `internal/http/feed.go` (new file) — implement `handleGetFeed`:
  1. Parse query params: `limit` (int, default 50, max 200), `before` (int64, cursor), `types` (comma-separated string → `[]string`).
  2. Call `p.Idx.ListEvents(limit, before, types)`.
  3. Derive `next_cursor`: if result length == limit, set to the last row's ID; otherwise null.
  4. Respond with:
     ```json
     {
       "events": [...],
       "next_cursor": 42
     }
     ```

### Acceptance criteria

- `GET /api/p/{project}/feed` returns events newest-first with default limit 50.
- `?limit=10` limits to 10 results.
- `?limit=999` is capped to 200.
- `?before=42` returns only events with `id < 42`.
- `?types=status_transition,agent_finished` filters to those types.
- `next_cursor` is the ID of the last returned event when the page is full, `null` otherwise.
- Endpoint responds in < 50 ms for 50 events on a 10,000-row table (NFR-1).
- `go build ./...` and `go vet ./...` pass.

---

## Milestone 6 — Broadcast feed events over WebSocket

### Description

When a new event is inserted, broadcast a `feed.new` WebSocket event so the [[project-feed-4-fe]] frontend can prepend it in real time without polling the REST endpoint.

### Files to change

- `internal/http/transition.go`, `internal/http/write.go`, `internal/http/idea_chat.go`, `internal/agent/agent.go`, `internal/watcher/watcher.go` — after each `InsertEvent` call, broadcast:
  ```go
  hub.Broadcast(hub.Event{
      Type:    "feed.new",
      Payload: eventRow,
  })
  ```

  This is a single additional broadcast alongside the existing domain event broadcast. The frontend listens for `feed.new` to update the feed view in real time.

### Acceptance criteria

- Every inserted event produces a `feed.new` WebSocket message containing the full `EventRow` as JSON.
- Existing domain broadcasts (`artifact.indexed`, `agent.started`, etc.) remain unchanged.
- The `WsEventType` union on the frontend will need updating (see [[project-feed-4-fe]]).
- `go build ./...` and `go vet ./...` pass.
