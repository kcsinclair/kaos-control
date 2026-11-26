# Local-Model Operability

Workstream 3 of the [Open Provider Support](open-provider-support.md) epic.
The `openai-compatible` driver makes a local inference server (llama.cpp
`llama-server`, Ollama's `/v1` endpoint) *reachable* — this covers what it
takes for a small local model (8B–30B parameters) to actually produce
**usable lifecycle artifacts**, and for an operator to understand what
happened when it doesn't:

1. **Local-model-tuned prompt fallbacks** — compact, single-step-ordered
   prompts used when an agent has no `prompt_templates` entry for its role.
2. **Model availability preflight** — fails fast, before a run starts, if the
   configured model isn't present on the provider.
3. **Warmup / lazy-load detection** — live UI feedback while a local server
   loads multi-gigabyte weights into memory, instead of an apparently frozen
   run.
4. **Structured error taxonomy** — every run failure classifies into one of a
   fixed set of reason codes with actionable, provider-specific remediation
   steps, surfaced in the UI instead of a raw stack trace.

## Why local models need this

kaos-control's agent prompts were originally written for frontier cloud
models (Claude, GPT-4o), which tolerate long, multi-phase prose instructions.
Small local models don't. The `agent-benchmark` harness referenced by
[[open-provider-support-2]] scored 13 local models against the same five
coding problems (68 checks total) using the *same* tool-calling loop:

| Score | Models | Verdict |
|---|---|---|
| 65–68/68 | `gemma-4-26B`, `gpt-oss-120b`, `gpt-oss-20b` (Q8), Qwen3.6-27B/35B, `gemma-4-31B` | fully agent-capable |
| 37–57/68 | Qwen3-Coder-30B (57), Qwen3VL-8B (41), `gpt-oss-20b` **Q4** (37) | usable, degraded |
| 0–28/68 | Muse-Glimmer-30B (28), Hermes-4-14B (8), Llama-3-14B/3.1-8B (0) | not agent-capable |

Two takeaways that shaped this feature: **quantization matters as much as
model choice** (`gpt-oss-20b` scores 68/68 at Q8 and 37/68 at Q4 — always
state the quant when recommending a model), and **prompt shape is one of the
few levers that moves capability at a fixed model/quant** — hence the local
prompt fallbacks below.

## 1. Local-model-tuned prompt fallbacks

`internal/agent/prompt_defaults.go` defines `LocalModelPromptDefaults`, a map
from standard agent name (`requirements-analyst`, `planning-analyst`,
`backend-developer`, `frontend-developer`, `test-developer`, `qa`,
`tech-writer`) to a compact fallback prompt (under ~1200 tokens each).
`Manager.StartRun` uses one of these automatically whenever an agent's
`prompt_templates` config has no entry for its active role — no
configuration is required to opt in.

Each fallback:

- Is a numbered, single-step-ordered instruction list ("1. Read X. 2. Write
  Y. 3. Output a summary.") rather than prose, so a model that drifts partway
  through still completes discrete, checkable steps.
- Includes a concrete YAML frontmatter few-shot example, not just a field
  list — local models corrupt frontmatter syntax far more often when only
  told the schema in words.
- States mandatory `##` section headings explicitly, in order.
- Ends with an explicit negative constraint ("Do not write to any other
  file.") rather than relying on the model to infer scope from context.

You can still override any role's prompt per-agent via `prompt_templates` in
`lifecycle/config.yaml` exactly as before — the fallback only applies when
that key is absent. `internal/initcmd/templates/config.yaml.tmpl` documents
this behaviour inline for newly-scaffolded projects.

This project's own `qa` agent (bound to a local llama.cpp provider —
see [Example configuration](#example-configuration) below) uses a
hand-tuned local prompt rather than the fallback, since QA's tool-use
pattern (run fixed shell commands, then read a bounded set of files) needed
tighter step-gating than the generic default provides.

## 2. Model availability preflight

Before a run starts, `internal/agent/openai_preflight.go`'s
`CheckModelAvailability` issues `GET <base_url>/v1/models` with a **3-second
timeout** and checks whether the configured `model` is present in the
response. `Manager.StartRun` calls this before acquiring the lineage lock for
any `openai-compatible` agent, so a missing model or an unreachable endpoint
fails within 3 seconds — no lock held, no artifact touched, no git commits.

- Model present → run proceeds.
- Model absent from a reachable provider → fails with `model_not_found`.
- Connection refused, DNS failure, timeout, or non-2xx `/v1/models` response
  → fails with `endpoint_unreachable`.

The same probe opportunistically reads a provider-specific `state` field on
each model entry (a llama.cpp extension — `"unloaded"` / `"loading"` /
`"loaded"`); when present, the backend broadcasts an `agent.status` event with
`state: "model_loading"` before the run starts.

## 3. Warmup and lazy-load detection

Local servers commonly lazy-load multi-gigabyte GGUF weights on the first
request, which can pause 30–120 seconds with no visible output. Without
signaling, that's indistinguishable from a hung process. The
`openai-compatible` driver (`internal/agent/openai_compatible.go`) tracks
this per turn:

- If **5 seconds** elapse with no token of any kind, it emits a `warming_up`
  progress event (`"Awaiting first token (model may be warming up)..."`).
- A **dedicated load timeout** (`model_loading_timeout_seconds`, app-level
  config under `agent:`, default **60s**) bounds the wait independently of
  the run's overall `timeout_minutes`. If it elapses with no token at all,
  the run fails with `model_unloaded` (`ErrModelLoadTimeout`).
- The watchdog is satisfied by **any** sign of generation — `content`,
  `reasoning_content`, or a tool-call delta — not just prose content. Earlier
  builds gated only on `delta.content`, which killed reasoning models
  (Qwen3, gpt-oss) mid-stream while they were actively producing
  `reasoning_content`; see
  [[openai-driver-ttft-ignores-reasoning-content]] for the fixed defect.
  Reasoning and generating are tracked as distinct progress stages, so a long
  think shows as active rather than stalled.

In the UI, this state is exposed as `warmup_state` (`'model_loading' |
'warming_up' | 'generating'`) on the running job, client-side only (never
persisted — it clears once the run reaches a terminal status):

- `AgentRunningBanner.vue` — an animated amber "Warming up model weights..."
  badge next to the run timer.
- `AgentsRunsView.vue` — the same pulsing badge on the active run's row.
- `RunDetailModal.vue` — a "Warming up model weights in memory..." line in
  the turn timeline, with the driver's live warmup message if one arrived.

## 4. Structured error taxonomy

`internal/agent/errors.go`'s `ClassifyRunError` inspects a driver failure
(sentinel error, HTTP status, or stderr text) and maps it to one of a fixed
set of `failure_reason` codes, each with ordered remediation steps:

| Code | Meaning |
|---|---|
| `tools_unsupported` | The model/chat-template silently dropped or explicitly rejected `tools`. |
| `model_not_found` | The configured model isn't in the provider's `/v1/models` listing. |
| `model_unloaded` | The provider returned HTTP 503, or the load-timeout watchdog fired. |
| `endpoint_unreachable` | Connection refused, DNS failure, or timeout reaching `base_url`. |
| `provider_disconnected` | The connection dropped **mid-stream** — a different failure from unreachable: the server was reachable and responding, then reset the stream (observed with model-swapping servers like `llama-swap`, and server-side idle/request timeouts on long reasoning phases). |
| `context_window_exceeded` | The model's context window was exceeded. |
| `turn_token_ceiling` | The model hit its per-turn generation token cap (`finish_reason: length`) without completing a tool call. |
| `max_iterations_reached` | The agent hit `max_tool_iterations` (default 25) without `finish_reason: stop`. |
| `auth_error` | HTTP 401/403 — bad or missing credentials. |
| `timeout` | The run exceeded its configured `timeout_minutes`. |

`AgentRunRow` (and the `agent.failed` WebSocket broadcast) carries
`failure_reason`, `remediation` (the ordered steps), and `error_details` — a
sanitized map (provider, model, base_url, HTTP status, masked error message).
`maskSecretsInText` redacts `Bearer <token>` and `api_key`/`access_token`/
`auth_token`-shaped values before any of this is persisted or broadcast,
per [[standards/secrets-handling]].

### UI surfacing

`RunFailureBanner.vue` renders a dedicated heading, explanation, and numbered
remediation list per code (static copy in `web/src/lib/failureReasons.ts`,
overridden by the backend's own `remediation` when present), plus links to
**Provider Settings** (`/p/{project}/settings/providers`) and **Agent
Config** (`/p/{project}/agents`) for codes where those are the fix. It
renders with `role="alert"`, and — because `error_details` is already
secret-masked server-side — never re-exposes a raw credential, only the
`***` marker the backend sent.

`RunDetailModal.vue` shows the same banner on a failed run's detail view,
plus a collapsible "Diagnostic Info" panel listing whatever `error_details`
the backend attached (provider, endpoint, status code, sanitized message).
`AgentsRunsView.vue` shows a compact failure badge in the run history table
and the full banner in the row's expanded detail.

## Example configuration

A Provider pointing at a local llama.cpp server (`~/.kaos-control/config.yaml`):

```yaml
providers:
  - name: leia-llamacpp
    base_url: http://leia.packsin.com:7442   # llama-server, started with --jinja
    driver: openai-compatible
```

An agent using it, relying entirely on the local-model prompt fallback and
availability/warmup handling described above — no `prompt_templates` entry
needed:

```yaml
agents:
  - name: qa
    role: [qa]
    provider: leia-llamacpp
    driver: openai-compatible
    model: gemma-4-26B-A4B-it-UD-Q8_K_XL
    max_tool_iterations: 40
    timeout_minutes: 30       # explicit — 0 means "10 minutes" on this driver, not unlimited
    allowed_write_paths:
      - lifecycle/defects
      - lifecycle/architecture/decisions
```

App-level tuning of the load-detection window (`~/.kaos-control/config.yaml`):

```yaml
agent:
  model_loading_timeout_seconds: 60   # default; raise for very large/slow-loading models
```

### Recommended models

Verified round-trip (2026-08-24) against the benchmark's five coding
problems — state the quantization when recommending any of these, since it
materially changes capability:

| Server | Model | Notes |
|---|---|---|
| llama.cpp (`--jinja`) | `gemma-4-26B-A4B-it-UD-Q8_K_XL` | Documented baseline — clean `tool_calls`, full round-trip in 5s. |
| llama.cpp (`--jinja`) | `gpt-oss-20b-Q8_0` | Verified alternative, 11s round-trip. **Q4 drops to 37/68** — always use Q8. |
| Ollama `/v1` | `qwen3-coder:30b` | Clean native `tool_calls` through Ollama; emits its own text-based tool-call syntax through llama.cpp instead — a server/template property, not a model property (recovered automatically, see [open-provider-support.md](open-provider-support.md#native-tool-call-recovery)). |
| Ollama `/v1` | `gemma4:26b` | Clean native `tool_calls`. |

## Troubleshooting

**Run fails instantly with `model_not_found` or `endpoint_unreachable`.**
The 3-second preflight probe against `/v1/models` failed. Verify the
`model:` value matches an ID the provider actually serves (`ollama list`, or
`Settings → Providers → Models`), and that the server is reachable at
`base_url`.

**A run shows "Warming up model weights..." for a long time, then fails
with `model_unloaded`.** The server didn't produce a single token within
`model_loading_timeout_seconds` (default 60s). For large quantizations on
slow storage/RAM, raise this app-level setting rather than the run's
`timeout_minutes` — they're independent, and only the loading timeout gates
this wait.

**A reasoning model (Qwen3, gpt-oss) gets killed mid-response even though
the log shows it streaming.** This was [[openai-driver-ttft-ignores-reasoning-content]]
— fixed. If you see it recur, check whether `delta.reasoning_content` or a
tool-call delta is arriving at all; the watchdog now clears on any of the
three signal types.

**`provider_disconnected` on a server that swaps models on demand (e.g.
`llama-swap`).** A second model load reset the in-flight stream. Pin the
agent's model so it isn't sharing server capacity with a model-swapping
router, or avoid driving two different local models through the same
router concurrently.

**`tools_unsupported` on a model that "should" support tool calling.**
Confirm `llama-server` was started with `--jinja` — required for the chat
template tool-calling depends on — and that the *specific* model/quant
supports tools (a template gap is per-model: on the same `--jinja` server,
`gemma-4-26B` and `gpt-oss-20b` inject tools cleanly while
`Dolphin-Llama3.1-8B` drops them silently).

## Related

- [open-provider-support.md](open-provider-support.md) — the `Provider`
  entity, the `openai-compatible` driver's request/tool-calling loop, and
  provider switching/failover, which this feature builds on.
- [[local-model-operability]] (idea), [[local-model-operability-2]]
  (requirement), [[local-model-operability-3-be]] /
  [[local-model-operability-4-fe]] / [[local-model-operability-5-test]]
  (plans).
- [[local-llm-operability]] — the feature record for this capability.
- [[ollama-local-llms]] — the retired native-`ollama`-driver feature this
  epic replaced; superseded by [[provider-management]] and this feature.
