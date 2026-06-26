# `claude-env` Driver: Full Claude Code Loop on Local / Custom Endpoints

The `claude-env` driver lets you run the **full Claude Code agentic loop** — tool
use, file edits, streamed progress, run logging — against any Anthropic-compatible
endpoint. The primary supported target is [Ollama's Anthropic compatibility
shim](https://ollama.com/blog/openai-compatibility); other compatible endpoints
(LiteLLM, vLLM Anthropic proxy, etc.) work on a best-effort basis.

---

## Why `claude-env` instead of the `ollama` driver?

kaos-control ships two separate ways to reach local models:

| Feature | `ollama` driver | `claude-env` driver |
|---|---|---|
| Execution model | Single batch prompt/response | Full Claude Code agentic loop |
| Tool use | No | Yes |
| File edits | No | Yes |
| Streamed progress events | No | Yes |
| Run log files | No | Yes |
| TTFT measurement | No | Yes |
| Process kill / cancellation | No | Yes |
| Endpoint target | Ollama native API | Any Anthropic-compatible endpoint |

Use `claude-env` whenever you need an agent that can **read and write files,
call tools, and make multi-step decisions** — all the things that make Claude
Code useful — but want the inference to happen locally or against a self-hosted
model.

---

## How it works

`claude-env` is a thin wrapper around the same `claude` CLI binary used by
`claude-code-cli`. On each run it:

1. Builds the same argument vector as `claude-code-cli` (bypass-permissions,
   `--output-format stream-json`, `--model <model>`, and the agent prompt).
2. Inherits the current process environment, then **appends**
   `ANTHROPIC_BASE_URL` and `ANTHROPIC_AUTH_TOKEN`, overriding any inherited
   values.
3. Spawns `claude` in the project root and hands the subprocess to the shared
   `startCommandProcess` launcher.
4. Streams `stream-json` progress events, records time-to-first-token, writes
   the per-run log file, and supports `Kill()` / `Wait()` — identical in every
   respect to `claude-code-cli`.

---

## Configuration

### Fields

All three fields are required for `driver: claude-env`:

| Field | Type | Description |
|---|---|---|
| `base_url` | string | Anthropic-compatible endpoint base URL. Must be a valid `http` or `https` URL. |
| `auth_token` | string | Bearer token sent as `ANTHROPIC_AUTH_TOKEN`. For the Ollama shim, use the literal string `"ollama"`. |
| `model` | string | Model tag passed to `claude` as `--model`. Must match a model available at the configured endpoint. |

`base_url` and `auth_token` are specific to `driver: claude-env`; all other
fields (`name`, `role`, `active_status`, `timeout_minutes`, `allowed_write_paths`,
`prompt_templates`, etc.) follow the same conventions as other drivers.

### Minimal example — Ollama running locally

```yaml
agents:
  - name: local-backend-developer
    role:
      - backend-developer
    driver: claude-env
    base_url: http://localhost:11434
    auth_token: ollama
    model: qwen2.5-coder:32b
    active_status: in-development
    source_types:
      - plan-backend
    timeout_minutes: 60
    allowed_write_paths:
      - internal
      - cmd
    git_identity:
      name: Local Backend Developer
      email: local-backend-developer@kaos-control.local
    prompt_templates:
      backend-developer: |
        You are a backend developer. Read the plan at {target_path} and
        implement it in Go.
```

### Remote / air-gapped endpoint example

```yaml
agents:
  - name: airgapped-analyst
    role:
      - analyst
    driver: claude-env
    base_url: https://llm-gateway.internal:8080
    auth_token: sk-...
    model: llama3.3:70b
    active_status: clarifying
    source_types:
      - idea
    timeout_minutes: 30
    allowed_write_paths:
      - lifecycle/requirements
    git_identity:
      name: Air-gapped Analyst
      email: analyst@kaos-control.local
    prompt_templates:
      analyst: |
        You are an analyst. Read the idea at {target_path} and produce a
        requirement artifact.
```

---

## Setting up Ollama

1. **Install Ollama** — [ollama.com/download](https://ollama.com/download) or
   via your package manager.

2. **Pull a model** that supports tool use:
   ```sh
   ollama pull qwen2.5-coder:32b
   # or
   ollama pull llama3.3:70b
   ```

3. **Verify the Anthropic-compatibility shim** is accessible (Ollama exposes it
   on the same port as the native API, at `/v1/`):
   ```sh
   curl -s http://localhost:11434/v1/models | jq '.data[].id'
   ```
   You should see the models you have pulled listed.

4. **Configure kaos-control** as shown in the minimal example above. Point
   `base_url` at `http://localhost:11434` and set `auth_token: ollama`.

5. **Choose a model carefully.** The quality of tool use varies significantly
   between community models. Models explicitly trained for tool/function calling
   (e.g. `qwen2.5-coder`, `llama3.3`) produce better results than general
   chat models.

---

## Config validation

kaos-control validates agent config at startup. A `claude-env` agent fails
validation with a descriptive error if any of the following are true:

| Condition | Error message |
|---|---|
| `base_url` is absent | `agent "X" has driver=claude-env but missing base_url` |
| `base_url` is not a valid http/https URL | `agent "X" has driver=claude-env but base_url "Y" is not a valid http/https URL` |
| `auth_token` is absent | `agent "X" has driver=claude-env but missing auth_token` |
| `model` is absent | `agent "X" has driver=claude-env but missing model` |

---

## Security: secret hygiene

`auth_token` is treated as a secret throughout the system:

- It is **never written** to per-run log files.
- It is **never included** in `ProgressEvent.Raw` or the stderr tail that
  appears in run-failure records.
- When agent configuration is returned via the REST API, `auth_token` is
  **masked** as `***`. Only `base_url` and `model` are surfaced as plaintext.
- The run-log header records the argument vector and model but does not include
  the injected environment variables.

`auth_token` is stored in the project `config.yaml` on disk — the same trust
boundary as other kaos-control credentials (`ollama_instances.api_key`, auth
config). No additional secret-storage mechanism is provided in v1.

---

## Runtime behaviour

### Concurrency

`claude-env` runs are subject to the global `max_concurrent_agents` semaphore
and lineage locking, exactly like all other drivers.

### Cancellation and timeout

The `claude` subprocess is started with a Go `context.Context`. Calling
`Process.Kill()` cancels the context and sends SIGTERM to the subprocess.
`timeout_minutes` applies the same way as for `claude-code-cli`.

### Endpoint unreachable

If the configured endpoint is unavailable, the `claude` CLI exits with a
non-zero status. kaos-control surfaces that exit code and the stderr tail
through the normal run-failure path. The run does not hang beyond
`timeout_minutes`.

### Environment variable precedence

`ANTHROPIC_BASE_URL` and `ANTHROPIC_AUTH_TOKEN` are appended to the inherited
environment. Because Go's `exec.Cmd` uses the **last occurrence** of a
duplicated key, the configured values always take precedence over any
same-named variables in the operator's shell environment.

---

## Caveats and limitations

### Model tool-use fidelity

This driver's contract is "inject the environment and run `claude`". Whether a
given local model faithfully honours Claude Code's tool-use protocol is
determined by the model, not by kaos-control. Results vary widely between
models, quantisation levels, and context lengths. Expect:

- **Best results** with models explicitly trained for tool calling
  (`qwen2.5-coder`, `llama3.3`, `deepseek-coder-v2`).
- **Degraded results** with general instruction-tuned models that lack
  tool-use fine-tuning.
- **Broken tool use** with base (non-instruct) models.

Test your chosen model with a simple agent task before committing to it for
production workflows.

### Only the Ollama shim is officially tested

Other Anthropic-compatible endpoints — LiteLLM, vLLM's Anthropic proxy,
and similar gateways — are expected to work but are **not tested** by the
kaos-control test suite. If a remote endpoint is compatible with the Anthropic
Messages API, it should be usable with `claude-env`, but you may encounter
endpoint-specific incompatibilities.

### Config-file only in v1

There is no UI for creating or editing `claude-env` agents. Configuration must
be done by editing `lifecycle/config.yaml` (or the project config file directly)
and restarting kaos-control.

### No Ollama model management

`claude-env` does not list, pull, or delete Ollama models. It only needs a
reachable `base_url`, a token, and a model tag. Ollama model management (pull,
list, instance registration) is handled by the separate `ollama` driver and
its associated instance configuration — see the Ollama Agent Support
documentation if you need that functionality.

### Permission model

`claude-env` uses the same bypass-permissions argument vector as `claude-code-cli`.
It does not support hook-gated tool calls (`claude-mediated`-style permission
mediation) in v1. A mediated variant may be added in a future release.

### Context window constraints

Local models typically support significantly smaller context windows than hosted
Anthropic models. Long-running agent tasks that accumulate large amounts of tool
output may exceed the model's context limit. Prefer models with at least 32 k
tokens of context for non-trivial tasks.

---

## Troubleshooting

### `base_url` validation error at startup

Check that `base_url` is a fully-qualified URL including the scheme:

```yaml
# Wrong
base_url: localhost:11434

# Correct
base_url: http://localhost:11434
```

### Agent run fails immediately with connection refused

Ollama must be running before kaos-control starts the agent. Verify Ollama is up:

```sh
curl http://localhost:11434/v1/models
```

If you get `connection refused`, start Ollama:

```sh
ollama serve
```

### Model not found

Ensure the model tag in `model:` matches exactly what Ollama knows:

```sh
ollama list
```

The tag in `model:` must match one of the listed model names (e.g.
`qwen2.5-coder:32b`, not `qwen2.5-coder`).

### Tool calls not working / agent loops without making progress

The local model may not support tool use reliably. Switch to a model known for
tool-calling capability (`qwen2.5-coder:32b`, `llama3.3:70b`), or consider
using the hosted Anthropic API with `claude-code-cli` for tasks that require
reliable tool use.

### `auth_token` appearing in unexpected places

If you see the token value in a log, file, or API response, that is a bug —
please file a defect. The system is designed to mask and exclude `auth_token`
from all output paths.
