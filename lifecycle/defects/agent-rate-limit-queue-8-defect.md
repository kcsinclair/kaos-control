---
title: Dispatcher skips re-queued rate-limit jobs because artifact moved to active_status
type: defect
status: approved
lineage: agent-rate-limit-queue
parent: lifecycle/tests/agent-rate-limit-queue-6-test.md
labels:
    - defect
    - release-blocker
release: KC-Release1
assignees:
    - role: backend-developer
      who: agent
---

# Dispatcher skips re-queued rate-limit jobs because artifact moved to active_status

`TestQueue_RateLimit_AutoResume` (QR2) fails: after a rate-limit pause the
re-queued job (attempts=2) is dequeued but immediately skipped because
`ArtifactStatus` returns `clarifying` rather than `approved`.

## Root Cause

When an agent run starts (`Manager.StartRun`, `internal/agent/agent.go:523`),
the artifact is transitioned to the agent's `active_status`
(`clarifying` for `requirements-analyst`). This happens *before* the fake
claude binary runs. When the fake claude then emits a rate-limit event, the
artifact is already in `clarifying` state.

The dispatcher re-enqueues the job at the head of the queue and pauses. On
resume, `processNext` calls `pa.ArtifactStatus(job.ArtifactPath)` and finds
`clarifying`, which is not `approved`, so it skips the job with reason
`status_changed_to:clarifying`.

```
queue: skipping job — artifact no longer approved
  job_id=deee2b1dbda12393
  artifact=lifecycle/ideas/qr2-idea-1.md
  status=clarifying
```

This is incorrect: the artifact's status was changed by the previous agent run
start, not by a user action. A rate-limit retry should not be subject to the
"must still be approved" gate that prevents stale enqueues from running.

## Reproduction Steps

1. `go test -count=1 -tags integration -run TestQueue_RateLimit_AutoResume ./tests/integration/ -timeout 60s`
2. Observe: rate-limit occurs, queue pauses, queue is manually resumed, but
   the re-queued job is skipped instead of completing.

## Expected Behaviour

After resuming from a rate-limit pause, the re-queued job (attempts=2) picks up
where it left off, runs the agent, and reaches `state=completed` in the recent
list.

## Actual Behaviour

The re-queued job is silently skipped:

```
queue: skipping job — artifact no longer approved
  artifact=lifecycle/ideas/qr2-idea-1.md status=clarifying
```

Test fails:
```
queue_rate_limit_test.go:215: expected re-queued job (attempts=2) to complete after resume
--- FAIL: TestQueue_RateLimit_AutoResume (15.41s)
```

## Suggested Fix

In `internal/queue/dispatcher.go`, `processNext()`, relax the approved-status
check for re-queued jobs (i.e., `job.Attempts > 1`):

```go
if status := pa.ArtifactStatus(job.ArtifactPath); status != "approved" {
    // Rate-limit retries have already been validated at first enqueue;
    // the active_status transition should not block re-runs.
    if job.Attempts <= 1 {
        reason := "status_changed_to:" + status
        // ... skip logic ...
        return
    }
}
```

Alternatively, the dispatcher could store the artifact's status at enqueue time
and use that stored value for re-queued attempts.

## Logs / Output

```
INFO  queue: skipping job — artifact no longer approved
      job_id=deee2b1dbda12393
      artifact=lifecycle/ideas/qr2-idea-1.md
      status=clarifying

queue_rate_limit_test.go:215: expected re-queued job (attempts=2) to complete after resume
--- FAIL: TestQueue_RateLimit_AutoResume (15.41s)
FAIL    github.com/kaos-control/kaos-control/tests/integration  20.149s
```
