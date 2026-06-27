---
title: Agent runs burn the full retry budget on transient auth (401) failures instead of failing fast and re-queuing
type: defect
status: draft
lineage: agent-auth-error-fail-fast
created: "2026-06-27T00:00:00+10:00"
priority: medium
labels:
    - defect
    - agent
    - queue
    - reliability
    - cost
release: KC-Release4
assignees:
    - role: backend-developer
      who: agent
---

# Agent runs burn the full retry budget on transient auth (401) failures instead of failing fast and re-queuing

## Reproduction Steps

1. Run a Claude-driver agent on a host where another Claude Code client (an
   interactive VS Code session, the Claude desktop app, etc.) shares the same
   Anthropic OAuth login.
2. While a long agent run is in flight, have one of those other clients refresh
   the OAuth token (the refresh token rotates), stranding the agent's process on
   the now-invalidated token.
3. Observe the run (real example: `215bc5d8c2773b49`, `test-developer`):
   - Claude emits repeated `{"type":"system","subtype":"api_retry", ...,
     "error_status":401,"error":"authentication_failed"}` events (72 auth/retry
     mentions in that log).
   - It grinds through Claude's 10-attempt retry budget with exponential
     backoff, then terminates with
     `{"type":"result","subtype":"success","is_error":true,
     "result":"Not logged in · Please run /login"}`.

## Expected Behaviour

A transient auth failure (401 `authentication_failed` / "Not logged in") should
**fail the run fast and re-enqueue it** — a retry picks up the freshly-rotated
token and succeeds (subsequent runs on the same host do). The queue should NOT
pause (this is not a rate limit), and the run should not waste wall-clock/cost.

## Actual Behaviour

The run stays alive for the full retry budget — **~58 minutes wall-clock and
$1.08** for `215bc5d8c2773b49` — doing ~7 min of real work
(`duration_ms=420730`, `duration_api_ms=320050`) before failing. kaos-control
correctly records `status=failed` (`exit_code=-1`) **after** the binary gives
up, but it does nothing to detect the auth failure early or to retry the run.
The expensive, slow dead-run is the symptom.

It also reads confusingly in the UI: the terminal result line is
`subtype:"success"` with `is_error:true`, so a run-detail summary can show a
"success"-looking envelope next to the failed status.

## Root Cause

Auth failures are **transient and non-recoverable within a session** (the
process holds a rotated-out OAuth token), but kaos-control has no handling for
them:

- The supervisor's broadcast closure
  ([internal/agent/agent.go:736-745](../../internal/agent/agent.go#L736)) detects
  rate-limit / overload payloads via `extractRateLimitText` and re-broadcasts
  `queue.rate_limit`, but there is no equivalent for `api_retry` events carrying
  `error_status:401` / `error:"authentication_failed"`.
- So the run is left to Claude's internal retry loop (up to 10 attempts) and is
  only marked failed when the process finally exits.

This is distinct from rate-limit handling
([internal/queue/dispatcher.go:400](../../internal/queue/dispatcher.go#L400),
`handleRateLimit`): a rate limit should **pause** the whole queue until reset; an
auth-rotation 401 should **not** pause — it should kill just this run and
re-enqueue it immediately.

## Suggested Fix

1. Detect auth failures in the supervisor's event stream — `api_retry` events
   with `error_status == 401` / `error == "authentication_failed"`, and/or the
   terminal `type:result` with `is_error:true` and a "Not logged in" / login
   `result` — mirroring how `extractRateLimitText` is used.
2. On detection (e.g. after N consecutive auth retries, to avoid reacting to a
   single blip), **kill the run early** (don't wait out the retry budget) and
   broadcast a new event (e.g. `queue.auth_error`).
3. In the dispatcher, treat `queue.auth_error` as **fail + immediate
   re-enqueue** (bounded by `max_attempts`), **without** pausing the queue — a
   short pause/jitter is fine to let token rotation settle, but no long
   rate-limit pause.
4. (UI nicety) When `is_error:true`, don't surface the result `subtype:"success"`
   as a success indicator — show the `result` error text instead.

## Mitigation (not the fix, but recommended)

Give agents their own credential so they don't share the rotating OAuth token
with interactive/desktop Claude clients: point them at a dedicated
`ANTHROPIC_API_KEY` via the existing **`claude-env`** driver
(`Run.BaseURL` / `Run.AuthToken`). API keys don't rotate, so the race disappears.

## Verification

- Inject an `api_retry`/`authentication_failed` event sequence into a fake-claude
  fixture and assert the supervisor kills the run early and the dispatcher
  re-enqueues it (attempts++), without setting the queue paused state.
- Assert a single isolated auth blip does not trigger a kill (threshold).
