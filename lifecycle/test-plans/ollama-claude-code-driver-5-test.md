---
title: "Env-Override Claude Code Driver — Test Plan"
type: plan-test
status: in-development
lineage: ollama-claude-code-driver
parent: lifecycle/requirements/ollama-claude-code-driver-2.md
---

# Env-Override Claude Code Driver — Test Plan

## Overview

Tests verifying the `claude-env` driver from [[ollama-claude-code-driver]]:
config validation, environment injection + precedence, `--model` pass-through,
stream-json/TTFT/log/kill parity with `claude-code-cli`, secret hygiene, and
no-regression for the other drivers.

The driver shells out to `claude`. To test env injection and process semantics
deterministically **without** a real `claude` binary or a live endpoint, the
driver tests follow the existing pattern in `internal/agent/codex_cli_test.go`
(`TestCodexCLIDriver_Start`, `TestCodexCLIDriver_DetachedChildHoldsPipes`):
swap the spawned binary for a small shell/echo stub that prints its environment
and emits canned `stream-json` lines on stdout. Pure-logic tests
(`buildArgs`, `driverEmitsResultEvent`, config `Validate`) need no subprocess.

Cross-references: [[ollama-claude-code-driver]] backend plan (BE-1..BE-6 are the
units under test), [[ollama-agent-support]] (regression target — its driver must
stay green and is **not** exercised by `claude-env`).

---

## Milestone T-1 — Config validation unit tests

**Description.** Table-driven tests for the BE-2 `claude-env` validation branch,
alongside the existing `config` tests (`internal/config/config_test.go`).

**Files to change.**
- `internal/config/config_test.go` (new test, e.g. `TestValidateClaudeEnvAgent`).

**Acceptance criteria (one sub-case each).**
- Valid `claude-env` agent (`base_url` http(s), non-empty `auth_token`, non-empty
  `model`) → `LoadProject`/`validateProject` returns no error and the fields are
  populated.
- Missing `base_url` → error contains the agent name and "base_url".
- Malformed `base_url` (`"not-a-url"`, and a non-http scheme such as
  `"ftp://x"`) → error names the agent and signals invalid http/https URL.
- Missing `auth_token` → error names the agent and "auth_token".
- Missing `model` → error names the agent and "model".
- Unchanged: a known-good `ollama`, `claude-mediated`, and `gemini` agent each
  still validate exactly as before (guards NFR-2).

---

## Milestone T-2 — `buildArgs` parity + result-event classification

**Description.** Confirm `claude-env` produces the same CLI args as
`claude-code-cli` and is classified as a stream-json driver.

**Files to change.**
- `internal/agent/claude_env_test.go` (new).
- Extend `internal/agent/agent_test.go`'s `TestDriverEmitsResultEvent`.

**Acceptance criteria.**
- For a given `Run`, `ClaudeEnvDriver`'s effective arg vector equals
  `(&ClaudeCodeDriver{}).buildArgs(run)` — includes `--permission-mode
  bypassPermissions`, `--dangerously-skip-permissions`, `-p <prompt>`,
  `--output-format stream-json`, `--verbose`.
- `--model <model>` is present when `run.Model != ""` and absent when empty
  (FR-3 mirror of `TestBuildArgs_ModelFlag`).
- `driverEmitsResultEvent("claude-env") == true`; values for `claude-code-cli`,
  `claude-mediated`, `ollama`, `gemini`, `codex-cli`, `shell-stub` unchanged.

---

## Milestone T-3 — Environment injection + precedence (FR-2)

**Description.** The core behaviour. Using a stub binary that echoes its
environment to stdout (à la `codex_cli_test.go`), start the driver and assert
the injected variables and their precedence.

**Files to change.**
- `internal/agent/claude_env_test.go`.

**Acceptance criteria.**
- The spawned process environment contains exactly one
  `ANTHROPIC_BASE_URL` equal to `run.BaseURL` and exactly one
  `ANTHROPIC_AUTH_TOKEN` equal to `run.AuthToken`.
- **Precedence:** when the parent process already exports
  `ANTHROPIC_BASE_URL=inherited` / `ANTHROPIC_AUTH_TOKEN=inherited` (set via
  `t.Setenv`), the child sees the *configured* values, not the inherited ones.
- No `ANTHROPIC_*` variable other than `BASE_URL` and `AUTH_TOKEN` is added or
  removed relative to the parent environment.
- `cmd.Dir` is set to `run.ProjectRoot`.

---

## Milestone T-4 — Streaming, TTFT, log file, kill/Wait parity (FR-7)

**Description.** Drive a stub that emits canned `stream-json` lines (an
`assistant` text event then a terminal `result` event) and assert process
behaviour matches `claude-code-cli`.

**Files to change.**
- `internal/agent/claude_env_test.go`.

**Acceptance criteria.**
- `Process.Progress()` yields the emitted events in order, with the `result`
  line parsed into `Event` (closes on exit).
- `run.OnTTFT` is invoked once with a non-negative millisecond value on the
  first assistant content token (`isFirstContentToken`).
- When `run.LogPath` is set, a log file is written with the standard header
  (`# agent=… model=… args=…`) plus the streamed lines and a `# finished=` footer.
- `Process.Kill()` terminates an in-flight stub run and `Process.Wait()` returns
  the resulting exit error; a clean stub exit returns `nil` from `Wait()`.

---

## Milestone T-5 — Secret hygiene (NFR-1)

**Description.** Assert the token never leaks to the log, stderr tail, progress
stream, or the agents API.

**Files to change.**
- `internal/agent/claude_env_test.go` (log/stderr/progress).
- `internal/http/agents_test.go` (API surface; create if absent, following the
  existing `internal/http` handler-test pattern).

**Acceptance criteria.**
- After a stub run with `auth_token: "s3cr3t-token"`, the on-disk log file
  contains neither the literal token nor `ANTHROPIC_AUTH_TOKEN=…` with the value.
- The token literal does not appear in any `ProgressEvent.Raw` nor in
  `StderrTail()`.
- `GET /api/p/:project/agents` for a project with a `claude-env` agent returns
  `driver`, `model`, and `base_url`, and the serialized response body contains
  **no** occurrence of the token literal and no `auth_token` field.

---

## Milestone T-6 — Driver-map wiring + concurrency (FR-1, FR-6, semaphore)

**Description.** Verify the runner selects the driver purely by name and honours
the global semaphore, without special-casing.

**Files to change.**
- `internal/agent/agent_test.go` (or a manager-level test file), reusing the
  existing `Manager` test scaffolding.

**Acceptance criteria.**
- A `Manager` built from a config containing a `claude-env` agent resolves it
  via the driver map; an agent referencing a truly unknown driver still yields
  the `unknown driver` error from `StartRun`.
- A `claude-env` run acquires the `max_concurrent_agents` semaphore; with the
  limit at its cap, an additional start returns `ErrBusy`, and the slot is
  released after the run completes (parity with existing semaphore tests).

---

## Milestone T-7 — No-regression sweep (NFR-2)

**Description.** Guard the four "existing drivers unaffected" acceptance
criteria by running the full existing suite plus an explicit assertion that the
driver map still contains every previously-registered driver.

**Files to change.**
- None beyond the above (this milestone is the green-suite gate).

**Acceptance criteria.**
- `go test ./... -short` passes, including all pre-existing `internal/agent` and
  `internal/config` tests.
- The `m.drivers` map after `New(...)` contains `claude-code-cli`,
  `claude-mediated`, `codex-cli`, `ollama`, `gemini`, `gemini-cli`,
  `shell-stub`, **and** `claude-env` — nothing removed or renamed.
- `make lint` (`go vet` + `staticcheck`) is clean for the new files.

---

## Optional — live round-trip (manual / tagged)

Mirroring `TestCodexCLIDriver_LiveRoundTrip`, an optional build-tagged or
env-gated test can run a real `claude` against a local Ollama shim
(`base_url: http://localhost:11434`, `auth_token: "ollama"`) to confirm an
end-to-end agentic run. Scoped to the Ollama shim only (resolved question 5:
other Anthropic-compatible endpoints are best-effort/untested). Not part of the
default CI suite; documented for manual verification.
