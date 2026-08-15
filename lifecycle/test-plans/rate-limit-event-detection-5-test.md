---
title: Test Plan — Detect rate_limit_event for Precise Quota Signalling
type: plan-test
status: approved
lineage: rate-limit-event-detection
parent: lifecycle/requirements/rate-limit-event-detection-2.md
created: "2026-08-15T00:00:00+10:00"
labels:
    - agent
    - queue
    - observability
    - test
release: KC-Release5
---

# Test Plan — Detect `rate_limit_event` for Precise Quota Signalling

Parent requirement: [[rate-limit-event-detection]]
(`lifecycle/requirements/rate-limit-event-detection-2.md`).
Companion plans: [[rate-limit-event-detection]] backend (`-3-be`) and frontend
(`-4-fe`).

## Strategy

Coverage maps 1:1 to the requirement's acceptance criteria AC1–AC8. The bulk is
Go unit/component tests co-located with the code under test:
- Parser + defensive cases → table tests in `internal/agent/agent_test.go`
  (extend the existing `extractRateLimitText` table pattern at
  `agent_test.go:74`).
- Broadcast + debounce + cache lifecycle → `supervise` fixture tests using the
  existing `superviseTestRun` helper (`internal/agent/precheck_test.go:340`) and
  a test hub to capture emitted events.
- Dispatcher precise-vs-fallback reset → `internal/queue` dispatcher tests with
  an injected clock (`d.cfg.clock()`) and the existing `ParseResetTime` fixtures
  (`internal/queue/parser_test.go`) as the fallback oracle.
- Frontend store/dispatch behaviour → Vitest over the Pinia store and WS
  dispatcher.

All new Go tests must pass under `go test -race`. No existing assertion is
modified (additive-only, per NFR1/FR8).

---

## Milestone 1 — Parser unit tests (AC1, AC2, AC3, NFR4)

**Description.** Table-driven tests for `extractRateLimitInfo`, wrapping each
event JSON in an `agent.progress` payload map exactly as the existing helper at
`agent_test.go:77` does.

**Files to change.**
- `internal/agent/agent_test.go` — new `TestExtractRateLimitInfo` table.

**Acceptance criteria / cases.**
- **AC1**: the requirement's exact sample payload → `ok=true`,
  `Bucket="five_hour"`, `Status="allowed"`, `ResetsAtUnix=1778911200`,
  `OverageAvailable=false`, `OverageDisabledReason="out_of_credits"`.
- Non-`rate_limit_event` (a normal assistant/result event) → `ok=false`.
- Payload missing the `rate_limit_info` object → `ok=false`.
- **AC2**: `rateLimitType:"weekly"` → `Bucket="weekly"`; `rateLimitType:"lunar"`
  → `Bucket="unknown"`.
- **AC3**: event missing `resetsAt` → `ResetsAtUnix=0`; missing `status` →
  `Status="unknown"`; missing `overageStatus`/`isUsingOverage` →
  `OverageAvailable` computed without panic. No case panics.
- **NFR4**: unknown `status` (e.g. `"throttled"`) and unknown `overageStatus`
  surface as `unknown`/best-effort, `ok=true`, event not dropped.
- `OverageAvailable` truth table: `isUsingOverage:true` → `true`;
  `isUsingOverage:false, overageStatus:"rejected"` → `false`;
  `isUsingOverage:false, overageStatus:"available"` → `true`.

---

## Milestone 2 — Broadcast + content-change debounce (AC4, AC5)

**Description.** Drive fixture `rate_limit_event`s through `supervise` and assert
the `agent.quota_status` events captured on a test hub. Add a
`ProgressEvent`-emitting fixture stream (extend `superviseTestRun` /
`precheck_test.go` fixtures) that yields a sequence of `rate_limit_event`
payloads.

**Files to change.**
- `internal/agent/agent_test.go` (or a new `quota_status_test.go`) — supervise
  fixture tests; a hub recorder capturing `agent.quota_status` payloads.

**Acceptance criteria / cases.**
- **AC4**: a stream with one `rate_limit_event` → exactly one
  `agent.quota_status` with the normalised payload; `resets_at` is the
  RFC3339-UTC render of `resetsAt` (assert exact string for
  `1778911200`), and equals UTC (NFR2).
- **AC5**: `[evA, evA]` (identical tuple) → one broadcast; `[evA, evA, evB]`
  where `evB` differs in exactly one tuple field (each of `bucket`, `status`,
  `resets_at`, `overage_available`, `overage_disabled_reason` exercised in
  sub-cases) → a second broadcast.
- `ResetsAtUnix=0` → `resets_at` key omitted from the payload.
- **NFR1**: no `agent.progress`, `queue.rate_limit`, or other existing event
  payload is altered by the presence of quota events (assert their shapes are
  unchanged in the same recorder).

---

## Milestone 3 — Per-run quota cache lifecycle (AC6)

**Description.** Unit test the `Manager` cache accessors and cleanup, mirroring
the existing `runPolicies`/`deniedCalls` cleanup test style.

**Files to change.**
- `internal/agent/agent_test.go` or `manager_test.go`.

**Acceptance criteria / cases.**
- After `setRunQuota(runID, info)`, `getRunQuota(runID)` returns `(info, true)`.
- **AC6**: after `cleanupRunState(runID)`, `getRunQuota(runID)` returns
  `(_, false)` — entry removed.
- Concurrent `setRunQuota`/`getRunQuota`/`cleanupRunState` across goroutines is
  clean under `-race`.
- A supervise run that emits a `rate_limit_event` and then reaches terminal
  state leaves no `runQuota` entry (end-to-end of FR5/AC6).

---

## Milestone 4 — Dispatcher prefers precise reset (AC7)

**Description.** Test `handleRateLimit` / `watchRunEvents` with an injected
clock. Verify the precise path bypasses `ParseResetTime` and the fallback path
matches current behaviour.

**Files to change.**
- `internal/queue/dispatcher_test.go` (extend existing rate-limit tests).

**Acceptance criteria / cases.**
- **AC7 (precise)**: a `queue.rate_limit` with `resets_at_unix=T` and a
  `raw_text` that `ParseResetTime` would parse to a *different* time → the queue
  pauses until `time.Unix(T,0) + resumeGrace`, proving the typed value won
  (assert `paused_until`/`setPausedUntil` argument equals the precise value, not
  the text-parsed one).
- **AC7 (fallback)**: a `queue.rate_limit` with `resets_at_unix=0`/absent and a
  parseable `raw_text` → pause equals `ParseResetTime(raw_text, now)+grace`,
  byte-identical to a control run of the pre-change path.
- `watchRunEvents` decodes `resets_at_unix` from the hub JSON into
  `runResult.ResetsAtUnix` and threads it to `handleRateLimit`.
- `overloaded` kind with no `resets_at_unix` still uses `OverloadPause`.

---

## Milestone 5 — Degradation / no-event runs (AC8, FR8)

**Description.** Confirm the whole feature is inert when no `rate_limit_event`
appears — the universal text fallback is untouched.

**Files to change.**
- `internal/agent/*_test.go`, `internal/queue/dispatcher_test.go`.

**Acceptance criteria / cases.**
- **AC8**: a supervise run whose stream contains no `rate_limit_event` emits
  **zero** `agent.quota_status` events.
- A Format-3 denial with no prior `rate_limit_event` → `queue.rate_limit` has no
  usable `resets_at_unix`, and `handleRateLimit` pauses via `ParseResetTime`
  exactly as today.
- **FR8**: existing `parser_test.go` rows are unchanged and still pass;
  `ParseResetTime` is not modified.
- A non-Claude driver (Ollama fixture) run emits no quota events (NFR3).

---

## Milestone 6 — Frontend tests (supports [[rate-limit-event-detection]] `-4-fe`)

**Description.** Vitest coverage for the store cache and graceful handling.

**Files to change.**
- `web/src/` `__tests__` for the run/agent Pinia store and WS dispatcher.

**Acceptance criteria / cases.**
- Dispatching a mocked `agent.quota_status` populates `quotaForRun(run_id)`; a
  second event replaces it; a terminal run event clears it.
- Receiving `agent.quota_status` with no consumer produces no `console.warn` and
  does not interrupt a following `agent.progress` dispatch (NFR1).
- `vue-tsc` type-checks the new discriminated-union member.

---

## Exit criteria (feature done)

- AC1–AC8 all covered by passing tests.
- `make lint`, `make test-unit`, and
  `go test -race ./internal/agent/... ./internal/queue/...` green.
- `pnpm build` + frontend unit tests green.
- No pre-existing test expectation weakened or removed (additive-only).
