---
title: "Test plan — gemini-cli agy stream-json telemetry"
type: plan-test
status: approved
lineage: gemini-cli-stream-json
parent: lifecycle/requirements/gemini-cli-stream-json-2.md
created: "2026-08-25T14:00:00+10:00"
---

# Test plan — gemini-cli agy stream-json telemetry

Covers the acceptance criteria of [[gemini-cli-stream-json]] (requirement
`gemini-cli-stream-json-2`). Companion plans: backend `-3-be`, frontend `-4-fe`.

## Architecture conformance

Pure-Go unit tests in `internal/agent` (table-driven, no external process where
avoidable) plus optional integration coverage in `tests/`. No new dependency; no
deviation from the go-vue stack or standards.

## Fixture

Author one canonical agy NDJSON fixture from the requirement's verified schema —
one `init`, four `step_update` (at least one carrying a non-empty `text_delta`,
at least one without), one `result` with `status:"SUCCESS"` and the recorded
`usage`. Keep it as a Go string constant or a `testdata/agy_run.ndjson` file
reused across the parser and TTFT tests. Derive variants (non-success status,
missing result, malformed line) from it.

## Milestone 1 — buildArgs arg-vector (FR-1)

**Description.** Update `TestGeminiCliDriver_BuildArgs`
(`internal/agent/gemini_cli_test.go`) so all three sub-tests assert
`--output-format stream-json` in its fixed position, with every pre-existing arg
unchanged in value and relative order.

**Files to change.** `internal/agent/gemini_cli_test.go`.

**Acceptance criteria.** The three sub-tests pass against the updated
`buildArgs`; a regression that drops or reorders the flag fails the test.

## Milestone 2 — ParseAgyResultLine happy path (FR-3)

**Description.** Unit-test `ParseAgyResultLine` against the SUCCESS fixture:
assert `NumTurns`, `DurationMs`/`DurationApiMs` (= round(seconds×1000)),
`Usage.InputTokens`, `Usage.OutputTokens`, `Usage.CacheReadInputTokens`,
`Usage.CacheCreationInputTokens == 0`, `TotalCostUSD == 0`,
`Subtype == "success"`, `IsError == false`, `Result == result.response`.

**Files to change.** `internal/agent/agy_result_test.go` (new).

**Acceptance criteria.** All mapped fields match the fixture; cost and
cache-creation are exactly zero (never fabricated).

## Milestone 3 — Success/failure mapping (FR-4)

**Description.** Table-driven cases over `result.status`:
- `SUCCESS` → `IsError=false`, reason empty.
- e.g. `ERROR` / `CANCELLED` / any non-`SUCCESS` → `IsError=true`,
  `Subtype` = lowercased status, and the status/message surfaced in `Result`.

**Files to change.** `internal/agent/agy_result_test.go`.

**Acceptance criteria.** Exactly `status == "SUCCESS"` yields a non-error result;
every other status yields an error with the reason surfaced.

## Milestone 4 — Missing terminal result → truncated_stream (FR-4)

**Description.** Two layers:
1. Parser: a log with `init` + `step_update`s but **no** `result` returns
   `errNoAgyResultLine`.
2. Supervisor semantics: a `gemini-cli` run that exits cleanly with no `result`
   event is recorded `failed` with `failure_reason="truncated_stream"`, and this
   is distinguishable from a recorded non-success status. Assert via a driver
   test using a shim agy that prints partial NDJSON then exits 0, or a
   supervisor-level test that drives the drain loop with such a stream.

**Files to change.** `internal/agent/agy_result_test.go`,
`internal/agent/gemini_cli_test.go` (or supervisor test).

**Acceptance criteria.** No-result run → `failed` + `truncated_stream`;
non-success-status run → `failed` with the status as reason; the two are
distinguishable.

## Milestone 5 — TTFT on first text_delta (FR-5)

**Description.** Drive `GeminiCliDriver.Start` with a shim (re-exec pattern,
like the existing `GO_WANT_HELPER_PROCESS` mock) that emits the fixture stream,
wiring a capturing `run.OnTTFT`. Assert the callback fires exactly once and on
the first `step_update` bearing a non-empty `text_delta` (not on `init` or on a
`text_delta`-less step).

**Files to change.** `internal/agent/gemini_cli_test.go`.

**Acceptance criteria.** `OnTTFT` invoked exactly once, keyed to the first
`text_delta`-bearing `step_update`.

## Milestone 6 — Progress events surface (FR-5)

**Description.** Consuming `proc.Progress()` over the fixture yields a
`ProgressEvent` per agy line, each `step_update` carrying the parsed object with
`text_delta` accessible; raw NDJSON is teed to the log verbatim.

**Files to change.** `internal/agent/gemini_cli_test.go`.

**Acceptance criteria.** One progress event per line; `step_update` payloads
expose `text_delta`; log file contains the raw NDJSON lines.

## Milestone 7 — Malformed/unknown tolerance & degradation (NFR-2)

**Description.** Feed a stream with a malformed JSON line and a plain-text line
interleaved with valid agy events. Assert neither is fatal: they become raw
`output` progress events, valid events still parse, and a run that emits only
plain text completes (recorded failed / `truncated_stream`) rather than
crashing.

**Files to change.** `internal/agent/gemini_cli_test.go`,
`internal/agent/agy_result_test.go`.

**Acceptance criteria.** Malformed/unknown lines never fail the run; degraded
(text-only) runs complete and log output with a warning.

## Milestone 8 — No cross-driver regression (NFR-1)

**Description.** Run the full `internal/agent` and `internal/http` suites; assert
`ParseResultLine` and Claude/Ollama/openai-compatible driver tests are unchanged
and green. Confirm the driver-dispatch helper (plan `-3-be` M3) routes
non-`gemini-cli` drivers to `ParseResultLine`.

**Files to change.** None (existing suites); add a dispatch unit test if the
helper is separately testable.

**Acceptance criteria.** `go test ./internal/agent/... ./internal/http/...`
green; `make lint` clean.

## Milestone 9 — End-to-end summary card (manual/integration)

**Description.** Verify against a real `gemini-cli` run (or a recorded fixture
fed through the WS pipeline) that `RunSummaryCard` renders turns, duration, and
token usage, with cost shown as zero, and that `step_update` events appear as
streaming progress in the Agents view (ties to frontend plan `-4-fe`).

**Files to change.** Optionally `tests/integration/` if a scripted agy fixture
run is added; otherwise a documented manual check.

**Acceptance criteria.** Summary card populated (cost zero) and live progress
streams for a `gemini-cli` run.
