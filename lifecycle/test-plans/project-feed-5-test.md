---
title: 'Test Plan: Project Feed'
type: plan-test
status: in-development
lineage: project-feed
created: "2026-04-29T00:00:00+10:00"
parent: lifecycle/requirements/project-feed-2.md
assignees:
    - role: product-owner
      who: agent
---

# Test Plan: Project Feed

relates-to: [[project-feed]]

## Overview

Integration tests for the Project Feed feature covering the `events` table persistence, the `GET /api/p/{project}/feed` REST endpoint, event recording at broadcast sites, WebSocket delivery of `feed.new` events, and startup pruning. Tests use the existing `testEnv` harness from `tests/integration/helpers_test.go`. These tests validate the backend delivered by [[project-feed-3-be]] and indirectly support the frontend in [[project-feed-4-fe]].

---

## Milestone 1 — Events table and basic CRUD

### Description

Verify that the `events` table is created on project open, that events can be inserted and queried, and that the table survives schema rebuilds.

### Files to change

- `tests/integration/feed_test.go` (new file)
  - `TestFeedEventsTableExists` — open a `testEnv`, query `sqlite_master` via the index's DB handle (or hit the feed endpoint) to confirm the `events` table and its indices exist.
  - `TestFeedInsertAndQuery` — insert events directly via `p.Idx.InsertEvent(...)` with varying types and timestamps, then call `p.Idx.ListEvents(50, 0, nil)` and assert:
    - Events are returned in reverse-chronological order.
    - All inserted fields round-trip correctly (`event_type`, `timestamp`, `actor`, `artifact_path`, `run_id`, `summary`, `payload_json`).
    - An empty result returns an empty slice, not nil.

### Acceptance criteria

- The `events` table and both indices (`idx_events_timestamp`, `idx_events_event_type`) exist after project open.
- Insert + query round-trips all fields correctly.
- Empty query returns `[]`, not `nil`.
- Tests pass with `go test ./tests/integration/ -run TestFeed -tags integration`.

---

## Milestone 2 — Feed REST endpoint: basic pagination

### Description

Test the `GET /api/p/{project}/feed` endpoint for correct pagination behaviour.

### Files to change

- `tests/integration/feed_test.go`
  - `TestFeedEndpointDefaults` — seed 5 events (via artifact creation or direct insert), call `GET /feed` with no params, assert:
    - Response has `events` array with 5 entries, newest first.
    - `next_cursor` is `null` (fewer events than default limit).
  - `TestFeedEndpointLimit` — seed 10 events, call `GET /feed?limit=3`, assert:
    - Exactly 3 events returned.
    - `next_cursor` equals the ID of the 3rd (last) event in the response.
  - `TestFeedEndpointCursorPagination` — seed 10 events, fetch page 1 with `limit=4`, then page 2 with `before={next_cursor}&limit=4`, then page 3. Assert:
    - No event ID appears in more than one page.
    - Events are globally ordered newest-first across pages.
    - Final page has `next_cursor: null`.
  - `TestFeedEndpointLimitCap` — call `GET /feed?limit=999`, assert the response contains at most 200 events (or fewer if not enough seeded).

### Acceptance criteria

- Default request returns all events (up to 50) newest-first.
- `limit` param restricts result count.
- Cursor-based pagination produces disjoint, correctly ordered pages.
- Limit is capped at 200.
- All tests pass.

---

## Milestone 3 — Feed REST endpoint: type filtering

### Description

Test the `types` query parameter for filtering events by category.

### Files to change

- `tests/integration/feed_test.go`
  - `TestFeedEndpointTypeFilter` — seed events of multiple types (e.g. `status_transition`, `artifact_created`, `agent_started`). Call `GET /feed?types=status_transition`, assert:
    - Only `status_transition` events are returned.
    - Other types are excluded.
  - `TestFeedEndpointMultiTypeFilter` — call `GET /feed?types=status_transition,agent_started`, assert both types are present and no others.
  - `TestFeedEndpointFilterWithPagination` — seed 20 events of mixed types, call `GET /feed?types=status_transition&limit=3`, then page 2 with cursor. Assert pagination works correctly within the filtered set.

### Acceptance criteria

- Single-type filter returns only matching events.
- Multi-type (comma-separated) filter returns events matching any listed type.
- Filtering composes correctly with cursor pagination.
- All tests pass.

---

## Milestone 4 — Automatic event recording: status transitions

### Description

Verify that performing a status transition via the API automatically inserts a `status_transition` event into the feed.

### Files to change

- `tests/integration/feed_test.go`
  - `TestFeedTransitionEvent` — seed an artifact with `status: draft`, log in as `admin@test.local`, POST `/artifacts/{path}/transition` with `{"to": "approved"}`. Then call `GET /feed` and assert:
    - A `status_transition` event exists with the correct artifact path.
    - The summary contains the old and new status.
    - The actor is the authenticated user's email.
    - The event timestamp is within a few seconds of now.

### Acceptance criteria

- Transitioning an artifact produces exactly one `status_transition` feed event.
- Event fields (actor, summary, artifact_path, timestamp) are correct.
- Test passes.

---

## Milestone 5 — Automatic event recording: artifact creation

### Description

Verify that creating an artifact via the API inserts an `artifact_created` event.

### Files to change

- `tests/integration/feed_test.go`
  - `TestFeedArtifactCreatedEvent` — POST `/artifacts` to create a new artifact, then call `GET /feed` and assert:
    - An `artifact_created` event exists with the correct path.
    - The summary mentions the artifact title and type.
    - Actor is the authenticated user.

### Acceptance criteria

- Creating an artifact produces exactly one `artifact_created` feed event.
- Event fields are populated correctly.
- Test passes.

---

## Milestone 6 — Automatic event recording: agent lifecycle events

### Description

Verify that agent start/finish/fail broadcasts produce corresponding feed events. Since launching a real agent in integration tests is complex, these tests should either:
- Use the existing agent test helpers (`tests/integration/agent_helpers_test.go`) if available, or
- Insert events directly via the index layer and verify they appear in the feed endpoint (already covered in Milestones 1-3), while separately verifying the broadcast-to-insert wiring by checking that the agent manager's broadcast calls include `InsertEvent`.

### Files to change

- `tests/integration/feed_test.go`
  - `TestFeedAgentEvents` — if agent helpers support it, start an agent run on a test artifact, wait for completion, then query `GET /feed?types=agent_started,agent_finished` and assert:
    - At least one `agent_started` event with the correct `run_id`.
    - At least one `agent_finished` (or `agent_failed`) event.
    - Events have the agent name as actor.
  - If a real agent run is impractical, test the index-layer wiring: insert `agent_started` and `agent_finished` events manually and verify they appear in the feed with correct type filtering.

### Acceptance criteria

- Agent lifecycle events appear in the feed with correct type, actor, and run_id.
- Test passes.

---

## Milestone 7 — Event pruning

### Description

Verify that `PruneEvents` correctly deletes old events and enforces the max-count cap.

### Files to change

- `tests/integration/feed_test.go`
  - `TestFeedPruneByAge` — insert events with timestamps 40 days ago and 10 days ago. Call `PruneEvents(30, 10000)`. Assert only the 10-day-old event survives.
  - `TestFeedPruneByCount` — insert 20 events all with recent timestamps. Call `PruneEvents(365, 10)`. Assert exactly 10 events remain (the 10 newest).
  - `TestFeedPruneCombined` — insert a mix of old and recent events. Call `PruneEvents(30, 5)`. Assert that both conditions are applied: old events are deleted AND total is capped at 5.

### Acceptance criteria

- Age-based pruning deletes events older than the threshold.
- Count-based pruning keeps only the N most recent events.
- Combined pruning applies both rules (whichever is more aggressive).
- All tests pass.

---

## Milestone 8 — WebSocket `feed.new` event delivery

### Description

Verify that when a feed event is recorded, a `feed.new` WebSocket message is broadcast to connected clients.

### Files to change

- `tests/integration/feed_ws_test.go` (new file)
  - `TestFeedWebSocketNewEvent` — connect a WebSocket client to `/api/p/{project}/ws`, then trigger a status transition via the REST API. Assert:
    - A `feed.new` WS message is received within 2 seconds.
    - The message payload contains the event's `id`, `event_type`, `summary`, and `timestamp`.
    - The event type in the WS message matches `status_transition`.
  - Use the existing WebSocket test patterns from `tests/integration/agent_ws_test.go` for connection setup and message reading.

### Acceptance criteria

- `feed.new` WebSocket events are delivered to connected clients.
- The payload contains the full event row.
- Message is received promptly after the triggering action.
- Test passes.

---

## Milestone 9 — Feed endpoint performance

### Description

Verify the < 50 ms response time NFR for the default page size on a table with 10,000 events.

### Files to change

- `tests/integration/feed_test.go`
  - `TestFeedEndpointPerformance` — insert 10,000 events into the events table (use a batch insert for speed). Time a `GET /feed` request (default limit=50). Assert response time is < 50 ms. Run the request 5 times and assert the median is under the threshold.

### Acceptance criteria

- With 10,000 events, the feed endpoint responds in < 50 ms for the default page size.
- Test passes consistently (not flaky under normal CI load).

---

## Milestone 10 — Test artifact in lifecycle/tests/

### Description

Create a companion test artifact documenting what the test suite covers.

### Files to change

- `lifecycle/tests/project-feed-6-test.md` (new file) — frontmatter:
  ```yaml
  title: "Project Feed Integration Tests"
  type: test
  status: draft
  lineage: project-feed
  parent: lifecycle/test-plans/project-feed-5-test.md
  ```
  Body: summarise the scenarios covered by `tests/integration/feed_test.go` and `tests/integration/feed_ws_test.go`, referencing each test function.

### Acceptance criteria

- The test artifact exists in `lifecycle/tests/` with correct frontmatter.
- The body lists all test functions and the scenarios they cover.
- Lineage index (6) follows the test plan (5) monotonically.

---

## Resolved Questions

**Blocking — test implementation cannot proceed until the backend is built.**

1. **Backend not implemented.** The test plan assumes that `project-feed-3-be` has been delivered: specifically the `Index.InsertEvent(...)`, `Index.ListEvents(limit, cursor, types)`, and `Index.PruneEvents(maxAgeDays, maxCount)` methods on `internal/index/index.go`, the `events` table in SQLite, and the `GET /api/p/{project}/feed` REST endpoint. As of 2026-04-29 none of these exist in the codebase (`lifecycle/backend-plans/project-feed-3-be.md` is `status: approved` but not yet merged). All milestones (1–9) depend on this code being present and compilable before any integration test can be written against it.

   **Question for product-owner:** Please assign and complete the `backend-developer` run for `project-feed-3-be` first, then re-assign the test-developer to this plan once the backend is merged.

2. **Method signatures unconfirmed.** The test plan specifies calls such as `p.Idx.InsertEvent(...)`, `p.Idx.ListEvents(50, 0, nil)`, and `p.Idx.PruneEvents(30, 10000)` but does not define the concrete Go signatures or the `EventRow` struct fields. If the backend implementation deviates from these shapes (e.g. uses a different cursor type, a struct argument, or different parameter order), the tests will need adjustment.

   **Question for product-owner / backend-developer:** Once the backend is implemented, please confirm the exact Go signatures for `InsertEvent`, `ListEvents`, and `PruneEvents`, and the fields of `EventRow`.

3. **`feed.new` WebSocket payload shape.** Milestone 8 asserts the `feed.new` message contains `id`, `event_type`, `summary`, and `timestamp`. The backend plan (milestone 5 of `project-feed-3-be`) describes the broadcast but does not canonically define the JSON key names or whether all `EventRow` fields are included. Tests against the wrong shape will give false negatives.

   **Question for backend-developer:** Please document the exact JSON structure of the `feed.new` WebSocket message once it is implemented.

> Backend development completed.
