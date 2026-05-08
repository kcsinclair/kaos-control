---
title: Clear stuck lineage locks from the UI
type: idea
status: approved
lineage: clear-stuck-lineage-locks
created: "2026-04-28T16:00:00+10:00"
priority: normal
labels:
    - feature
    - frontend
    - workflow
    - operability
release: KC-Release1
---

# Clear stuck lineage locks from the UI

## Problem

When an agent run crashes (or kaos-control itself crashes), the lineage lock written at agent-start stays in the `lineage_locks` SQLite table until either the server restarts long enough for the stale-lock reaper to fire, or the lock is manually deleted.

The reaper at [internal/lock/lock.go:80](internal/lock/lock.go#L80) clears any lock whose `last_heartbeat` is older than 5 minutes — but it only runs while the server is up, and the user has no visibility into "this is locked, you can't transition this artifact, here's why, and here's the holder."

Today the user's options when stuck are:

1. Wait ~5 min for the reaper.
2. SSH in and `DELETE FROM lineage_locks WHERE lineage = '…';`.
3. `curl -X DELETE` against the existing endpoint.

None of these are reasonable for a non-technical product owner using the SPA.

## Reproducer

A real instance occurred after a kaos-control crash mid-agent-run. The artifact `artefact-inline-status-change` could not be transitioned because the lock from a now-dead agent process was still present:

```
lineage "artefact-inline-status-change" is locked by 0e5e1eebfb65843f (agent): lineage already locked
```

The lock cleared naturally after ~5 minutes once the server was back up.

## What already exists

- `DELETE /api/p/:project/locks/:lineage` endpoint at [internal/http/locks.go:71](internal/http/locks.go#L71) — releases a lock unconditionally.
- Stale-lock reaper at [internal/lock/lock.go:80](internal/lock/lock.go#L80) — 60s tick, 5-min heartbeat threshold.
- Crash recovery for runs at `index.RecoverRunningRuns` — marks orphaned `running` agent runs as `failed` on startup.

So the backend pieces are all there. What's missing is:

1. **Surfacing the lock state in the UI** when an action fails because of it.
2. **A force-release affordance** for the product-owner role.
3. **Symmetric crash recovery** for *agent locks* on startup, parallel to what `RecoverRunningRuns` does for run rows.

## Proposed approach

Three pieces, in order of cheapness / value:

### 1. Inline "Force release" on lock-conflict toasts (cheapest, biggest immediate win)

When the SPA receives a `409 locked` (or equivalent) response from `transition`, `agents/:name/run`, or artifact save endpoints, the error toast should include the lock's holder and kind, plus a **Force release** button visible only to users with the `product-owner` role. Clicking it calls the existing `DELETE /locks/:lineage` and retries the original action.

### 2. Locks admin panel (more general)

A small page (or sidebar panel on the artifact view) listing all current locks: lineage, holder, kind, acquired-at, last-heartbeat. One row per lock, "Force release" button per row, product-owner only. Useful when running multi-agent campaigns to see what's stuck without having to trigger a conflict first.

### 3. Symmetric startup recovery for orphan agent locks (one-shot fix on the backend)

In `lock.Manager.New()` (or alongside it), at startup release any lock of kind `agent` whose `holder` (the run_id) does not correspond to a `running` agent run in the index. This catches the exact crash scenario from the reproducer above without making the user wait 5 minutes after restart.

The existing reaper handles the live-system case; this would handle the crash-restart case.

## Scope

In scope:

- Frontend toast button (#1).
- Optional: locks admin panel (#2).
- Backend startup recovery for orphan agent locks (#3).

Out of scope (separate ideas if needed):

- Per-user lock claim transfer ("steal this lock from another user").
- Editor lock auto-release on tab close that already exists.
- Distributed/multi-server lock coordination — kaos-control is single-binary, single-host.

## Acceptance criteria

- Pressing **Force release** on a lock-conflict toast clears the lock, the toast disappears, and the user can retry the action.
- The button is invisible to non-product-owner users.
- After a kaos-control crash with at least one in-flight agent run, restarting the server clears the orphaned agent lock automatically (no 5-min wait).
- Existing editor-lock auto-release on close is unaffected.
