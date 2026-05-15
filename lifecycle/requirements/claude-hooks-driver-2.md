---
title: Mediated Claude Driver with Permission Hooks
type: requirement
status: planning
lineage: claude-hooks-driver
created: "2026-05-15T12:00:00+10:00"
priority: high
parent: lifecycle/ideas/claude-hooks-driver.md
labels:
    - agent
    - security
    - backend
release: KC-Release2
assignees:
    - role: product-owner
      who: agent
---

# Mediated Claude Driver with Permission Hooks

Parent: [[claude-hooks-driver]].

## Problem

The existing `ClaudeCodeDriver` invokes Claude Code with
`--dangerously-skip-permissions` and `--permission-mode bypassPermissions`.
During a run the agent is unconstrained — it can write anywhere, run
arbitrary shell commands, and make network calls. The `AllowedPaths`
allowlist is only enforced post-hoc when scoping the git commit.

This is acceptable for trusted single-user environments but provides no
real sandbox. Multi-user installs, hosted deployments, and agents
consuming less-trusted prompts need run-time enforcement of write paths
and command restrictions before tool calls execute, not after.

## Goals / Non-goals

### Goals

1. Provide an **alternate driver** (`claude-mediated`) that invokes
   Claude Code without bypass mode and mediates every tool call through
   Kaos Control's permission endpoint via Claude Code's hooks API.
2. Enforce `AllowedPaths` as a **hard run-time barrier** — writes
   outside allowed paths are denied before they execute.
3. Support per-agent **bash allowlist / denylist** configuration with
   sensible shipped defaults.
4. Enforce **lineage-scoped writes** — refuse writes outside the
   current lineage's paths even when `AllowedPaths` would permit them.
5. Provide a **dry-run / observe-only mode** that logs decisions
   without enforcing them, for operators to preview policy impact.
6. Surface permission decisions (allow/deny) in the **run timeline UI**
   alongside existing `agent.progress` events.
7. Produce a full **audit trail** of every tool-call decision with
   structured logging.

### Non-goals

- Replacing the existing `claude-code-cli` driver. It remains the
  default and is unchanged.
- Per-tool allowlisting via `--allowedTools` (Claude Code flag) — this
  requirement uses hooks, not flag-level filtering.
- Network egress filtering (e.g. blocking outbound HTTP from the agent
  process). May be addressed later.
- Automatic remediation of denied tool calls (retrying with corrected
  paths, etc.).
- Sandboxing at the OS level (containers, seccomp, etc.).

## Detailed Requirements

### Driver registration

- **FR1 — New driver type.** A new `ClaudeHooksDriver` is registered
  under the name `claude-mediated` in the driver map. It implements
  the same `Driver` interface as `ClaudeCodeDriver`.

- **FR2 — Invocation without bypass.** The new driver invokes `claude`
  **without** `--dangerously-skip-permissions` and **without**
  `--permission-mode bypassPermissions`. It passes
  `--settings <path>` pointing to a per-run generated settings file
  that wires the `PreToolUse` hook.

- **FR3 — Driver reuse.** The new driver reuses the existing
  `ClaudeCodeDriver` logic for stream-JSON parsing, progress
  reporting, cost tracking, and run lifecycle. Only the invocation
  arguments, precheck, and hook configuration differ.

### Hook helper

- **FR4 — Subcommand implementation.** A new `kaos-control hook-helper`
  subcommand of the main binary reads tool-call JSON from stdin, POSTs
  it to the local permission endpoint, prints the JSON response to
  stdout, and exits. No external scripts or shell wrappers.

- **FR5 — Per-run authentication.** Each agent run generates a
  cryptographically random secret (minimum 32 bytes, hex-encoded).
  The secret is passed to the subprocess via an environment variable
  (e.g. `KC_HOOK_SECRET`). The hook helper includes this secret in
  every request to the permission endpoint. The endpoint rejects
  requests with missing or invalid secrets.

- **FR6 — Per-run settings generation.** Before spawning `claude`, the
  driver writes a temporary `settings.json` containing the
  `PreToolUse` hook configuration pointing to `kaos-control hook-helper`.
  The file is cleaned up when the run completes (success or failure).

### Permission endpoint

- **FR7 — HTTP endpoint.** New route:
  `POST /api/agent/{run_id}/permission`. Request body is the
  tool-call JSON from Claude Code's hook contract. Response body:

  ```json
  {
    "decision": "allow" | "deny",
    "reason": "optional human-readable explanation"
  }
  ```

  The endpoint is authenticated by the per-run secret (FR5).

- **FR8 — Request validation.** The endpoint validates that the
  `run_id` corresponds to an active run, the secret matches, and the
  request body contains at minimum the tool name. Invalid requests
  return HTTP 403 (bad secret) or 400 (malformed body).

### Permission policy

- **FR9 — AllowedPaths enforcement.** For `Write`, `Edit`, and any
  other file-mutating tool calls, the policy checks the target path
  against the agent's `AllowedPaths` configuration. Paths outside the
  allowlist are denied.

- **FR10 — Lineage scope enforcement.** When a run is associated with
  a specific lineage, write operations are further restricted to paths
  relevant to that lineage. A write to a path that is within
  `AllowedPaths` but outside the lineage scope is denied with a
  specific reason.

- **FR11 — Bash command filtering.** For `Bash` tool calls, the policy
  evaluates the command string against:
  1. The agent's `bash_denylist` (if matched → deny).
  2. The agent's `bash_allowlist` (if configured and not matched → deny).
  Denylist is checked first; a command matching the denylist is denied
  even if it also matches the allowlist.

- **FR12 — Default denylist.** The binary ships with a built-in bash
  denylist applied to all `claude-mediated` runs unless overridden:
  - `rm -rf /` and variants targeting system paths
  - `curl ... | sh`, `wget ... | sh` (pipe-to-shell patterns)
  - `sudo` (any command prefixed with sudo)
  - Writes to `$HOME` outside the project root
  - `chmod 777`, `chown` on system paths

  The exact patterns are glob/regex-based and documented in
  `lifecycle/config.yaml` comments.

- **FR13 — Read operations pass through.** `Read`, `Glob`, `Grep`, and
  other read-only tool calls are allowed by default. The policy does
  not restrict what the agent can read.

### Denial behaviour

- **FR14 — Configurable denial action.** When a tool call is denied,
  the behaviour is configurable per-agent in `lifecycle/config.yaml`
  via a `on_denial` field:
  - `continue` (default): The deny response is returned to Claude,
    which skips the tool call and continues the run. The denial is
    logged to the job log.
  - `abort`: The run is terminated immediately on the first denial.

- **FR15 — No auto-commit on denial.** If any tool call was denied
  during a run (regardless of `on_denial` setting), the run must
  **not** auto-commit. The run completes (or aborts) with a
  `denied_tool_calls` flag. The agent queue is paused. The operator
  must manually review and resume.

- **FR16 — Queue pause on denial.** When a run finishes with one or
  more denials, the agent queue for that project is paused
  automatically. The queue remains paused until an operator explicitly
  resumes it via the UI or API.

### Observe-only mode

- **FR17 — Dry-run mode.** A per-agent boolean `observe_only`
  (default `false`) causes the permission endpoint to log every
  decision but always return `allow`. The log entry includes what the
  decision **would have been** under enforcement. This lets operators
  audit policy impact without blocking agent work.

### Precheck

- **FR18 — Hook-aware precheck.** For the `claude-mediated` driver,
  the precheck verifies:
  1. The init event does **not** report `permissionMode == "bypassPermissions"`.
  2. The init event confirms hooks are configured (implementation may
     check for a hook-related field in the init payload, or simply
     verify the settings file was accepted without error).
  If either check fails, the run is terminated with a structured error
  analogous to the existing precheck (see [[agent-permission-precheck]]).

### Audit and observability

- **FR19 — Structured logging.** Every permission decision (allow or
  deny) is logged as a structured JSON line in the run log with:
  `run_id`, `tool_name`, `target_path` (if applicable), `command`
  (if Bash), `decision`, `reason`, `policy_rule` (which rule matched),
  `timestamp`.

- **FR20 — WebSocket events.** Each permission decision is broadcast
  as an `agent.permission` WebSocket event with the same fields as
  FR19. The UI run timeline renders these alongside existing
  `agent.progress` events.

- **FR21 — Denied-calls summary.** When a run completes, the
  `agent.completed` or `agent.failed` event includes a
  `denied_tool_calls` array summarising each denial (tool, path,
  reason). The UI renders this prominently on the run detail view.

### Configuration

- **FR22 — Per-agent driver selection.** In `lifecycle/config.yaml`,
  each agent's `driver` field selects the driver. Example:

  ```yaml
  agents:
    - name: backend-developer
      driver: claude-code-cli        # existing, unchanged
    - name: backend-developer-strict
      driver: claude-mediated
      bash_allowlist: ["go test", "go build", "go vet", "git status"]
      bash_denylist: ["rm -rf", "sudo"]
      on_denial: continue
      observe_only: false
  ```

- **FR23 — Existing driver unchanged.** The `claude-code-cli` driver
  is not modified. `RequireBypassPermissions` config stays in place
  for the existing driver. Selecting `claude-mediated` does not
  affect runs using `claude-code-cli`.

## Non-functional requirements

- **NFR1 — Latency.** Permission round-trips over loopback must
  complete in < 10 ms p99 under normal load. The hook helper must
  not add more than 50 ms end-to-end per tool call (including
  process spawn of the subcommand).

- **NFR2 — Failure resilience.** If the Kaos Control HTTP server is
  unreachable when the hook helper POSTs, the helper retries once
  after 500 ms. If still unreachable, the helper returns `deny` and
  logs a warning. The agent run will likely fail, which is the safe
  default.

- **NFR3 — Secret entropy.** The per-run secret must be generated
  from `crypto/rand` with at least 32 bytes of entropy.

- **NFR4 — Temporary file cleanup.** Per-run `settings.json` files
  and any other temporary files are removed in a deferred cleanup
  that runs on run completion, including after panics and SIGKILLs
  (best-effort for the latter).

- **NFR5 — Claude Code version.** The minimum supported Claude Code
  version for the hooks API must be documented. The driver should
  check `claude --version` at startup and log a warning if the
  version is below the minimum.

- **NFR6 — Backwards compatibility.** Adding the new driver must not
  change any existing behaviour. All existing tests must continue to
  pass without modification.

## Acceptance Criteria

- [ ] **AC1** — `claude-mediated` driver is selectable in
  `lifecycle/config.yaml` and spawns `claude` without bypass flags.
- [ ] **AC2** — `kaos-control hook-helper` subcommand reads stdin,
  POSTs to the permission endpoint with the run secret, and prints
  the JSON response.
- [ ] **AC3** — `POST /api/agent/{run_id}/permission` accepts
  tool-call payloads, evaluates the permission policy, and returns
  `allow` or `deny` with a reason.
- [ ] **AC4** — A Write tool call targeting a path outside
  `AllowedPaths` is denied before execution. Verified by test with
  a mock claude binary emitting a Write event to a disallowed path.
- [ ] **AC5** — A Bash tool call matching the denylist (e.g. `sudo rm
  -rf /`) is denied. A Bash call matching the allowlist is allowed.
- [ ] **AC6** — A run with any denied tool calls does **not**
  auto-commit. The queue is paused. The run's `denied_tool_calls`
  flag is set.
- [ ] **AC7** — With `observe_only: true`, denied-by-policy calls are
  logged but allowed. The log shows the would-be decision.
- [ ] **AC8** — Permission decisions appear in the UI run timeline as
  `agent.permission` events. Denied calls are visually distinct
  (e.g. red icon/badge).
- [ ] **AC9** — The run detail view shows a denied-calls summary when
  denials occurred.
- [ ] **AC10** — The `claude-code-cli` driver is unaffected. Existing
  agent runs work identically to before this change.
- [ ] **AC11** — Per-run `settings.json` is generated before spawn and
  cleaned up after the run completes.
- [ ] **AC12** — The per-run secret is validated on every permission
  request. Requests with wrong or missing secrets return HTTP 403.
- [ ] **AC13** — Precheck for `claude-mediated` fails the run if the
  init event reports bypass mode or if hooks are not configured.
  See [[agent-permission-precheck]].

## Resolved Questions

1. **Hook helper process overhead.** The hook helper is a new process spawn per tool call. Should we consider a long-running sidecar process instead, or is per-call spawn acceptable given loopback latency? (The idea doc leans toward per-call; confirm after benchmarking.)

> Long-running sidecar works.

2. **Lineage scope at run time.** How does the driver determine the current lineage for a run? Is it passed explicitly in the agent config, inferred from the ticket/artifact being worked on, or derived from the prompt? This affects FR10 implementation.

> It is inferred from the artifact being worked on.

3. **Bash pattern matching semantics.** Should `bash_allowlist` / `bash_denylist` use glob patterns, regex, or prefix matching? Glob is simplest; regex is most powerful but harder to configure safely. The idea doc uses glob-like syntax — confirm the contract.

> Globs work for v1

4. **Hook helper fallback on server unreachable.** FR's NFR2 specifies deny-on-unreachable as the safe default. The idea doc suggests a local-allowlist fallback instead. Which is correct? Deny-on-unreachable is safer; local-allowlist is more available. Recommend deny-on-unreachable with a config escape hatch.

> deny-on-unreachable with a config escape hatch
