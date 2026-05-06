---
title: "artifact.indexed WebSocket event not emitted after release rename propagation"
type: defect
status: draft
lineage: releases-and-roadmaps
parent: lifecycle/tests/releases-and-roadmaps-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# artifact.indexed WebSocket event not emitted after release rename propagation

## Reproduction Steps

1. Register a hub listener channel via `env.proj.Hub.Register`.
2. Create a release and assign at least one artifact to it.
3. Rename the release via `PUT /api/p/testproject/releases/:id` with a new name.
4. Wait up to 3 seconds for WebSocket events on the listener channel.
5. Check whether an `artifact.indexed` event is received.

## Expected Behaviour

When a release is renamed, the handler rewrites the `release:` field in all assigned artifact files on disk and re-indexes them. Each re-indexed artifact should broadcast an `artifact.indexed` event via the hub, observable on the registered listener.

A `release.updated` event is also expected; the test waits for **both**.

## Actual Behaviour

The `release.updated` event is received (rename is recorded), but `artifact.indexed` is never emitted within the 3-second window. Artifact files may be rewritten without the hub being notified, or the re-index step is omitting the broadcast.

## Logs / Output

```
releases_ws_test.go:191: did not receive artifact.indexed event after rename propagation
--- FAIL: TestReleaseWebSocket_RenamePropagate (3.12s)
```

The 3.12 s elapsed time confirms the test waited the full deadline before failing.
