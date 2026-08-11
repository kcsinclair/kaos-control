---
title: Cannot cancel a broken DevOps pipeline stuck in "Running" (orphaned run, in-memory cancel state)
type: defect
status: approved
lineage: devops-cancel-orphaned-running-pipeline
created: "2026-08-11T00:00:00+10:00"
priority: medium
release: KC-Release5
labels:
    - defect
    - devops
    - reliability
assignees:
    - role: backend-developer
      who: agent
---

# Cannot cancel a broken DevOps pipeline stuck in "Running"

## Source

GitHub issue [#17](https://github.com/kcsinclair/kaos-control/issues/17) —
reported by **aburow**, 2026-06-23.

## Summary

A DevOps pipeline ("Ollama Git Repository Updater", 9 steps) is shown as
**Running** with a spinner, but its recent runs are all `failed` and none of the
step indicators advance. Clicking **Cancel** does not clear it — the pipeline
stays stuck in Running and cannot be cancelled or re-run.

## Likely root cause

Cancellation is implemented
([internal/http/devops.go L177-195](../../internal/http/devops.go#L177)
`handleCancelPipeline` → `p.DevopsRunner.Cancel(runID)`), but the runner tracks
cancellation state **in memory** (`activeRun` holds the cancel func + done
channel — [internal/devops/runner.go L21](../../internal/devops/runner.go#L21)).
If the process/server restarted, or the run was reaped while its persisted
status stayed `Running`, the in-memory `activeRun` entry is gone. `Cancel(runID)`
then finds nothing to cancel (404 / no-op) while the UI still renders the run as
Running — an **orphaned** run that can't be cleared. (This is the same class of
in-memory-vs-persisted desync seen elsewhere; the persisted run status is never
reconciled to a terminal state when the in-memory handle is lost.)

## Steps to reproduce (inferred)

1. Start a pipeline that wedges (e.g. the Ollama updater with an unreachable
   backend); it sits in `Running`.
2. Lose the in-memory run handle (restart, or the run's goroutine ends without
   writing a terminal status).
3. Click **Cancel** → nothing happens; the pipeline stays Running with no way to
   clear it.

## Expected

Cancel always resolves a stuck run to a terminal state: if no live handle
exists, mark the persisted run `cancelled`/`failed` so the UI clears and the
pipeline can be re-run. Ideally, orphaned `Running` runs are reconciled to a
terminal status on startup (a run with no live handle cannot still be running).
