# Open Provider Support: Providers, the OpenAI-Compatible Driver, and Failover

kaos-control can run agents against any endpoint that speaks the OpenAI
`/v1/chat/completions` wire format — local inference (Ollama, llama.cpp),
gateways (OpenRouter), and direct cloud APIs (OpenAI, Groq, Together, Azure)
— through one configuration surface and one driver. This replaces the old
per-vendor approach where local models could only *talk* (single-shot
completion, no tool use) and connection details were duplicated across every
agent declaration.

This document covers the two shipped workstreams of the **Open Provider
Support** epic:

1. **Provider entity + `openai-compatible` driver** — a first-class
   `Provider` record, a management REST API and UI, and a driver that runs
   the full tool-calling agent loop against any OpenAI-compatible endpoint.
2. **Dynamic provider switching and failover** — automatic failover on
   upstream overload/rate-limit/outage, manual switch/restore controls, and
   provider templates.

A third workstream, **local-model operability** (prompt tuning, richer
availability signals), and two smaller follow-ups —
[[inline-driver-provider-abstraction]] (giving the idea-chat conversational
completer the same provider abstraction) and
[[agent-logging-provider-driver]] (recording provider/driver on every run
record) — are still in progress; see [Related and in-progress work](#related-and-in-progress-work).

---

## 1. The Provider entity

A **Provider** is a named, reusable connection to an OpenAI-compatible
endpoint. It lives in **app-level** configuration
(`~/.kaos-control/config.yaml`), shared across every project, so updating a
base URL or rotating a key is one edit instead of N agent edits.

```yaml
providers:
  - name: loki                                # unique; how agents reference it
    base_url: http://leia.packsin.com:11434    # server root — driver POSTs to <base_url>/v1/chat/completions
    driver: openai-compatible                  # defaults to openai-compatible
    api_key: ""                                # optional — sent as "Authorization: Bearer <token>"
    extra_headers: {}                          # optional — arbitrary request headers

  - name: llamacpp-gemma
    base_url: http://leia.packsin.com:7442
    driver: openai-compatible

  - name: openrouter
    base_url: https://openrouter.ai/api
    driver: openai-compatible
    api_key: sk-or-...
    extra_headers:
      HTTP-Referer: https://kaos-control.local
      X-Title: kaos-control
```

| Field | Required | Notes |
|---|---|---|
| `name` | yes | Unique across `providers`; referenced by `AgentConfig.provider`. |
| `base_url` | yes | Must be a valid `http`/`https` URL. The driver targets `<base_url>/v1/chat/completions`. |
| `driver` | yes | `openai-compatible` for every current provider. |
| `api_key` | no | Sent as `Authorization: Bearer <token>` when non-empty. Masked (`***`) everywhere it could otherwise leak — see [Secret hygiene](#secret-hygiene). |
| `extra_headers` | no | Arbitrary headers, e.g. OpenRouter's `HTTP-Referer` / `X-Title` for attribution. Optional even for OpenRouter — requests succeed with or without them. |

Config validation (`config.Validate`) rejects a Provider with an empty
`name`, a duplicate `name`, an empty `base_url`, or a `base_url` that isn't
a valid `http`/`https` URL.

### Migrating from `ollama_instances`

The legacy `ollama_instances` block and the native `ollama` driver (a
single-shot `/api/chat` client with no tool support) have been **removed
outright** — there is no `driver: ollama` any more. On startup, any
`ollama_instances` entries with no `providers` yet configured are
automatically migrated into `providers` records with
`driver: openai-compatible`, preserving `name`, `base_url`, and `api_key`.
Ollama exposes an OpenAI-compatible `/v1/` endpoint, so the migrated
provider reaches the exact same server and models with no special-casing.
`OllamaInstance` remains in config only as the migration source type.

The old `/api/ollama/instances` REST endpoints and the `OllamaSettingsView`
frontend page are gone; use the Provider management API and **Provider
Settings** view described below instead. See
[[ollama-local-llms]] for the retired feature.

---

## 2. Provider management API

App-level (not project-scoped) REST API under `/api/providers`:

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/providers` | List providers (`api_key` masked). |
| `POST` | `/api/providers` | Create a provider. |
| `POST` | `/api/providers/test` | Test a provider's connectivity without saving it. |
| `GET` | `/api/providers/{name}` | Get one provider. |
| `PUT` | `/api/providers/{name}` | Update a provider. |
| `DELETE` | `/api/providers/{name}` | Delete a provider (blocked if any agent, in any project, still references it — the response names the referencing agents). |
| `GET` | `/api/providers/{name}/health` | Live reachability probe against `<base_url>/v1/models`. |
| `GET` | `/api/providers/{name}/models` | List models the endpoint currently advertises, via `/v1/models`. |

`api_key` is never returned in plaintext by any of these — list/get/create/
update responses mask it as `***`.

### Provider Settings UI

**Settings → Providers** (`ProviderSettingsView.vue`, route
`/p/{project}/settings/providers` — this replaces the old
`/settings/ollama` route, which redirects here). It lists configured
providers with live health status, and lets you add, edit, test, and delete
a provider, and browse the models it currently advertises (including
whether each one reports tool-calling support).

---

## 3. Agents as `{provider, model}` pairs

An agent no longer embeds raw connection details for HTTP-based drivers.
`AgentConfig` in `lifecycle/config.yaml` carries:

```yaml
agents:
  - name: local-backend-developer
    role: [backend-developer]
    driver: openai-compatible
    provider: loki                 # name of a configured Provider
    model: qwen3-coder:30b         # model id sent as the request "model" field
    max_tool_iterations: 25        # optional — per-agent override of the tool-call loop cap (default 25)
    active_status: in-development
    source_types: [plan-backend]
    timeout_minutes: 60
    allowed_write_paths: [internal, cmd]
    git_identity:
      name: Local Backend Developer
      email: local-backend-developer@kaos-control.local
    prompt_templates:
      backend-developer: |
        You are a backend developer. Read the plan at {target_path} and
        implement it in Go.
```

`config.Validate` rejects an `openai-compatible` agent whose `provider`
doesn't name a configured Provider, or whose `model` is empty — each
failure names the offending agent.

The Agent Editor (`AgentConfigForm.vue`) selects a registered provider and
then a model discovered from it (via `/api/providers/{name}/models`)
instead of free-typing connection fields.

---

## 4. The `openai-compatible` driver

Registered under the driver name **`openai-compatible`**, alongside
`claude-code-cli`, `claude-mediated`, `claude-env`, `codex-cli`, `gemini`,
`gemini-cli`, and `shell-stub`. It gives a provider running behind an
OpenAI-compatible endpoint the **full agentic loop** — tool use, file
edits, streamed progress, run logging — that previously only
`claude-code-cli`/`claude-env` had.

### Request construction

On `Start`, the driver POSTs to `<base_url>/v1/chat/completions` with
`model`, a `messages` array (system + user, built from the run's prompt via
the same `---SYSTEM--- / ---USER---` split every driver uses), a non-empty
`tools` array, and `stream: true`. `Authorization: Bearer <token>` is set
only when the provider has an `api_key`.

### Tool-calling loop

The v1 tool set is fixed and sandboxed — no `bash` tool, deliberately, since
shell execution is the widest surface and small local models are the least
trustworthy with it:

| Tool | Arguments | Behaviour |
|---|---|---|
| `read_file` | `path` | Reads a file's full contents. |
| `write_file` | `path`, `content` | Writes full content to a file. Refused with a tool-error result (not a crash, not an out-of-scope write) if `path` falls outside the agent's `allowed_write_paths`. |
| `list_dir` | `path` | Lists a directory's entries. |
| `grep` | `pattern`, `path` (optional, defaults to `.`) | Regex search within a directory or file. |

All four tools execute through the same sandbox path resolver every other
driver uses, so reads and writes can never escape the project root.

When a response has `finish_reason: tool_calls`, the driver executes each
call locally, appends the assistant's `tool_calls` message and one `tool`
result message per call (keyed by `tool_call_id`), and re-issues the
completion. This repeats until the model returns a final assistant message
with `finish_reason: stop`, or the loop hits its cap — **25 by default**,
overridable per agent via `max_tool_iterations` — at which point the run
ends with a clear terminal status rather than looping unbounded.

### Native tool-call recovery

Some models/back-ends emit tool calls in their own text syntax instead of
the endpoint parsing them into `tool_calls` (observed: Qwen3-Coder does this
*through llama.cpp*, but returns clean `tool_calls` *through Ollama* — this
is a server/chat-template property, not a model property). When a turn
returns no `tool_calls` but non-empty `content`, the driver attempts a
fallback parse for two known native encodings —
`<function=NAME><parameter=KEY>VALUE</parameter></function>` and
`<tool_call>{json}</tool_call>` — before treating the turn as a final
answer. Recovered calls are normalised to the OpenAI shape and executed the
same way, but counted and logged separately from native OpenAI-shaped
calls, since an off-the-shelf client would not have recovered them.

### Preflight tool-capability check

An HTTP 200 does not prove a model saw the advertised tools — the most
dangerous observed failure mode is a **silent drop**: the server ignores
`tools` and the model answers fluently and confidently with a fabricated,
indistinguishable-from-real response. To catch this, the driver compares
`usage.prompt_tokens` for the request *with* `tools` against the same
request *without* them before/at the first turn. An identical count means
the tools were silently dropped, and the run **hard-fails** — it does not
degrade to a chat-only completion, and no artifacts are written. An
explicit server rejection (e.g. Ollama's HTTP 400
`"<model> does not support tools"`) is surfaced verbatim and is not
retried.

### Streaming, progress, logging, cancellation

- Consumes the endpoint's streamed delta response and emits `ProgressEvent`s
  (raw line + parsed JSON where available), matching the streaming
  convention other drivers use.
- Records time-to-first-token on the first streamed content token.
- Writes a per-run log (when `Run.LogPath` is set) with a header
  (run id, agent, role, `driver=openai-compatible`, `base_url`, model, start
  time), the system/user prompts, each turn's tool calls and results, the
  final assistant message, and a finish-time footer.
- `Process.Kill()` cancels the in-flight request and terminates the loop;
  `Process.Wait()` returns the terminal error. Runs respect
  `timeout_minutes` and the global `max_concurrent_agents` semaphore.
- HTTP 529/429/gateway-unreachable failures are parsed and re-broadcast as
  `queue.rate_limit` — the same signal the work queue already reacts to for
  pause/retry — so failover logic (§5) is uniform across every driver.

### Secret hygiene

A configured `api_key` never appears in run logs, `ProgressEvent.Raw`,
stderr tails, or any API response that returns agent or provider config —
it is masked as `***` wherever it would otherwise surface, matching the
existing `ollama_instances.api_key` convention. `base_url` and any token
live in the app/project config file on disk, the same trust boundary as
other kaos-control credentials — no new secret-storage mechanism.

---

## 5. Dynamic provider switching and failover

When an upstream provider becomes unavailable (HTTP 529 overload, HTTP 429
quota exhaustion, or an unreachable endpoint), agents can fail over to a
configured fallback provider automatically, or an operator can switch them
manually — without hand-editing `lifecycle/config.yaml` and waiting for a
restart.

### Per-agent fallback configuration

```yaml
agents:
  - name: backend-developer
    provider: anthropic-direct
    model: claude-sonnet-5
    fallback_provider: openrouter     # switched to on a matching failure
    fallback_model: anthropic/claude-sonnet-5
    # primary_provider / primary_model are populated automatically by a
    # failover and cleared on restore — do not set them by hand.
```

### Automatic failover

Project-level policy (`ProviderFailoverConfig` in `lifecycle/config.yaml`):

```yaml
provider_failover:
  enabled: true                 # gates the whole subsystem; manual switch/restore still works when false
  auto_switch: false            # opt-in per project — the dispatcher only auto-switches when true
  switch_on_kinds: [overloaded, rate_limit, unreachable]  # default: all three
  max_failovers_per_run: 1      # bounds cascading switches for one queued job
  probe_interval_seconds: 60    # how often the recovery prober re-checks a switched-away primary
```

When `auto_switch` is on, the queue dispatcher reacts to a matching failure
kind on a queued job by switching the affected agent to its configured
`fallback_provider`/`fallback_model` and re-enqueueing the job immediately,
instead of pausing the whole queue for the usual 30–60 minute backoff. The
switch preserves the agent's original `provider`/`model` as
`primary_provider`/`primary_model` so it can be restored later, and is
recorded in the project's event feed and git commit log (atomic config
rewrite + hot-reload via `project.ReloadConfig()` — no server restart).

A **recovery prober** re-probes each switched-away primary provider (a
bounded `GET <base_url>/v1/models`, non-5xx counts as healthy) every
`probe_interval_seconds`, so operators are notified when it's safe to
restore.

### Manual controls

REST API (project-scoped, `RolesDevopsOrAdmin`):

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/p/{project}/provider-switch/status` | Current failover status — which agents are on a fallback, primary health. |
| `POST` | `/api/p/{project}/agents/{name}/switch-provider` | Switch one agent to a named provider/model. |
| `POST` | `/api/p/{project}/agents/{name}/restore-provider` | Restore one agent to its `primary_provider`/`primary_model`. |
| `POST` | `/api/p/{project}/provider-switch/switch-all` | Batch-switch every agent on one provider to another. |
| `POST` | `/api/p/{project}/provider-switch/restore-all` | Restore every agent currently on a fallback. |
| `GET` | `/api/p/{project}/provider-templates` | List named provider templates (bulk provider/model presets across roles). |
| `POST` | `/api/p/{project}/provider-templates/apply` | Apply a named template across the project's agents. |

UI: the **Agents** panel shows a fallback badge and a **Restore Primary**
button on any agent currently failed over, plus a switch-provider action
per agent card; a global header control offers the batch switch/restore
actions.

---

## Troubleshooting

**Run fails immediately with "tool calling is unsupported" / a provider and
model named in the error.** The preflight detected either a silent
`tools` drop (identical `prompt_tokens` with and without `tools`) or an
explicit server rejection. This is deliberate — there is no degraded
chat-only fallback, because a model that can't see the tools will still
answer fluently and the answer will be fabricated. Pick a model/endpoint
combination known to support tool calling (see the requirement's verified
matrix: `gemma-4-26B-A4B-it-UD-Q8_K_XL` and `gpt-oss-20b-Q8_0` on llama.cpp
`--jinja`; `qwen3-coder:30b` / `gemma4:26b` on Ollama's `/v1` endpoint), or
check the provider's `/v1/models` response via **Settings → Providers →
Models**.

**Agent loop ends at the iteration cap without finishing.** Either the task
genuinely needs more than 25 tool calls, or the model is looping without
converging. Raise `max_tool_iterations` on the agent for slower/cheaper
local models if the former; otherwise treat it as a model capability issue.

**Provider delete fails with a list of agent names.** Providers in use by
at least one agent in any project can't be deleted. Repoint or remove those
agents first.

**Config validation error naming a `provider` field.** The agent's
`provider` doesn't match any configured Provider `name`, or the Provider
itself has an invalid `base_url`. Check **Settings → Providers**.

**A switched-over agent never recovers automatically.** Automatic
*failback* is not implemented — the recovery prober only notifies that the
primary is healthy again; you still restore explicitly via **Restore
Primary** or `POST .../restore-provider`.

---

## Related and in-progress work

- [[provider-model-for-agents]] — the Provider entity, management API, and
  agent `{provider, model}` shape (§1–3 above). **Done.**
- [[open-provider-support]] (this epic's driver requirement,
  [[open-provider-support-2]]) — the `openai-compatible` driver itself
  (§4 above). **Done.**
- [[switch-provider]] — dynamic switching and automated failover (§5
  above). **Done.**
- [[local-model-operability]] — workstream 3: local-model-tuned prompt
  templates, richer availability checks, error surfacing. Not yet planned
  in detail.
- [[inline-driver-provider-abstraction]] — extending the same Provider
  abstraction to the conversational idea-chat completer (currently
  in planning/development).
- [[agent-logging-provider-driver]] — recording the provider and driver
  used on every agent run record, so run history and the usage report can
  break down by provider (currently in planning/development).
- [[reporting-open-provider-gemini-stream]] — reporting changes to account
  for per-provider agent runs and Gemini's JSON streaming shape (approved,
  not yet built).
- [[ollama-local-llms]] — the retired native-`ollama`-driver feature this
  epic replaces.
- `docs/ollama-claude-code-driver.md` — the `claude-env` driver, which is
  unaffected by this epic and remains the way to run the full Claude Code
  loop against an Anthropic-compatible endpoint (as distinct from an
  OpenAI-compatible one).
