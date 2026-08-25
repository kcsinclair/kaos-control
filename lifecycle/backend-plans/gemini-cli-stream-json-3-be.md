---
title: Backend plan — gemini-cli driver consumes agy stream-json for run telemetry
type: plan-backend
status: done
lineage: gemini-cli-stream-json
created: "2026-08-25T14:00:00+10:00"
parent: lifecycle/requirements/gemini-cli-stream-json-2.md
release: KC-Release6
---

# Backend plan — gemini-cli driver consumes agy stream-json

Implements the backend of [[gemini-cli-stream-json]] (requirement
`gemini-cli-stream-json-2`). Companion plans: frontend `-4-fe`, test `-5-test`.

## Architecture conformance

Confirmed against `lifecycle/architecture/`:

- **Modular monolith / go-vue, single binary** — all work is pure-Go inside
  `internal/agent` and `internal/http`; no new dependency, no cgo, no external
  service. Conforms to the single-self-contained-binary invariant in
  `architecture-summary.md`.
- **No shared-schema break** — `RunResult` / `RunResultUsage` are reused
  as-is. Per FR-6 we deliberately do **not** widen the struct for
  `thinking_tokens`; that avoids a schema change rippling across the API type
  and the summary card, so no ADR is triggered.
- **[[adr-0006-mediated-agent-driver-permission-model]]** — agy is still
  invoked with `--dangerously-skip-permissions` and tool-call mediation for agy
  remains out of scope (requirement non-goal). This plan does **not** regress
  the mediation model; it also does not add agy mediation. That remaining gap is
  pre-existing and unchanged — no new ADR, but noted here so it is not lost.

No deviation from the recorded architecture, stack, or standards is required.

## Key facts established from the code

- `GeminiCliDriver` has its **own** `Start()` with its own stdout scanner
  goroutine (`internal/agent/gemini_cli.go`), separate from the shared
  claude-style scanner in `agent.go`. TTFT and per-line parsing for agy must be
  wired **inside this driver's goroutine**, not the shared one.
- Post-run telemetry is produced by re-reading the log and calling
  `ParseResultLine` in **two** places: the supervisor broadcast/metrics path
  (`internal/agent/agent.go:1343`) and the on-demand REST endpoint
  (`internal/http/agents.go:465`). Both must dispatch by driver.
- The mid-stream `broadcast` closure (`agent.go:965`) already runs for the
  gemini path (the `default` drain branch at `agent.go:1058` forwards through
  it), and it sets `resultEventSeen` / `resultEventSuccess` via
  `isResultEvent` / `resultEventIsError`. Those helpers currently match only the
  Claude shape (`ev["type"]=="result"`), so agy's `ev["event"]=="result"` slips
  through and the run is wrongly flagged `truncated_stream`. They must become
  agy-aware.
- `driverEmitsResultEvent` (`agent.go:2059`) gates both the truncated-stream
  check (`agent.go:1161`) and TTFT wiring (`agent.go:693`). Adding
  `gemini-cli` there is the single switch that turns both on for this driver.
- The discriminator in agy NDJSON is **`event`**, and each payload nests under a
  key equal to the event name (`init`, `step_update`, `result`). See the
  verified schema in the requirement.

---

## Milestone 1 — Request stream-json output (FR-1)

**Description.** Make `GeminiCliDriver.buildArgs` opt into agy's NDJSON stream.
Append `--output-format stream-json`. Preserve the value and relative order of
every existing arg (`--dangerously-skip-permissions`, `--add-dir`,
`--print-timeout`, `--prompt`). Choose a stable insertion point (recommended:
immediately after `--dangerously-skip-permissions`, before the conditional
`--add-dir`) and lock that order in the test.

**Files to change.**
- `internal/agent/gemini_cli.go` — `buildArgs`.
- `internal/agent/gemini_cli_test.go` — update all three arg-vector
  expectations (`withProjectRootAndUnlimitedTimeout`, `withExplicitTimeout`,
  `withoutProjectRoot`) to include the new flag in the exact expected position.
  (Covered in detail by the test plan `-5-test`.)

**Acceptance criteria.**
- `buildArgs` output contains `--output-format stream-json`.
- The three existing sub-tests pass with the flag asserted in a fixed position,
  and all prior args are byte-identical in value and relative order.

---

## Milestone 2 — agy result parser (FR-2, FR-3)

**Description.** Add a **separate**, agy-shaped result parser; do not reuse
`ParseResultLine`. Introduce `ParseAgyResultLine(logContent string)
(*RunResult, error)` plus the agy decode structs (e.g. `agyResultEvent`,
`agyResult`, `agyUsage`). Scan the log from the end for a line whose top-level
`event == "result"`, decode the nested `result` object, and map onto the
existing `RunResult`:

| `RunResult` field | Source |
|---|---|
| `NumTurns` | `result.num_turns` |
| `DurationMs` | round(`result.duration_seconds` × 1000) |
| `DurationApiMs` | round(`result.duration_seconds` × 1000) |
| `Usage.InputTokens` | `result.usage.input_tokens` |
| `Usage.OutputTokens` | `result.usage.output_tokens` |
| `Usage.CacheReadInputTokens` | `result.usage.cache_read_tokens` |
| `Usage.CacheCreationInputTokens` | 0 (no agy equivalent) |
| `Subtype` | lowercased `result.status` |
| `IsError` | `result.status != "SUCCESS"` |
| `Result` | `result.response` on success; failure text/status otherwise |
| `TotalCostUSD` | 0 (agy reports none — never fabricated, FR-6) |

Return a sentinel error (e.g. `errNoAgyResultLine`, mirroring
`errNoResultLine`) when no `result` line exists, so callers treat "absent" as a
normal, non-fatal condition and the supervisor's truncated-stream logic (M4)
can act on it.

Place the parser in a new file `internal/agent/agy_result.go` (keeps the two
schemas visibly distinct per FR-2) or in `result.go` under a clear section
comment — either is acceptable; new file preferred for isolation.

**Files to change.**
- `internal/agent/agy_result.go` (new) — `ParseAgyResultLine`, agy structs,
  `errNoAgyResultLine`.

**Acceptance criteria.**
- A recorded agy fixture (`init` + `step_update`×N + `result`) parses into a
  `RunResult` with correct `NumTurns`, `DurationMs`/`DurationApiMs`, and token
  counts, `CacheCreationInputTokens == 0`, `TotalCostUSD == 0`.
- `status:"SUCCESS"` → `IsError == false`, `Subtype == "success"`,
  `Result == result.response`.
- Any other status → `IsError == true`, `Subtype` = lowercased status, and the
  status/message surfaced in `Result`.
- A log with no `result` line returns `errNoAgyResultLine`.
- `ParseResultLine` is untouched and still ignores agy input.

---

## Milestone 3 — Driver-aware result-parsing dispatch (FR-3)

**Description.** Route post-run parsing to the right parser by driver. Add a
small helper (e.g. `parseRunResult(driver, logContent)`) that calls
`ParseAgyResultLine` for `gemini-cli` and `ParseResultLine` otherwise, so the
two call sites share one dispatch rule.

- `internal/agent/agent.go:~1343` — replace the direct `ParseResultLine` call
  with the dispatch helper using `run.Driver`. The downstream
  `index.AgentRunMetrics` population and `agent.finished`/`agent.failed`
  broadcast payload are unchanged (they read `RunResult` fields the agy parser
  now fills).
- `internal/http/agents.go:~465` — the on-demand run-result endpoint. Look up
  the run's driver (from the agent-run row / project agent config) and dispatch
  the same way. If the driver cannot be resolved, fall back to the existing
  Claude parser (preserves today's behaviour).

**Files to change.**
- `internal/agent/agent.go` — supervisor post-run parse site + new dispatch
  helper.
- `internal/http/agents.go` — `handleGetAgentRunResult` parse site.

**Acceptance criteria.**
- A completed `gemini-cli` run yields a non-nil `RunResult` in both the
  `agent.finished` broadcast and the `GET …/result` endpoint, with turns,
  duration, and token metrics persisted to `AgentRunMetrics`.
- Claude / Ollama / openai-compatible runs still parse via `ParseResultLine`
  (no behavioural change).

---

## Milestone 4 — Terminal-event detection & success/failure (FR-4)

**Description.** Make the supervisor's terminal-event machinery agy-aware so
success, non-success, and the "no result event at all" case are correctly
distinguished, composing with the existing `IsError`-over-`Subtype` rule
(commit `7caa1536`).

1. **Enable the contract for gemini-cli.** Add `"gemini-cli"` to
   `driverEmitsResultEvent` (`agent.go:2059`). This turns on (a) TTFT wiring at
   `agent.go:693` and (b) the truncated-stream check at `agent.go:1161`.
2. **Teach the detectors agy's shape.** Update `isResultEvent` (`agent.go:2085`)
   to also match `payload["event"] == "result"`, and `resultEventIsError`
   (`agent.go:2096`) to read agy's nested `result.status != "SUCCESS"` when the
   agy shape is present, while leaving the Claude `type`/`is_error` path intact.
   Keep these branches explicit and commented so the two schemas stay legible.
   (Alternative if preferred: dedicated `isAgyResultEvent` /
   `agyResultIsError` helpers dispatched by driver — either is fine provided the
   `broadcast` closure sets `resultEventSeen`/`resultEventSuccess` correctly for
   agy.)
3. **Result-present success semantics.** With (1)+(2), a `result` with
   `SUCCESS` sets `resultEventSeen=true, resultEventSuccess=true`, so a nonzero
   agy exit code cannot override a clean success (`agent.go:1138`), and a
   non-success `result` leaves `resultEventSuccess=false`.
4. **Result-absent failure.** A `gemini-cli` run that exits `done` with no
   `result` event trips the truncated-stream branch → `status="failed"`,
   `failureReason="truncated_stream"`. This is the required, distinguishable
   "no terminal event" failure, separate from a recorded non-success status
   (whose reason comes from the parsed `RunResult`).

**Files to change.**
- `internal/agent/agent.go` — `driverEmitsResultEvent`, `isResultEvent`,
  `resultEventIsError` (or new agy helpers).

**Acceptance criteria.**
- `status:"SUCCESS"` result → run recorded successful, `IsError=false`, and a
  nonzero exit does not flip it to failed.
- Any non-success status → run recorded failed with the status surfaced as the
  reason (via the parsed `RunResult.Result`/`Subtype`).
- Process exits with **no** `result` event → `failed` with
  `failure_reason="truncated_stream"`, distinguishable from a non-success
  status.
- No consumer of `RunResult` (`agents.go`, summary card) needs changing —
  verified by existing tests staying green.

---

## Milestone 5 — Progress events & TTFT in the driver (FR-5)

**Description.** Rework the stdout goroutine in `GeminiCliDriver.Start`
(`gemini_cli.go:118`) to understand agy NDJSON:

- Continue teeing every raw line to the log file unchanged (FR-5, NFR-2).
- For each line, `json.Unmarshal` into a generic map. When it decodes and has an
  `event` key, forward it as `ProgressEvent{Raw: line, Event: parsed}` so the
  frontend (plan `-4-fe`) can render the native agy shape. When it does not
  decode as JSON, keep today's raw-text wrapping
  (`{type:"output", text: line+"\n"}`) — this is the NFR-2 degradation path.
- **TTFT.** Track a `firstTokenSeen` bool in the goroutine. On the first
  `step_update` whose nested `step_update.text_delta` is a non-empty string,
  call `run.OnTTFT(time.Since(runStart).Milliseconds())` once. Capture
  `runStart` right after `cmd.Start()` succeeds (mirror `agent.go:271`). The
  callback is nil-guarded (`run.OnTTFT` is wired by M4 step 1 via
  `driverEmitsResultEvent`).

Keep the detached-grandchild handling (async `cmd.Wait`, `waitErr`) exactly as
is — it is load-bearing (see `TestGeminiCliDriver_DetachedChildHoldsPipes`).

**Files to change.**
- `internal/agent/gemini_cli.go` — stdout goroutine: agy-aware parse, TTFT hook,
  `runStart` capture.

**Acceptance criteria.**
- Each `step_update` line produces a `ProgressEvent` carrying the parsed agy
  object (with `text_delta` available to the UI).
- `run.OnTTFT` fires exactly once, on the first non-empty `text_delta`.
- Raw NDJSON is still teed to the log verbatim.
- Non-JSON / malformed lines still produce a raw `output` progress event and are
  never fatal (composes with NFR-2).

---

## Milestone 6 — Backward tolerance & degradation (NFR-1, NFR-2, NFR-3)

**Description.** Guarantee graceful degradation and no cross-driver regression.

- If a future/older agy rejects `--output-format stream-json` or emits plain
  text anyway, the run must still complete and log its output: unparseable lines
  fall through to the raw-text `output` event path (M5), and the absence of a
  `result` event is handled by M4 (recorded as a failure with a clear log
  warning rather than a crash). Add a single `slog.Warn` when a `gemini-cli` run
  finishes with no parseable agy `result` event, to make the degradation
  visible.
- Confirm no change to `ParseResultLine` or the Claude/Ollama/openai-compatible
  code paths (NFR-1).
- No credentials introduced; nothing in the NDJSON is treated as trusted
  (NFR-3) — unchanged.

**Files to change.**
- `internal/agent/gemini_cli.go` and/or `internal/agent/agent.go` — degradation
  warning only; no other behavioural change.

**Acceptance criteria.**
- A simulated agy that emits only plain text still completes, logs its output,
  and is recorded as failed (`truncated_stream`) rather than crashing, with a
  warning logged.
- `go test ./internal/agent/... ./internal/http/...` green, including untouched
  Claude/Ollama/openai-compatible driver tests (NFR-1).
- `make lint` (go vet + staticcheck) clean.

---

## Out of scope (restated from requirement)

- `--input-format stream-json`, tool-call mediation for agy, cost reporting, and
  capturing `thinking_tokens`. See requirement Goals/Non-goals and FR-6.
