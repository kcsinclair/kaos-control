---
title: gemini-cli driver should use agy's stream-json output
type: idea
status: done
lineage: gemini-cli-stream-json
created: "2026-08-25T12:40:00+10:00"
priority: high
labels:
    - idea
    - agent
    - agent-runner
    - driver
    - gemini
    - observability
release: KC-Release6
---

# gemini-cli driver should use agy's stream-json output

The Antigravity CLI (`agy`) now supports **NDJSON streaming** in print mode via
`--output-format stream-json`, emitting `init`, `step_update`, and terminal
`result` events. The `gemini-cli` driver does not use it, so every Gemini run
loses the structured telemetry that Claude Code runs get.

## Why this matters now

Development on this project has moved onto `gemini-cli`, so this is the driver
most runs go through — and it is currently the least observable one.

`GeminiCliDriver.buildArgs` passes `--dangerously-skip-permissions`,
`--add-dir`, `--print-timeout` and `--prompt`, but **never `--output-format`**,
so `agy` falls back to its default `text`. The driver does attempt a
`json.Unmarshal` per stdout line, but against plain prose that essentially never
succeeds, so every line becomes an unstructured raw progress event and no
`RunResult` is ever produced.

Consequences today:

- **No run summary card** for Gemini runs — no turns, duration, or token usage.
- **No success/failure signal from the CLI**, so the run outcome rests entirely
  on process exit status.
- **No streaming progress or TTFT**, despite `agy` emitting incremental
  `text_delta` updates.

## Verified event shape

Probed directly against the installed `agy` (2026-08-25) rather than assumed —
and it is **not** Claude-shaped. The discriminator is `event`, not `type`, and
each payload nests under a key matching the event name:

```json
{"event":"init","conversation_id":"…","init":{"cwd":"…","tools":[…]}}
{"event":"step_update","step_update":{"step_index","step_type","state",
    "text_delta"?,"duration_seconds"?,"usage"?}}
{"event":"result","result":{"status":"SUCCESS","response":"OK\n",
    "num_turns":1,"duration_seconds":4.363532,
    "usage":{"input_tokens":19425,"output_tokens":25,"thinking_tokens":24,
             "cache_read_tokens":0,"total_tokens":19450}}}
```

This matters: `ParseResultLine` scans for Claude's `{"type":"result",…}` with
`is_error`, so it classifies **none** of these. A separate agy-shaped parser is
required — reusing the Claude one would silently produce nothing, the same trap
as the `is_error` bug fixed in `7caa1536`.

## Shape of the work

- Pass `--output-format stream-json` in `buildArgs`.
- Parse the `event`-keyed NDJSON: `result` → `RunResult`, `step_update` →
  progress events and TTFT.
- Map `status: "SUCCESS"` to the run outcome, and `usage` / `num_turns` /
  `duration_seconds` onto the existing summary fields.

Two mapping gaps to settle: `agy` reports **no cost**, and it has a
`thinking_tokens` field with no home in `RunResultUsage`.

Related: [[open-provider-support]] (the provider/driver direction generally),
and the `is_error` display fix (`7caa1536`) for why driver-specific result
semantics must not be assumed.
