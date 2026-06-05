---
title: "Successful Claude agent runs reported FAILED (false truncated_stream)"
type: defect
status: draft
lineage: agent-truncated-stream-detection
created: "2026-06-05T00:00:00+10:00"
labels:
  - defect
assignees:
  - role: backend-developer
    who: agent
---

## Reproduction Steps

1. Run any `claude-code-cli` (or `claude-mediated`) agent on a deployment built
   from commit `8392a946` or later (e.g. sol.packsin.com).
2. Let the agent complete its task normally — the on-disk run log ends with a
   `{"type":"result","subtype":"success","is_error":false,...}` event and the
   process exits 0.
3. Observe that kaos-control records the run with `status=failed` and
   `failure_reason=truncated_stream`, and broadcasts `agent.failed`.

A captured example: a `qa` agent run that read its test artifact, ran vitest,
created two defect files, and emitted a clean `result` event with
`is_error:false` — yet was reported FAILED.

## Expected Behaviour

A run that exits 0 and emits a terminal `result` event should be recorded as
`status=done`. The truncated-stream check (added in `8392a946`) should only
fire when the stream genuinely ends without a `result` event.

## Actual Behaviour

Every clean `claude-code-cli` / `claude-mediated` run is downgraded
`done → failed` with `failure_reason=truncated_stream`.

Root cause: the truncated-stream check at
[internal/agent/agent.go:807](internal/agent/agent.go#L807) requires
`resultEventSeen`, which is only flipped inside the `broadcast` closure for
events actually read off the progress channel. But `runPrecheck` /
`runMediatedPrecheck` stop reading the channel the instant they see the
`system/init` event ([internal/agent/precheck.go:121-122](internal/agent/precheck.go#L121-L122)),
and `supervise` has no drain loop on the precheck-pass path
([internal/agent/agent.go:720-768](internal/agent/agent.go#L720-L768)). Every
post-init event — including the terminal `result` — is left unread in the
buffered progress channel (`cap 64`), so `resultEventSeen` stays `false` and
the check flips the status to failed.

Two further consequences of the same missing drain:
- A run emitting more than 64 events fills the buffer, blocks the stdout
  reader, back-pressures Claude's stdout pipe, and stalls the run until
  timeout.
- Post-init events never reach WebSocket subscribers, so the live agent view
  is missing all tool calls after `init`.

The commit's unit tests (`TestDriverEmitsResultEvent`, `TestIsResultEvent`)
only exercise the predicate functions in isolation; nothing drives `supervise`
end-to-end, so the regression was not caught.

## Logs / Output

```
agent: stream-json run exited cleanly without emitting a terminal result event
  — marking failed (truncated stream)
  run_id=... agent=qa driver=claude-code-cli
```

…despite the run log containing:

```json
{"type":"result","subtype":"success","is_error":false,"terminal_reason":"completed",...}
```

## Fix

After the precheck passes, `supervise` must continue draining and forwarding
the remaining stream events through the existing `broadcast` closure (which
sets `resultEventSeen` and feeds the WS view). This also removes the >64-event
stall. Add a `supervise`-level regression test with a fake process that emits
`init` + `result` and assert the run ends `done`.
