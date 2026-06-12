---
title: "Release WebSocket events (created/updated) not broadcast; API write emits spurious watcher event"
type: defect
status: draft
lineage: release-websocket-events-not-broadcast
created: "2026-06-12T00:00:00+10:00"
labels:
  - defect
assignees:
  - role: backend-developer
    who: agent
---

# Release WebSocket events (created/updated) not broadcast; API write emits spurious watcher event

## Reproduction Steps

### Part A — Missing events
1. Subscribe to the project WebSocket.
2. Create a release via POST `/api/p/<project>/releases`.
3. Wait 2 seconds for a `release.created` WS event.

Or:
1. Subscribe to the project WebSocket.
2. Update a release via PATCH `/api/p/<project>/releases/<slug>`.
3. Wait 2 seconds for a `release.updated` WS event.

Or:
1. Subscribe to the project WebSocket.
2. Rename a release (which triggers propagation to linked artifacts).
3. Wait for a `release.updated` event after rename propagation completes.

### Part B — Spurious watcher event
1. Subscribe to the project WebSocket.
2. Write a release via the API (PUT or PATCH) — this should produce exactly 1 `release.changed` event (the API-issued one).
3. Observe the event count.

## Expected Behaviour

- **Part A**: `release.created` fires within 2 s of a new release being saved; `release.updated` fires within 2 s of any update; propagation rename also emits `release.updated`.
- **Part B**: Exactly 1 `release.changed` event (the API's own broadcast). The fsnotify watcher must suppress a second event for API-driven writes.

## Actual Behaviour

- **Part A**: No `release.created`, `release.updated`, or rename-propagation `release.updated` event arrives within the test timeout.
- **Part B**: 2 `release.changed` events are received (the API event plus an additional watcher-triggered one).

```
releases_ws_test.go:58:  did not receive release.created event within 2 seconds
releases_ws_test.go:94:  did not receive release.updated event within 2 seconds
releases_ws_test.go:190: did not receive release.updated event after rename propagation
releases_watcher_test.go:204: expected exactly 1 release.changed event (API only), got 2
```

## Failing Tests

- `TestReleaseWebSocket_Created`
- `TestReleaseWebSocket_Updated`
- `TestReleaseWebSocket_RenamePropagate`
- `TestAPIWriteDoesNotProduceWatcherEvent`

## Fix

Two issues are likely present:

1. **Missing events**: The release HTTP handlers (create, update) are not broadcasting the `release.created` / `release.updated` WS events after persisting. Confirm that `hub.Broadcast` is called with the correct event type after every successful write path.

2. **Spurious watcher event**: The fsnotify watcher path for `lifecycle/releases/` is not suppressing events triggered by API writes. A write-token or in-flight set (keyed by path + recent-write timestamp) should prevent the watcher from re-broadcasting what the API handler already emitted.
