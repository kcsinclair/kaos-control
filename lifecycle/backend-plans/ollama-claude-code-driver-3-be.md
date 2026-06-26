---
title: "Env-Override Claude Code Driver — Backend Plan"
type: plan-backend
status: approved
lineage: ollama-claude-code-driver
parent: lifecycle/requirements/ollama-claude-code-driver-2.md
---

# Env-Override Claude Code Driver — Backend Plan

## Overview

Add a new agent driver, **`claude-env`**, that runs the standard `claude` CLI
agentic loop (tool use, file edits, `stream-json` progress, run logging) but
retargets it to an Anthropic-compatible endpoint (Ollama's compatibility shim,
a self-hosted gateway, etc.) by injecting `ANTHROPIC_BASE_URL` and
`ANTHROPIC_AUTH_TOKEN` into the subprocess environment, plus `--model`.

The driver is deliberately thin: it reuses `ClaudeCodeDriver.buildArgs` (so the
CLI runs in `bypassPermissions` mode exactly as `claude-code-cli`) and the
shared `startCommandProcess` launcher, differing only in that it sets `cmd.Env`
before launch. The existing `claude-mediated` driver already demonstrates the
env-injection pattern (`cmd.Env = append(os.Environ(), "KC_HOOK_SECRET="+secret)`
at `internal/agent/claude_mediated.go:70`).

This plan covers config schema + validation, the driver itself, runner wiring,
secret hygiene, and unit tests. It does **not** add a frontend (resolved
question 4: no UI for v1 — see [[ollama-claude-code-driver]] frontend plan) and
does **not** register or manage Ollama instances (that is
[[ollama-agent-support]]'s job; this driver only needs a base URL, token, and
model tag).

Cross-references: [[ollama-claude-code-driver]] frontend plan (verification
only), [[ollama-claude-code-driver]] test plan (integration + secret-hygiene
tests).

---

## Milestone BE-1 — Config schema: `base_url` + `auth_token` fields

**Description.** Extend `config.AgentConfig` (and its YAML decode shim
`agentConfigRaw`) with two new fields used only when `driver: claude-env`:
`base_url` and `auth_token`. These mirror the naming already used by
`OllamaInstance.base_url` / `api_key`.

**Files to change.**
- `internal/config/config.go`
  - Add `BaseURL string` (`yaml:"base_url,omitempty"`) and
    `AuthToken string` (`yaml:"auth_token,omitempty"`) to `AgentConfig` (the
    struct near line 388).
  - Add the same two fields to `agentConfigRaw` (near line 429) and copy them
    in `UnmarshalYAML` (near line 460) so the custom unmarshaler does not drop
    them.

**Acceptance criteria.**
- A `lifecycle/config.yaml` agent block with `driver: claude-env`, `base_url:`,
  `auth_token:`, and `model:` round-trips through `LoadProject` with all four
  values populated on the resulting `AgentConfig`.
- Existing agents (no `base_url`/`auth_token`) load unchanged; the new fields
  default to empty strings.
- `go build ./...` and `go vet ./...` pass.

---

## Milestone BE-2 — Config validation for `claude-env`

**Description.** Add a `driver == "claude-env"` branch to `validateProject`'s
per-agent loop (`internal/config/config.go`, near line 680, alongside the
existing `ollama` / `claude-mediated` / `gemini` branches). It must reject the
agent when `base_url` is empty, when `base_url` is not a valid `http`/`https`
URL, when `auth_token` is empty, or when `model` is empty — each error naming
the offending agent. Reuse `net/url.ParseRequestURI` + scheme check exactly as
the `ollama_instances` validation does (`validateApp`, line 187).

**Files to change.**
- `internal/config/config.go` — new validation branch in `validateProject`.

**Acceptance criteria.**
- A `claude-env` agent missing `base_url` → error containing the agent name and
  "base_url".
- A `claude-env` agent with `base_url: "not-a-url"` (or `ftp://…`) → error
  naming the agent and indicating an invalid http/https URL.
- A `claude-env` agent missing `auth_token` → error naming the agent and
  "auth_token".
- A `claude-env` agent missing `model` → error naming the agent and "model".
- A fully-valid `claude-env` agent → no error.
- Validation for `ollama`, `claude-mediated`, and `gemini` agents is byte-for-
  byte unchanged (existing `config` tests stay green).

---

## Milestone BE-3 — Carry `base_url` / `auth_token` onto the `Run`

**Description.** The `Run` struct (`internal/agent/agent.go`, near line 73) is
the only thing a driver receives. Add two fields so the driver can read the
configured endpoint and token, and populate them in `Manager.StartRun` where
the `Run` is constructed (near line 560).

**Files to change.**
- `internal/agent/agent.go`
  - Add `BaseURL string` and `AuthToken string` to `Run` (group them with the
    other driver-specific fields, near the Ollama fields). Add a comment that
    `AuthToken` is a secret and must never be logged or echoed.
  - In `StartRun`, set `BaseURL: ag.BaseURL, AuthToken: ag.AuthToken` on the
    `Run` literal.

**Acceptance criteria.**
- A run started for a `claude-env` agent produces a `Run` whose `BaseURL` and
  `AuthToken` equal the agent config values.
- No other driver reads these fields, so behaviour for `claude-code-cli`,
  `claude-mediated`, `ollama`, `gemini`, and `codex-cli` is unchanged.

---

## Milestone BE-4 — The `claude-env` driver

**Description.** Add `ClaudeEnvDriver` in a new file
`internal/agent/claude_env.go`. Its `Start`:
1. Builds args by reusing the `claude-code-cli` invocation — instantiate a
   `ClaudeCodeDriver` and call its `buildArgs(run)` (bypassPermissions +
   `stream-json` + `--model` when `run.Model != ""`). This guarantees argument
   parity and means FR-3 (`--model <model>`) is satisfied for free.
2. Creates the command: `exec.CommandContext(ctx, "claude", args...)`, sets
   `cmd.Dir = run.ProjectRoot`.
3. **Injects env (FR-2):**
   `cmd.Env = append(os.Environ(), "ANTHROPIC_BASE_URL="+run.BaseURL,
   "ANTHROPIC_AUTH_TOKEN="+run.AuthToken)`.
   Because Go's `exec` uses the *last* occurrence of a duplicated key, appending
   after `os.Environ()` makes the injected values take precedence over any
   inherited `ANTHROPIC_BASE_URL` / `ANTHROPIC_AUTH_TOKEN` (FR-2 precedence
   requirement). No other `ANTHROPIC_*` var is set or unset (FR-2).
4. Delegates to the shared `startCommandProcess(ctx, cmd, run, args, "claude")`
   so progress streaming, TTFT, the per-run log file, kill, and `Wait()` are all
   identical to `claude-code-cli` (FR-1, FR-7).

Because `startCommandProcess` writes only `args`, `model`, and identity into the
log header (not `cmd.Env`), the injected token never reaches the log file — this
is relied on by NFR-1 and verified in BE-6.

**Files to change.**
- `internal/agent/claude_env.go` (new).

**Acceptance criteria.**
- `(&ClaudeEnvDriver{}).Start(ctx, run)` spawns `claude` with the same arg
  vector `ClaudeCodeDriver` would produce for the same `run`.
- The spawned process's environment contains exactly one
  `ANTHROPIC_BASE_URL=<run.BaseURL>` and one
  `ANTHROPIC_AUTH_TOKEN=<run.AuthToken>`, and these override any value present
  in the parent environment (asserted with a fake/echo binary in the test plan).
- No `ANTHROPIC_*` variable other than `BASE_URL` and `AUTH_TOKEN` is added.
- The returned `Process` supports `Progress()`, `Wait()`, `Kill()`, and
  `StderrTail()` with the same semantics as `cliProcess`.

---

## Milestone BE-5 — Runner wiring: driver map + result-event classification

**Description.** Register the driver and classify it as a stream-json
(result-event-emitting) driver so TTFT recording and truncated-stream detection
work (FR-7, NFR-2).

**Files to change.**
- `internal/agent/agent.go`
  - In `New`, add `"claude-env": &ClaudeEnvDriver{}` to the `m.drivers` map
    (near line 442).
  - In `driverEmitsResultEvent` (near line 1558), add `"claude-env"` to the
    `case "claude-code-cli", "claude-mediated":` list so the driver is treated
    as result-event-emitting. This wires `run.OnTTFT` (StartRun line 583) and
    enables the truncated-stream check (supervise line 848).

**Supervisor note (no change required).** `supervise`'s `switch run.Driver`
(line 755) has explicit cases only for `claude-code-cli` (bypass precheck) and
`claude-mediated` (mediated precheck); every other driver falls through to the
`default` drain branch. `claude-env` therefore drains and forwards events like
`ollama`/`gemini`/`codex-cli` — satisfying FR-6 ("no call site outside the
driver map and config validation special-cases it"). The `broadcast` closure
still runs in the default branch, so `resultEventSeen` is tracked and the
truncated-stream guard at line 848 functions correctly. The bypass-mode precheck
(a `claude-code-cli` safety) is intentionally skipped for v1; see "Permission
model" below.

**Acceptance criteria.**
- An agent with `driver: claude-env` resolves to `ClaudeEnvDriver` via the
  driver map; an unknown driver still returns `unknown driver` from `StartRun`
  (line 537), unchanged.
- `driverEmitsResultEvent("claude-env") == true`;
  `driverEmitsResultEvent` for every other existing driver is unchanged.
- A `claude-env` run records `ttft_ms` on first content token and is flagged
  `failed`/`truncated_stream` if it exits 0 without a terminal `result` event.
- `claude-env` runs acquire and release the global `max_concurrent_agents`
  semaphore via the existing `StartRun` path (no driver-specific bypass).

---

## Milestone BE-6 — Secret hygiene (NFR-1)

**Description.** Guarantee `auth_token` never leaks. Three surfaces:

1. **Run log / stderr / `ProgressEvent.Raw`.** Already safe by construction: the
   token is injected via `cmd.Env`, never via `args`, `PromptText`, or `Model`,
   and `startCommandProcess` does not print the env. Add a regression test
   (test plan) asserting the token string is absent from the written log file.
2. **API responses.** `handleListAgents` (`internal/http/agents.go:26`) builds
   an `agentSummary` that today exposes no `base_url`/`auth_token`. Keep
   `auth_token` **out** of every API response. Optionally surface `base_url`
   (non-secret) on the summary for observability. If a future need surfaces the
   token, mask it to `"***"` exactly like `maskedInstances`
   (`internal/http/ollama.go:20`). For v1: add `base_url` to `agentSummary`
   (plain) and do **not** add `auth_token`.
3. **No new secret store (NFR-4).** Token stays in the on-disk project config,
   same trust boundary as `ollama_instances.api_key`. No code change needed.

**Files to change.**
- `internal/http/agents.go` — add `BaseURL string json:"base_url,omitempty"` to
  `agentSummary` and populate it; do not add an `auth_token` field.

**Acceptance criteria.**
- `GET /api/p/:project/agents` for a project containing a `claude-env` agent
  returns the agent with `driver`, `model`, and `base_url`, and the response
  body contains **no** occurrence of the configured token string.
- The per-run log file for a `claude-env` run contains neither
  `ANTHROPIC_AUTH_TOKEN` nor the token value.
- `ProgressEvent.Raw` / stderr tail for a run never contain the token value.

---

## Milestone BE-7 (optional / stretch) — Mediated variant

**Description.** Resolved question 3 notes that using `claude-mediated`-style
hook-gated permissions with a local model "is a good option" if feasible. This
is explicitly a *later enhancement*, not a v1 acceptance criterion (all v1 ACs
are bypass-oriented). If pursued, the cleanest shape is an env-injecting wrapper
around `ClaudeHooksDriver` (which already sets `cmd.Env`) so both the hook
secret and the `ANTHROPIC_*` overrides are present, plus a supervisor case so
`claude-env-mediated` runs the mediated precheck. Defer unless the bypass path
proves insufficient.

**Acceptance criteria (only if implemented).**
- A mediated env-override run gets both `KC_HOOK_SECRET` and the `ANTHROPIC_*`
  overrides in its environment and passes the mediated precheck.
- v1 `claude-env` behaviour is unaffected.

---

## Permission model (v1 decision)

v1 ships the **bypass** path: `claude-env` reuses `ClaudeCodeDriver.buildArgs`
(`--permission-mode bypassPermissions --dangerously-skip-permissions`), matching
the idea's `claude --model …` example. The supervisor does not run the
bypass-mode precheck for `claude-env` (it is not in the `claude-code-cli` case),
which keeps FR-6 satisfied and avoids special-casing. A mediated variant is
captured as BE-7.

## Out of scope (per requirement non-goals)

- No new inference client / wire-protocol implementation.
- No Ollama instance/model management ([[ollama-agent-support]]).
- No guarantee of tool-use fidelity for any particular community model.
- No frontend driver picker (resolved question 4).
