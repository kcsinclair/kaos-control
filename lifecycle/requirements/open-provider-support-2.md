---
title: OpenAI-Compatible Agent Driver (Tool-Calling)
type: requirement
status: done
lineage: open-provider-support
created: "2026-08-11T18:15:34+10:00"
priority: normal
parent: lifecycle/ideas/open-provider-support.md
labels:
    - driver
    - agent
    - agent-runner
    - backend
    - go
    - portability
    - feature
    - provider
    - open-provider-support
release: KC-Release6
assignees:
    - role: product-owner
      who: agent
---

# OpenAI-Compatible Agent Driver (Tool-Calling)

## Scope (generalised from the llama.cpp requirement)

This requirement was originally scoped to a `llama-cpp` driver. It is now the
**workstream 1 driver requirement** of the [[open-provider-support]] epic:
a single first-party driver that speaks the OpenAI-compatible
`/v1/chat/completions` endpoint with a tool-calling agent loop, reached through
a configured **Provider** record (`{name, base_url, api_key, driver,
extra_headers}`) rather than a per-vendor driver name.

The functional requirements below are unchanged in substance — the multi-turn
tool-call loop, sandbox/`allowed_write_paths` scoping, and the
ProgressEvent/TTFT contract are the hard part and apply identically to every
target. What changes is that llama.cpp is now a **verification target, not the
subject**. The same driver serves OpenAI, OpenRouter, Ollama, Groq,
Together and Azure, so [[openai-api-integration]],
[[openrouter-llm-integration]] and [[llama-cpp-driver]] are all satisfied by
this one deliverable.

Verification targets (both live and confirmed to serve `/v1/models`):

- **llama.cpp** — `leia.packsin.com:7442`, `llama-server --jinja`, model
  **`gemma-4-26B-A4B-it-UD-Q8_K_XL`** (verified 2026-08-24: tools injected,
  clean `tool_calls`, full two-turn round-trip in 5 s). `gpt-oss-20b-Q8_0` is a
  verified alternative (11 s). `--jinja` provides the chat template that
  tool-calling depends on — but see FR-5b: the template must also *support*
  tools, which is per-model.
- **Ollama** — `leia.packsin.com:11434` (`qwen3-coder:30b`, `gemma3:12b`).
  Ollama answering `/v1/models` is precisely why the **native `ollama` driver is
  removed outright** rather than maintained alongside this one.

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

- Add a new first-party agent driver (`driver: openai-compatible`) that drives a
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
- A *driver*-picker UI. Provider/model selection **is** in scope, but as the
  Provider settings surface delivered by the epic (`OllamaSettingsView` →
  provider settings) — not a per-driver picker. See Resolved Question 6.
- Streaming partial tool-call arguments token-by-token to the UI. Assembling a
  complete tool call before executing it is sufficient for v1.

## Detailed Requirements

### Functional

#### FR-1: Driver registration and selection

- A new driver is registered in the agent `Manager`'s driver map under the
  stable name **`openai-compatible`**, selected purely on the `AgentConfig.driver`
  field, identical to how `ollama`/`gemini`/`codex-cli` are selected. Unknown
  driver names are rejected as today.
- The driver satisfies the existing `agent.Driver` interface
  (`Start(ctx, Run) (Process, error)`) and returns a `Process` implementing
  `Wait`, `Kill`, `Progress`, and `StderrTail`.

#### FR-2: Configuration fields

Connection identity lives on the **Provider record**, not on the agent — an
agent is a `{provider, model}` pair (see [[provider-model-for-agents]]).

- A **Provider** (app-level, generalising today's `ollama_instances` → `providers`):
  - `name` (string, **required**) — unique; how agents reference it.
  - `base_url` (string, **required**) — server root, e.g.
    `http://leia.packsin.com:7442`. The driver targets
    `<base_url>/v1/chat/completions`.
  - `driver` (string, **required**) — `openai-compatible` for this requirement.
  - `api_key` (string, **optional**) — sent as `Authorization: Bearer <token>`
    when non-empty (e.g. `llama-server --api-key`, OpenRouter, OpenAI).
  - `extra_headers` (map, **optional**) — arbitrary request headers; this is
    what makes [[openrouter-llm-integration]] pure configuration
    (`HTTP-Referer`, `X-Title`).
- `AgentConfig` gains:
  - `provider` (string, **required**) — the name of a configured Provider.
  - `model` (string, **required**) — the model identifier sent as the request
    `model` field (e.g. `qwen3-coder:30b`, `Dolphin3.0-Llama3.1-8B-Q4_K_M`).
  - `max_tool_iterations` (int, **optional**) — per-agent override of the
    default tool-call cap (see FR-5).
- Field naming reuses existing conventions (`base_url`/`api_key` already exist on
  `OllamaInstance`, which the Provider record replaces; secret masking carries
  over — `api_key` must never be logged or returned by the agents API).

#### FR-3: Config validation

- `config.Validate` rejects a Provider with an empty `name`, a duplicate `name`,
  an empty `base_url`, or a `base_url` that is not a valid `http`/`https` URL.
- `config.Validate` rejects an agent whose `provider` names no configured
  Provider, or whose `model` is empty. Each failure message names the offending
  agent or provider.
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
  `read_file`, `write_file`, `list_dir`, `grep` — no shell tool in v1) is
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
- The loop is bounded by a configurable maximum iteration count (default **25**,
  overridable per agent via `max_tool_iterations` — see Resolved Questions 3).
  Hitting the cap ends the run with a clear terminal status and a logged reason
  rather than looping unbounded.

#### FR-5a: Native-format tool-call recovery

Some models emit tool calls in their **own syntax** and llama.cpp passes the
text through as plain `content` instead of parsing it into `tool_calls`.
Qwen3-Coder does exactly this *through llama.cpp* — yet returns clean,
OpenAI-shaped `tool_calls` through Ollama (verified 2026-08-24). Tool-call
formatting is therefore a **server + chat-template property, not a model
property**: the same model must be assumed to behave differently on different
back ends, and the driver cannot key this behaviour off the model name. A driver that only reads `message.tool_calls`
silently scores zero on those models — they appear to "not call tools" when in
fact the server did not parse them.

- When a turn returns **no `tool_calls` but non-empty `content`**, the driver
  attempts a fallback parse for the known native encodings before treating the
  turn as a final answer:
  - `<function=NAME><parameter=KEY>VALUE</parameter></function>`
  - `<tool_call>{json}</tool_call>`
- Recovered calls are normalised to the OpenAI `tool_calls` shape and executed
  through the same path, but are **counted and logged separately** from native
  OpenAI-shaped calls — an off-the-shelf agent would not see them, so conflating
  the two would measure the server rather than the model.
- A turn with neither `tool_calls`, recoverable native calls, nor content is a
  terminal condition, not a retry.
- Gateways that normalise across vendors (OpenRouter) are expected to yield
  **zero** recovered calls; a non-zero count there is a finding worth logging,
  not something to paper over silently.

Reference implementation: `benchmark/run_agent.py` in
[`~/Code/agent-benchmark`](../ideas/llama-cpp-driver.md) (`parse_native_calls`,
`NATIVE_FN` / `TOOL_CALL_JSON`).

#### FR-5b: Preflight tool-capability verification

Sending `tools` to an endpoint has **four** observed outcomes, and one of them is
silent (see Prior art and measured evidence). The driver must not assume a
successful HTTP 200 means the model saw the tools.

- Before the first agent turn (or on the first turn), the driver performs a
  **capability preflight**: it compares `usage.prompt_tokens` for the request
  *with* `tools` against the same request *without* them. An **identical count
  means the server silently dropped the tools** and the run cannot be an agent
  run.
- On a detected silent drop the run **hard-fails** — it does not degrade to a
  chat-only completion. The terminal status names the provider and model and
  states that tool calling is unsupported, and **no artifacts are written**.
  This is deliberate: the observed failure produced a fluent, confident,
  entirely fabricated answer indistinguishable from success, so continuing
  without tools risks writing hallucinated content into `lifecycle/`.
- An explicit server rejection (e.g. Ollama's HTTP 400
  `"<model> does not support tools"`) is surfaced verbatim as the failure
  reason; it is the *desirable* failure mode and must not be retried.
- Where the provider advertises capability up front (OpenRouter exposes
  `supported_parameters` containing `tools` on `/v1/models`), the driver
  **should** consult it and refuse to start rather than discovering the problem
  mid-run. This metadata is gateway-only; llama.cpp and Ollama do not expose it,
  which is exactly why the token-delta preflight above is the general mechanism.

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
  (`run id`, `agent`, `role`, `driver=openai-compatible`, `base_url`, `model`, start
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

- All existing driver unit tests continue to pass. Adding `openai-compatible` to the
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

- [ ] An agent declared with `driver: openai-compatible`, a valid `base_url`, and
      `model` loads and validates without error; a configured token is accepted.
- [ ] Config validation rejects an `openai-compatible` agent missing `base_url`, with a
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

## Prior art and measured evidence

`~/Code/agent-benchmark` already measures the exact risk this requirement
carries — whether a model can drive a `write_file` tool loop — across 13 local
models on the same five coding problems, with a working OpenAI-compatible
reference loop in `benchmark/run_agent.py`.

**Agent-track results (score out of 68 checks):**

| band | models | reading |
|---|---|---|
| 65–68 | gemma-4-26B, gpt-oss-120b, gpt-oss-20b Q8, Qwen3.6-27B/35B, gemma-4-31B | fully agent-capable |
| 37–57 | Qwen3-Coder-30B (57), Qwen3VL-8B (41), gpt-oss-20b **Q4** (37) | usable, degraded |
| 0–28 | Muse-Glimmer-30B (28), Hermes-4-14B (8), Llama-3-14B (0), Llama-3.1-8B (0) | not agent-capable |

Consequences for this driver:

- **Capability is not a given.** Llama-3-14B called a tool *zero* times and
  answered in prose; Llama-3.1-8B called tools but wrote the wrong files. Both
  are terminal outcomes the driver must report clearly rather than retry.
- **Quantisation matters as much as model choice** — gpt-oss-20b scores 68/68 at
  Q8 and 37/68 at Q4. Model recommendations must state the quant.
- **Per-turn token ceiling is a real failure mode.** Muse-Glimmer recorded
  exactly 8192 completion tokens (the per-turn cap) on three of five problems
  with zero tool calls — it never finished reasoning, so it never called a tool.
  That is a token ceiling, not an inability to drive an agent, so `max_tokens`
  must be configurable per agent and truncation must be logged distinctly.
- **Empirical loop bound.** The benchmark harness defaults to `max_turns: 12`
  with `max_tokens: 8192` and completed the suite. The 25 default set in
  Resolved Question 3 is therefore comfortably above the observed need; 12 is
  evidence that 25 is a safety cap rather than a working limit.

**Live probe of both local servers (2026-08-24).** Sending an identical
`read_file` tool definition produced **four distinct outcomes**:

| Server · model | `tools` injected? | Outcome | Mode |
|---|---|---|---|
| llama.cpp `:7442` · `Dolphin3.0-Llama3.1-8B-Q4_K_M` | ❌ **dropped** (prompt_tokens 27 → 27) | confidently **hallucinated** the file as "Kubernetes Helm config"; `finish_reason: stop`; `tool_choice:"required"` also ignored | **A — silent drop (most dangerous: looks successful)** |
| Ollama · `gemma3:12b` | — | HTTP 400 `"gemma3:12b does not support tools"` | **B — explicit rejection (safest failure)** |
| Ollama · `qwen3-coder:30b` | ✅ +269 tokens | clean native `tool_calls` → `read_file({"path":"lifecycle/config.yaml"})` | **C — clean success** |
| Ollama · `gemma4:26b` | ✅ +40 tokens | clean native `tool_calls` | **C — clean success** |
| llama.cpp · **`gemma-4-26B-A4B-it-UD-Q8_K_XL`** | ✅ +43 tokens | clean `tool_calls`; full 2-turn round-trip, correct answer, **5 s** | **C — clean success (documented target)** |
| llama.cpp · `gpt-oss-20b-Q8_0` | ✅ +43 tokens | clean `tool_calls`; full round-trip, correct answer, 11 s | **C — clean success (verified alternative)** |
| llama.cpp · `gemma-4-26B…heretic-Q4_K_M` | ✅ +43 tokens | clean `tool_calls` | **C — clean success** |
| llama.cpp · Qwen3-Coder (per benchmark) | ✅ | native syntax passed through as `content` | **D — needs FR-5a recovery** |

Two conclusions that shaped the requirements above:

1. **The dangerous case is silent.** Mode A returns HTTP 200, `finish_reason:
   stop`, and a fluent, entirely fabricated answer. Nothing in the response
   distinguishes it from success — which is why FR-5b makes the `prompt_tokens`
   delta a mandatory preflight rather than an optimisation.
2. **Behaviour is per server *and* model — probe, never infer.** Qwen3-Coder is
   mode D on llama.cpp and mode C on Ollama, so one model differs by back end.
   The converse also holds: on the *same* llama.cpp server, gemma-4-26B and
   gpt-oss-20b inject tools cleanly while Dolphin-Llama3.1-8B drops them.
   **The `:7442` server is correctly configured** — `--jinja` works; the silent
   drop is a property of the Dolphin / Llama-3.1-8B chat template, which carries
   no tool support (consistent with the benchmark, where Llama-3.1-8B scores
   0/68). A model whose template lacks tools cannot be rescued by server flags,
   which is why FR-5b probes per provider+model rather than per server.

**OpenRouter probe (2026-08-24, this analysis):** `/v1/models` returns 419
models, **83% advertising `tools` in `supported_parameters`** — so capability is
discoverable up front and the driver can reject an unsuitable model before a run
rather than failing mid-loop. A full tool-call round-trip
(`read_file` → tool result → final answer) succeeded natively, confirming
gateways normalise tool calls. Two caveats observed: `HTTP-Referer`/`X-Title`
are **optional** (both present and absent returned 200 — `extra_headers` is
attribution, not a functional requirement), and several catalogued model IDs
returned **404 on `/v1/chat/completions`**, so "listed" does not imply
"routable".

## Resolved Questions

Resolved 2026-08-24 during the [[open-provider-support]] epic dedup — product
owner decisions, plus two answered by the dedup itself.

1. **Tool-execution layer ownership — driver owns it.**
   [[ollama-agents-need-execution-layer]] is abandoned, so no shared execution
   layer is landing. This driver implements its own local tool executor against
   the sandbox resolver.
2. **v1 tool set — `read_file`, `write_file`, `list_dir`, and a `grep`/search
   tool.** Search is included because "find where X is defined" is a real
   multi-step need; **no `bash` tool in v1** — shell execution is the widest
   surface and small local models are the least trustworthy with it.
3. **Maximum loop iterations — default 25, per-agent overridable.** Enough for
   genuine multi-step artifact work, low enough to catch runaway loops; exposed
   as a per-agent config field so slower/cheaper local models can be tuned.
4. **Permission model — auto-approve within `allowed_write_paths`.** Writes stay
   hard-scoped by the sandbox resolver (the same guarantee every other agent
   gets); no interactive gate in v1. Routing through the
   [[adr-0006-mediated-agent-driver-permission-model]] mediation path (denial
   recording, queue pause, `on_denial`) is a deliberate follow-up, not v1.
5. **Endpoint/flag compatibility — committed, tested configuration.**
   `llama-server --jinja` running **`gemma-4-26B-A4B-it-UD-Q8_K_XL`** is the
   documented llama.cpp baseline (`gpt-oss-20b-Q8_0` a verified alternative),
   and Ollama's `/v1` endpoint with `gemma4:26b` / `qwen3-coder:30b` the
   documented Ollama baseline — all round-trip verified 2026-08-24. Other
   server builds and models are best-effort/untested, mirroring how
   [[ollama-claude-code-driver]] scoped its shim. **Model selection is part of
   the supported configuration, not an operator detail**: a template without
   tool support fails FR-5b regardless of server flags.
6. **Frontend exposure — in scope, via Provider settings.** Moot as originally
   posed: there is no `llama-cpp` driver type to surface. Workstream 1 replaces
   `OllamaSettingsView` with provider settings (`ollama_instances` → `providers`,
   `/ollama/instances` → the provider API), so provider/model selection is part
   of the epic by definition.
7. **Model without tool support — hard-fail, no degraded mode.** When FR-5b
   detects a silent drop, or the server rejects `tools` outright, the run fails
   with a clear reason and writes nothing. There is no "warn and continue"
   fallback and no per-agent opt-out in v1: the failure this guards against
   (Dolphin returning a confident hallucination at HTTP 200) is exactly the case
   where continuing *looks* successful and corrupts artifacts.
