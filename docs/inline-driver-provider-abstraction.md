# Inline Conversational Driver Provider Abstraction

The `openai-compatible` async agent driver (see
[open-provider-support.md](open-provider-support.md)) already lets any agent
in the `Manager` driver map target Claude, OpenAI, OpenRouter, Ollama, or a
local `llama-server` via a first-class `Provider` record. Until this feature,
kaos-control's **second** LLM execution path — the inline conversational and
single-shot generation flows in `internal/ideachat/` — could not: it was
hard-coupled to shelling out to the `claude` CLI binary. This document covers
the completer abstraction that closes that gap.

## Who this is for

- **Operators** who want the Idea Capture chat and the idea/defect/doc
  generation flows to run against a provider other than the Claude CLI (an
  OpenAI-compatible gateway, Ollama, or a local inference server) — see
  [Configuration](#configuration).
- **Developers** extending `internal/ideachat/` or adding a new completer
  implementation — see [Architecture](#architecture).

---

## The two LLM execution paths

| | Async agent path (`internal/agent/`) | Inline path (`internal/ideachat/`) |
|---|---|---|
| Consumers | Requirements, plans, backend/frontend/test dev, QA, tech-writer agents — anything run via the `Manager` / work queue | Idea Capture chat (`idea-capture`), and single-shot `idea-generate` / `defect-generate` / `doc-generate` |
| Execution model | Full agentic loop — tool use, file edits, streamed progress, run logging, kill/timeout supervision | One blocking chat completion; the server (not the model) parses the response and writes the artifact |
| Provider abstraction | `openai-compatible` driver against a `Provider` record (done in [[provider-model-for-agents]]) | `Completer` interface against the same `Provider` records (this feature) |
| Tool calling | Yes (`read_file`, `write_file`, `list_dir`, `grep`) | No — deliberately out of scope; see [Why the inline path stays tool-free](#why-the-inline-path-stays-tool-free) |

Both paths now resolve provider identity from the same `{provider, model}`
config shape and the same app-level `providers:` list, but they remain two
separate execution engines. This feature does **not** merge them, add inline
agents to the async run pipeline, or give the inline path tool-calling.

---

## Architecture

### The `Completer` interface

`internal/ideachat/completer.go` defines:

```go
type Completer interface {
	Complete(ctx context.Context, cfg ModelConfig, messages []LLMMessage) (string, error)
}
```

`CallLLM` — the package-level function every inline consumer (`converse.go`,
`generate.go`) calls — is a **reassignable `var`** that in production points
at a dispatcher:

```go
var CallLLM = dispatchComplete

func dispatchComplete(ctx context.Context, cfg ModelConfig, messages []LLMMessage) (string, error) {
	completer, err := selectCompleter(cfg)
	if err != nil {
		return "", err
	}
	return completer.Complete(ctx, cfg, messages)
}
```

`selectCompleter` picks the implementation from `cfg.Provider`:

- `cfg.Provider == nil` → `claudeCLICompleter{}` (the default).
- `cfg.Provider.Driver == "openai-compatible"` → `&openAICompleter{provider: *cfg.Provider}`.
- Any other driver value → a named error (defensive; config validation is
  expected to prevent this occurring in practice).

Tests continue to reassign `CallLLM` directly to a fake, bypassing the
dispatcher entirely — this was true before the abstraction and remains true
after it, so no existing test needed to change.

### `ModelConfig` carries provider identity, never a secret surface

```go
type ModelConfig struct {
	Model        string
	SystemPrompt string
	MaxTokens    int

	// Provider identifies the app-level provider + driver to route this call
	// through. Nil means "use the Claude CLI default".
	Provider *config.ProviderConfig
}
```

`ModelConfig` has no struct tags and is never marshalled to JSON/YAML or
returned by any HTTP handler — it exists only to carry enough information
from `resolveIdeaCaptureConfig` (see below) to the dispatcher. Embedding the
full `config.ProviderConfig` (including `api_key`, `extra_headers`) here is
therefore safe: there is no serialisation path that could leak it.

### The Claude CLI completer (default)

`internal/ideachat/claude_cli.go`'s `claudeCLICompleter` is the pre-existing
`callLLMImpl` behaviour, refactored **verbatim**: it builds the same prompt
via `buildPrompt` (folding system prompt + `Human:`/`Assistant:`-tagged
history into one string) and shells out to:

```sh
claude --dangerously-skip-permissions -p <prompt> --model <model>
```

An inline agent with **no** `provider` configured — the state of every
shipped inline agent (`idea-capture`, `idea-generate`, `defect-generate`,
`doc-generate`) — routes here and behaves exactly as it did before this
feature existed. There is no CLI sentinel provider name in v1; "no provider"
*is* the signal for the CLI default.

### The OpenAI-compatible inline completer

`internal/ideachat/openai_completer.go`'s `openAICompleter` is a plain,
**non-streaming**, tool-free client against
`<base_url>/v1/chat/completions`, reusing the app-level `Provider` record
(`base_url`, `api_key`, `extra_headers`) that the async driver already
validates and manages. It is intentionally a small, standalone
implementation rather than a reuse of the async `openai-compatible` driver's
client (`internal/agent/openai_compatible.go`), which is welded to
streaming, tool-calling, preflight, and native-tool-call recovery — none of
which the inline path needs.

Request construction:

- `model` — from `cfg.Model`.
- `messages` — a `system` message first (only when `cfg.SystemPrompt` is
  non-empty), then each `LLMMessage` mapped in order with its `role`
  (`user`/`assistant`) preserved.
- `max_tokens` — included only when `cfg.MaxTokens > 0`.
- **No `tools` key is ever sent.**

Headers: `Content-Type: application/json` always; `Authorization: Bearer
<api_key>` only when the provider has a non-empty key; each `extra_headers`
entry applied.

The response's `choices[0].message.content` is decoded and returned,
trimmed — matching the CLI completer's return contract exactly. `content`
may be a plain string or an array of `{"type":"text","text":"..."}` parts;
both shapes are handled.

Failure handling is bounded and non-retrying: a non-2xx response, malformed
JSON, an empty `choices` array, or a transport/context error each return a
plain Go `error` — never a panic, never an unbounded retry — and the error
text is scrubbed of the provider's `api_key` value before it can reach a log
line or an API response.

### Provider resolution

`internal/http/idea_chat.go`'s `resolveIdeaCaptureConfig` is the single
function that builds a `ModelConfig` for both the converse endpoint
(`handleIdeaConverse`) and the generate endpoint (`handleIdeaGenerate`), for
all four template keys. It:

1. Looks up the owning agent (`idea-capture` for `idea-capture`/
   `idea-generate`/`defect-generate`; `docs-capture` for `doc-generate`).
2. Resolves `model`, falling back to `claude-sonnet-4-6` if unset.
3. Calls `resolveAgentProvider(p, agent.Provider)`, which looks up
   `agent.Provider` by name in the project's app-level `Providers` snapshot
   and returns a pointer to the matching `config.ProviderConfig`, or `nil`
   if the agent's `provider` field is empty (or, defensively, unregistered —
   config validation is expected to prevent that case reaching here).
4. Resolves the system prompt from the agent's `prompt_templates`, falling
   back to a built-in default template, exactly as before this feature.

The same resolver feeds all four inline template keys, so a single
`provider:` binding on the `idea-capture` (or `docs-capture`) agent covers
conversational capture and every single-shot generation flow.

### Why the inline path stays tool-free

The inline path returns a single completion string that **server-side Go
code** (`generate.go`) parses and writes to a fixed location under the
agent's `allowed_write_paths` — the model never issues a Write/Edit/Bash
tool call. This keeps the change outside the scope of
[[adr-0006-mediated-agent-driver-permission-model]]: there is no
model-driven tool execution here for mediation to govern. If the inline
abstraction is ever extended to let a model issue tool calls, those calls
**must** route through the mediated driver path — this feature deliberately
keeps the completer tool-free so no such gap is opened in the meantime.

---

## Configuration

### Default behaviour (no config change required)

An inline agent with no `provider` field — the shipped default for
`idea-capture` and `docs-capture` — keeps using the Claude CLI exactly as
before:

```yaml
agents:
  - name: idea-capture
    driver: inline
    model: claude-sonnet-4-6
    # no `provider:` — routes to the Claude CLI completer
    prompt_templates:
      idea-capture: |
        You are helping a user capture a new idea...
```

### Pointing an inline agent at a Provider

Register (or reuse) an app-level Provider — the same `providers:` entries
used by the async `openai-compatible` driver — in
`~/.kaos-control/config.yaml`:

```yaml
providers:
  - name: loki
    base_url: http://leia.packsin.com:11434   # Ollama's /v1 endpoint
    driver: openai-compatible
```

Then bind the inline agent to it by name in the project's
`lifecycle/config.yaml`:

```yaml
agents:
  - name: idea-capture
    driver: inline
    provider: loki                # name of a configured Provider
    model: qwen3-coder:30b        # model id sent as the request "model" field
    prompt_templates:
      idea-capture: |
        You are helping a user capture a new idea...
```

With this binding, both the Idea Capture chat and `idea-generate` /
`defect-generate` (owned by the same `idea-capture` agent) route through the
OpenAI-compatible completer against `loki`. Bind `docs-capture` the same way
to cover `doc-generate` independently.

No UI exists for this in v1 — provider selection for the inline path is
config-only. Editing the file and restarting (or triggering a config reload)
is the only way to change it.

### Local / offline capture and generation

Because the OpenAI-compatible completer is a plain HTTP client with no
external dependency, pointing an inline agent at a **local** provider (e.g.
`http://localhost:11434` for Ollama, or a local `llama-server`) lets Idea
Capture and generation work with **zero internet connectivity** — a
capability the CLI-only path never offered. See
[local-model-operability.md](local-model-operability.md) for guidance on
model selection and quantisation for local inference.

### Config validation

`config.Validate` (project config) and the new `config.ValidateAgentProviders`
(invoked at project load, once app-level providers are available) enforce:

| Condition | Result |
|---|---|
| Inline agent with `provider` set and empty `model` | Rejected — error names the agent. |
| Inline agent with `provider` naming a provider absent from app config | Rejected — error names the agent and the missing provider. |
| Inline agent with no `provider` | Validates exactly as before this feature — no new mandatory field. |

Existing app and project config files load and validate unchanged; the
`inline` driver marker and the shipped config with no `provider` set on any
inline agent are unaffected by this feature.

---

## Secret hygiene

`api_key` and `extra_headers` credential material:

- Is carried only on the internal, never-serialised `ModelConfig` /
  `config.ProviderConfig`.
- Is sent by the OpenAI-compatible completer solely as request headers
  (`Authorization: Bearer <api_key>`, plus each `extra_headers` entry) — it
  is never placed in a prompt, conversation history, or generated artifact.
- Is scrubbed from every error string the completer returns before it can
  reach a log line or an API response.
- Remains masked (`***`) in every Provider-listing API response, matching
  the existing convention documented in
  [open-provider-support.md](open-provider-support.md#secret-hygiene).

This conforms to [[secrets-handling]].

---

## What did not change

- **No new UI.** Provider selection for the inline path is config-only;
  the Idea Capture chat and generate/preview flows call the same endpoints
  (`POST /ideas/converse`, `POST /ideas/generate`) with the same
  request/response shapes regardless of which completer backs them.
- **No tool-calling from the inline path.** See
  [Why the inline path stays tool-free](#why-the-inline-path-stays-tool-free).
- **No run recording.** Inline calls remain synchronous, in-request, and
  produce no `agent_runs` row or provider/driver attribution — that is
  tracked separately under [[agent-logging-provider-driver]].
- **No provider failover for inline calls.** A failed inline call surfaces
  a single bounded error; switching to a different provider is a manual
  config edit, not an automatic retry. Automatic inline failover may be
  revisited alongside [[provider-failover]].
- **No new third-party dependency.** The OpenAI-compatible inline completer
  uses only `net/http` and `encoding/json`, matching the async driver's
  stdlib-only approach.

---

## Troubleshooting

**Idea Capture / generation fails with a `claude` binary error.** Nothing
changed here — this is the same failure mode as before this feature. Verify
`claude` is on `PATH`, or bind the agent to a Provider instead (see
[Configuration](#configuration)) to remove the dependency.

**An inline agent bound to a Provider fails with a status-code or decode
error.** The OpenAI-compatible completer surfaces non-2xx responses,
malformed JSON, and empty-`choices` responses as a returned error. Check
the provider's reachability and model id via **Settings → Providers →
Models**, the same page the async driver uses.

**Config fails to load with an error naming an inline agent and a provider.**
The agent's `provider` field names a Provider that isn't registered in app
config, or the agent has `provider` set with no `model`. Fix the binding in
`lifecycle/config.yaml` or register the missing Provider in
`~/.kaos-control/config.yaml`.

**I want the model to write files or call tools from the inline chat.** Not
supported — the inline path is a plain chat completion; use an async agent
(`internal/agent/`, the work queue) for anything requiring tool use or file
edits.

---

## Related

- [open-provider-support.md](open-provider-support.md) — the `Provider`
  entity, the async `openai-compatible` driver, and provider
  switching/failover, whose `Provider` record this feature reuses.
- [local-model-operability.md](local-model-operability.md) — local-model
  operability guidance applicable to any provider-backed completer,
  including this inline one.
- [[inline-driver-provider-abstraction]] (idea),
  [[inline-driver-provider-abstraction-2]] (requirement, in
  `lifecycle/requirements/`), and the companion backend/frontend/test plans
  under the same lineage.
- [[inline-conversational-provider-abstraction]] — the feature record for
  this capability.
- [[provider-management]], [[provider-failover]] — the async-path features
  this one brings the inline path to parity with.
