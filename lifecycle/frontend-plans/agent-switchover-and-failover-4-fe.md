---
title: Agent Switchover and Failover — Frontend Plan
type: plan-frontend
status: done
lineage: agent-switchover-and-failover
parent: lifecycle/requirements/agent-switchover-and-failover-2.md
created: "2026-09-03T12:00:00+10:00"
---

# Agent Switchover and Failover — Frontend Plan

Implements the UI for [[agent-switchover-and-failover]]. Consumes the backend
routes and WS events defined in the backend plan (status from `operations.yaml`,
effective policy, reachability, partial pause, quota-gated recovery) — see
[[agent-switchover-and-failover]] for the shared lineage. Related UI already in
the tree: `web/src/stores/providerSwitch.ts`,
`web/src/components/provider/ProviderFailoverModal.vue`,
`web/src/components/agent/SwitchProviderModal.vue`,
`web/src/components/queue/QueuePauseBanner.vue`,
`web/src/lib/failureReasons.ts`.

Conforms to the tech stack (Vite 5 / Vue 3.5 SFC / TypeScript / Pinia / Vue
Router 4) and to [[secrets-handling]]: the UI shows provider/model **names** and
boolean/label state only — it never receives or renders secret material.

---

## Milestone 1 — Types, API client, and store for the new state shape

**Description.** Extend the wire types and store to carry the operations-backed
status: reachability per provider (all modes), `switched_at`, `reason`,
`resets_at_unix`, `bucket` (`five_hour` | `weekly`), FR-3.4 partial-pause,
FR-7.3 awaiting-operator-decision, and the effective event→action policy. Add
API calls for the policy route and the manual switch/failback routes, and wire
the WS events (`provider.switched`, `provider.restored`,
`provider.primary_recovered`, queue pause/resume) into the store.

**Files to change.**
- `web/src/types/providerSwitch.ts` — extend `FailoverAgent`/`FailoverStatus`
  with `reachability`, `resets_at_unix`, `bucket`, `partial_pause`,
  `awaiting_decision`; add `SwitchoverPolicy` (reason → action) and reachability
  types.
- `web/src/api/providerSwitch.ts` — add `getSwitchoverPolicy`, ensure manual
  `switchAll` / `restoreAll` / `restore` map to the guarded routes.
- `web/src/stores/providerSwitch.ts` — hold status + policy + reachability;
  actions to load them; subscribe to the WS events; getters for "mode"
  (single / multiple / manual / automated) and "current side" (primary vs
  secondary).

**Acceptance criteria.**
- The store exposes reachability for every provider in single-provider mode, not
  only failed-over primaries.
- The effective policy (with defaults) is loadable and every classified reason
  has a displayed action.
- No secret fields exist on any type (only names/labels/booleans).

## Milestone 2 — Status button: "Primary Agents" / "Secondary Agents"

**Description.** When both primary and secondary are configured, show a status
button reflecting the current side (FR-9.1): **"Primary Agents"** when all agents
are on primary, **"Secondary Agents"** when failed over, with a distinct
partial state when FR-3.4 leaves some agents paused. The button routes to the
failback screen (Milestone 3).

**Files to change.**
- `web/src/components/layout/AppHeader.vue` — render the status button from the
  store getter; hidden when no secondary is configured.
- `web/src/components/provider/` — small presentational component for the button
  if AppHeader grows unwieldy.

**Acceptance criteria.**
- The button reads "Primary Agents" on primary and "Secondary Agents" after
  failover, and indicates the partial-pause state when only some agents switched.
- The button is absent when no secondary is configured.

## Milestone 3 — Dedicated failback screen with authoritative reset time

**Description.** A new route + view presenting the information needed to judge
when to fail back (FR-9.2). It **must show the expected reset time** derived from
`resets_at_unix` and `bucket` (`five_hour` / `weekly`), per agent/provider. It
**must not** surface an unqualified `provider.primary_recovered`: any "recovered"
indicator is **suppressed or explicitly qualified until the recorded reset time
has passed** (FR-9.3). The screen offers the manual failback action (FR-8.1,
FR-8.3) and shows reachability and per-agent active-vs-primary detail.

**Files to change.**
- `web/src/views/project/FailbackView.vue` (new) — reset-time display
  (absolute + relative countdown), per-agent table (active vs primary, reason,
  switched_at, reachability), and a manual "Fail back to primary" action.
- `web/src/router/index.ts` — add a `failback` route under `/p/:project`.
- `web/src/components/layout/AppSidebar.vue` — nav entry (visible when a secondary
  is configured).

**Acceptance criteria.**
- The screen shows the expected reset time from `resets_at_unix`/`bucket`.
- No unqualified "recovered" state is shown before the reset time passes; any
  recovery hint is explicitly qualified until then.
- Manual failback is available here and reflects the guarded backend route.

## Milestone 4 — Event→action policy view & automated-switchover toggle

**Description.** Surface the effective event→action policy (FR-2.4): a read-only
(minimum) inspector listing every classified reason and its resolved action,
clearly marking defaulted vs operator-set entries. Expose the
`automated_switchover` toggle (**disabled by default**, prominently labelled per
FR-2.1) in provider settings.

**Files to change.**
- `web/src/views/project/ProviderSettingsView.vue` — policy inspector table +
  `automated_switchover` toggle with the "disabled by default" callout.
- `web/src/lib/failureReasons.ts` — extend with the full reason set and
  human-readable action labels.
- `web/src/stores/providerSwitch.ts` (or `projectConfig.ts`) — persist the toggle
  via the config API.

**Acceptance criteria.**
- Every classified reason appears with its effective action; defaulted entries
  are visually distinguished.
- The `automated_switchover` toggle shows disabled by default and its state
  round-trips through the backend.

## Milestone 5 — Manual-switch guard, partial-pause, and operator-decision surfaces

**Description.** When a manual switch is rejected because runs are in flight
(FR-8.2), show a warning listing the running jobs rather than a generic error.
Surface the FR-3.4 partial pause (agents with no secondary) and the FR-7.3
awaiting-operator-decision state (suspected partial commit) as actionable
prompts. Update the queue pause banner to distinguish policy-driven pauses
(rate-limit / disconnect-threshold) with the reason.

**Files to change.**
- `web/src/components/agent/SwitchProviderModal.vue` /
  `web/src/components/provider/ProviderFailoverModal.vue` — render the
  running-jobs rejection warning; block confirm while runs execute.
- `web/src/components/queue/QueuePauseBanner.vue` — show pause reason (rate-limit,
  disconnect threshold, partial pause).
- `web/src/components/agent/RunFailureBanner.vue` or `RunDetailModal.vue` —
  operator-decision prompt for a suspected partial commit (no auto-rerun /
  auto-rollback; the human decides).

**Acceptance criteria.**
- A manual switch during an executing run shows a warning naming the running jobs
  and performs no switch.
- Partial pause and awaiting-operator-decision states are visibly surfaced with
  the reason.

---

## Cross-cutting acceptance (from the requirement)

- Status button shows "Primary Agents"/"Secondary Agents"; failback screen shows
  the expected reset time and never an unqualified "recovered" before reset.
- Reachability is displayed in single-provider mode.
- All displayed data is name/label/boolean only — no secrets ([[secrets-handling]]).
- See [[agent-switchover-and-failover]] backend and test plans.
