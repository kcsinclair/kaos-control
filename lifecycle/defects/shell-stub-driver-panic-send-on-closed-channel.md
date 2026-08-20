---
title: "ShellStubDriver panics with \"send on closed channel\" under concurrent integration test load"
type: defect
status: approved
lineage: shell-stub-driver-panic-send-on-closed-channel
parent: internal/agent/shell_stub.go
created: "2026-08-20T12:22:00+10:00"
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# ShellStubDriver panics with "send on closed channel" under concurrent integration test load

## Reproduction Steps

1. `make test-integration` (runs `go test ./... -count=1 -tags=integration`).
2. Run repeatedly if it doesn't reproduce on the first try — this is a goroutine race and is timing-dependent (it surfaced after ~4 minutes / thousands of requests into this run, at `TestRehydrateSkipsInvalidFrontmatter`'s neighbourhood, but the panic is not specific to that test — any test exercising `ShellStubDriver.Start` concurrently can trigger it).
3. Observe the whole `tests/integration` test binary crash with an unrecovered panic, which aborts every remaining test in that binary (not just one subtest).

## Expected Behaviour

`ShellStubDriver.Start` (`internal/agent/shell_stub.go:27`) runs its stdout- and stderr-reading goroutines to completion without panicking, regardless of which one finishes first.

## Actual Behaviour

`internal/agent/shell_stub.go` starts two goroutines per process:
- one (`shell_stub.go:53-67`) scans `stdout` and sends `ProgressEvent`s on `progressCh`
- another (`shell_stub.go:69-81`) drains `stderr` and, when done, unconditionally does `defer close(progressCh)` (`shell_stub.go:70`)

These two goroutines are not synchronized. If the stderr-drain goroutine finishes (and closes `progressCh`) before the stdout-scan goroutine's next send, the stdout goroutine panics with `send on closed channel` at the `case progressCh <- ev:` on line 63. Because this is an unrecovered panic in a background goroutine, it crashes the entire test process rather than failing a single test.

## Logs / Output

```
panic: send on closed channel

goroutine 188002 [running]:
github.com/kaos-control/kaos-control/internal/agent.(*ShellStubDriver).Start.func1()
	/Users/keith/Code/kaos-control/internal/agent/shell_stub.go:63 +0x98
created by github.com/kaos-control/kaos-control/internal/agent.(*ShellStubDriver).Start in goroutine 187871
	/Users/keith/Code/kaos-control/internal/agent/shell_stub.go:53 +0x274
FAIL	github.com/kaos-control/kaos-control/tests/integration	239.123s
```

## Fix guidance

Only the goroutine that owns the *last* writer to `progressCh` should close it, and only after all writers have stopped. Options:
- Have the stdout-scan goroutine be the one that closes `progressCh` after `sc.Scan()` returns (it's the only writer), and have the stderr-drain goroutine not touch the channel at all.
- Or synchronize both goroutines with a `sync.WaitGroup` and close `progressCh` only after both have returned, from a third coordinating goroutine (or the caller after `cmd.Wait()`).
Add a regression test that runs many concurrent `ShellStubDriver.Start` calls (ideally under `go test -race`) to catch this class of bug going forward.
