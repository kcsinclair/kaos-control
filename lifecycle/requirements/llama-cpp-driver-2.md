---
title: llama.cpp Agent Driver for Local Models
type: requirement
status: blocked
lineage: llama-cpp-driver
priority: normal
parent: lifecycle/ideas/llama-cpp-driver.md
labels:
    - driver
    - agent
    - agent-runner
    - backend
    - go
    - portability
    - feature
release: KC-Release5
assignees:
    - role: product-owner
      who: agent
---

# llama.cpp Agent Driver for Local Models

## Problem

kaos-control can already reach local models two ways, but neither gives a
locally-hosted model the *full agent-mode* experience:

- The native **`ollama`** driver ([[ollama-agent-support]]) is a single-shot
  `/api/chat` / `/api/generate` client — one prompt, one response, **no tool
  use, no file edits, no multi-turn loop**.
- The **`claude-env`** driver ([[ollama-claude-code-driver]]) does give the full
  agentic loop, but only by shelling out to the `claude` binary retargeted at an
  Anthropic-compatible endpoint. It depends on that external CLI and its
  Anthropic wire protocol; it is not a first-party path to a raw llama.cpp
  server.

llama.cpp's server exposes an OpenAI-compatible `/v1/chat/completions` endpoint
that natively supports tool/function calling and multi-turn conversation. There
is currently no driver that speaks this endpoint directly, so a raw
`llama-server` instance cannot participate in the same agent-mode workflow
(multi-turn conversation with tool use that reads and writes artifacts) as the
cloud-backed providers. This is the same "local models can talk but cannot
*act*" gap identified in [[ollama-agents-need-execution-layer]].

## Goals / Non-goals

### Goals

- Add a new first-party agent driver (`driver: llama-cpp`) that drives a
  llama.cpp server over its OpenAI-compatible `/v1/chat/completions` endpoint.
- Support **agent mode**: a bounded multi-turn loop that advertises tool
  definitions, receives `tool_calls` from the model, executes them locally, and
  feeds results back until the model returns a final assistant message with no
  further tool calls (or a safety cap is hit).
- Execute tool calls against the same sandbox and `allowed_write_paths` scoping
  that governs other agents, so a local model can produce and edit artifacts
  under `lifecycle/` exactly as a cloud agent does.
- Emit the same `ProgressEvent` stream, per-run log file, TTFT measurement, and
  cancellation/kill behaviour as the existing drivers, and respect the global
  `max_concurrent_agents` semaphore.
- Ship example configurations and tests in `~/Code/agent-benchmark` covering
  driver initialisation, a single tool-call round-trip, and a multi-step agent
  run, documenting the required `llama-server` flags and recommended models.
- Leave every existing driver (`claude-code-cli`, `claude-mediated`,
  `claude-env`, `ollama`, `gemini`, `codex-cli`, `shell-stub`) behaviourally
  unchanged.

### Non-goals

- Managing the llama.cpp server lifecycle (build, launch, model download,
  GPU/CPU flags). The driver is a client of an already-running `llama-server`;
  operators start and configure it out of band.
- Re-implementing the OpenAI protocol beyond the subset needed for chat
  completions with tool calls (`messages`, `tools`, `tool_calls`,
  `tool_choice`, streaming deltas, `finish_reason`).
- Guaranteeing tool-use fidelity of any particular GGUF model. The driver's
  contract is "advertise tools, execute what the model calls, loop"; whether a
  given model reliably emits well-formed tool calls is the model's concern and a
  documentation/recommendation matter, not a correctness requirement.
- A frontend driver-picker or instance-management UI. v1 is config-file driven;
  UI surfacing is an open question.
- Streaming partial tool-call arguments token-by-token to the UI. Assembling a
  complete tool call before executing it is sufficient for v1.

## Detailed Requirements

### Functional

#### FR-1: Driver registration and selection

- A new driver is registered in the agent `Manager`'s driver map under the
  stable name **`llama-cpp`**, selected purely on the `AgentConfig.driver`
  field, identical to how `ollama`/`gemini`/`codex-cli` are selected. Unknown
  driver names are rejected as today.
- The driver satisfies the existing `agent.Driver` interface
  (`Start(ctx, Run) (Process, error)`) and returns a `Process` implementing
  `Wait`, `Kill`, `Progress`, and `StderrTail`.

#### FR-2: Configuration fields

- `AgentConfig` gains fields used only when `driver: llama-cpp`:
  - `base_url` (string, **required**) — the llama.cpp server root, e.g.
    `http://localhost:8080`. The driver targets `<base_url>/v1/chat/completions`.
  - `model` (string, **required**) — the model identifier passed as the request
    `model` field. (llama.cpp typically serves one model; the value is still
    sent for compatibility and logging.)
  - `api_key` / `auth_token` (string, **optional**) — sent as
    `Authorization: Bearer <token>` when non-empty, for servers started with
    `--api-key`.
- Field naming reuses existing conventions (`base_url` already exists on
  `OllamaInstance`; secret masking mirrors `ollama_instances.api_key`).

#### FR-3: Config validation

- `config.Validate` rejects a `llama-cpp` agent when `base_url` is empty, when
  `base_url` is not a valid `http`/`https` URL, or when `model` is empty. Each
  failure message names the offending agent.
- Existing per-driver validation for other drivers is unchanged.

#### FR-4: Request construction

- On `Start`, the driver POSTs a JSON body to `<base_url>/v1/chat/completions`
  containing at minimum: `model`, a `messages` array, a `tools` array (the
  advertised tool schemas, see FR-5), and `stream: true`.
- The initial `messages` array is built from the run's prompt using the existing
  `---SYSTEM--- / ---USER---` split convention (`splitPrompt`), producing a
  `system` message (when present) and a `user` message.
- `Content-Type: application/json` is set; `Authorization: Bearer <token>` is
  set only when a token is configured.

#### FR-5: Tool advertisement and execution loop

- The driver advertises a defined set of tool functions to the model via the
  `tools` parameter (OpenAI function-calling schema). The v1 tool set MUST be
  sufficient to create and edit lifecycle artifacts; the concrete set (e.g.
  `read_file`, `write_file`, `list_dir`, and optionally a shell/patch tool) is
  an Open Question, but at minimum a file-read and a file-write tool are
  required.
- When the model response has `finish_reason: tool_calls`, the driver executes
  each requested tool call **locally**, appends an assistant message carrying
  the `tool_calls` and a matching `tool` message per result (keyed by
  `tool_call_id`), and re-issues the completion — repeating until the model
  returns an assistant message with no tool calls (`finish_reason: stop`).
- Tool execution MUST go through the existing sandbox path resolver so reads and
  writes cannot escape the project root, and writes MUST be constrained to the
  agent's configured `allowed_write_paths`. A write outside the allowed paths is
  refused and reported back to the model as a tool error (it does not crash the
  run or write outside scope).
- The loop is bounded by a configurable maximum iteration count (default TBD,
  see Open Questions). Hitting the cap ends the run with a clear terminal
  status and a logged reason rather than looping unbounded.

#### FR-6: Streaming, progress, and TTFT

- The driver consumes the endpoint's SSE / streamed delta response and emits
  `ProgressEvent`s (raw line plus parsed JSON where available), mirroring the
  `ollama` driver's progress convention.
- TTFT is recorded on the first streamed content token via the `Run.OnTTFT`
  callback (when set), matching streaming drivers.
- A terminal `completed` `ProgressEvent` carrying the final assistant text is
  emitted at end of run, and a per-turn/summary record is written to the log.

#### FR-7: Run logging

- When `Run.LogPath` is set, the driver writes a per-run log with a header
  (`run id`, `agent`, `role`, `driver=llama-cpp`, `base_url`, `model`, start
  time), the system/user prompts, each turn's tool calls and tool results, the
  final assistant message, and a footer with the finish time — consistent in
  shape with the `ollama` driver's log format.

#### FR-8: Cancellation and timeout

- `Process.Kill()` cancels the in-flight HTTP request(s) and terminates the
  loop; `Process.Wait()` returns the run's terminal error (nil on success).
- The run is bounded by the agent's `timeout_minutes` (falling back to the
  existing default when unset); an unreachable or hung endpoint fails within
  that bound rather than hanging indefinitely.

### Non-functional

#### NFR-1: Secret hygiene

- A configured `api_key` / `auth_token` MUST NOT appear in run log files,
  `ProgressEvent.Raw`, stderr tails, or any HTTP/API response that exposes agent
  configuration; it is masked (e.g. `"***"`) wherever agent config is returned,
  matching the `ollama_instances.api_key` convention.

#### NFR-2: No regression

- All existing driver unit tests continue to pass. Adding `llama-cpp` to the
  driver map does not alter result-event or progress behaviour for any other
  driver.

#### NFR-3: Failure behaviour

- Non-2xx responses from the endpoint, malformed JSON, and malformed/unknown
  tool calls surface as run failures (or per-tool errors fed back to the model,
  as appropriate) with a useful stderr tail — never a panic and never an
  unbounded retry loop.

#### NFR-4: Trust boundary

- `base_url` and any token live in the project/app config file on disk, the same
  trust boundary as existing credentials. No new secret-storage mechanism is
  required for v1.

## Acceptance Criteria

- [ ] An agent declared with `driver: llama-cpp`, a valid `base_url`, and
      `model` loads and validates without error; a configured token is accepted.
- [ ] Config validation rejects a `llama-cpp` agent missing `base_url`, with a
      malformed `base_url`, or missing `model`, with a message naming the agent.
- [ ] Starting a run POSTs to `<base_url>/v1/chat/completions` with `model`,
      `messages` (system+user derived from the prompt), a non-empty `tools`
      array, and streaming enabled; `Authorization: Bearer` is present iff a
      token is configured.
- [ ] A single tool-call round-trip works end-to-end: the model requests one
      tool call, the driver executes it, appends the `tool` result message, and
      obtains a final assistant message. (Covered by the `~/Code/agent-benchmark`
      example.)
- [ ] A multi-step agent run works end-to-end: at least two sequential tool
      calls (e.g. read then write) complete and the run terminates on
      `finish_reason: stop`. (Covered by the `~/Code/agent-benchmark` example.)
- [ ] A write tool call targeting a path outside the agent's
      `allowed_write_paths` is refused, reported to the model as a tool error,
      and does not write outside scope or crash the run.
- [ ] The agentic loop terminates at the configured maximum-iteration cap with a
      clear terminal status when the model never stops requesting tools.
- [ ] A run emits streaming `ProgressEvent`s, records TTFT on the first content
      token, and writes a per-run log in the shape used by the `ollama` driver.
- [ ] `Process.Kill()` cancels an in-flight run; `Process.Wait()` returns the
      terminal error; the run respects `timeout_minutes`.
- [ ] A configured token does not appear in the run log, stderr tail, or any API
      response returning agent config (it is masked).
- [ ] Existing drivers (`claude-code-cli`, `claude-mediated`, `claude-env`,
      `ollama`, `gemini`, `codex-cli`, `shell-stub`) are unaffected (existing
      tests green) and runs respect `max_concurrent_agents`.
- [ ] `~/Code/agent-benchmark` contains an example config documenting the
      required `llama-server` flags (tool-calling / grammar / `--jinja` /
      `--api-key` as applicable) and recommended agent-capable models.
- [ ] Lineage artifacts for [[llama-cpp-driver]] (this requirement and its
      backend/frontend/test plans) link via `parent:` correctly, and related
      work [[ollama-claude-code-driver]], [[ollama-agent-support]], and
      [[ollama-agents-need-execution-layer]] is referenced without duplication.

## Open Questions

1. **Tool-execution layer ownership.** Does this driver implement its own
   local tool executor (file read/write/list against the sandbox), or should it
   consume a shared execution layer proposed in
   [[ollama-agents-need-execution-layer]]? If the shared layer lands first, this
   driver should depend on it rather than duplicate it. Which sequences first?
2. **v1 tool set.** Minimum viable is `read_file` + `write_file`. Do we also
   ship `list_dir`, a `grep`/search tool, and/or a constrained `bash` tool in
   v1, or defer those? A broader tool set improves capability but widens the
   sandbox/security surface.
3. **Maximum loop iterations.** What default cap on tool-call rounds balances
   real multi-step tasks against runaway loops (e.g. 10, 25, 50), and should it
   be a per-agent config field?
4. **Permission model.** Should tool calls be auto-approved (like
   `claude-code-cli` bypass) or routed through the mediation/precheck path used
   by `claude-mediated`? v1 baseline is auto-approve within
   `allowed_write_paths`; confirm.
5. **Endpoint/flag compatibility scope.** llama.cpp tool calling depends on
   server flags and model chat templates (`--jinja`, grammar/tool-call parsing).
   Do we commit to a documented, tested `llama-server` configuration and treat
   other builds as "best effort, untested" (as [[ollama-claude-code-driver]]
   scoped the Ollama shim)?
6. **Frontend exposure.** Is config-file-only acceptable for v1, or must the
   agent create/edit UI offer `llama-cpp` as a driver type with
   base-url/model/token fields?
