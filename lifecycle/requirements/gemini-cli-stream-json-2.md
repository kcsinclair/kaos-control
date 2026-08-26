---
title: gemini-cli driver — consume agy stream-json for run telemetry
type: requirement
status: done
lineage: gemini-cli-stream-json
parent: lifecycle/ideas/gemini-cli-stream-json.md
created: "2026-08-25T12:45:00+10:00"
priority: high
labels:
    - requirement
    - agent
    - agent-runner
    - driver
    - gemini
    - observability
release: KC-Release6
---

# gemini-cli driver — consume agy stream-json for run telemetry

## Problem

`GeminiCliDriver` runs `agy` in print mode without `--output-format`, so `agy`
uses its default `text` output. The driver's per-line `json.Unmarshal`
(`internal/agent/gemini_cli.go`) therefore almost never succeeds, every stdout
line becomes an unstructured `ProgressEvent{Raw: line}`, and **no `RunResult` is
produced for any Gemini run**.

The result is that `gemini-cli` — now the project's primary driver — is the
least observable one: no run summary card, no token or duration metrics, no
TTFT, and no CLI-reported success/failure signal.

`agy` has supported NDJSON streaming in print mode for some time
(`--output-format` accepts `text`, `json`, `stream-json`); the driver simply
never opted in.

## Goals / Non-goals

### Goals

- Have the `gemini-cli` driver request and consume `agy`'s `stream-json` output.
- Produce a `RunResult` for Gemini runs so the existing run summary card,
  metrics, and reports work for them as they do for Claude Code runs.
- Emit structured progress events and record TTFT from incremental output.
- Derive run success/failure from the CLI's own terminal event rather than
  process exit status alone.

### Non-goals

- Changing any other driver's behaviour, or the `RunResult` consumers
  (`internal/http/agents.go`, the summary card).
- `--input-format stream-json` (feeding NDJSON *to* agy). Only output is in
  scope.
- Tool-call mediation or permission interception for agy. `agy` is invoked with
  `--dangerously-skip-permissions` today and that is unchanged here.
- Cost reporting. `agy` emits no cost field; see FR-6.

## Verified event schema

Captured from the installed `agy` on 2026-08-25 (`--output-format stream-json`,
trivial prompt). **The discriminator is `event`, not `type`**, and each payload
nests under a key matching the event name:

```json
{"event":"init","conversation_id":"…","init":{"cwd":"…","tools":[…]}}
{"event":"step_update","step_update":{"conversation_id","step_index","step_type",
    "state","text_delta"?,"duration_seconds"?,"usage"?}}
{"event":"result","result":{"conversation_id","status":"SUCCESS","response":"OK\n",
    "num_turns":1,"duration_seconds":4.363532,
    "usage":{"input_tokens":19425,"output_tokens":25,"thinking_tokens":24,
             "cache_read_tokens":0,"total_tokens":19450}}}
```

Observed sequence for a one-turn run: one `init`, four `step_update`, one
`result` (6 lines).

## Detailed Requirements

### Functional

#### FR-1: Request stream-json output

- `GeminiCliDriver.buildArgs` appends `--output-format stream-json`.
- Existing args (`--dangerously-skip-permissions`, `--add-dir`,
  `--print-timeout`, `--prompt`) are unchanged in value and relative order;
  `internal/agent/gemini_cli_test.go` asserts the exact arg vector and must be
  updated to match.

#### FR-2: Parse agy-shaped NDJSON

- Add a **separate** agy result parser (e.g. `ParseAgyResultLine`) alongside the
  existing Claude-shaped `ParseResultLine`.
- `ParseResultLine` MUST NOT be reused: it scans for `{"type":"result",…}` with
  `is_error`, and would silently match nothing in agy output — failing quietly
  rather than loudly. Keep the two schemas explicitly distinct.
- Malformed or unrecognised lines are tolerated: they are logged/teed as raw
  progress, never fatal to the run.

#### FR-3: Populate RunResult from the terminal result event

Map `result.*` onto the existing `RunResult`:

| `RunResult` field | Source |
|---|---|
| `NumTurns` | `result.num_turns` |
| `DurationMs` | `result.duration_seconds` × 1000 |
| `DurationApiMs` | `result.duration_seconds` × 1000 (agy reports no separate API time) |
| `Usage.InputTokens` | `result.usage.input_tokens` |
| `Usage.OutputTokens` | `result.usage.output_tokens` |
| `Usage.CacheReadInputTokens` | `result.usage.cache_read_tokens` |
| `Subtype` | lowercased `result.status` (e.g. `success`) |
| `IsError` | `result.status != "SUCCESS"` |
| `Result` | `result.response` on success; the failure text/status otherwise |

- `Usage.CacheCreationInputTokens` has no agy equivalent and is left zero.

#### FR-4: Success and failure signalling

- A `result` event with `status: "SUCCESS"` marks the run successful; any other
  status marks it failed, with the status (and any message) recorded as the
  failure reason.
- Absence of a terminal `result` event when the process exits is itself a
  failure condition and must be distinguishable from a run that reported a
  non-success status.
- This must compose correctly with the existing `IsError`-over-`Subtype` rule
  (commit `7caa1536`): consumers already prefer `IsError`, so the mapping in
  FR-3 is sufficient — no consumer changes should be required.

#### FR-5: Progress events and TTFT

- Each `step_update` produces a `ProgressEvent`, with `text_delta` (when
  present) carried as the human-readable content so the Agents view streams
  incrementally.
- TTFT is recorded via the run's `OnTTFT` callback on the **first**
  `step_update` carrying a non-empty `text_delta`, matching how streaming
  drivers behave elsewhere.
- The raw NDJSON continues to be teed to the run log file unchanged.

#### FR-6: Fields with no kaos-control home

- **Cost** — `agy` reports none. `TotalCostUSD` stays `0`; it MUST NOT be
  fabricated or estimated. The summary card already renders a zero cost without
  error.
- **`thinking_tokens`** — no field exists on `RunResultUsage`. *Decision taken:*
  leave it uncaptured in v1 rather than widen the shared struct for one driver.
  Revisit if per-driver token breakdowns are wanted later; capturing it would
  mean a schema change across `RunResultUsage`, the API type, and the summary
  card.

### Non-functional

#### NFR-1: No regression for other drivers

- `ParseResultLine` and the Claude/Ollama/openai-compatible paths are untouched.
  Existing driver tests remain green.

#### NFR-2: Backward tolerance

- If a future/older `agy` rejects `--output-format stream-json`, or emits text
  anyway, the run must still complete and log its output rather than failing
  outright — degrade to today's raw-line behaviour with a clear warning.

#### NFR-3: Secret hygiene

- Unchanged: nothing in the NDJSON stream is treated as trusted, and no
  credentials are introduced by this work.

## Acceptance Criteria

- [ ] `buildArgs` includes `--output-format stream-json`; the arg-vector test in
      `gemini_cli_test.go` is updated and passes.
- [ ] A recorded agy NDJSON fixture (init + step_update ×N + result) parses into
      a `RunResult` with the correct turns, duration, and token counts.
- [ ] A `result` with `status: "SUCCESS"` yields `IsError=false`; any other
      status yields `IsError=true` with the status surfaced as the reason.
- [ ] A run whose process exits with **no** `result` event is recorded as failed
      and is distinguishable from a non-success status.
- [ ] TTFT is recorded on the first `text_delta`-bearing `step_update`.
- [ ] `step_update` events appear as streaming progress in the Agents view.
- [ ] The run summary card renders for a real `gemini-cli` run — turns,
      duration, and token usage populated; cost shown as zero.
- [ ] Malformed/unknown NDJSON lines do not fail the run.
- [ ] Claude Code, Ollama, and `openai-compatible` runs are unaffected
      (existing tests green).

## Notes

Schema captured empirically on 2026-08-25 against the installed `agy`
(`/Users/keith/.local/bin/agy`) rather than from documentation — the observed
shape differs from a Claude-style reading, which is the main implementation
risk this requirement exists to de-risk.
