---
title: "Project Feed Integration Tests"
type: test
status: draft
lineage: project-feed
parent: lifecycle/test-plans/project-feed-5-test.md
created: "2026-05-06T00:00:00+10:00"
---

# Project Feed Integration Tests

## Overview

Integration tests for the Project Feed feature, covering the `events` table,
the `GET /api/p/{project}/feed` REST endpoint, automatic event recording, WebSocket
delivery, and startup pruning. All tests require the `integration` build tag and
are located in `tests/integration/`.

---

## Test Files

### `tests/integration/feed_test.go`

Covers milestones 1–7 and 9 from the test plan.

| Test function | Scenario |
|---|---|
| `TestFeedEventsTableExists` | Confirms the `events` table and both indices (`idx_events_timestamp`, `idx_events_event_type`) exist in SQLite after project open. |
| `TestFeedInsertAndQuery` | Inserts three events of different types and verifies round-trip of all fields (`event_type`, `timestamp`, `actor`, `artifact_path`, `run_id`, `summary`, `payload_json`). Asserts reverse-chronological ordering and that an empty table returns a zero-length result. |
| `TestFeedEndpointDefaults` | Seeds 5 events, calls `GET /feed` with no params, asserts 5 events newest-first and `next_cursor: null`. |
| `TestFeedEndpointLimit` | Seeds 10 events, calls `GET /feed?limit=3`, asserts 3 events returned and `next_cursor` equals the last event's ID. |
| `TestFeedEndpointCursorPagination` | Seeds 10 events, pages through with `limit=4` using cursor-based pagination, asserts disjoint pages, global newest-first order, and `next_cursor: null` on the final page. |
| `TestFeedEndpointLimitCap` | Calls `GET /feed?limit=999`, asserts the server caps the response at 200 events maximum. |
| `TestFeedEndpointTypeFilter` | Seeds three event types, calls `?types=status_transition`, asserts only matching events are returned. |
| `TestFeedEndpointMultiTypeFilter` | Calls `?types=status_transition,agent_started`, asserts both types are present and no others. |
| `TestFeedEndpointFilterWithPagination` | Seeds 20 mixed-type events, verifies type filtering composes correctly with cursor pagination across two pages. |
| `TestFeedTransitionEvent` | POSTs a `draft → clarifying` transition, queries the feed, and asserts exactly one `status_transition` event with the correct actor, artifact_path, and timestamp. |
| `TestFeedArtifactCreatedEvent` | POSTs `POST /artifacts` to create an artifact, queries the feed, and asserts exactly one `artifact_created` event with the correct actor and artifact_path. |
| `TestFeedAgentEvents` | Inserts `agent_started` and `agent_finished` events directly via the index layer, queries `?types=agent_started,agent_finished`, and asserts correct type, actor, and run_id on each. |
| `TestFeedPruneByAge` | Inserts one 40-day-old and one 10-day-old event, calls `PruneEvents(30, 10000)`, asserts only the recent event survives. |
| `TestFeedPruneByCount` | Inserts 20 recent events, calls `PruneEvents(365, 10)`, asserts exactly 10 (the newest) survive. |
| `TestFeedPruneCombined` | Inserts 5 recent + 5 old events, calls `PruneEvents(30, 3)`, asserts age rule removes old events and count cap limits the total to 3. |
| `TestFeedEndpointPerformance` | Batch-inserts 10,000 events and asserts the median of 5 `GET /feed` requests is under 50 ms. |

### `tests/integration/feed_ws_test.go`

Covers milestone 8 from the test plan.

| Test function | Scenario |
|---|---|
| `TestFeedWebSocketNewEvent` | Registers a hub channel, triggers a `draft → clarifying` transition, and asserts a `feed.new` WebSocket event is received within 2 seconds containing a non-zero `id`, `event_type: status_transition`, a non-empty `summary`, and a non-zero `timestamp`. |

---

## Running the Tests

```sh
go test ./tests/integration/ -run TestFeed -tags integration -v
```
