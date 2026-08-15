---
title: Frontend Plan — Consume agent.quota_status (wire-level, no badge)
type: plan-frontend
status: in-development
lineage: rate-limit-event-detection
parent: lifecycle/requirements/rate-limit-event-detection-2.md
created: "2026-08-15T00:00:00+10:00"
labels:
    - agent
    - observability
    - frontend
release: KC-Release5
---

# Frontend Plan — Consume `agent.quota_status` (wire-level, no badge UI)

Parent requirement: [[rate-limit-event-detection]]
(`lifecycle/requirements/rate-limit-event-detection-2.md`).
Companion plans: [[rate-limit-event-detection]] backend (`-3-be`) and test
(`-5-test`).

## Scope note (read first)

The requirement puts the **quota indicator UI** ("quota: 4h 12m to reset"
badge / global indicator) explicitly **out of scope** — it is a separate scoped
follow-up (see requirement §Out of scope). NFR1 states the frontend *may ignore
the event until a UI consumer ships*. Accordingly this plan does **not** build a
badge. It does the minimum that keeps the additive WS surface honest:

1. Type the new `agent.quota_status` event so it is a first-class, typed member
   of the WS union (not an `any` that silently slips through).
2. Cache the latest quota status per run in the existing agent/run store so a
   future badge is a pure presentation layer over already-available state.
3. Guarantee graceful handling: an unknown/absent-consumer event must never warn,
   throw, or disrupt existing WS handling.

If the architecture had promoted `standards/` (it has not yet — only the catalog
ships today), those UI/usability standards would gate the deferred badge; this
plan deliberately stops short of UI so no such gate applies.

## Architecture conformance

Go + Vue / Local Web architecture (per catalog + `CLAUDE.md`). Work is additive
Vue 3 + Pinia + TypeScript over the existing per-project WebSocket hub client.
No new dependency, view, or route. No deviation → no ADR.

## Grounding to confirm during M1

Before editing, locate (they exist but exact paths must be confirmed):
- The WS client / hub event dispatch in `web/src/` (search for existing event
  types such as `agent.progress`, `queue.rate_limit`, `queue.paused`).
- The Pinia store that already tracks per-run state (search for `run_id`
  handling around `agent.progress` / `queue.*`).
- The shared TypeScript type/union describing hub event payloads.

---

## Milestone 1 — Type the `agent.quota_status` event (NFR1)

**Description.** Add a TypeScript interface for the new payload and extend the
WS event union so the dispatcher recognises `agent.quota_status` as a known,
typed case. Mirror the backend payload exactly (snake_case on the wire):

```ts
interface QuotaStatusPayload {
  run_id: string
  bucket: 'five_hour' | 'weekly' | 'unknown'
  status: 'allowed' | 'warning' | 'rejected' | 'unknown'
  resets_at?: string            // RFC3339 UTC; omitted when unknown
  overage_available: boolean
  overage_disabled_reason?: string
}
```

Keep the union literals as strings from the backend but allow forward-compatible
`unknown` (matches NFR4 — the backend already normalises novel values to
`unknown`, so the frontend never receives an unmodelled literal).

**Files to change.**
- `web/src/` WS types module (the file declaring existing hub event payload
  types) — add `QuotaStatusPayload` and the `'agent.quota_status'` union member.

**Acceptance criteria.**
- `pnpm build` / `vue-tsc` type-checks with the new event as a discriminated
  union member.
- No existing event type is altered (NFR1).

---

## Milestone 2 — Cache latest quota status per run in the store

**Description.** In the Pinia store that already tracks per-run state, add a
reactive map `quotaByRun: Record<string, QuotaStatusPayload>` and, in the WS
message handler, upsert on `agent.quota_status` keyed by `run_id`. Expose a
getter `quotaForRun(runId)`. Because the backend debounces on content change
(FR4), each received event is a genuine transition and can overwrite blindly.
Clear the run's entry when the run reaches a terminal state (reuse whatever
`agent.finished` / `agent.failed` cleanup the store already performs for per-run
state), so the store mirrors the backend's per-run cache lifecycle (FR5/AC6).

**Files to change.**
- `web/src/` run/agent Pinia store — new state field, handler branch, getter,
  and terminal-state cleanup.

**Acceptance criteria.**
- Dispatching a mocked `agent.quota_status` populates `quotaForRun(run_id)` with
  the payload.
- A subsequent event for the same run replaces the cached value.
- On the run's terminal event the entry is removed.
- No badge/DOM is rendered (UI deferred — see Scope note).

---

## Milestone 3 — Graceful no-consumer handling (NFR1)

**Description.** Ensure the WS dispatcher's default/fallthrough path treats an
event with no active consumer as a no-op — no `console.warn`, no thrown error,
no disruption to sibling handlers. If the existing dispatcher already warns on
unknown types, the M1 union addition prevents that; add a light test to lock the
behaviour in.

**Files to change.**
- `web/src/` WS dispatcher (only if it currently warns/throws on unmodelled
  types; otherwise no code change — covered by the test in `-5-test`).

**Acceptance criteria.**
- Receiving `agent.quota_status` before any badge/consumer exists produces no
  console warning and does not interrupt handling of a following
  `agent.progress` event.
- Existing WS handling paths are unchanged.

---

## Out of scope (this plan)

- The quota indicator badge / global reset-countdown UI (separate follow-up).
- Timezone rendering / countdown formatting (deferred with the badge; backend
  emits UTC only per NFR2).
- Any cross-project quota aggregation (requirement §Out of scope — an app-global
  hub is not in scope).
