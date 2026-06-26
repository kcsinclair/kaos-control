---
title: Env-Override Claude Code Driver (Ollama / Anthropic-compatible endpoints)
type: requirement
status: planning
lineage: ollama-claude-code-driver
priority: high
parent: lifecycle/ideas/ollama-claude-code-driver.md
labels:
    - agent
    - agent-runner
    - driver
    - ollama
    - integration
    - portability
release: KC-Release4
assignees:
    - role: product-owner
      who: agent
---

# Env-Override Claude Code Driver

## Problem

kaos-control's richest agent execution path is the Claude Code CLI driver
(`claude-code-cli` and `claude-mediated`): it gives agents the full Claude Code
agentic loop — tool use, file edits, streamed `stream-json` progress, run
logging, and permission mediation. Today that path is hard-wired to Anthropic's
hosted API: the `claude` subprocess inherits the operator's ambient
credentials and talks to `api.anthropic.com`.

The existing `ollama` driver (see [[ollama-agent-support]]) reaches local
models, but it is a *native* `/api/chat` / `/api/generate` client: a single
batch prompt/response with **no tool use, no file edits, and no agentic loop**.
So users who want a local or self-hosted model must give up everything that
makes the Claude Code drivers useful.

The Claude Code CLI already supports retargeting via two environment
variables — `ANTHROPIC_BASE_URL` and `ANTHROPIC_AUTH_TOKEN` — letting it speak
to any Anthropic-compatible endpoint, including Ollama's compatibility shim
(`http://host:11434`, token literal `"ollama"`) or a self-hosted gateway. There
is currently no supported, config-driven way to inject those variables per
agent. As a result there is no path to run the *full Claude Code agent* against
a local / air-gapped / cost-free model.

## Goals / Non-goals

### Goals

- Add a new agent driver that runs the standard `claude` CLI agentic loop but
  retargets it to an Anthropic-compatible endpoint via injected environment
  variables (`ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`, and `--model`).
- Make endpoint, auth token, and model configurable per agent in
  `lifecycle/config.yaml`, with the token resolvable from config without being
  echoed back in API responses or run logs.
- Preserve every behaviour that the existing Claude Code drivers provide:
  `stream-json` progress events, per-run log files, TTFT measurement, run
  cancellation/kill, and the global `max_concurrent_agents` semaphore.
- Leave the existing `claude-code-cli`, `claude-mediated`, `ollama`, `gemini`,
  and `codex-cli` drivers behaviourally unchanged (no regression).

### Non-goals

- Building a new model-inference client. This driver shells out to the existing
  `claude` binary; it does not re-implement the Anthropic or Ollama wire
  protocol (that is the `ollama` driver's job).
- Managing Ollama models (pull/delete/list) or registering Ollama *instances*
  through this driver — instance/model management is covered by
  [[ollama-agent-support]]. This driver only needs a base URL, token, and model
  tag.
- Guaranteeing correctness or tool-use fidelity of any particular community
  model. The driver's contract is "inject the env and run `claude`"; whether a
  given local model honours tool calls is the model's concern, not this
  requirement's.
- A frontend instance-picker UI. v1 is config-file driven; UI surfacing is an
  open question (see below).

## Detailed Requirements

### Functional

#### FR-1: New driver registration

- A new driver is registered in the agent `Manager`'s driver map under a stable
  name. Default name: **`claude-env`**. (Final name is an open question; the
  implementation MUST use a single agreed string and reject unknown drivers as
  today.)
- The driver reuses the existing Claude Code CLI invocation and the shared
  `startCommandProcess` launcher — i.e. it spawns the same `claude` binary,
  pipes stdout/stderr, parses `stream-json`, and writes the run log exactly as
  `claude-code-cli` does.

#### FR-2: Environment injection

- On `Start`, the driver sets the subprocess environment to the inherited
  parent environment **plus** the following overrides:
  - `ANTHROPIC_BASE_URL=<configured base_url>`
  - `ANTHROPIC_AUTH_TOKEN=<configured auth_token>`
- Injected overrides MUST take precedence over any same-named variable already
  present in the parent environment.
- No other `ANTHROPIC_*` variables are set or unset by the driver.

#### FR-3: Model selection

- The agent's `model` config value is passed through to the `claude` CLI as
  `--model <model>` (reusing existing `buildArgs` behaviour).
- `model` is **required** for this driver; config validation fails fast with a
  clear message if it is empty.

#### FR-4: Configuration fields

- `AgentConfig` gains two fields used only when `driver: claude-env`:
  - `base_url` (string, required) — the Anthropic-compatible endpoint, e.g.
    `http://localhost:11434` or `http://leia.packsin.com:11434`.
  - `auth_token` (string, required) — the bearer/auth token, e.g. the literal
    `"ollama"` for the Ollama shim.
- These reuse, or are consistent with, existing config field naming
  (`OllamaInstance.base_url` already exists). If an `auth_token` value would be
  shared across agents, it MAY be referenced from app-level config rather than
  duplicated inline (resolution mechanism is an open question; inline value is
  the v1 baseline).

#### FR-5: Config validation

- `config.Validate` rejects a `claude-env` agent when `base_url` is empty, when
  `base_url` is not a valid `http`/`https` URL, when `auth_token` is empty, or
  when `model` is empty. Each failure names the offending agent.
- Existing per-driver validation for `ollama`, `claude-mediated`, and `gemini`
  is unchanged.

#### FR-6: Driver selection at runtime

- The agent runner selects this driver purely on the `driver` field, identical
  to how `ollama`/`gemini`/`codex-cli` are selected today. No call site outside
  the driver map and config validation needs to special-case it.

#### FR-7: Progress, logging, cancellation parity

- Runs using this driver MUST emit the same `ProgressEvent` stream as
  `claude-code-cli`, record TTFT on first content token, write a per-run log
  file when `LogPath` is set, support `Process.Kill()` (context cancel), and
  return the subprocess exit status from `Process.Wait()`.

### Non-functional

#### NFR-1: Secret hygiene

- `auth_token` MUST NOT appear in run log files, `ProgressEvent.Raw`, stderr
  tails, or any HTTP/API response that exposes agent configuration. Where agent
  config is returned over the API, the token is masked (e.g. `"***"`), matching
  the `ollama_instances` `api_key` masking convention in [[ollama-agent-support]].
- The run-log header (which today prints `args` and model) MUST NOT include the
  injected environment.

#### NFR-2: No regression

- All existing driver unit tests continue to pass. Adding `claude-env` to the
  driver map does not change `driverEmitsResultEvent` behaviour for other
  drivers; `claude-env` is treated as a `stream-json` driver (result-event
  emitting) like `claude-code-cli`.

#### NFR-3: Failure behaviour

- If the configured endpoint is unreachable, the driver surfaces the `claude`
  CLI's own non-zero exit and stderr tail through the normal run-failure path —
  it does not hang beyond the agent's configured `timeout_minutes`.

#### NFR-4: Trust boundary

- `auth_token` is stored in the project/app config file on disk, the same trust
  boundary as existing credentials (`ollama_instances.api_key`,
  `auth` config). No new secret-storage mechanism is required for v1.

## Acceptance Criteria

- [ ] An agent declared with `driver: claude-env`, a valid `base_url`,
      `auth_token`, and `model` loads and validates without error.
- [ ] Config validation rejects a `claude-env` agent that is missing
      `base_url`, has a malformed `base_url`, is missing `auth_token`, or is
      missing `model`, with a message naming the agent.
- [ ] When the driver starts, the spawned `claude` subprocess environment
      contains `ANTHROPIC_BASE_URL` and `ANTHROPIC_AUTH_TOKEN` set to the
      configured values, overriding any inherited values.
- [ ] The configured `model` is passed to the CLI as `--model <model>`.
- [ ] A run using this driver emits `stream-json` `ProgressEvent`s, records
      TTFT, and writes a per-run log file identical in shape to
      `claude-code-cli`.
- [ ] `Process.Kill()` cancels an in-flight run; `Process.Wait()` returns the
      subprocess exit status.
- [ ] `auth_token` does not appear in the run log, stderr tail, or any API
      response that returns agent config (it is masked).
- [ ] Existing `claude-code-cli`, `claude-mediated`, `ollama`, `gemini`, and
      `codex-cli` agents are unaffected (existing tests green).
- [ ] Runs on this driver respect the global `max_concurrent_agents` semaphore.
- [ ] [[ollama-claude-code-driver]] lineage artifacts (this requirement, its
      backend/frontend/test plans) link via `parent:` correctly.
- [ ] Related work [[ollama-agent-support]] is referenced and not duplicated:
      this driver does not register or list Ollama instances.

## Resolved Questions

1. **Driver name.** `claude-env` (descriptive of mechanism) vs `claude-ollama`
   (descriptive of primary use case). The idea text suggested either
   `driver: ollama` (already taken) or `driver: claude-env`. Recommend
   `claude-env`. Confirm the canonical string.

> claude-env

2. **Shared token resolution.** Should `auth_token` be referenceable from
   app-level config (like Ollama instances are shared across projects per
   [[ollama-agent-support]] resolved question 2), to avoid duplicating the
   token across agents? Or is an inline per-agent value sufficient for v1?

> For Ollama the token is irrelevant, for other local AI it will be unique to the server instance.

3. **Permission model.** Should this driver mirror `claude-code-cli`
   (bypass-permissions) or `claude-mediated` (hook-gated tool calls)? The idea's
   example uses plain `claude --model …` (bypass). Recommend starting from the
   bypass path and treating a mediated variant as a later enhancement —
   confirm.

> If claude-mediated can be used with the local model that is a good option.

4. **Frontend exposure.** Is config-file-only acceptable for v1, or must the
   agent creation/edit UI offer `claude-env` as a driver type with
   base-url/token/model fields (mirroring the Ollama picker)?

> For this version no frontend required.

5. **Endpoint compatibility scope.** Do we commit to any Anthropic-compatible
   endpoint (LiteLLM, vLLM Anthropic shim, etc.) or scope v1 docs/testing to the
   Ollama shim only, with others "best effort, untested"?

> scope v1 docs/testing to the Ollama shim only, with others "best effort, untested"?
