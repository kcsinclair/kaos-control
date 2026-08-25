---
title: "Frontend Plan — Surface Run Provider and Driver in the UI"
type: plan-frontend
status: draft
lineage: agent-logging-provider-driver
parent: lifecycle/requirements/agent-logging-provider-driver-2.md
created: "2026-08-25T11:20:00+10:00"
---

# Frontend Plan — Surface Run Provider and Driver in the UI

Implements FR-5 of [[agent-logging-provider-driver]]. Requirement:
[agent-logging-provider-driver-2.md](../requirements/agent-logging-provider-driver-2.md).

Depends on the backend plan ([[agent-logging-provider-driver]]) exposing
`driver` and `provider` on the run payload. Verified by the test plan
([[agent-logging-provider-driver]]).

## Architecture conformance

Conforms to the embedded-SPA / go-vue stack
([[adr-0004-embedded-spa-single-binary]], `go-vue.md`). No secret material
reaches the client ([[secrets-handling]]): the payload carries only the driver
id and provider **name**, never `base_url`/`api_key`/`auth_token`. No
architecture-breaking change; no new ADR.

## Current-state findings (verified in code)

The run-history/run-detail views **currently derive driver and provider from the
live agent config**, not from the run record — which is precisely the
attribution defect the requirement calls out (config is mutable, so a past run
shows the *current* driver/provider, not the one that ran):

- `web/src/views/project/AgentsRunsView.vue` — the Driver column
  (`:319`–`:325`) reads
  `store.agents.find(a => a.name === run.agent_name)?.driver`, i.e. the current
  config, not `run.driver`. There is no Provider column.
- `web/src/components/agent/RunDetailModal.vue` —
  `providerNameFor(agentName)` (`:136`) and the token-metrics driver check
  (`:131`) both look up `agentsStore.agents.find(...)`. The field grid
  (`:221`–`:272`) shows Agent/Role/Status/etc. but no Driver/Provider row.
- `web/src/types/api.ts` — `AgentRunRow` interface (`:313`) has neither
  `driver`, `provider`, nor `model`.

Goal: display the run's **recorded** `driver`/`provider`, falling back to the
config-derived value only when the run record's field is empty (legacy rows).

## Milestone 1 — Type + store

**Description.** Extend the `AgentRunRow` type so the new fields are available to
all consumers.

**Files to change.**
- `web/src/types/api.ts` — add to `AgentRunRow` (`:313`):
  ```ts
  /** Driver id that executed this run, recorded at run-start (immutable). */
  driver?: string
  /** Provider name that backed this run; empty for CLI drivers. */
  provider?: string
  ```
  (Optional/possibly-empty for legacy rows.) No store logic change is required —
  `useAgentsStore().runs` already holds `AgentRunRow[]`
  (`web/src/stores/agents.ts:192`).

**Acceptance criteria.**
- `pnpm build` / `vue-tsc` typechecks with the new fields referenced.
- Consuming `run.driver` / `run.provider` in components is type-safe.

## Milestone 2 — Runs table: driver + provider from the run record

**Description.** Change the Driver column to read the recorded `run.driver`, and
add provider display. Fall back to the config lookup only when `run.driver` is
empty (legacy rows), preserving today's behaviour for old data.

**Files to change.**
- `web/src/views/project/AgentsRunsView.vue`:
  - Add a helper (script) `runDriverLabel(run)` that prefers `run.driver` and
    maps it to a friendly label using the existing `driverLabel`/`agentDriver`
    mapping (`:43`–`:50`), falling back to the config-derived driver when
    `run.driver` is empty.
  - Driver cell (`:319`–`:325`): bind the badge text and `:data-driver` to
    `run.driver || <config driver>` rather than the config lookup alone.
  - Provider: show `run.provider` when non-empty. Per resolved question 1,
    when provider is empty fall back to displaying the driver (do **not**
    fabricate a provider). Add it either as a small sub-label under the driver
    badge or a dedicated "Provider" column — keep it compact; a sub-label under
    the driver badge is preferred to avoid widening the table.

**Acceptance criteria.**
- A historical run whose agent has since switched driver/provider shows the
  driver/provider **it actually ran with** (`run.driver`/`run.provider`), not
  the agent's current config.
- An `openai-compatible` run shows its bound provider name.
- A CLI run (`gemini-cli`, empty provider) shows the driver and no fabricated
  provider (falls back to driver per resolved Q1).
- A legacy run (`run.driver` empty) still shows the config-derived driver as
  before — no regression.

## Milestone 3 — Run detail modal: Driver + Provider fields

**Description.** Add explicit Driver and Provider rows to the run-detail field
grid, sourced from the run record; keep the config lookup only as an
empty-value fallback.

**Files to change.**
- `web/src/components/agent/RunDetailModal.vue`:
  - Field grid (`:221`–`:272`): add a "Driver" row bound to
    `run.driver || <config driver> || '—'` and a "Provider" row bound to
    `run.provider || '—'` (empty provider renders `—`, per FR-5 "empty
    serialises as empty string", displayed as a dash, not an error).
  - `providerNameFor` (`:136`): prefer `run.provider`, fall back to the current
    config lookup only when the record's provider is empty. (The RunSummaryCard
    `:provider-name` binding at `:283` should follow the same precedence so the
    summary card labels the provider that actually ran.)
  - Leave the token-metrics capability check (`:131`) as-is (it is a
    driver-capability decision about the current agent, not run attribution) —
    but if `run.driver` is present, prefer it for that check too for
    consistency.

**Acceptance criteria.**
- Opening a run detail shows Driver and Provider reflecting the recorded run
  values.
- For a run with empty provider, the Provider row renders `—` and nothing
  throws.
- For an `openai-compatible` run, Driver = `openai-compatible` and Provider =
  the bound provider name from the record.

## Milestone 4 — Unit tests (component)

**Description.** Cover the new display + fallback logic at the component level
(Vitest), complementing the backend integration tests in the test plan.

**Files to change.**
- `web/src/components/agent/__tests__/RunDetailModal.spec.ts` (new or extend if
  present): given an `AgentRunRow` with `driver`/`provider` set, assert both are
  rendered; given empty `provider`, assert `—`; given empty `driver`, assert the
  config fallback is used.
- `web/src/views/project/__tests__/AgentsRunsView.spec.ts` (extend if present):
  assert the Driver cell prefers `run.driver` over the current config driver.

**Acceptance criteria.**
- `pnpm test` (Vitest) passes.
- A test proves that when the agent's current config driver differs from
  `run.driver`, the UI displays `run.driver`.

## Out of scope

- Any analytics/report UI for provider/driver breakdown
  ([[agent-usage-analytics-report]]) — follow-up.
- Changing provider-switch UI ([[switch-provider]]).
