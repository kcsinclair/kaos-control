---
title: "Tests — Detect rate_limit_event for Precise Quota Signalling"
type: test
status: draft
lineage: rate-limit-event-detection
parent: lifecycle/test-plans/rate-limit-event-detection-5-test.md
created: "2026-08-15T00:00:00+10:00"
labels:
    - agent
    - queue
    - observability
    - test
release: KC-Release5
---

# Tests — Detect `rate_limit_event` for Precise Quota Signalling

Implements the integration-level slice of the companion test plan
(`-5-test`). Per `CLAUDE.md`'s role split, `test-developer`'s write scope is
`tests/**` (integration tests) + this artifact — co-located Go unit tests in
`internal/agent`/`internal/queue` and frontend Vitest are `backend-developer`
/`frontend-developer` responsibilities. See **Coverage map** below for exactly
which test-plan milestones/ACs are covered here versus elsewhere, and the one
gap that remains open.

---

## New files

- `tests/integration/quota_status_test.go` — Mode 1 (observability)
- `tests/integration/queue_rate_limit_precise_test.go` — Mode 2 (precise reset)

Both drive the real HTTP + project-hub stack with a fake `claude` binary
emitting crafted stream-json lines (no white-box access to
`extractRateLimitInfo` / `Manager.runQuota` / `handleRateLimit`), so every
assertion is on externally observable behaviour: WebSocket broadcast payloads
and `GET /api/queue` snapshot state.

## Coverage map

| AC / NFR | Covered by | Notes |
|---|---|---|
| AC1 — parse the captured event | `TestQuotaStatus_AC1_ExactSamplePayload` | replays the requirement's exact sample payload verbatim |
| AC1 — non-`rate_limit_event` / missing `rate_limit_info` → no broadcast | `TestQuotaStatus_AC1_NonRateLimitEventPayload_NoBroadcast` | malformed lines produce zero broadcasts and don't block a later valid one |
| AC2 — weekly / unknown bucket | `TestQuotaStatus_AC2_BucketDiscrimination` | subtests `weekly`, unrecognised → `unknown` |
| AC3 — defensive parse, missing fields | `TestQuotaStatus_AC3_DefensiveParse_MissingFields` | `resetsAt`/`status`/overage fields all absent; no panic, `resets_at` omitted, `status="unknown"` |
| AC4 — broadcast on stream | `TestQuotaStatus_AC1_ExactSamplePayload` | asserts exactly one `agent.quota_status` |
| AC5 — content-change debounce | `TestQuotaStatus_AC5_ContentChangeDebounce` | table of 5 subtests, one per debounce-tuple field (`bucket`, `status`, `resets_at`, `overage_available`, `overage_disabled_reason`) |
| AC6 — per-run cache cleared | **not covered here** | see Known gap below |
| AC7 — Mode-2 precise reset preferred | `TestQueue_RateLimit_PrecisePreferred` | precise `resetsAtUnix` wins over a materially different `ParseResetTime` result; asserts exact `paused_until` |
| AC7 — fallback when `resets_at_unix` absent | pre-existing `tests/integration/queue_rate_limit_test.go` (`TestQueue_RateLimit_FromSampleLog`, `TestQueue_RateLimit_FallbackOnUnparseable`) | unchanged by this feature — these fixtures never emit a `rate_limit_event`, so they continue to exercise the untouched `ParseResetTime` path, proving FR8 "byte-identical" behaviour |
| AC7 — overloaded kind, no `resets_at_unix` → `OverloadPause` | `TestQueue_RateLimit_OverloadedFallback_NoResetsAtUnix` | distinguishes `OverloadPause` (~5min) from the longer rate-limit `FallbackPause` (~30min) via a wall-clock band, since the integration env has no clock injection |
| AC8 — no `rate_limit_event` → zero `agent.quota_status` | `TestQuotaStatus_AC8_NoRateLimitEvent_ZeroBroadcasts` | Mode-1 half |
| AC8 — Format-3 denial, no prior event → `ParseResetTime` as today | pre-existing `TestQueue_RateLimit_FromSampleLog` | Mode-2 half, unchanged path |
| NFR1 — additive WS surface | `TestQuotaStatus_NFR1_ExistingEventShapesUnchanged` | `agent.progress` payload keys (`run_id`/`line`/`raw`/`event`) unaltered alongside a quota broadcast |
| NFR2 — timezone-free, RFC3339-UTC | `TestQuotaStatus_AC1_ExactSamplePayload` | asserts the exact UTC string for the requirement's sample `resetsAt` |
| NFR3 — vendor coupling isolated (non-Claude driver unaffected) | `TestQuotaStatus_NFR3_OllamaDriver_NoQuotaEvents` | an Ollama-driven run emits zero `agent.quota_status` |
| NFR4 — forward-compatible unknown values | `TestQuotaStatus_NFR4_UnknownValuesSurfaceAsUnknown`, `TestQuotaStatus_OverageAvailableTruthTable` | novel `status`/`overageStatus` values surface as `unknown`/best-effort without dropping the event |

### Known gap — AC6 (per-run quota cache cleared on `cleanupRunState`)

`runQuota` is private state on `internal/agent.Manager`; there is no HTTP/WS
surface that observes its removal directly (a terminated run emits no further
events by definition, so there's nothing to black-box-assert against). Closing
this needs a package-internal test alongside the existing
`runPolicies`/`deniedCalls` cleanup tests, mirroring the backend plan's
Milestone 2 acceptance criterion ("after `cleanupRunState(runID)` the
`runQuota` entry for that run is gone (assertable directly...)") — that test
does not yet exist in `internal/agent/*_test.go`. This is out of
`test-developer`'s `tests/**` write scope for this run; flagging here rather
than silently claiming full AC6 coverage.

### Frontend (Milestone 6 of the test plan)

Already fully covered by `frontend-developer`'s own Vitest suite (not
duplicated here):
`web/src/stores/__tests__/agents.spec.ts` (quota cache populate/replace/clear)
and `web/src/api/__tests__/ws.test.ts` (no-consumer graceful handling).

## Exit criteria status

- `make lint` — clean (`go vet` clean on the new files; no existing test
  modified).
- `go test -tags=integration -race ./tests/integration/... -run
  'TestQuotaStatus|TestQueue_RateLimit'` — 16/16 pass.
- All pre-existing `tests/integration/queue_rate_limit_test.go` tests still
  pass unmodified (additive-only, per NFR1/FR8).
