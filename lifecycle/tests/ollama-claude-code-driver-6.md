---
title: "Env-Override Claude Code Driver — Test Suite"
type: test
status: draft
lineage: ollama-claude-code-driver
parent: lifecycle/test-plans/ollama-claude-code-driver-5-test.md
---

# Env-Override Claude Code Driver — Test Suite

Covers all milestones from the test plan at
`lifecycle/test-plans/ollama-claude-code-driver-5-test.md`.

## Test files

| File | Milestones |
|---|---|
| `internal/config/config_test.go` | T-1 |
| `internal/agent/agent_test.go` | T-2 (extension) |
| `internal/agent/claude_env_test.go` | T-2, T-3, T-4, T-5 |
| `internal/agent/manager_test.go` | T-6, T-7 |
| `internal/http/agents_test.go` | T-5 (API surface) |

## Scenarios covered

### T-1 — Config validation (`TestValidateClaudeEnvAgent`)

Table-driven subtests verify `validateProject` for the `claude-env` driver:

- Valid agent with `http://` and `https://` base_url, non-empty auth_token and
  model → `LoadProject` returns no error; all three fields are populated on the
  struct.
- Missing `base_url` → error names the agent and mentions "base_url".
- Malformed `base_url` (`not-a-url`, `ftp://x`) → error names the agent.
- Missing `auth_token` → error names the agent and mentions "auth_token".
- Missing `model` → error names the agent and mentions "model".
- Regression: `ollama`, `claude-mediated`, and `gemini` agents still validate
  without errors (NFR-2 guard).

### T-2 — `buildArgs` parity + result-event classification

`TestClaudeEnvDriver_BuildArgsParity` (in `claude_env_test.go`) verifies that
`ClaudeCodeDriver.buildArgs` (which `ClaudeEnvDriver` delegates to) produces the
expected flag set: `--permission-mode bypassPermissions`,
`--dangerously-skip-permissions`, `-p <prompt>`, `--output-format stream-json`,
`--verbose`, and `--model <model>` only when `run.Model` is non-empty.

`TestDriverEmitsResultEvent` (in `agent_test.go`) extended with
`{"claude-env", true}` to confirm the driver is classified as a stream-json
result-event emitter.

### T-3 — Environment injection + precedence (`TestClaudeEnvDriver_EnvInjection`)

Uses a shell stub (fake `claude` on PATH via `t.Setenv`) that prints
`PWD_STUB=$(pwd)` and `ANTHROPIC_*` env vars:

- `ANTHROPIC_BASE_URL` equals `run.BaseURL` in the child process.
- `ANTHROPIC_AUTH_TOKEN` equals `run.AuthToken` in the child process.
- When parent exports `ANTHROPIC_BASE_URL=inherited` / `ANTHROPIC_AUTH_TOKEN=inherited`
  via `t.Setenv`, the child sees the configured values (last-wins Go exec behaviour).
- `cmd.Dir` is set to `run.ProjectRoot` (verified via the `PWD_STUB=` line,
  with symlink resolution for macOS).

### T-4 — Streaming, TTFT, log file, kill/Wait parity

`TestClaudeEnvDriver_StreamingAndTTFT`: stub emits an assistant content event
then a terminal result event. Asserts events arrive in order with parsed `Event`
maps, `OnTTFT` is called once with a non-negative value, and the log file
contains the run header, both event lines, and the `# finished=` footer.

`TestClaudeEnvDriver_KillAndWait`: stub sleeps indefinitely; `Kill()` terminates
it and `Wait()` returns a non-nil error within 5 seconds.

`TestClaudeEnvDriver_CleanExitReturnsNilWait`: stub exits cleanly; `Wait()`
returns `nil`.

### T-5 — Secret hygiene

`TestClaudeEnvDriver_SecretHygiene` (agent-level): after a clean-exit stub run
with a known token, asserts:
- No `ProgressEvent.Raw` contains the token literal.
- `StderrTail()` does not contain the token.
- The on-disk log file does not contain the token, either bare or as
  `ANTHROPIC_AUTH_TOKEN=<token>`.

`TestHandleListAgents_ClaudeEnvSecretHygiene` (API surface, `agents_test.go`):
calls `handleListAgents` for a project whose agent.Manager holds a claude-env
agent. Asserts:
- HTTP 200 response includes `driver`, `model`, and `base_url` fields.
- The raw response body contains no occurrence of the token literal.
- The raw response body contains no `auth_token` field name.

### T-6 — Driver-map wiring + concurrency (`manager_test.go`)

`TestManager_DriverMapComplete`: after `agent.New(...)`, all eight drivers
(`claude-code-cli`, `claude-mediated`, `claude-env`, `codex-cli`, `ollama`,
`gemini`, `gemini-cli`, `shell-stub`) are present in `m.drivers` (T-6, T-7
regression guard).

`TestManager_ClaudeEnvDriverWired`: `StartRun` with a claude-env agent config
does not return an "unknown driver" error (the driver is wired).

`TestManager_UnknownDriverReturnsError`: `StartRun` with driver
`truly-unknown-driver` returns an error containing "unknown driver".

`TestManager_SemaphoreErrBusy`: with `maxConcurrent=1` and a long-running
shell-stub run holding the semaphore, a second `StartRun` returns `ErrBusy`.

`TestManager_SemaphoreReleasedAfterRun`: after a quick-exit shell-stub run
completes (polled via `GetRun` status), a subsequent `StartRun` does not return
`ErrBusy`.

### T-7 — No-regression sweep

Running `go test ./internal/config/ ./internal/agent/ ./internal/http/ -short`
passes all pre-existing and new tests. `go vet ./...` and `staticcheck` are
clean for all new files.
