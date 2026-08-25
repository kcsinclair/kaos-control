---
title: "Test Plan — Record Provider and Driver on Every Agent Run"
type: plan-test
status: draft
lineage: agent-logging-provider-driver
parent: lifecycle/requirements/agent-logging-provider-driver-2.md
created: "2026-08-25T11:20:00+10:00"
---

# Test Plan — Record Provider and Driver on Every Agent Run

Verifies [[agent-logging-provider-driver]] against the backend
([[agent-logging-provider-driver]]) and frontend
([[agent-logging-provider-driver]]) plans. Requirement:
[agent-logging-provider-driver-2.md](../requirements/agent-logging-provider-driver-2.md).

Covers the requirement's Acceptance Criteria: DB columns + idempotent migration,
run-start population, API-driver vs CLI-driver behaviour, per-driver header
consistency, immutability across metrics/model updates, API payload exposure,
and secret hygiene.

## Architecture conformance

Go integration tests under `tests/integration/` (the `test-developer` target),
plus Go unit tests co-located with `internal/index` and `internal/agent` where a
unit test is the tighter fit. Uses the existing `newAgentTestEnv*` /
`startAgentRun` / `waitForRunCompletion` harness
(`tests/integration/agent_helpers_test.go`). No new dependency; conforms to
[[adr-0003-pure-go-sqlite-index]].

## Test conventions (verified in repo)

- `newAgentTestEnvWithCfg(t, cfgYAML, seeds)` builds a project with a custom
  agent config (`agent_helpers_test.go:125`); `startAgentRun`
  (`:333`) POSTs a run and returns the run id; `waitForRunCompletion` (`:349`)
  polls `GET /api/p/testproject/agents/runs/{run_id}`.
- The env auto-logs in as admin; devops-style URL helpers return full URLs — see
  the project memory note on the test harness.
- Shell-stub / fake-claude drivers are the deterministic way to exercise a run
  without a live model (`setupFakeClaude*`, `agent_helpers_test.go:297`;
  `openai_driver_test.go` uses a local `httptest` server for
  `openai-compatible`).
- Existing precedent for asserting run-row fields via the API:
  `agent_metrics_test.go`, `openai_agent_run_test.go`.

## Milestone 1 — Migration + data-model unit tests

**Description.** Prove the schema migration is idempotent and non-destructive and
that the row round-trips the new fields.

**Files to change / add.**
- `internal/index/index_agent_runs_test.go` (new, `-short` unit test):
  - Open a fresh index; assert `agent_runs` has `driver` and `provider` columns
    (`PRAGMA table_info(agent_runs)`).
  - Simulate a legacy DB: open an index, drop the `driver`/`provider` columns is
    not straightforward in SQLite — instead assert idempotency by calling the
    `ensureAgentRunsTable` path twice (reopen) and confirming no error and no
    rebuild of the versioned schema (rows survive).
  - `InsertAgentRun` a row with `Driver`/`Provider` set → `GetAgentRun` returns
    them; insert with empty provider → returns `""`.

**Acceptance criteria.**
- Fresh DB exposes both columns.
- Reopening an existing DB does not error and preserves existing `agent_runs`
  rows (no rebuild triggered by the new columns).
- Round-trip: values written at insert read back identically; empty provider
  reads back as `""` (not NULL-scan panic).

## Milestone 2 — Immutability unit test

**Description.** Prove no update path mutates `driver`/`provider` (FR-3).

**Files to change / add.**
- `internal/index/index_agent_runs_test.go`:
  - Insert a run with `Driver="openai-compatible"`, `Provider="prov-a"`.
  - Call `SetAgentRunModel(runID, "model-x")`, then
    `UpdateAgentRunMetrics(runID, AgentRunMetrics{Model:"model-y", ...})`, then
    `UpdateAgentRun(...)` with changed status.
  - Re-read: `model` reflects the later value, but `driver` and `provider` are
    **unchanged** (`openai-compatible` / `prov-a`).

**Acceptance criteria.**
- After all three update calls, `driver`/`provider` equal the run-start values.
- `model` is allowed to change (contrast assertion) — confirms the test is
  actually exercising the update paths.

## Milestone 3 — API-driver end-to-end (non-empty provider)

**Description.** A full `openai-compatible` run records driver + bound provider
on the DB row, exposes them on the API, and writes them in the log header.

**Files to change / add.**
- `tests/integration/agent_run_provider_driver_test.go` (new):
  - Config an `openai-compatible` agent bound to a named provider (pattern from
    `openai_agent_run_test.go:60`–`:106` + local `httptest` upstream from
    `openai_driver_test.go`).
  - Start a run; `waitForRunCompletion`.
  - Assert the run JSON from `GET …/agents/runs/{run_id}` has
    `driver == "openai-compatible"` and `provider == "<provider name>"`.
  - Read the run log via `GET …/agents/runs/{run_id}/log` and assert the header
    contains `driver=openai-compatible` and `provider=<name>` before the first
    output line.

**Acceptance criteria.**
- DB row (via API) shows the correct driver id and provider name.
- Log header contains both tokens with the provider name.
- Maps to requirement AC "API driver records both driver id and bound provider".

## Milestone 4 — CLI-driver end-to-end (empty provider)

**Description.** A CLI driver run records the driver id, an empty provider, and
still writes both `driver=` and `provider=` tokens in the header.

**Files to change / add.**
- `tests/integration/agent_run_provider_driver_test.go`:
  - Use a fake-claude (`claude-code-cli`) run via `setupFakeClaude*` +
    `startAgentRun`, and/or a `shell-stub` agent.
  - Assert API row: `driver == "claude-code-cli"` (or `shell-stub`),
    `provider == ""`.
  - Assert log header contains the literal `provider=` (empty token) and
    `driver=<id>` before output.

**Acceptance criteria.**
- CLI run row shows the driver id and empty provider.
- Header includes both `driver=` and `provider=` (empty token, not omitted, not
  a fabricated placeholder) — resolved question 3.
- Maps to requirement AC "CLI driver records driver id, empty provider, writes
  both tokens".

## Milestone 5 — Header consistency across every driver

**Description.** Assert the header block for every driver that writes a log
contains both fields in a consistent shape (FR-4).

**Files to change / add.**
- `internal/agent/log_header_test.go` (new unit test) OR extend existing
  per-driver tests: for each driver Start path that opens a `LogPath`
  (`claude-code-cli` via `startCommandProcess`, `claude-env`, `claude-mediated`,
  `codex-cli`, `gemini-cli`, native `gemini`, `openai-compatible`,
  `shell-stub`), drive a minimal run with a temp `LogPath` and assert the file's
  first header block matches a regex requiring `driver=` and `provider=`.
  - Prefer testing the shared `writeRunLogHeader` helper (if the backend plan
    extracts one) directly for the format, plus one integration touch per driver
    family to prove the helper is actually called.
  - `shell-stub` specifically: assert it now writes a header (it wrote none
    before this change).

**Acceptance criteria.**
- Every listed driver's header contains `driver=` and `provider=` in the same
  block/order.
- `shell-stub` produces a header when `LogPath` is set.
- A driver added without the shared helper would fail this test (guards against
  future drift).

## Milestone 6 — Secret-hygiene assertion (NFR-1)

**Description.** Prove no credential material lands in the new columns or the
header.

**Files to change / add.**
- Extend `agent_run_provider_driver_test.go` (API-driver case, which has a
  `base_url`/token):
  - Configure the provider with a recognisable fake token
    (e.g. `auth_token: SECRET-DO-NOT-LOG`).
  - After the run, assert the `driver`/`provider` fields do **not** contain the
    token, and the full run-log text does not contain `SECRET-DO-NOT-LOG` nor an
    `Authorization:`/`api_key` value newly introduced by this change.

**Acceptance criteria.**
- Neither column nor the log header contains the secret token.
- Test maps to requirement AC on secret hygiene ([[secrets-handling]]).

## Milestone 7 — Frontend component tests

**Description.** Cover the UI precedence (record over config) and empty-provider
rendering (mirrors the frontend plan's Milestone 4).

**Files to change / add.**
- `web/src/components/agent/__tests__/RunDetailModal.spec.ts` and
  `web/src/views/project/__tests__/AgentsRunsView.spec.ts` (Vitest): given a run
  whose `driver`/`provider` differ from the current agent config, assert the UI
  shows the **run's** values; given empty provider, assert `—`; given empty
  `driver`, assert the config fallback.

**Acceptance criteria.**
- `pnpm test` passes.
- A test fails if the UI reverts to config-derived driver/provider for a run
  that carries its own recorded values.

## Traceability to requirement Acceptance Criteria

| Requirement AC | Covered by |
| --- | --- |
| Columns added via idempotent migration; no rebuild; existing rows load | M1 |
| `AgentRunRow` populated from `Run.Driver`/`ProviderName` at insert | M1, M3, M4 |
| API driver records driver id + bound provider (row + header) | M3 |
| CLI driver records driver id, empty provider, both tokens in header | M4 |
| Every driver header has `driver=`/`provider=` before first output | M5 |
| Config change / failover after start does not mutate the row | M2 |
| Metrics/model update leaves driver/provider unchanged | M2 |
| Run-detail API payload includes driver + provider; UI can display | M3, M4, M7 |
| No secret in columns or newly in any header | M6 |
| Integration tests: API-driver + CLI-driver, header presence, immutability | M2–M6 |

## Out of scope

- Analytics per-provider/per-driver report tests
  ([[agent-usage-analytics-report]]) — follow-up.
- Backfill of legacy rows.
