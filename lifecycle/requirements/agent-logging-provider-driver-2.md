---
title: Record Provider and Driver on Every Agent Run
type: requirement
status: planning
lineage: agent-logging-provider-driver
created: "2026-08-25T10:05:00+10:00"
priority: normal
parent: lifecycle/ideas/agent-logging-provider-driver.md
labels:
    - agent
    - agent-runner
    - driver
    - provider
    - observability
    - runs
    - persistence
release: KC-Release6
assignees:
    - role: product-owner
      who: agent
---

# Record Provider and Driver on Every Agent Run

Parent: [[agent-logging-provider-driver]] (idea). Related: [[switch-provider]],
[[provider-model-for-agents]], [[artifact-agent-run-history]],
[[agent-usage-analytics-report]].

## Problem

Every agent run produces two run records: an on-disk job log file (at
`Run.LogPath`) and a database row in the `agent_runs` table
(`AgentRunRow`, `internal/index/index.go`). Today neither record reliably
captures **which driver executed the run** or **which provider backed it**:

- The `agent_runs` table has columns for `run_id`, `agent_name`, `role`,
  `model`, and metrics, but **no `driver` column and no `provider` column**.
  A run's backing driver/provider cannot be recovered from the database.
- The on-disk log header is **inconsistent across drivers**. The
  `claude-code-cli`, `gemini-cli`, and `codex-cli` drivers write only
  `# agent=… role=… model=…` — no driver, no provider. The native `gemini`
  driver writes `driver=gemini` but no provider. Only `openai-compatible`
  writes both `driver=…` and `provider=…`.

Because agent configuration is mutable over time — switching providers on 529
overload ([[switch-provider]]), swapping drivers, or re-pointing an agent at a
different `{provider, model}` ([[provider-model-for-agents]]) — a run record
that omits provider/driver makes attribution, cost analysis, and debugging
unreliable: given a piece of produced work, you cannot tell which provider or
driver produced it.

The values are already available at run start — `Run.Driver` and
`Run.ProviderName` are populated on the `Run` struct before the driver starts —
so this is a persistence gap, not a data-availability gap.

## Goals / Non-goals

### Goals

- Persist the **driver** and **provider** active at run-start on both run
  records: the `agent_runs` database row and the on-disk job log header.
- Write the values **at the start of the run**, before any agent output is
  recorded, and treat them as **immutable** for that run record thereafter — a
  later config change (or an in-flight provider failover) must not rewrite the
  stored driver/provider of a completed or already-started run.
- Make the on-disk log header **consistent across all drivers**: every driver
  writes both `driver=` and `provider=` in its header block.
- Surface the recorded `driver` and `provider` on the run-detail API payload so
  the run-history UI ([[artifact-agent-run-history]]) can display them.
- Keep the change compliant with [[secrets-handling]]: driver and provider are
  non-secret identifiers; no credential material is added to logs or records.

### Non-goals

- Adding provider/driver **breakdown dimensions** to the agent-usage analytics
  aggregation ([[agent-usage-analytics-report]]) — capturing the data is in
  scope; new report groupings are a follow-up (see Open Questions).
- Backfilling driver/provider onto **historical** run rows that predate this
  change — legacy rows remain valid with empty values.
- Changing how providers or drivers are **resolved, selected, or switched**
  ([[provider-model-for-agents]], [[switch-provider]]).
- Recording any secret (API key, auth token) on the run — explicitly forbidden.

## Detailed Requirements

### Architecture-Breaking Requirements

Reviewed against `lifecycle/architecture/architecture-summary.md` and the
recorded standards/ADRs. **No architecture-breaking requirement is introduced.**
Evaluation of each standing constraint:

1. **Single self-contained binary.** New work is two nullable `TEXT` columns
   added to the existing `agent_runs` table via idempotent `ALTER TABLE`
   migration (same pattern as the `model` column) plus string fields in the log
   header. Pure Go, no new dependency, no cgo, no external service.
   → **Satisfied.**
2. **Local filesystem is the source of truth ([[index-is-a-cache]]).** The
   `agent_runs` table holds runtime run telemetry, not a cache of `lifecycle/`
   markdown; it is not reconstructed from a disk scan of artifacts, so this
   change does not make the index authoritative over any artifact. The on-disk
   job log remains the durable, human-readable record of the run.
   → **Satisfied / not applicable.**
3. **Agents execute mediated tool calls ([[adr-0006-mediated-agent-driver-permission-model]]).**
   This change is observational only — it records metadata about the run and
   does not touch tool mediation, `allowed_write_paths`, or command policy.
   → **Satisfied.**
4. **Secrets hygiene ([[secrets-handling]]).** `driver` and `provider` are
   non-secret identifiers (a driver id such as `openai-compatible`; a provider
   *name/slug* such as `gemini-cloud`, never a `base_url` with credentials, an
   `api_key`, or an `auth_token`). No secret is added to any record.
   → **Satisfied.**

**Conclusion:** No conflict with the recorded architecture, stack, or
standards. No new ADR required.

### Functional Requirements

#### FR-1: Run-record data model

- `AgentRunRow` (`internal/index/index.go`) gains two fields:
  - `driver` (string): the driver id that executed the run, sourced from
    `Run.Driver`.
  - `provider` (string): the provider name that backed the run, sourced from
    `Run.ProviderName`. **May be empty** for CLI drivers that have no provider
    binding (e.g. `claude-code-cli`, `gemini-cli`, `codex-cli`).
- The `agent_runs` table gains `driver TEXT` and `provider TEXT` columns via
  idempotent `ALTER TABLE ADD COLUMN` migrations (consistent with the existing
  `model` migration). Both are nullable; legacy rows remain valid.

#### FR-2: Written at run-start

- `driver` and `provider` are written when the run row is first inserted
  (`InsertAgentRun`), i.e. at run-start, **before** any agent output, metrics,
  or `type:result` line is recorded — not on a later update path.
- The on-disk job log **header** (the `# kaos-control agent run …` block each
  driver writes before streaming output) includes `driver=<driver>` and
  `provider=<provider>` for **every** driver, written before the first output
  line. Where provider is empty the value is written as empty/omitted
  consistently (e.g. `provider=`), not as a fabricated placeholder.

#### FR-3: Immutability for the run

- Once written, a run's `driver` and `provider` are **immutable**. No code path
  (config reload, provider failover mid-fleet, `UpdateAgentRun`,
  `UpdateAgentRunMetrics`, `SetAgentRunModel`, TTFT stamping) may overwrite them
  after the run row is inserted.
- Rationale contrast: `model` on the row *is* refined from the `type:result`
  line to reflect the model actually used after any fallback; `driver` and
  `provider` are **not** refined — they reflect the resolved configuration at
  run-start and are frozen there.

#### FR-4: Consistency across all drivers

- Every driver that writes a log header emits both fields: `claude-code-cli`,
  `claude-mediated`/claude-hooks, `claude-env`, `gemini-cli`, native `gemini`,
  `codex-cli`, `openai-compatible`, and `shell-stub`. The header format is
  uniform across drivers (driver and provider appear in the same header block
  for all).

#### FR-5: Surfaced on the run API

- The run-detail / run-history API payload that serialises `AgentRunRow`
  includes the `driver` and `provider` fields (JSON keys `driver`,
  `provider`), so [[artifact-agent-run-history]] can display which driver and
  provider produced each run. Empty provider serialises as an empty string /
  omitted field, not an error.

### Non-Functional Requirements

#### NFR-1: Secret hygiene ([[secrets-handling]])

- Only the non-secret driver id and provider **name** are recorded. `base_url`,
  `api_key`, `AuthToken`, and any `Authorization` header value must never be
  written to the `provider`/`driver` columns or added to the log header by this
  change. Existing header content is not made less redacted.

#### NFR-2: Backward compatibility

- The migration is idempotent and non-destructive; opening an existing index
  adds the columns without a rebuild. Rows written before this change (empty
  `driver`/`provider`) load and serialise without error.

#### NFR-3: No behavioural change to runs

- Recording this metadata must not alter run scheduling, driver selection,
  exit status, metrics, or produced artifacts. It is observational only.

## Acceptance Criteria

- [ ] `agent_runs` has `driver` and `provider` columns, added via idempotent
      `ALTER TABLE` migration; opening a pre-existing index does not trigger a
      rebuild and existing rows still load.
- [ ] `AgentRunRow` exposes `driver` and `provider`, populated from
      `Run.Driver` and `Run.ProviderName` at `InsertAgentRun` (run-start).
- [ ] A run started with an API driver (`openai-compatible`) records both the
      driver id and the bound provider name on the DB row and in the log header.
- [ ] A run started with a CLI driver (`claude-code-cli`, `gemini-cli`,
      `codex-cli`) records the driver id, an empty provider, and writes both
      `driver=` and `provider=` in the log header.
- [ ] Every driver's on-disk log header contains `driver=` and `provider=` in a
      consistent format, written before the first output line.
- [ ] Changing the agent's provider/driver config (or a provider failover per
      [[switch-provider]]) after a run has started does **not** mutate that
      run's recorded `driver`/`provider`.
- [ ] Updating run metrics / model on finish (`UpdateAgentRunMetrics`,
      `SetAgentRunModel`) leaves `driver` and `provider` unchanged.
- [ ] The run-detail API payload includes `driver` and `provider`; the run
      history UI ([[artifact-agent-run-history]]) can display them.
- [ ] No secret (`api_key`, `auth_token`, `base_url` credential) appears in the
      `driver`/`provider` columns or is newly added to any log header
      ([[secrets-handling]]).
- [ ] Integration tests cover: an API-driver run (non-empty provider) and a
      CLI-driver run (empty provider), header presence for each driver, and
      immutability across a metrics update.

## Resolved Questions

1. **Provider name vs. driver+base_url for CLI drivers.** CLI drivers have no
   provider binding, so `provider` will be empty for them. Is an empty
   `provider` acceptable in the UI/report, or should CLI runs display a synthetic
   label (e.g. the driver id) in the provider column? *Recommendation:* leave
   `provider` empty and let the UI fall back to `driver` for display.

> Proceed with recommendation.

2. **Analytics breakdown.** Should [[agent-usage-analytics-report]] gain
   per-provider / per-driver series and totals once the data is captured?
   Proposed as a follow-up, out of scope here.

> Out of scope here.  Will raise enhancement to handle.

3. **Header value when provider is empty.** Emit `provider=` (empty token) for
   every driver for grep-consistency, or omit the token entirely for CLI
   drivers? *Recommendation:* always emit `provider=` so the header shape is
   uniform and greppable.

> Proceed with recommendation.
