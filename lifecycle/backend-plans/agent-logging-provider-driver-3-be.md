---
title: "Backend Plan — Record Provider and Driver on Every Agent Run"
type: plan-backend
status: approved
lineage: agent-logging-provider-driver
parent: lifecycle/requirements/agent-logging-provider-driver-2.md
created: "2026-08-25T11:20:00+10:00"
---

# Backend Plan — Record Provider and Driver on Every Agent Run

Implements the persistence + log-header side of
[[agent-logging-provider-driver]]. Requirement:
[agent-logging-provider-driver-2.md](../requirements/agent-logging-provider-driver-2.md).

Pairs with the frontend plan (surfacing `driver`/`provider` on the run-detail
UI, [[agent-logging-provider-driver]]) and the test plan (integration coverage,
[[agent-logging-provider-driver]]).

## Architecture conformance

Reviewed against `lifecycle/architecture/architecture-summary.md`, the ADRs, and
`standards/`. This change is:

- **Pure-Go, single-binary** — two nullable `TEXT` columns added by idempotent
  `ALTER TABLE` (same pattern as the existing `model` migration at
  `internal/index/index.go:1864`), no new dependency, no cgo
  ([[adr-0003-pure-go-sqlite-index]], [[adr-0004-embedded-spa-single-binary]]).
- **Index-is-a-cache safe** ([[index-is-a-cache]]) — `agent_runs` is runtime run
  telemetry, not a cache of `lifecycle/` markdown; it is already excluded from
  the schema rebuild (`internal/index/index.go:1819`). No file content is
  stamped from the index path.
- **Secrets-clean** ([[secrets-handling]]) — only the non-secret driver id
  (`Run.Driver`) and provider **name** (`Run.ProviderName`) are recorded. No
  `base_url`, `api_key`, or `AuthToken` is added to any column or log header.
- **Observational only** ([[adr-0006-mediated-agent-driver-permission-model]]) —
  no change to tool mediation, scheduling, or driver selection.

**No architecture-breaking requirement; no new ADR required** (concurs with the
requirement's own §Architecture-Breaking Requirements analysis).

## Current-state findings (verified in code)

- `AgentRunRow` struct: `internal/index/index.go:1292`. Has `model` and metrics
  but no `driver`/`provider`.
- `agent_runs` DDL + migrations: `ensureAgentRunsTable`,
  `internal/index/index.go:1842`; migrations block at `:1863`.
- `InsertAgentRun` (run-start insert): `internal/index/index.go:1346`. The
  `INSERT` column list is fixed and omits driver/provider today.
- `SELECT` column lists to keep in sync: `GetAgentRun` (`:1387`),
  `ListAgentRuns` (`:1397`), `ListAgentRunsByTargetPath` (`:1435`).
- Row scanners to keep in sync: `scanAgentRun` (`:1519`) and `scanAgentRunRow`
  (`:1602`).
- Mutation paths that must **not** touch driver/provider (immutability, FR-3):
  `UpdateAgentRun` (`:1359`), `UpdateAgentRunMetrics` (`:1478`),
  `SetAgentRunModel` (`:1507`), `SetAgentRunTTFT` (`:1514`). None reference the
  new columns — immutability is preserved by omission; the plan only requires we
  keep it that way.
- Run-start insert call site: `internal/agent/agent.go:711` builds `runRow`
  without driver/provider though `run.Driver`/`run.ProviderName` are already
  populated (`agent.go:671`, `:683`).
- Log-header writers (must all emit `driver=` and `provider=`, FR-4):
  - Shared `startCommandProcess` (`agent.go:254`) — covers `claude-code-cli`,
    `claude-env`, `claude-mediated`. Header currently: `agent= role= model=`.
  - `codex_cli.go:83` — `agent= role= model=`.
  - `gemini_cli.go:91` — `agent= role= model=`.
  - `gemini.go:166` — `driver=gemini model=` (has driver, no provider).
  - `openai_compatible.go:167` — already `driver= provider= base_url= model=`
    (the reference format; keep `base_url` where it already is — pre-existing,
    non-secret host, per requirement NFR-1 "existing header content is not made
    less redacted").
  - `shell_stub.go:28` (`ShellStubDriver.Start`) — writes **no** log header
    today; must gain one for FR-4.
- Secondary `InsertAgentRun` callers to consider:
  `recordPreflightFailure` (`agent.go:882`) and `internal/triage/run.go:343`.

## Milestone 1 — Data model + migration

**Description.** Add `driver` and `provider` to `AgentRunRow` and the
`agent_runs` schema.

**Files to change.**
- `internal/index/index.go`:
  - `AgentRunRow` struct (`:1292`): add
    `Driver string \`json:"driver"\`` and
    `Provider string \`json:"provider"\``. Use plain `string` (not `*string`):
    empty string is the natural "no provider" / legacy value and serialises as
    `""` per FR-5. Place them near `Model` with a doc comment noting they are
    written once at run-start and are immutable (FR-3).
  - `ensureAgentRunsTable` (`:1863` migrations block): append
    `_, _ = idx.db.Exec(\`ALTER TABLE agent_runs ADD COLUMN driver TEXT\`)` and
    `_, _ = idx.db.Exec(\`ALTER TABLE agent_runs ADD COLUMN provider TEXT\`)`,
    matching the existing idempotent, error-discarding pattern.

**Acceptance criteria.**
- `go build ./...` compiles.
- Opening a pre-existing index (with rows lacking the columns) does **not**
  trigger a rebuild and does not error (`ALTER` duplicate-column errors are
  discarded, consistent with existing migrations).
- Fresh DB: `agent_runs` has `driver TEXT` and `provider TEXT` columns
  (verifiable via `PRAGMA table_info(agent_runs)`).

## Milestone 2 — Persist at run-start + read paths

**Description.** Write driver/provider on the run-start `INSERT`, and read them
back on every select. Deliberately leave all update paths untouched so the
values are immutable (FR-3).

**Files to change.**
- `internal/index/index.go`:
  - `InsertAgentRun` (`:1346`): add `driver, provider` to the column list and
    bind `r.Driver, r.Provider`. This is the only write path for these columns.
  - `GetAgentRun` (`:1387`), `ListAgentRuns` (`:1397`),
    `ListAgentRunsByTargetPath` (`:1435`): add `driver, provider` to each
    `SELECT` list. (Legacy rows return SQL `NULL`; scan into `sql.NullString`
    and map to `""`.)
  - `scanAgentRun` (`:1519`) and `scanAgentRunRow` (`:1602`): add two
    `sql.NullString` locals, add them to the `Scan(...)` arg list **in the same
    column order** as the SELECT, and set `r.Driver = v.String` when `Valid`
    (else `""`), same for provider.
  - Leave `UpdateAgentRun`, `UpdateAgentRunMetrics`, `SetAgentRunModel`,
    `SetAgentRunTTFT` **unchanged** — none may reference driver/provider (FR-3).
- `internal/agent/agent.go`:
  - Run-start `runRow` literal (`:711`): set
    `Driver: run.Driver` and `Provider: run.ProviderName`.

**Acceptance criteria.**
- A run inserted with `Driver="openai-compatible"`, `ProviderName="gemini-cloud"`
  reads back with both fields populated via `GetAgentRun`/`ListAgentRuns`.
- A run inserted with a CLI driver (`Driver="gemini-cli"`, `ProviderName=""`)
  reads back `driver="gemini-cli"`, `provider=""` (no error, not NULL-panic).
- After calling `UpdateAgentRunMetrics` (with a `Model`) and `SetAgentRunModel`
  on an existing run, re-reading the row shows `driver`/`provider` **unchanged**
  (immutability). `model` may change; driver/provider must not.
- Legacy rows (columns NULL) load through both scanners without error and expose
  `driver=""`, `provider=""`.

## Milestone 3 — Uniform log header across all drivers

**Description.** Every driver's on-disk header emits both `driver=<Run.Driver>`
and `provider=<Run.ProviderName>`, written before the first output line, in a
consistent block. Empty provider is emitted as the literal token `provider=`
(FR-2 / resolved question 3 — always emit for grep-uniformity).

**Files to change.**
- `internal/agent/agent.go` — `startCommandProcess` header (`:254`): change to
  `# agent=%s role=%s driver=%s provider=%s model=%s` binding
  `run.Driver, run.ProviderName`. Covers `claude-code-cli`, `claude-env`,
  `claude-mediated`.
- `internal/agent/codex_cli.go` (`:83`): same header shape, `run.Driver` will be
  `codex-cli`, provider empty.
- `internal/agent/gemini_cli.go` (`:91`): same, driver `gemini-cli`.
- `internal/agent/gemini.go` (`:166`): replace the hard-coded `driver=gemini`
  with `driver=%s provider=%s` bound to `run.Driver, run.ProviderName` (keep
  driver value dynamic rather than literal so it always matches the row).
- `internal/agent/openai_compatible.go` (`:167`): already emits
  `driver= provider=`; verify field ordering matches the others
  (`driver= provider=` before `base_url= model=`). Leave `base_url` as-is
  (pre-existing, non-secret host — NFR-1).
- `internal/agent/shell_stub.go` — `ShellStubDriver.Start` (`:28`): add
  log-file opening + header write mirroring `startCommandProcess` (open
  `run.LogPath` when non-empty, `MkdirAll` the dir, write the same
  `# kaos-control agent run … / # agent= role= driver= provider= model=` block,
  ensure the file handle is closed on process exit via `stubProcess`). Guard on
  `run.LogPath != ""`.

Recommended: extract a small helper
`writeRunLogHeader(w io.Writer, run Run, args any)` (in `agent.go`) that formats
the single canonical header line, and call it from all six sites, so the format
cannot drift again. Keep the `# args=%v` line only where callers already emit it
(agent.go/codex/gemini-cli) — do not add args to headers that omit it today.

**Acceptance criteria.**
- Each driver's log file, when `LogPath` is set, contains a header line matching
  `driver=<id> provider=<name-or-empty>` before any streamed output line.
- For a CLI driver the header contains the literal `provider=` (empty token),
  not a fabricated placeholder and not an omitted token.
- For `openai-compatible` the header still contains `driver=` and `provider=`
  and no new secret material was introduced.
- `shell-stub` now produces a header when `run.LogPath` is set (previously
  none).
- No `api_key`, `auth_token`, or `Authorization` value appears in any header
  (grep the produced logs in tests — see test plan).

## Milestone 4 — Secondary insert call sites

**Description.** Decide and implement behaviour for the two non-primary
`InsertAgentRun` callers so they do not regress and, where cheap, carry the
data.

**Files to change.**
- `internal/agent/agent.go` — `recordPreflightFailure` (`:882`): this records a
  failure before a full `Run` exists. Populate `Driver` (and `Provider` when
  resolvable) from the agent config already available in the caller if it can be
  threaded without a signature churn; otherwise leave empty and add a code
  comment that preflight-failure rows may carry empty driver/provider (they
  predate driver selection). Empty is acceptable per FR-1.
- `internal/triage/run.go` (`:343`): triage-run rows have no agent driver
  binding; leave `Driver`/`Provider` empty. Confirm the struct literal still
  compiles with the new fields (they default to `""`).

**Acceptance criteria.**
- `go build ./...` and `make lint` pass with both call sites updated/confirmed.
- Preflight-failure rows and triage rows load and serialise with empty
  driver/provider (no panic, no error).

## Milestone 5 — API surface confirmation

**Description.** Confirm the run-detail/run-history API payload now includes the
fields (FR-5). No handler change is expected because handlers serialise
`AgentRunRow` directly.

**Files to change.**
- None expected. Verify:
  - `internal/http/agents.go` — `handleGetAgentRun` (`:219`) returns
    `map[string]any{"run": run}` where `run` is `*index.AgentRunRow`; the new
    JSON keys `driver`/`provider` appear automatically.
  - `handleListAgentRuns` (`:193`) likewise.
- If any handler copies `AgentRunRow` into a hand-built DTO (grep for a
  per-field struct in `internal/http/`), add the two fields there too. Current
  reading shows direct serialisation, so this is a verification step.

**Acceptance criteria.**
- `GET /api/p/{project}/agents/runs/{run_id}` response `run` object includes
  `"driver"` and `"provider"` keys (provider may be `""`).
- `GET /api/p/{project}/agents/runs` list entries include both keys.
- No secret field is present in the payload
  ([[secrets-handling]] — assert in the integration test).

## Out of scope (per requirement)

- Analytics per-provider/per-driver breakdowns in the usage report
  ([[agent-usage-analytics-report]]) — follow-up enhancement.
- Backfilling historical rows.
- Any change to how providers/drivers are resolved, selected, or switched
  ([[switch-provider]], [[provider-model-for-agents]]).
