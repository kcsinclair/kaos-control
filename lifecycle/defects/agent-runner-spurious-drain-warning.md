---
title: "\"stdout readers did not drain within grace\" warns on every run over 5 seconds"
type: defect
status: draft
lineage: agent-runner-spurious-drain-warning
created: "2026-08-25T12:10:00+10:00"
priority: low
labels:
    - defect
    - agent
    - agent-runner
    - observability
    - logging
release: KC-Release6
assignees:
    - role: backend-developer
      who: agent
---

# "stdout readers did not drain within grace" warns on every run over 5 seconds

## Reproduction Steps

1. Run any agent whose work takes longer than 5 seconds.
2. Watch the server log:
   `grep "did not drain within grace" ~/.kaos-control/logs/kaos-control.log`
3. Observe a WARN for that run, whether or not anything is wrong.

## Expected Behaviour

The warning should indicate an **actual abnormality** — a detached grandchild
holding the stdout pipe write end open past process exit, which is what its
message claims.

## Actual Behaviour

It fires for essentially every non-trivial run. In the current log, all 8
occurrences map to runs of 149 s–2532 s, and **7 of the 8 completed
successfully** (`status=done`, `exit_code=0`):

```
  1cc491a2ab  done exit=0 wall=2532s
  225f10dc92  done exit=0 wall=259s
  610c74b2ee  done exit=0 wall=1578s
  8c648afcae  done exit=0 wall=149s
  a1e2f7ef87  done exit=0 wall=1209s
  dc92ccf089  done exit=0 wall=1004s
  ff7c97ac53  done exit=0 wall=654s
  e698997b31  failed exit=-1 wall=1128s
```

## Root cause

In `internal/agent/agent.go` the watcher goroutine is started **at process
launch**, not at exit:

```go
readersDone := make(chan struct{})
go func() { wg.Wait(); close(readersDone) }()
select {
case <-readersDone:
case <-time.After(readerDrainGrace):   // readerDrainGrace = 5 * time.Second
        slog.Warn("agent: stdout readers did not drain within grace; reaping anyway (possible detached grandchild holding the pipe)", ...)
}
err := cmd.Wait()
```

The stdout readers only reach EOF when the process exits and closes the pipe.
So for any run longer than the 5-second grace the timer necessarily wins, the
warning is logged, and `cmd.Wait()` then blocks normally until real exit. The
grace is effectively "has this run finished within 5 seconds?", not "is a
grandchild holding the pipe?".

## Impact

Log noise, but of the actively harmful kind: the message asserts a specific
abnormal cause ("possible detached grandchild holding the pipe") that is
usually false. While investigating
[[agent-run-wall-clock-exceeds-reported-duration]] this warning appeared to be
the smoking gun for a 16-minute stall and was nearly recorded as its cause; it
was only excluded by cross-referencing the other run IDs and finding they had
all succeeded.

## Fix guidance

The intent — don't let a pipe-holding grandchild hang the reaper — is sound;
only the signalling is wrong. Options, roughly in order of preference:

- **Start the grace at process exit, not at launch.** Wait for the process to
  exit first, then give the readers a bounded grace to drain. Then the warning
  means what it says.
- If the current structure must be kept, **only warn when the grace expires
  *and* the process has already exited**, and downgrade the ordinary
  still-running case to debug (or drop it).
- Re-word the message so it does not assert a cause that has not been
  established.

Worth a test that a normal long-running agent produces **no** WARN.
