---
title: Inline Conversational Driver Provider Abstraction
type: requirement
status: planning
lineage: inline-driver-provider-abstraction
created: "2026-08-25T14:00:00+10:00"
priority: normal
parent: lifecycle/ideas/inline-driver-provider-abstraction.md
labels:
    - driver
    - provider
    - agent
    - ideachat
    - inline
    - open-provider-support
    - backend
release: KC-Release6
assignees:
    - role: product-owner
      who: agent
---

# Inline Conversational Driver Provider Abstraction

Parent: [[inline-driver-provider-abstraction]] (idea). Related:
[[provider-model-for-agents]], [[open-provider-support]], [[switch-provider]],
[[agent-logging-provider-driver]], [[local-model-operability]].

## Problem

kaos-control has **two independent LLM execution paths**, and only one of them
was ever brought under the Provider abstraction delivered by the
[[open-provider-support]] epic:

1. **The asynchronous agent path** (`internal/agent/`, the `Manager` driver map)
   now speaks to a first-class **Provider** record
   (`{name, base_url, api_key, driver, extra_headers}`) via the
   `openai-compatible` driver, so an agent is a portable `{provider, model}`
   pair ([[provider-model-for-agents]]). This path can already target Claude,
   OpenAI, OpenRouter, Ollama, or a local `llama-server`.

2. **The inline conversational path** (`internal/ideachat/`) has **not**. The
   conversational and generation agents declared `driver: inline` in
   `lifecycle/config.yaml` — `idea-capture` (multi-turn chat, `converse.go`),
   `idea-generate`, `defect-generate`, and `doc-generate` (single-shot,
   `generate.go`) — all funnel through one hard-coded function,
   `callLLMImpl` in `internal/ideachat/llm.go`, which shells out to the
   **`claude` CLI binary**:

   ```go
   exec.CommandContext(ctx, "claude", "--dangerously-skip-permissions", "-p", prompt, "--model", model)
   ```

Consequences of the coupling:

- **No provider choice.** The inline path is Claude-only and cannot be pointed
  at OpenAI, a gateway, Ollama, or a local `llama-server`, even though the app
  already holds validated Provider records for exactly those. Selecting an
  alternative requires a code change, not config.
- **A hidden external dependency.** Every inline agent silently requires the
  `claude` binary on `PATH`; there is no config-visible way to see or change
  this, and it defeats the project's tool-agnostic stance ([[adr-0005-agents-md-primary-directives]]).
- **`inline` is not a real driver.** The name is a config marker consumed by
  `internal/http/idea_chat.go` (`resolveIdeaCaptureConfig`); it is absent from
  the `Manager` driver map. So the async path's provider work never reached it.
- **Fields are silently ignored.** `ideachat.ModelConfig` carries `MaxTokens`,
  but the CLI path drops it; an inline agent's `provider` field
  (`AgentConfig.Provider` already exists) is never read.

This requirement breaks the inline path's compile-time coupling to Claude by
introducing a small provider abstraction that `CallLLM` delegates to, selected
by the agent's `{provider, model}` config — with the existing Claude CLI as the
default so current behaviour is preserved exactly. It brings the inline path to
parity with the async path's Provider spine without duplicating the async
driver's tool-calling machinery, which the inline path does not use.

## Goals / Non-goals

### Goals

- Replace the single hard-coded `callLLMImpl` with a **completer interface** in
  `internal/ideachat/` that `CallLLM` dispatches to based on resolved
  configuration, so provider selection is a config/runtime concern, not a
  compile-time dependency.
- Keep the **Claude CLI implementation as the default**: an inline agent with no
  `provider` (or the explicit CLI sentinel) behaves **exactly as today**,
  including its dependency on the `claude` binary and its current prompt shape.
- Add an **OpenAI-compatible inline completer** — a plain (no tool-calling)
  `/v1/chat/completions` client — that reuses the **existing app-level Provider
  record** (`config.ProviderConfig`) resolved by name, so the inline path can
  target any provider the async path can (Claude via API, OpenAI, OpenRouter,
  Ollama `/v1`, local `llama-server`).
- Resolve the completer from the agent's `{provider, model}` config
  (`AgentConfig.Provider` / `.Model`), the same identifiers used by the async
  path ([[provider-model-for-agents]]).
- Honour `ModelConfig.MaxTokens` and the system/user/assistant message roles on
  the OpenAI-compatible path.
- Conform to [[secrets-handling]]: provider `api_key` is never logged, never
  placed in a prompt or artifact, and stays masked in API responses.
- Leave both inline consumers (`converse.go` multi-turn, `generate.go`
  single-shot) and all four inline agents working, with no change to their
  produced-artifact shape.

### Non-goals

- **Tool-calling / file-writing from the inline path.** The inline path is a
  plain chat completion whose text output is parsed by Go (`generate.go`) and
  written to a fixed location by the server, *not* by model tool calls. The
  tool-call loop, native-format recovery, and preflight capability checks of the
  async `openai-compatible` driver ([[open-provider-support]]) are explicitly out
  of scope here.
- Adding inline agents to the `Manager` driver map or the async run pipeline
  (run rows, `ProgressEvent` streaming, kill/timeout supervision). Inline calls
  remain synchronous, in-request.
- Provider **failover** for inline calls ([[switch-provider]]) — see Open
  Questions.
- New provider *types* beyond what app config already validates
  (`openai-compatible`); no per-vendor inline driver.
- Changing how Provider records are stored, validated, discovered, or surfaced
  in the UI — that is [[provider-model-for-agents]], already done.
- A UI for choosing the inline provider; selection is config-only in v1.

## Detailed Requirements

### Architecture-Breaking Requirements

Reviewed against `lifecycle/architecture/architecture-summary.md`, the promoted
architecture ([[modular-monolith]]), tech stack ([[go-vue]]), the ADRs, and the
standards. **No architecture-breaking requirement is introduced.** Each standing
constraint:

1. **Single self-contained binary.** The completer interface, the dispatcher,
   and the OpenAI-compatible inline client use only the Go standard library
   (`net/http`, `encoding/json`) — the same building blocks as the async
   `openai-compatible` driver. No new dependency, no cgo, no external datastore.
   The one *reduction* in coupling: the Claude CLI dependency, previously
   implicit and mandatory, becomes an explicitly-selected default that operators
   can replace with an in-process HTTP provider. → **Satisfied.**
2. **Local filesystem is the source of truth ([[index-is-a-cache]]).** Inline
   agent bindings live in `lifecycle/config.yaml`; Provider records live in
   `~/.kaos-control/config.yaml`. Disk stays authoritative; nothing here touches
   the index or makes it authoritative. → **Satisfied / not applicable.**
3. **Agents execute mediated tool calls ([[adr-0006-mediated-agent-driver-permission-model]],
   [[filesystem-sandboxing]]).** The inline path performs **no** model-driven
   tool execution — it returns a single completion string that server-side Go
   code parses and writes to the agent's fixed `allowed_write_paths`. ADR-0006
   mediation governs model-issued Write/Edit/Bash calls, of which the inline path
   issues none, so the trust model is not weakened. **Explicit constraint for the
   future:** if the inline abstraction is ever extended to let the model issue
   tool calls, those calls MUST route through the mediated driver path
   (ADR-0006); this requirement deliberately keeps the inline completer
   tool-free so no such gap is opened. → **Satisfied (with recorded constraint).**
4. **Secrets hygiene ([[secrets-handling]]).** The OpenAI-compatible completer
   sends `api_key` as `Authorization: Bearer` and reuses the existing masking
   applied to Provider records; the key never enters a prompt, a log line, or an
   API response. → **Satisfied.**
5. **Direct-served, no trusted proxy hop ([[adr-0001-no-header-based-client-ip-trust]]).**
   Unchanged: the server calls the provider endpoint directly; the browser never
   contacts a provider or handles a key. → **Satisfied / not applicable.**

**Conclusion:** No conflict with the recorded architecture, stack, or
standards; no new ADR is required. The change is a portability improvement fully
aligned with the [[provider-model-for-agents]] Provider spine.

### Functional Requirements

#### FR-1: Completer interface

- `internal/ideachat/` defines an interface abstracting a single LLM completion,
  with the same contract as today's `CallLLM`:
  ```go
  type Completer interface {
      Complete(ctx context.Context, cfg ModelConfig, messages []LLMMessage) (string, error)
  }
  ```
- The package-level `CallLLM` variable is retained (tests replace it with a
  deterministic fake — see `generate_test.go`), but its production implementation
  becomes a **dispatcher** that selects a `Completer` and delegates. Reassigning
  `CallLLM` in tests continues to short-circuit the dispatcher, so existing
  fakes keep working unchanged (**NFR-2**).

#### FR-2: Claude CLI completer is the default

- The current `callLLMImpl` (shelling out to `claude … -p … --model …`) is
  refactored **verbatim in behaviour** into a `Completer` implementation (e.g.
  `claudeCLICompleter`).
- It is selected when the resolving agent declares **no `provider`** (the status
  quo for `idea-capture`, `idea-generate`, `defect-generate`, `doc-generate`),
  or names the explicit CLI sentinel provider (see Open Question 3). Its prompt
  construction (`buildPrompt`) and CLI flags are unchanged, so existing config
  produces byte-identical requests.

#### FR-3: OpenAI-compatible inline completer

- A new `Completer` speaks the OpenAI `/v1/chat/completions` wire format against
  an app-level `config.ProviderConfig` (`base_url`, `api_key`, `extra_headers`).
- The request body carries `model` (from `ModelConfig.Model`), a `messages`
  array (FR-5), and `max_tokens` when `ModelConfig.MaxTokens > 0`. It does
  **not** send a `tools` array (this path is deliberately tool-free — Non-goals).
- `Content-Type: application/json` is set; `Authorization: Bearer <api_key>` is
  set only when the provider has a non-empty key; `extra_headers` are applied.
- The completer returns the final assistant message text (the concatenated
  content of the single completion) as a string, matching the CLI completer's
  return contract. Non-streaming is sufficient for v1 (see Open Question 1).

#### FR-4: Provider selection from `{provider, model}` config

- The inline dispatch resolves the completer from the agent's config:
  - `AgentConfig.Provider` **empty** → Claude CLI completer (FR-2).
  - `AgentConfig.Provider` **set** → look up the named `ProviderConfig` in app
    config and select the completer keyed by that provider's `driver`
    (`openai-compatible` → FR-3).
- `resolveIdeaCaptureConfig` (`internal/http/idea_chat.go`) is extended to carry
  the resolved provider identity (name/driver + `base_url`/`api_key`/
  `extra_headers`) alongside the existing `{Model, SystemPrompt}` so the
  dispatcher can pick and configure the completer. `ModelConfig` is extended as
  needed to carry this without leaking secrets to any serialised surface.
- The four inline template keys (`idea-capture`, `idea-generate`,
  `defect-generate`, `doc-generate`) all resolve through the same mechanism.

#### FR-5: Message mapping

- **CLI completer:** unchanged — `buildPrompt` folds system + history into the
  single `-p` prompt, preserving `Human:` / `Assistant:` turn markers.
- **OpenAI-compatible completer:** `ModelConfig.SystemPrompt` (when non-empty)
  becomes a `system` message; each `LLMMessage` maps to a `messages` entry with
  its `role` (`user` / `assistant`) preserved in order.

#### FR-6: Config validation

- `config.Validate` (project) rejects an `inline` agent whose `provider` names a
  provider **not** present in app config, with a message naming the offending
  agent — consistent with the async-path rule in [[provider-model-for-agents]].
- An `inline` agent with a `provider` set must have a non-empty `model`.
- An `inline` agent with **no** `provider` validates exactly as today (CLI
  default); no `model` requirement change and no new mandatory field.

#### FR-7: Failure behaviour

- A missing `claude` binary (CLI completer) surfaces the existing clear error,
  unchanged.
- The OpenAI-compatible completer surfaces non-2xx responses, malformed JSON,
  and unreachable/hung endpoints as a returned `error` (bounded by the request
  `context`), never a panic and never an unbounded retry. The error propagates
  to the calling generate/converse endpoint as it does today.

#### FR-8: No behavioural change for existing agents

- With the shipped `lifecycle/config.yaml` (inline agents declaring `model:
  claude-sonnet-4-6` and **no** `provider`), every inline agent continues to use
  the Claude CLI completer and produces the same requests and artifact output as
  before this change.

### Non-Functional Requirements

#### NFR-1: Secret hygiene ([[secrets-handling]])

- `api_key` / `extra_headers` credential material MUST NOT appear in prompt text,
  conversation history, generated artifacts, log output, or any API response
  that returns agent/provider config. Reuse the masking already applied to
  Provider records.

#### NFR-2: No regression

- All existing `internal/ideachat` tests pass unchanged, including those that
  reassign `CallLLM`. The dispatcher, CLI completer, and OpenAI completer are
  independently unit-testable.

#### NFR-3: Single-binary / stdlib only

- No new third-party dependency, no cgo. The OpenAI completer uses `net/http` +
  `encoding/json`, sharing wire conventions with the async `openai-compatible`
  driver (see Open Question 2 on extracting a shared client).

#### NFR-4: Backward compatibility

- Existing app and project config load and validate without edits. The `inline`
  driver marker keeps working; `provider` on an inline agent is purely additive.

#### NFR-5: Offline capability

- An inline agent pointed at a **local** provider (`http://localhost:11434`
  Ollama `/v1`, or a local `llama-server`) performs conversational capture and
  generation with **zero** internet connectivity — a capability the CLI-only
  path did not offer. (Consistent with [[local-model-operability]]; offline is
  not a *required* baseline per the architecture summary, so enabling it is a
  gain, not a constraint change.)

## Acceptance Criteria

- [ ] `internal/ideachat` defines a `Completer` interface; production `CallLLM`
      dispatches to a selected `Completer`, and reassigning `CallLLM` in a test
      still bypasses the dispatcher (existing fakes unchanged). ([[open-provider-support]])
- [ ] The Claude CLI implementation is a `Completer` selected when the agent has
      no `provider`; an inline agent with the shipped config produces the same
      `claude … -p … --model …` invocation as before. ([[provider-model-for-agents]])
- [ ] An inline agent declaring `provider: <name>` (driver `openai-compatible`)
      and a `model` routes to the OpenAI-compatible completer, which POSTs to
      `<base_url>/v1/chat/completions` with `model`, mapped `messages`, and
      `max_tokens` when set, and returns the assistant text.
- [ ] `Authorization: Bearer` is present iff the provider has an `api_key`;
      `extra_headers` are applied; the OpenAI path sends **no** `tools` array.
- [ ] System prompt maps to a `system` message and history `LLMMessage` roles
      (`user`/`assistant`) are preserved in order on the OpenAI path.
- [ ] Config validation rejects an inline agent whose `provider` is not a
      registered provider, and an inline agent with `provider` set but empty
      `model`, each error naming the agent; an inline agent with no `provider`
      validates as today.
- [ ] All four inline consumers (`idea-capture` conversational, `idea-generate`,
      `defect-generate`, `doc-generate`) work through both completers.
- [ ] A non-2xx / malformed / unreachable OpenAI provider surfaces as a returned
      error (bounded by context), never a panic or unbounded retry; a missing
      `claude` binary error is unchanged.
- [ ] No `api_key` / credential material appears in prompts, artifacts, logs, or
      any config-returning API response. ([[secrets-handling]])
- [ ] No new third-party dependency or cgo is introduced; existing
      `internal/ideachat` tests pass. ([[modular-monolith]], [[go-vue]])
- [ ] An inline agent pointed at a local provider completes capture/generation
      offline. ([[local-model-operability]])

## Resolved Questions

1. **Streaming vs. blocking for conversational capture.** The current CLI path
   is blocking (`exec … .Output()`) and the endpoints return a full string, so a
   non-streaming OpenAI completer is sufficient for v1. Should the conversational
   `idea-capture` flow stream tokens for responsiveness, or is blocking
   acceptable until a later UX pass? *Recommendation:* blocking in v1, matching
   current behaviour.

> Proceed with recommendation

2. **Shared HTTP client with the async driver.** The async `openai-compatible`
   driver already has a `/v1/chat/completions` client (with tool-calling,
   streaming, preflight). Should the inline completer **reuse** a shared,
   tool-free subset of that client to avoid a second implementation, or stay a
   small standalone client to avoid dragging in the agent-loop machinery?
   *Recommendation:* extract a minimal shared request builder if it is clean;
   otherwise keep the inline client small and separate.

> Proceed with recommendation

3. **Explicit CLI sentinel vs. empty provider.** Is "no `provider`" a sufficient
   signal for the Claude CLI default, or should there be an explicit provider
   entry (e.g. `driver: claude-cli`) so the CLI path is visible and selectable in
   config like any other? *Recommendation:* keep empty-`provider` = CLI default
   for zero-config compatibility, and consider a named CLI provider as a later
   nicety.

> Proceed with recommendation

4. **Failover participation.** Should inline calls honour provider failover
   ([[switch-provider]]) on a transient upstream error, or is a single attempt
   with a clear error sufficient for these short, interactive calls?
   *Recommendation:* single attempt in v1; revisit with [[switch-provider]].

> Switch provider will be a manual step at this time, will investigate automated switching later.

5. **Run recording for inline calls.** Inline calls are not async runs and today
   produce no `agent_runs` row or provider/driver record
   ([[agent-logging-provider-driver]]). Should inline conversational/generation
   calls gain lightweight provider/driver attribution for observability, or
   remain unrecorded? *Recommendation:* out of scope here; raise separately if
   attribution is wanted.

> Out of scope.
