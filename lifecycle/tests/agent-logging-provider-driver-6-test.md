---
title: "Tests — Record Provider and Driver on Every Agent Run"
type: test
status: draft
lineage: agent-logging-provider-driver
parent: lifecycle/test-plans/agent-logging-provider-driver-5-test.md
created: "2026-08-26T12:30:00+10:00"
---

# Tests — Record Provider and Driver on Every Agent Run

Automated coverage for [agent-logging-provider-driver-2.md](../requirements/agent-logging-provider-driver-2.md)
against the backend implementation (already `status: done` on
[agent-logging-provider-driver-3-be.md](../backend-plans/agent-logging-provider-driver-3-be.md)),
per the test plan's Milestones 1–6 (Go). Milestone 7 (frontend Vitest
component tests) is out of scope for this artifact/lineage step — it is
frontend-developer's target (`web/src/**`), not `test-developer`'s
(`tests/**`); see [[frontend-plan]] Milestone 4, already implemented per
`a312f263`.

## Scenarios covered

### Milestone 1 — Migration + data-model unit tests

`internal/index/index_agent_runs_test.go`:

| Test function | Plan scenario |
|---|---|
| `TestAgentRunsTable_HasDriverProviderColumns` | Fresh index exposes `driver`/`provider` via `PRAGMA table_info(agent_runs)` |
| `TestAgentRunsTable_ReopenIsIdempotent` | Closing and reopening an existing DB does not error and the inserted row (with driver/provider) survives |
| `TestInsertAgentRun_DriverProviderRoundTrip` | Insert with `Driver`/`Provider` set → `GetAgentRun`/`ListAgentRuns` return them unchanged; insert with empty provider → reads back `""` (no NULL-scan panic) |

### Milestone 2 — Immutability unit test

`internal/index/index_agent_runs_test.go`:

| Test function | Plan scenario |
|---|---|
| `TestAgentRun_DriverProviderImmutable` | After `SetAgentRunModel`, `UpdateAgentRunMetrics`, and `UpdateAgentRun`, re-reading the row shows `driver`/`provider` unchanged while `model` (the contrast case) has changed |

### Milestone 3 — API-driver end-to-end (non-empty provider)

`tests/integration/agent_run_provider_driver_test.go`:

| Test function | Plan scenario |
|---|---|
| `TestAgentRunProviderDriver_APIDriver_RecordsRow` | An `openai-compatible` run (mock upstream via `testutil.MockOpenAIServer`) records `driver=="openai-compatible"` and `provider=="test-provider"` on the API run payload, and the log header (before the first output line) contains `driver=openai-compatible provider=test-provider` |

### Milestone 4 — CLI-driver end-to-end (empty provider)

`tests/integration/agent_run_provider_driver_test.go`:

| Test function | Plan scenario |
|---|---|
| `TestAgentRunProviderDriver_CLIDriver_RecordsRow` | A `claude-code-cli` run (fake `claude` stub via `setupFakeClaude`) records `driver=="claude-code-cli"` and `provider==""` on the API run payload, and the log header contains the literal empty `provider=` token alongside `driver=claude-code-cli` |

### Milestone 5 — Header consistency across every driver

`internal/agent/log_header_test.go`, plus existing driver tests extended in
this change's scope:

| Test function | Plan scenario |
|---|---|
| `TestWriteRunLogHeader_Format` | The shared `writeRunLogHeader` helper emits `driver=`/`provider=` on the header line for both a bound provider and an empty one, and only appends the `# args=` line when args is non-nil |
| `TestShellStubDriver_LogHeader_IncludesDriverProvider` | `shell-stub` — previously silent — now writes a header containing `driver=shell-stub provider=` |
| `TestCodexCLIDriver_LogHeader_IncludesDriverProvider` | `codex-cli` header (via the shared helper) contains `driver=codex-cli provider=` |
| `TestGeminiCliDriver_LogHeader_IncludesDriverProvider` | `gemini-cli` header (via the shared helper) contains `driver=gemini-cli provider=` |

The remaining driver families are already covered by pre-existing tests that
this change's implementation now satisfies without modification:

- `claude-code-cli`/`claude-env`/`claude-mediated` (shared `startCommandProcess`
  path): `TestAgentRunProviderDriver_CLIDriver_RecordsRow` above (Milestone 4).
- native `gemini`: `internal/agent/gemini_test.go::TestGeminiDriver_Start_Success`
  already asserts `driver=gemini provider=` in the header.
- `openai-compatible`: `tests/integration/openai_driver_test.go` already
  asserts `driver=openai-compatible` / `provider=test-prov` in the header;
  extended in Milestone 3 above with a full DB-row assertion.

### Milestone 6 — Secret-hygiene assertion (NFR-1)

`tests/integration/agent_run_provider_driver_test.go`:

| Test function | Plan scenario |
|---|---|
| `TestAgentRunProviderDriver_SecretHygiene` | An `openai-compatible` provider configured with a recognisable fake `api_key` (`SECRET-DO-NOT-LOG`, enforced by the mock server's `RequireAuthToken`) completes a run; the `driver`/`provider` API fields and the full run log text are asserted to never contain the token or a raw `Authorization:` header value |

### Milestone 7 — Frontend component tests

Out of scope for this artifact (see note above). Already implemented per
`lifecycle/frontend-plans/agent-logging-provider-driver-4-fe.md` Milestone 4
(`web/src/components/agent/__tests__/RunDetailModal.spec.ts`,
`web/src/views/project/__tests__/AgentsRunsView.spec.ts`).

## Verification

- `go build ./...`, `go vet ./...` — clean.
- `go test ./internal/index/... ./internal/agent/...` — pass, including all
  new tests above.
- `go test ./tests/integration/... -tags=integration -run TestAgentRunProviderDriver` — pass (3/3).
- Full `go test ./tests/integration/... -tags=integration` run: 13 pre-existing
  failures (`TestBackfill_*`, `TestFailover_AutoSwitch_*`,
  `TestSecrets_FailoverAudit`, `TestOpenAIDriver_LiveTarget_*`,
  `TestOpenAIRegression_ExistingDriversWork/ollama`,
  `TestOnboard_ExistingMode_AlreadyInitialised`, `TestProviderAPI_Delete`,
  `TestAgentDirectives_ADRAuthoringWritePath`) verified unrelated to this
  change — reproduced identically with the new test files removed from the
  tree.

## Test files

- `internal/index/index_agent_runs_test.go`
- `internal/agent/log_header_test.go`
- `tests/integration/agent_run_provider_driver_test.go`
