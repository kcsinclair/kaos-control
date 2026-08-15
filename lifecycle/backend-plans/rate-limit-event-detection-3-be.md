---
title: Backend Plan — Detect rate_limit_event for Precise Quota Signalling
type: plan-backend
status: in-development
lineage: rate-limit-event-detection
parent: lifecycle/requirements/rate-limit-event-detection-2.md
created: "2026-08-15T00:00:00+10:00"
labels:
    - agent
    - queue
    - observability
    - backend
release: KC-Release5
---

# Backend Plan — Detect `rate_limit_event` for Precise Quota Signalling

Parent requirement: [[rate-limit-event-detection]]
(`lifecycle/requirements/rate-limit-event-detection-2.md`).
Companion plans: [[rate-limit-event-detection]] frontend (`-4-fe`) and test
(`-5-test`).

## Architecture conformance

Verified against `lifecycle/architecture/` before planning. That directory
currently ships only the **catalog** (`architectures/`, `tech-stacks/`,
`README.md`); this project has not yet promoted an `architecture-summary.md`,
`decisions/`, or `standards/` of its own. The project's de-facto architecture
is **Local Web Application** + **Go + Vue** (per the catalog entries and
`CLAUDE.md`): a single Go binary serving an embedded SPA, with a per-project
WebSocket hub as the live event channel.

All work here is **strictly additive** within existing packages
(`internal/agent`, `internal/queue`): one new stream shape parsed behind an
isolated helper, one new additive hub event, one new optional field on an
internal struct. It introduces no new service, datastore, dependency, or
cross-boundary coupling. **No architecture deviation → no new ADR required.**
The vendor-specific `rate_limit_event` shape stays isolated behind
`extractRateLimitInfo` and is only reachable for stream-json Claude drivers
(NFR3), preserving the existing driver-abstraction boundary.

## Grounding (verified in code)

- `extractRateLimitText(payload map[string]any) (rawText, kind, ok)` —
  `internal/agent/agent.go:1578`. Handles Formats 1–3; **does not** handle the
  mid-stream `rate_limit_event` shape.
- Supervisor broadcast closure — `internal/agent/agent.go:799`. Inspects each
  `agent.progress` event; already re-broadcasts `queue.rate_limit` (with
  `run_id`, `raw_text`, `kind`) at `agent.go:804`. This is the single choke
  point every forwarded stream event passes through (precheck + drain loops).
- Per-run state maps on `Manager` — `runPolicies` / `deniedCalls` declared at
  `agent.go:405`, initialised in `New` at `agent.go:462`, cleared in
  `cleanupRunState` at `agent.go:1366`. `cleanupRunState` is called at
  `agent.go:904`, **after** the drain loop (so the cache is still live when the
  terminal Format-3 result passes through the broadcast closure — this ordering
  is what makes FR6 work).
- Dispatcher: `runResult` struct — `internal/queue/dispatcher.go:322`;
  `watchRunEvents` decodes hub events into `runResult` at `dispatcher.go:342`
  (JSON struct at `:364`, `queue.rate_limit` case at `:403`); `handleRateLimit`
  — `dispatcher.go:427`, calls `ParseResetTime` at `:429`, adds `resumeGrace()`
  at `:439`.
- `ParseResetTime(text, now)` — `internal/queue/parser.go:28`. Unchanged by this
  work (FR8).

## Sequencing

Ship **Mode 1** (M1–M4: parse + observability broadcast) before **Mode 2**
(M5–M6: dispatcher plumbing), matching the requirement's "smallest viable
slice" (FR1–FR5 first). Each milestone is independently mergeable and leaves the
build green.

---

## Milestone 1 — `extractRateLimitInfo` parser + typed struct (FR1, FR2)

**Description.** Add a `RateLimitInfo` struct and an
`extractRateLimitInfo(payload map[string]any) (RateLimitInfo, bool)` helper that
parses a decoded `rate_limit_event` `agent.progress` payload. Parsing is
defensive: unknown `rateLimitType`/`status` map to `"unknown"`; missing numeric
fields default to `0`; `ok=false` for any event whose inner
`event.type != "rate_limit_event"` or that lacks a `rate_limit_info` object.
Note the payload nesting mirrors `extractRateLimitText`: the real event lives
under `payload["event"]` (see `agent.go:1579`), so read
`event["rate_limit_info"]`.

Field mapping (per FR1 table):

| Struct field | Source | Rule |
|---|---|---|
| `Bucket string` | `rate_limit_info.rateLimitType` | `five_hour`/`weekly`, else `unknown` |
| `Status string` | `rate_limit_info.status` | `allowed`/`warning`/`rejected`, else `unknown` |
| `ResetsAtUnix int64` | `rate_limit_info.resetsAt` | JSON number → int64; `0` if absent |
| `OverageAvailable bool` | `isUsingOverage` OR `overageStatus != "rejected"` | best-effort |
| `OverageDisabledReason string` | `overageDisabledReason` | free-form, may be `""` |

Guard the numeric read: JSON decodes numbers as `float64`, so
`resetsAt` must be read as `float64` (or `json.Number`) and converted, never
type-asserted directly to `int64`.

**Files to change.**
- `internal/agent/agent.go` — add `RateLimitInfo` type (near
  `RateLimitKind`, ~`:1546`) and `extractRateLimitInfo` (near
  `extractRateLimitText`, ~`:1578`), plus small bucket/status normalisation
  helpers.

**Acceptance criteria.**
- **AC1**: `extractRateLimitInfo` on the requirement's sample payload returns
  `ok=true`, `Bucket="five_hour"`, `Status="allowed"`,
  `ResetsAtUnix=1778911200`, `OverageAvailable=false`,
  `OverageDisabledReason="out_of_credits"`.
- A payload whose `event.type != "rate_limit_event"` (e.g. a normal assistant
  event) returns `ok=false`.
- **AC2**: `rateLimitType:"weekly"` → `Bucket="weekly"`; an unrecognised value
  → `Bucket="unknown"`.
- **AC3**: a `rate_limit_event` missing `resetsAt`/`overageStatus`/`status`
  parses without panic, yielding `ResetsAtUnix=0` and `Status="unknown"`.
- **NFR4**: novel `rateLimitType`/`status`/`overageStatus` values surface as
  `"unknown"` and never drop the event or panic.
- `make lint` clean; no change to any existing exported signature.

---

## Milestone 2 — Per-run quota cache on `Manager` (FR5, AC6)

**Description.** Add a `runQuota map[string]RateLimitInfo` field to `Manager`
(guarded by the existing `m.mu`), initialise it in `New`, and delete the run's
entry in `cleanupRunState`. This cache holds the most recent parsed
`rate_limit_info` per `run_id`; it backs both the FR4 debounce comparison
(M3) and the FR6 Mode-2 lookup. Add small mutex-guarded accessors
(`setRunQuota(runID, RateLimitInfo)` / `getRunQuota(runID) (RateLimitInfo, bool)`)
so the broadcast closure never touches `m.mu` inline.

Note the cleanup ordering: `cleanupRunState` runs at `agent.go:904`, *after* the
drain loop finishes forwarding every stream event — so the terminal Format-3
event (M4) still sees a live cache entry before it is cleared.

**Files to change.**
- `internal/agent/agent.go` — field at `Manager` (~`:406`), init in `New`
  (~`:462`), delete in `cleanupRunState` (~`:1368`), accessor helpers.

**Acceptance criteria.**
- **AC6**: after `cleanupRunState(runID)` the `runQuota` entry for that run is
  gone (assertable directly, matching the existing `runPolicies` cleanup test
  style).
- Accessors are race-free under `-race` (covered by M-tests in `-5-test`).
- `runQuota` is initialised in `New`; a nil-map write can never occur.

---

## Milestone 3 — `agent.quota_status` broadcast with content-change debounce (FR3, FR4)

**Description.** In the supervisor broadcast closure (`agent.go:799`), after the
existing `extractRateLimitText` branch, add: if
`info, ok := extractRateLimitInfo(payload); ok`, compare `info` against the
cached value from M2 for this `run_id`. Broadcast a new `agent.quota_status`
hub event **only** when the debounce tuple
`(Bucket, Status, ResetsAtUnix, OverageAvailable, OverageDisabledReason)`
differs from the last broadcast for the run (or none exists yet); then update
the cache via `setRunQuota`. Identical consecutive events are suppressed.

Payload (FR3), with `resets_at` = RFC3339-UTC of `ResetsAtUnix`, **omitted when
`ResetsAtUnix == 0`** (NFR2 — always UTC, never localised):

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

Update the cache on **every** parsed `rate_limit_event` even when the broadcast
is suppressed, so the M4 Mode-2 lookup always reflects the latest signal.
Render `resets_at` with `time.Unix(sec, 0).UTC().Format(time.RFC3339)`.

**Files to change.**
- `internal/agent/agent.go` — extend the broadcast closure (~`:801`–`:838`);
  a small helper to build the payload + RFC3339-UTC render.

**Acceptance criteria.**
- **AC4**: a run whose stream carries one `rate_limit_event` produces exactly
  one `agent.quota_status` hub event with the normalised payload (verified via a
  fixture stream through `supervise` in `-5-test`).
- **AC5**: two identical consecutive `rate_limit_event`s produce **one**
  broadcast; a third differing in any tuple field produces a second.
- `resets_at` is RFC3339-UTC and is omitted when `ResetsAtUnix == 0`.
- **NFR1**: no existing event payload shape changes; `agent.quota_status` is
  purely additive. `allowed` runs proceed normally — no queue/dispatcher effect.

---

## Milestone 4 — Attach precise reset to `queue.rate_limit` on denial (FR6)

**Description.** When the existing `extractRateLimitText` branch fires `ok` (any
of Formats 1–3, including the terminal Format-3 result), look up the cached
`RateLimitInfo` for this `run_id` (M2). If `ResetsAtUnix > 0`, add
`resets_at_unix` to the `queue.rate_limit` payload alongside the existing
`raw_text` and `kind`. When the cache is empty or `ResetsAtUnix == 0`, emit the
payload exactly as today (no new key, or a zero the dispatcher ignores).

This is where the M2 cleanup ordering matters: the terminal Format-3 event is
forwarded through the broadcast closure during the drain loop (`agent.go:889`),
*before* `cleanupRunState` at `:904`, so the cached reset is still present.

**Files to change.**
- `internal/agent/agent.go` — extend the `queue.rate_limit` broadcast map
  (~`:804`–`:811`) to include `resets_at_unix` when non-zero.

**Acceptance criteria.**
- A run that saw a `rate_limit_event` with `resetsAt=T` and then hit a Format-3
  denial emits `queue.rate_limit` carrying `resets_at_unix=T`.
- A run with no prior `rate_limit_event` emits `queue.rate_limit` with no usable
  `resets_at_unix` (absent or `0`), preserving current payload shape (AC8
  precondition).
- No change to `raw_text`/`kind` semantics.

---

## Milestone 5 — Dispatcher prefers precise reset (FR7, FR8, AC7, AC8)

**Description.** Add `ResetsAtUnix int64` to `runResult`
(`dispatcher.go:322`). Extend `watchRunEvents`' inline JSON struct
(`dispatcher.go:364`) with `ResetsAtUnix int64 \`json:"resets_at_unix"\`` and
carry it into the `runResult` built in the `queue.rate_limit` case
(`dispatcher.go:405`). In `handleRateLimit` (`dispatcher.go:427`), when
`ResetsAtUnix > 0`, use `time.Unix(ResetsAtUnix, 0)` as `resetTime`
**without calling `ParseResetTime`**; otherwise fall back to
`ParseResetTime(rawText, now)` exactly as today. Apply `resumeGrace()` to the
chosen `resetTime` in both branches (unchanged at `dispatcher.go:439`).

`handleRateLimit`'s signature gains the reset value — pass it from the
`case "rate_limit"` call site at `dispatcher.go:309` (thread `result.ResetsAtUnix`
through). `ParseResetTime` and the entire text path stay byte-for-byte unchanged
(FR8).

**Files to change.**
- `internal/queue/dispatcher.go` — `runResult` field (`:322`), `watchRunEvents`
  struct + case (`:364`, `:405`), `handleRateLimit` signature/body (`:309`,
  `:427`).

**Acceptance criteria.**
- **AC7**: a `queue.rate_limit` carrying `resets_at_unix` pauses the queue until
  that instant `+ resumeGrace`, **without** invoking `ParseResetTime`; when
  `resets_at_unix` is absent/`0` the dispatcher pauses using the text-parsed
  value, byte-identical to current behaviour.
- **AC8**: a run with no `rate_limit_event` at all produces no
  `agent.quota_status` and, on a Format-3 denial, is handled via
  `ParseResetTime` exactly as today.
- Overloaded (`kind="overloaded"`) fallback path is unaffected when
  `resets_at_unix` is absent.
- `make lint` and `make test-unit` clean.

---

## Milestone 6 — Wiring verification & regression guard

**Description.** End-to-end verification that Mode 1 and Mode 2 coexist: a
single fixture stream containing an `allowed` `rate_limit_event` followed by a
Format-3 denial yields (a) one `agent.quota_status` and (b) a `queue.rate_limit`
whose `resets_at_unix` drives the pause. Confirm `-race` cleanliness of the new
cache under concurrent runs. Detailed cases live in the test plan.

**Files to change.**
- None beyond M1–M5 (this milestone is the integration checkpoint; test code
  lands under [[rate-limit-event-detection]] `-5-test`).

**Acceptance criteria.**
- Full `make lint`, `make test-unit`, and `go test -race ./internal/agent/...
  ./internal/queue/...` pass.
- No existing test in `agent_test.go` / `dispatcher` tests / `parser_test.go`
  is modified in a way that changes prior expectations (additive only).
