---
title: Tests — gemini-cli agy stream-json telemetry
type: test
status: draft
lineage: gemini-cli-stream-json
parent: lifecycle/test-plans/gemini-cli-stream-json-5-test.md
created: "2026-08-25T20:30:00+10:00"
---

# Tests — gemini-cli agy stream-json telemetry

Covers [[gemini-cli-stream-json]] (requirement `gemini-cli-stream-json-2`), test
plan `gemini-cli-stream-json-5-test.md`. All new tests live in
`tests/integration/gemini_cli_agy_test.go` (build tag `integration`, package
`integration`).

## Scope note (why this isn't the internal/agent unit tests the plan describes)

The test plan's Milestones 1–8 largely call for pure-Go unit tests inside
`internal/agent` (`gemini_cli_test.go`, a new `agy_result_test.go`), asserting
against unexported helpers (`buildArgs`, `isResultEvent`,
`resultEventIsError`, `driverEmitsResultEvent`). The test-developer role's
write scope is `tests/**`, `lifecycle/tests/`, and
`lifecycle/architecture/decisions/` only — it cannot write to `internal/`.

Cross-checked against the backend plan (`gemini-cli-stream-json-3-be.md`,
status `done`) and its commit history: the backend-developer agent already
added/updated in-package tests for Milestone 1 (`buildArgs`, commit
`bfeea7ee`) and exercised Milestones 4–6 incidentally via
`TestDriverEmitsResultEvent` and `TestGeminiCliDriver_Start` (commits
`1fbe8dee`, `be990af0`, `6d34772b`). However, `ParseAgyResultLine` — the core
of Milestones 2, 3, 4 (layer 1), and 7 — shipped with **no** test coverage at
all (commit `84d5f1ac` added only `agy_result.go`, no test file).

Since `ParseAgyResultLine`, `GeminiCliDriver`, `GeminiCliDriver.BinaryPath`,
`Run`, `Run.OnTTFT`, `Process`, and `ProgressEvent` are all exported, that gap
— plus Milestones 5, 6, and the driver-layer half of 7 — is fully testable
from an external package. `tests/` is a sibling of `internal/` under the same
module root, so it can import `internal/agent`'s exported surface without
violating Go's internal-package visibility rule. The two exceptions:

- **Milestone 1** (`buildArgs` arg-vector) is unexported and untestable from
  outside the package; it is already covered by the backend-developer's
  in-package update to `internal/agent/gemini_cli_test.go`. No action needed
  here.
- **Milestone 8** ("no cross-driver regression") has no files to change per
  the plan itself — it's satisfied by the existing `internal/agent` and
  `internal/http` suites staying green, verified by `go test
  ./internal/agent/... ./internal/http/...` (confirmed passing).

Milestone 4's second layer (supervisor-level `truncated_stream`
classification) and Milestone 9 (end-to-end summary-card telemetry + live
progress) both require driving a real run through the HTTP/supervisor stack —
squarely `tests/integration/` territory — and are covered by dedicated
end-to-end tests below using the same PATH-shadowing shim pattern the
existing `claude-code-cli` integration tests use (`setupFakeClaude` in
`agent_helpers_test.go`), since `GeminiCliDriver` has no config-level
binary-path override.

## Fixture

`agyInitLine`, `agyStepUpdateLines` (4 lines — index 1 carries the fixture's
only non-empty `text_delta`), and `agySuccessResultLine` in
`tests/integration/gemini_cli_agy_test.go` reproduce the verified schema from
the requirement (one `init`, four `step_update`, one `result`,
`status:"SUCCESS"`). `agyFixtureLines`/`agyFixtureLog` assemble variants;
`agyResultLine(status, response)` builds arbitrary terminal-event variants for
the status-mapping table.

## Scenarios covered

### Parser-level (`agent.ParseAgyResultLine`)

- `TestParseAgyResultLine_HappyPath` — Milestone 2. Asserts `NumTurns`,
  `DurationMs`/`DurationApiMs` (rounded from `duration_seconds`), all `Usage`
  fields including `CacheCreationInputTokens==0`, `TotalCostUSD==0`,
  `Subtype=="success"`, `IsError==false`, `Result` against the fixture.
- `TestParseAgyResultLine_StatusMapping` — Milestone 3. Table-driven over
  `SUCCESS`/`ERROR`/`CANCELLED`: `IsError`, lowercased `Subtype`, and the
  status/response surfaced in `Result`.
- `TestParseAgyResultLine_NoResultLine` — Milestone 4, layer 1. A log with
  `init` + `step_update`s and no `result` line returns a non-nil error and
  nil result.
- `TestParseAgyResultLine_ToleratesMalformedAndPlainTextLines` — Milestone 7
  (parser layer). A malformed JSON line and a plain-text line interleaved
  around the valid result line don't prevent it from parsing.

### Driver-level (`agent.GeminiCliDriver.Start` against a shim `agy` script)

- `TestGeminiCliDriver_TTFT` — Milestone 5. `run.OnTTFT` fires exactly once,
  and the fixture is asserted to have an `init` event and a delta-less
  `step_update` preceding the first `text_delta`, so the single TTFT call can
  only have fired on that first delta-bearing step.
- `TestGeminiCliDriver_ProgressEvents` — Milestone 6. One `ProgressEvent` per
  fixture line (plus the synthetic `started` marker), `step_update` events
  expose `text_delta`, and the raw NDJSON is teed to the log file verbatim.
- `TestGeminiCliDriver_MalformedAndUnknownLinesDegradeGracefully` — Milestone
  7 (driver layer). A malformed JSON line and a plain-text line both degrade
  to `type:"output"` progress events and are never fatal; a valid
  `step_update` after them still parses natively.

### End-to-end (`tests/integration` HTTP + supervisor stack, shim `agy` on `PATH`)

- `TestGeminiCliRun_EndToEnd_SuccessTelemetryAndProgress` — Milestone 9. A
  `gemini-cli` run streams an `agent.progress` `step_update` event carrying
  `text_delta` over the WebSocket hub, completes `status:"done"`, and `GET
  .../runs/{id}/result` returns turns/duration/usage populated with
  `total_cost_usd:0`.
- `TestGeminiCliRun_EndToEnd_NoResultEvent_RecordsTruncatedStream` —
  Milestone 4, layer 2. A clean exit with `init`+`step_update`s but no
  `result` event is recorded `status:"failed"`,
  `failure_reason:"truncated_stream"`.
- `TestGeminiCliRun_EndToEnd_PlainTextOnly_CompletesAsTruncatedStream` —
  Milestone 7 (NFR-2, end-to-end). An `agy` emitting only plain text (e.g. it
  rejected `--output-format stream-json`) still completes rather than hanging
  or crashing, landing on the same distinguishable `truncated_stream` outcome.

## Verification

```
go test -tags=integration ./tests/integration/ -run 'TestParseAgyResultLine|TestGeminiCliDriver_|TestGeminiCliRun_' -v
go test -tags=integration ./tests/...
go vet -tags=integration ./tests/integration/...
gofmt -l tests/integration/gemini_cli_agy_test.go
```

All pass as of this writing. `go test ./internal/agent/... ./internal/http/...`
(Milestone 8) also verified green — no cross-driver regression.
