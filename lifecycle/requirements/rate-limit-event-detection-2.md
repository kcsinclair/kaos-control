---
title: Detect Claude Code rate_limit_event for Precise Quota Signalling
type: requirement
status: draft
lineage: rate-limit-event-detection
parent: lifecycle/ideas/rate-limit-event-detection.md
created: "2026-06-12T00:00:00+10:00"
priority: medium
labels:
    - agent
    - queue
    - observability
    - backend
assignees:
    - role: product-owner
      who: agent
---

# Detect Claude Code `rate_limit_event` for Precise Quota Signalling

Parent: [[rate-limit-event-detection]].

## Goal

Consume the Claude Code `rate_limit_event` stream-json event in two additive
ways: (1) **observability** — re-broadcast each quota signal on the project hub
as a normalised `agent.quota_status` event so the UI can show how close a run is
to its 5-hour / weekly limit; and (2) **precise reset time** — when a run is
actually denied, prefer the typed Unix `resetsAt` from the most recent
`rate_limit_event` over the regex-parsed human-readable string. Both paths are
strictly additive: drivers and binaries that never emit the event keep their
current behaviour unchanged.

The Open Questions in the parent idea are resolved there (2026-06-12); this
requirement encodes those decisions. In particular: the event is **project-
scoped** (one hub per project — the quota is per-account but observed through
whichever project is running), debounce is **content-change based** (not a
timer), and **no proactive queue pausing** is in scope for this requirement.

## Background

`extractRateLimitText` in [internal/agent/agent.go](../../internal/agent/agent.go)
already recognises three failure shapes and returns
`(rawText, kind RateLimitKind, ok bool)`. The queue dispatcher's
`handleRateLimit` in [internal/queue/dispatcher.go](../../internal/queue/dispatcher.go)
calls `ParseResetTime` ([internal/queue/parser.go](../../internal/queue/parser.go))
on the human-readable text to decide how long to pause. The fourth event shape —
`rate_limit_event` — is emitted mid-stream and currently ignored:

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

`status: "allowed"` means the current call is going through; the structure
carries a precise Unix `resetsAt`, the bucket (`rateLimitType`), and whether
overage will rescue the run when the bucket empties.

## Functional requirements

### Detection

- **FR1 — `extractRateLimitInfo` helper.** A new helper in
  [internal/agent/agent.go](../../internal/agent/agent.go) parses a decoded
  `rate_limit_event` payload and returns a typed struct plus an `ok` bool:

  | Field | Source | Notes |
  |---|---|---|
  | `Bucket` | `rate_limit_info.rateLimitType` | `five_hour`, `weekly`, else `unknown` |
  | `Status` | `rate_limit_info.status` | `allowed`, `warning`, `rejected`, else `unknown` |
  | `ResetsAtUnix` | `rate_limit_info.resetsAt` | Unix UTC seconds; `0` if absent |
  | `OverageAvailable` | `rate_limit_info.isUsingOverage` OR `overageStatus != "rejected"` | best-effort bool |
  | `OverageDisabledReason` | `rate_limit_info.overageDisabledReason` | free-form, may be empty |

  `ok` is false for any event whose `type != "rate_limit_event"` or that lacks a
  `rate_limit_info` object. Parsing is **defensive**: unknown `rateLimitType` /
  `status` values map to `"unknown"` rather than erroring, and missing numeric
  fields default to `0` without panicking.

- **FR2 — Weekly bucket via field, not a second event.** The weekly limit is
  carried in the same shape with `rateLimitType: "weekly"`; FR1's `Bucket`
  mapping is the only handling required. No separate event type is consumed.

### Mode 1 — observability (always on)

- **FR3 — `agent.quota_status` broadcast.** When a `rate_limit_event` arrives on
  the `agent.progress` stream, the supervisor's broadcast closure in
  `supervise()` emits a new project-hub event:

  ```json
  {
    "type": "agent.quota_status",
    "payload": {
      "run_id": "<run id>",
      "bucket": "five_hour",
      "status": "allowed",
      "resets_at": "<RFC3339 UTC>",
      "overage_available": false,
      "overage_disabled_reason": "out_of_credits"
    }
  }
  ```

  `resets_at` is the RFC3339-UTC rendering of `ResetsAtUnix` (omitted/empty when
  `0`). This event changes **no** queue or dispatcher behaviour — `allowed` runs
  proceed normally.

- **FR4 — Content-change debounce.** A `agent.quota_status` is broadcast only
  when the tuple `(bucket, status, resets_at, overage_available,
  overage_disabled_reason)` differs from the last one broadcast for the same
  `run_id`. Identical consecutive events are suppressed. (Rationale: these
  fields are constant between bucket boundaries, so change-detection yields
  near-zero churn while never dropping a boundary transition — strictly better
  than a fixed time window.)

- **FR5 — Per-run quota cache.** The most recent parsed `rate_limit_info` for a
  run is cached on the `Manager` keyed by `run_id` (alongside the existing
  per-run maps such as `runPolicies` / `deniedCalls`). The cache backs both the
  FR4 debounce comparison and the Mode-2 lookup. It is cleared in
  `cleanupRunState(run_id)`.

### Mode 2 — precise reset time on actual denial

- **FR6 — Prefer typed reset over regex.** When `extractRateLimitText` returns
  `ok` on the terminal Format-3 result event, the supervisor includes the cached
  `resets_at_unix` (from FR5, when non-zero) alongside the existing `raw_text`
  and `kind` in the `queue.rate_limit` hub event.

- **FR7 — Dispatcher prefers precise value.** `runResult` in
  [internal/queue/dispatcher.go](../../internal/queue/dispatcher.go) carries an
  optional `ResetsAtUnix int64`. `handleRateLimit` uses it directly as the reset
  time when present (`> 0`); otherwise it falls back to
  `ParseResetTime(rawText, now)` exactly as today. The `resume_grace` is applied
  to the chosen reset time in both cases.

- **FR8 — Clean degradation.** `ParseResetTime` and the existing text path are
  retained unchanged as the universal fallback. Older Claude binaries that emit
  Format 3 without any prior `rate_limit_event`, and any non-Claude / sidecar
  driver, continue to work via the text path. Nothing in this requirement
  removes or weakens the regex parser.

## Non-functional requirements

- **NFR1 — Additive WS surface.** `agent.quota_status` is a new event type on
  the existing per-project hub. No existing event payload changes shape. The
  frontend may ignore the event until a UI consumer ships.

- **NFR2 — Timezone-free backend.** `resetsAt` is Unix-UTC; the backend emits
  RFC3339-UTC and never localises. Any TZ rendering is the frontend's concern.

- **NFR3 — Vendor coupling is isolated.** `rate_limit_event` is a
  Claude-Code-specific shape. Detection lives behind `extractRateLimitInfo` and
  is only reachable for stream-json Claude drivers; Ollama and other drivers are
  unaffected.

- **NFR4 — Forward compatibility.** New `rateLimitType` / `status` /
  `overageStatus` values introduced by future Claude versions surface as
  `"unknown"` and must not crash parsing or drop the event.

## Acceptance criteria

- **AC1 — Parse the captured event.** `extractRateLimitInfo` on the sample
  payload above returns `ok=true`, `Bucket="five_hour"`, `Status="allowed"`,
  `ResetsAtUnix=1778911200`, `OverageAvailable=false`,
  `OverageDisabledReason="out_of_credits"`. A non-`rate_limit_event` payload
  returns `ok=false`.

- **AC2 — Weekly discrimination.** The same payload with
  `rateLimitType:"weekly"` yields `Bucket="weekly"`; an unrecognised value
  yields `Bucket="unknown"`.

- **AC3 — Defensive parse.** A `rate_limit_event` missing `resetsAt` /
  `overageStatus` parses without panic, with `ResetsAtUnix=0` and `Status`
  defaulting to `"unknown"` when absent.

- **AC4 — Broadcast on stream.** A run whose stream contains a
  `rate_limit_event` produces exactly one `agent.quota_status` hub event with
  the normalised payload (verified via a fixture stream through `supervise`).

- **AC5 — Content-change debounce.** Two identical consecutive
  `rate_limit_event`s produce **one** `agent.quota_status`; a third event
  differing in any tuple field (FR4) produces a second broadcast.

- **AC6 — Per-run cache cleared.** After a run reaches a terminal state, its
  entry in the quota cache is gone (assert via `cleanupRunState`).

- **AC7 — Mode-2 precise reset preferred.** A `queue.rate_limit` carrying
  `resets_at_unix` causes `handleRateLimit` to pause until that time (+grace)
  **without** calling `ParseResetTime`; when `resets_at_unix` is absent/zero the
  dispatcher pauses using the text-parsed value, identical to current
  behaviour.

- **AC8 — Degradation.** A run with no `rate_limit_event` at all behaves exactly
  as today: no `agent.quota_status` events, and any Format-3 denial is handled
  via `ParseResetTime`.

## Out of scope

- **Proactive Mode-1 queue pausing.** Holding/refusing a job based on an
  `allowed` signal (e.g. `overageStatus=rejected` with a near reset) is a
  deferred product decision per the parent idea — explicitly not in this
  requirement. Revisit only if logs show runs dying expensively near the
  boundary.
- **Frontend quota indicator UI.** The "quota: 4h 12m to reset" badge / global
  indicator is a separate, scoped frontend follow-up; this requirement only
  guarantees the event is on the wire.
- **Removing `ParseResetTime`.** The regex path stays as the universal fallback.
- **An app-global hub.** Aggregating quota across projects, if ever wanted, is a
  frontend concern over the existing per-project events.

## Open questions

None blocking. The proactive-pause question is intentionally deferred (see Out
of scope); the smallest viable slice is FR1–FR5 (Mode 1) shipped before the
Mode-2 dispatcher plumbing (FR6–FR8), matching the proof-of-concept sequencing
in the parent idea.
