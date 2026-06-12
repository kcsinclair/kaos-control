---
title: Detect Claude Code rate_limit_event for Precise Quota Signalling
type: idea
status: draft
lineage: rate-limit-event-detection
priority: medium
labels:
    - agent
    - queue
    - observability
assignees:
    - role: product-owner
      who: agent
---

# Detect Claude Code `rate_limit_event` for Precise Quota Signalling

## Context

Today's rate-limit detection in
[internal/agent/agent.go](../../internal/agent/agent.go)'s
`extractRateLimitText` recognises three Claude Code stream-json shapes:

1. `{"error":"rate_limit", …}` — top-level error
2. `{"type":"error","error":{"type":"rate_limit_error", …}}` — nested error
3. `{"type":"result","is_error":true,"result":"You're out of extra usage · resets 11:10pm (Australia/Brisbane)"}` — terminal failure
   (added in `232eb4c6`)

Format 3 is the most common but the messiest: we regex-parse a
human-readable "resets HH:MMpm (Area/City)" string from a free-form
message field, then reconstruct a `time.Time` from it. The regex
already handles the documented Claude phrasings but is fragile to
phrasing drift.

While debugging quota events on 2026-05-16, Claude emitted a fourth
event shape mid-stream that we don't currently consume:

```json
{
  "type": "rate_limit_event",
  "rate_limit_info": {
    "status": "allowed",
    "rateLimitType": "five_hour",
    "resetsAt": 1778911200,
    "isUsingOverage": false,
    "overageStatus": "rejected",
    "overageDisabledReason": "out_of_credits"
  },
  "session_id": "...",
  "uuid": "..."
}
```

This event is **informational** — `status: "allowed"` means the
current call is going through. But the structure carries everything
we want: a precise Unix `resetsAt`, the bucket (`five_hour` or
weekly), and whether overage will rescue us when the bucket runs out.

## Proposal

Consume `rate_limit_event` in two distinct ways:

### Mode 1 — observability (always on)

Whenever a `rate_limit_event` arrives on the agent.progress stream,
broadcast it on the project hub as a new `agent.quota_status` event
with normalised fields:

```json
{
  "type": "agent.quota_status",
  "payload": {
    "run_id": "<run id>",
    "bucket": "five_hour" | "weekly",
    "status": "allowed" | "warning" | "rejected",
    "resets_at": "<RFC3339>",
    "overage_available": false,
    "overage_disabled_reason": "out_of_credits"
  }
}
```

The frontend can show a small "quota: 4h 12m to reset" indicator next
to running agents, or warn when overage is rejected and the run is
likely to fail at `resetsAt`.

No queue or dispatcher behaviour changes in this mode — `status:
"allowed"` runs proceed normally.

### Mode 2 — precise reset-time on actual denial

When the run *does* fail with the terminal Format-3 result event, we
have a much better source of truth than the regex: the most recent
`rate_limit_event` seen during the run carried `resetsAt` as a Unix
timestamp. Plumb that through to the queue dispatcher so
`handleRateLimit` can use the precise value directly instead of
calling `ParseResetTime` on the human-readable string.

Mechanically: cache the most recent `rate_limit_info` on the
supervisor's per-run state. When `extractRateLimitText` returns true
on Format 3, also emit the cached `resets_at_unix` alongside the
existing `raw_text` in the `queue.rate_limit` event. The dispatcher
prefers `resets_at_unix` when present; falls back to
`ParseResetTime(raw_text, now)` otherwise.

This degrades cleanly — older Claude versions that don't emit
`rate_limit_event` still go through the current text-parsing path.

## Implementation notes

- Detection lives in [internal/agent/agent.go](../../internal/agent/agent.go)
  alongside `extractRateLimitText`. New helper
  `extractRateLimitInfo` returns `(bucket, status, resetsAt, overage)`
  from the parsed event map; the supervise() broadcast closure routes
  these as `agent.quota_status`.
- Per-run cache of last `rate_limit_info` lives on `runState` in
  `Manager` (one entry per run). Cleared on `cleanupRunState`.
- Dispatcher [internal/queue/dispatcher.go](../../internal/queue/dispatcher.go)
  `handleRateLimit` accepts `ResetsAtUnix int64` on `runResult` as an
  optional override of the text-parsed reset.
- WS event type `agent.quota_status` is new — front-end can ignore it
  until a UI consumer is ready.

## Why bother

- **Robustness.** A regex on user-facing prose breaks whenever Claude
  rephrases the message. A typed event with a Unix timestamp doesn't.
- **Proactive UX.** Right now operators only learn quota is tight
  when a run dies. With this signal we can show "approaching 5-hour
  limit" hours earlier, or refuse to enqueue new work when overage is
  rejected and the bucket has minutes left.
- **Weekly limits.** Claude has both 5-hour and weekly buckets. The
  current terminal-result text doesn't always distinguish them; the
  event field `rateLimitType` does. Useful when deciding how long
  to pause the queue.
- **Telemetry hook.** A dashboard widget showing recent quota events
  per project becomes a one-table-and-one-WS-subscriber feature.

## Caveats

- **Vendor coupling.** `rate_limit_event` is a Claude-Code-specific
  shape; Ollama and any sidecar driver won't emit it. That's fine —
  it's purely additive and the existing rate-limit text path stays
  intact for everything else.
- **Event flood.** Claude may emit `rate_limit_event` frequently
  during a run. Mode 1 broadcasts should be debounced (e.g. at most
  one per minute per run) so the WS stream doesn't churn.
- **Time-zone display.** `resetsAt` is Unix-UTC; the UI must render
  in the user's TZ. Backend stays TZ-free.
- **Forward compatibility.** Claude could add more fields
  (`rateLimitType: "daily"`, new overage states). Parse defensively;
  unknown values surface as `status: "unknown"` rather than crashing.

## Effort estimate

| Piece | Effort |
|---|---|
| `extractRateLimitInfo` helper + unit tests | ~hour |
| Per-run cache + `agent.quota_status` broadcast | ~half day |
| `runResult.ResetsAtUnix` plumbing through dispatcher | ~hour |
| Debounce / event-flood guard | ~hour |
| `handleRateLimit` prefer-precise-when-available branch + test | ~hour |
| Frontend quota indicator (optional, scoped follow-up) | ~half day |
| **Backend total (no UI)** | **~1 day** |

## Smallest viable proof-of-concept

1. Add `extractRateLimitInfo` (parse the event, no plumbing yet).
   Two unit tests: status="allowed" → returns the parsed info;
   non-rate-limit event → returns false.
2. Broadcast as `agent.quota_status` on the project hub. Verify in a
   real run that the event appears in the WS stream.
3. Hold off on the dispatcher integration until the UI half is
   designed — Mode 1 alone is useful and contained.

Step 1 + 2 are about an hour of work and prove the event is reaching
us in the shape we expect before any further behaviour change.

## Resolved Questions

Resolved 2026-06-12 by reviewing the current code; four of five are settled
by the architecture, one (proactive pause) is a deferred product decision.

- **Project-scoped or app-global?** **Project-scoped.** There is one hub per
  project ([internal/project/project.go](../../internal/project/project.go),
  `hub.New()` per Project) and no app-global hub — emitting globally would
  require new cross-project infrastructure. Note the nuance: the quota is
  per-Claude-account / per-machine, *not* per-project (all projects on the box
  share one subscription/credit pool). So the model is "global truth, observed
  through whichever project is running": emit on the project hub (matches every
  other event), and if a global indicator is wanted, let the frontend treat the
  latest `quota_status` from *any* project as the account-wide state. Do not
  build a global hub for this.
- **Debounce window?** Debounce on **content-change, not on a timer.**
  `resetsAt` / `status` / `bucket` are constant between bucket boundaries, so
  broadcast only when `(status, bucket, resetsAt, overageStatus)` differs from
  the last one sent for that run. Near-zero churn *and* no dropped transitions
  (a 60 s window could swallow a boundary crossing). A per-minute floor is fine
  as a cheap secondary guard, but change-detection is the primary mechanism.
- **Proactively pause on Mode-1 signals?** **No — not in the first cut.**
  `status:"allowed"` means Claude would accept the call; refusing to start it
  trades certain throughput for an uncertain avoided failure. The Mode-2 denial
  path already re-enqueues at the head and pauses until reset
  ([handleRateLimit](../../internal/queue/dispatcher.go)), so a job that dies at
  `resetsAt` is delayed one attempt, not lost. Ship Mode 1 as pure
  observability; revisit a conservative gate (e.g. `overageStatus=rejected` AND
  `resetsAt < 2min`) only if logs later show runs dying expensively near the
  boundary.
- **Mode 2 replace `ParseResetTime` entirely?** **No — keep both,
  prefer-precise-when-present.** `rate_limit_event` is Claude-Code- and
  version-specific; older binaries and any sidecar / Ollama driver won't emit
  it. [ParseResetTime](../../internal/queue/parser.go) is the universal fallback
  for the text-only case and must stay.
- **Separate event for the weekly bucket?** **No — same shape, discriminated by
  `rateLimitType`.** In the captured payloads `rateLimitType` is a field inside
  `rate_limit_info` (`"five_hour"`); weekly arrives as `rateLimitType: "weekly"`
  in the identical structure. `extractRateLimitInfo` returning `bucket` from
  that field covers it — no second code path. Parse defensively (unknown value
  → `bucket:"unknown"`).
