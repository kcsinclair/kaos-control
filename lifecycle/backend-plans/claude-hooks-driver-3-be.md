---
title: "Backend Plan: Mediated Claude Driver with Permission Hooks"
type: plan-backend
status: done
lineage: claude-hooks-driver
parent: lifecycle/requirements/claude-hooks-driver-2.md
created: "2026-05-15T14:00:00+10:00"
---

# Backend Plan: Mediated Claude Driver with Permission Hooks

Parent: [[claude-hooks-driver]].

This plan covers the Go backend implementation for FR1–FR23 and NFR1–NFR6
from the requirement. Frontend work is in [[claude-hooks-driver-4-fe]];
test coverage is in [[claude-hooks-driver-5-test]].

---

## Milestone 1 — Configuration & AgentConfig Extensions

### Description

Extend `AgentConfig` with the fields needed by the `claude-mediated` driver
and add validation rules so that misconfigured agents are rejected at load
time.

### Files to change

- **`internal/config/config.go`** — Add fields to `AgentConfig`:
  - `BashAllowlist []string` (`yaml:"bash_allowlist,omitempty"`)
  - `BashDenylist []string` (`yaml:"bash_denylist,omitempty"`)
  - `OnDenial string` (`yaml:"on_denial,omitempty"`) — `"continue"` (default) or `"abort"`
  - `ObserveOnly bool` (`yaml:"observe_only,omitempty"`)
  Add validation in `validateProject()`: when `driver == "claude-mediated"`,
  reject if `OnDenial` is set to an unknown value; warn if `BashAllowlist`
  and `BashDenylist` overlap.

- **`lifecycle/config.yaml`** — Document the new fields in comments. No
  agents are switched to `claude-mediated` yet (FR23 — existing driver
  unchanged).

### Acceptance criteria

- [ ] `AgentConfig` struct has the four new fields with correct YAML tags.
- [ ] `validateProject` rejects unknown `on_denial` values for `claude-mediated` agents.
- [ ] Existing agents with `driver: claude-code-cli` or `driver: ollama` pass validation unchanged (NFR6).
- [ ] `make test-unit` passes.

---

## Milestone 2 — Permission Policy Engine

### Description

Implement the stateless permission policy evaluator. This is a pure-logic
package with no HTTP or driver dependencies, making it independently
testable.

### Files to change

- **`internal/agent/policy.go`** (new file) — Core types and evaluator:
  ```go
  type PolicyConfig struct {
      AllowedPaths   []string
      LineagePaths   []string // derived from lineage at run start
      BashAllowlist  []string
      BashDenylist   []string
      ObserveOnly    bool
  }

  type Decision struct {
      Action string // "allow" or "deny"
      Reason string
      Rule   string // e.g. "allowed_paths", "bash_denylist", "lineage_scope"
  }

  func Evaluate(cfg PolicyConfig, toolName string, toolInput map[string]any) Decision
  ```
  Logic (FR9–FR13):
  1. Read-only tools (`Read`, `Glob`, `Grep`, `WebFetch`, `WebSearch`,
     `Agent`, `TodoWrite`, `NotebookEdit`) → allow (FR13).
  2. File-mutating tools (`Write`, `Edit`) → extract `file_path`, resolve
     against `AllowedPaths` prefixes. If no prefix matches → deny
     `"allowed_paths"`. If `LineagePaths` configured and no lineage prefix
     matches → deny `"lineage_scope"` (FR10).
  3. `Bash` tool → extract `command` string. Check `BashDenylist` first
     (glob match) → deny `"bash_denylist"`. Then check `BashAllowlist` (if
     non-empty and no match) → deny `"bash_allowlist"` (FR11).
  4. All other tools → allow.

- **`internal/agent/policy_defaults.go`** (new file) — Default denylist
  (FR12):
  ```go
  var DefaultBashDenylist = []string{
      "rm -rf /",
      "rm -rf /*",
      "sudo *",
      "curl *|*sh",
      "wget *|*sh",
      "curl *| *sh",
      "wget *| *sh",
      "chmod 777 /*",
      "chown * /*",
  }
  ```
  The default list is merged with per-agent `bash_denylist` at run start.

- **`internal/agent/policy_test.go`** (new file) — Unit tests for every
  branch: allowed writes, denied writes, lineage scoping, bash
  allow/denylist precedence, read-only pass-through, default denylist
  patterns, observe-only mode flag plumbing.

### Acceptance criteria

- [ ] `Evaluate` returns correct decisions for all tool types and policy combinations.
- [ ] Default denylist matches `sudo rm -rf /`, `curl ... | sh`, `chmod 777 /etc`.
- [ ] Denylist takes precedence over allowlist (FR11).
- [ ] Read-only tools always return allow (FR13).
- [ ] `ObserveOnly` flag is passed through but does not change the decision (observe-only is handled by the endpoint, not the evaluator).
- [ ] `make test-unit` passes with >90% coverage on `policy.go`.

---

## Milestone 3 — Per-run Secret Generation & Settings File

### Description

Implement the per-run secret and the temporary `settings.json` that wires
the `PreToolUse` hook to `kaos-control hook-helper`.

### Files to change

- **`internal/agent/runsecret.go`** (new file) — Secret generation (FR5,
  NFR3):
  ```go
  func GenerateRunSecret() (string, error)
  ```
  Uses `crypto/rand` to produce 32 bytes, hex-encoded (64-char string).

- **`internal/agent/settings.go`** (new file) — Settings file generation
  (FR6):
  ```go
  func WriteHookSettings(dir string, binary string, serverAddr string, runID string) (path string, cleanup func(), err error)
  ```
  Writes a `settings.json` to `dir` containing:
  ```json
  {
    "hooks": {
      "PreToolUse": [
        {
          "type": "command",
          "command": "<binary> hook-helper --server <serverAddr> --run-id <runID>"
        }
      ]
    }
  }
  ```
  Returns a cleanup function that removes the file. Cleanup is safe to call
  multiple times (NFR4).

### Acceptance criteria

- [ ] `GenerateRunSecret` returns 64-char hex strings with no collisions across 10k calls.
- [ ] `WriteHookSettings` creates a valid JSON file parseable by Claude Code.
- [ ] The cleanup function removes the file and is idempotent.
- [ ] `settings.json` path does not collide across concurrent runs (use run ID in filename or temp dir).

---

## Milestone 4 — Permission HTTP Endpoint

### Description

Add the `POST /api/agent/{run_id}/permission` endpoint that the hook helper
calls (FR7, FR8).

### Files to change

- **`internal/agent/agent.go`** — Add to `Manager`:
  - `runSecrets map[string]string` — maps `runID → secret`, populated in
    `StartRun`, removed in `supervise` cleanup.
  - `PolicyForRun(runID string) (*PolicyConfig, error)` — returns the
    policy config for an active run.
  - `RecordDenial(runID string, d Decision, toolName string, toolInput map[string]any)` —
    appends to a per-run denial log, sets `denied_tool_calls` flag.

- **`internal/http/permission.go`** (new file) — Handler:
  ```go
  func (s *Server) handlePermission(w http.ResponseWriter, r *http.Request)
  ```
  1. Extract `run_id` from URL, `KC_HOOK_SECRET` from `Authorization`
     header (or `X-Hook-Secret` header).
  2. Validate secret against `Manager.runSecrets[runID]` → 403 if mismatch
     (FR8).
  3. Validate `run_id` is active → 400 if not.
  4. Decode request body (Claude Code hook contract: `{tool_name, tool_input}`).
  5. Call `policy.Evaluate(cfg, toolName, toolInput)`.
  6. If `ObserveOnly` → log the would-be decision, return `{"decision":"allow"}` (FR17).
  7. If denied → call `Manager.RecordDenial`, broadcast `agent.permission`
     WS event (FR20), log structured JSON (FR19). If `OnDenial == "abort"` →
     also kill the run (FR14).
  8. Return `{"decision": d.Action, "reason": d.Reason}`.

- **`internal/http/server.go`** — Register route. This endpoint is
  **exempt from session auth and CSRF** (it uses the per-run secret
  instead). Add a dedicated route group outside the session-protected
  block:
  ```go
  r.Post("/api/agent/{run_id}/permission", s.handlePermission)
  ```

### Acceptance criteria

- [ ] Endpoint returns 403 for missing/wrong secret (FR8).
- [ ] Endpoint returns 400 for unknown `run_id` or malformed body (FR8).
- [ ] Endpoint evaluates policy and returns correct JSON (FR7).
- [ ] `agent.permission` WS event is broadcast on every decision (FR20).
- [ ] Structured log line emitted per decision with all FR19 fields.
- [ ] `observe_only` mode logs but always returns allow (FR17).
- [ ] `on_denial: abort` kills the run on first deny (FR14).

---

## Milestone 5 — Hook Helper Subcommand

### Description

Implement `kaos-control hook-helper` — the process Claude Code spawns on
every `PreToolUse` event (FR4).

### Files to change

- **`cmd/kaos-control/hookcmd/hook.go`** (new package) — Subcommand:
  1. Parse `--server` and `--run-id` flags.
  2. Read `KC_HOOK_SECRET` from environment (FR5).
  3. Read tool-call JSON from stdin.
  4. POST to `http://<server>/api/agent/<run_id>/permission` with the
     secret in `Authorization: Bearer <secret>` header.
  5. On success: write response body to stdout, exit 0.
  6. On connection failure: retry once after 500ms (NFR2). If still
     unreachable: write `{"decision":"deny","reason":"server unreachable"}`
     to stdout, log warning to stderr, exit 0.

- **`cmd/kaos-control/main.go`** — Register `hook-helper` subcommand in
  the dispatch switch.

### Acceptance criteria

- [ ] `kaos-control hook-helper` reads stdin, POSTs to the endpoint, prints response (AC2).
- [ ] Per-run secret is read from `KC_HOOK_SECRET` env var (FR5).
- [ ] Retries once on connection failure, then returns deny (NFR2).
- [ ] Exit code is always 0 (Claude Code treats non-zero as an error).
- [ ] No external dependencies — uses only stdlib `net/http`.

---

## Milestone 6 — ClaudeHooksDriver Implementation

### Description

Implement the `ClaudeHooksDriver` that wires everything together (FR1–FR3).

### Files to change

- **`internal/agent/claude_mediated.go`** (new file) — Driver struct:
  ```go
  type ClaudeHooksDriver struct {
      ServerAddr string // e.g. "127.0.0.1:9600"
      BinaryPath string // path to kaos-control binary (os.Executable)
  }
  ```
  `Start(ctx, run)` method:
  1. Generate per-run secret via `GenerateRunSecret()`.
  2. Store secret in `Manager` (passed via a `SecretStore` interface or
     closure to avoid circular dependency).
  3. Write temp `settings.json` via `WriteHookSettings(...)`.
  4. Build args: `--settings <path>`, `-p`, `--output-format stream-json`,
     `--verbose`, `--model` (if set). **No** `--dangerously-skip-permissions`
     or `--permission-mode bypassPermissions` (FR2).
  5. Set `KC_HOOK_SECRET=<secret>` in subprocess env.
  6. Reuse `ClaudeCodeDriver`'s stream-JSON parsing, progress channel, log
     file, and stderr capture logic (FR3). Extract shared logic into
     unexported helper functions (e.g. `startClaudeProcess`) that both
     drivers call.
  7. Return `Process` with deferred cleanup of settings file (NFR4).

- **`internal/agent/agent.go`** — Register driver in `New()`:
  ```go
  "claude-mediated": &ClaudeHooksDriver{
      ServerAddr: serverAddr,
      BinaryPath: binaryPath,
  },
  ```
  Pass `serverAddr` and `binaryPath` through `New()` parameters (derived
  from app config and `os.Executable()`).

- **`internal/agent/agent.go`** — In `StartRun()`, when driver is
  `claude-mediated`:
  - Compute `LineagePaths` from the artifact's lineage and the agent's
    `AllowedPaths`.
  - Store secret in `m.runSecrets`.
  - Pass `PolicyConfig` to `Manager` for the permission endpoint to use.

### Acceptance criteria

- [ ] `claude-mediated` is selectable in config and spawns `claude` without bypass flags (AC1).
- [ ] Per-run `settings.json` is generated before spawn and cleaned up after (AC11).
- [ ] Stream-JSON parsing, progress, cost tracking reuse existing logic (FR3).
- [ ] Secret is stored and available to the permission endpoint.

---

## Milestone 7 — Hook-aware Precheck

### Description

Implement the precheck for `claude-mediated` runs that verifies Claude Code
is not in bypass mode and has hooks configured (FR18).

### Files to change

- **`internal/agent/precheck.go`** — Add `runMediatedPrecheck()`:
  1. Wait for `system/init` event (reuse timeout logic).
  2. Check `permissionMode` is **not** `"bypassPermissions"` — if it is,
     kill and fail with `"precheck_mediated_bypass"` reason.
  3. Optionally verify hooks presence in the init payload (if Claude Code
     surfaces this). If not verifiable from init, log a warning but pass
     (the settings file acceptance is the implicit check).
  4. After init, continue draining progress events normally.

- **`internal/agent/agent.go`** — In `supervise()`, branch on driver type:
  - `"claude-code-cli"` → existing `runPrecheck()`
  - `"claude-mediated"` → `runMediatedPrecheck()`
  - others → drain only

### Acceptance criteria

- [ ] Precheck fails the run if init reports `bypassPermissions` (AC13).
- [ ] Precheck passes when hooks are configured and permission mode is not bypass.
- [ ] Precheck timeout kills the run (same as existing behaviour).
- [ ] Existing `claude-code-cli` precheck is unchanged (NFR6).

---

## Milestone 8 — Denial Handling: No Auto-commit, Queue Pause

### Description

Implement the post-run logic for denied tool calls (FR15, FR16).

### Files to change

- **`internal/agent/agent.go`** — In `supervise()`, after process exits:
  1. Check `m.deniedCalls[runID]` (populated by `RecordDenial` during the
     run).
  2. If any denials exist:
     - Skip the `git.AddAndCommit` block (FR15).
     - Set `denied_tool_calls` flag on the `AgentRunRow`.
     - Pause the project's agent queue (FR16). Call
       `m.queue.Pause(projectID)` or equivalent.
     - Include `denied_tool_calls` array in `agent.finished`/`agent.failed`
       event payload (FR21).

- **`internal/agent/agent.go`** — Add `deniedCalls map[string][]DenialRecord`
  to `Manager`, with `DenialRecord` struct:
  ```go
  type DenialRecord struct {
      ToolName string `json:"tool_name"`
      Path     string `json:"path,omitempty"`
      Command  string `json:"command,omitempty"`
      Reason   string `json:"reason"`
      Rule     string `json:"rule"`
  }
  ```

- **`internal/index/index.go`** (or agent run schema) — Add
  `denied_tool_calls` JSON column to `agent_runs` table.

### Acceptance criteria

- [ ] A run with denied tool calls does not auto-commit (AC6).
- [ ] The agent queue is paused after a run with denials (FR16).
- [ ] `denied_tool_calls` array is included in the completion WS event (FR21).
- [ ] `denied_tool_calls` is persisted in the run record for later retrieval.
- [ ] A run with zero denials behaves identically to the existing flow.

---

## Milestone 9 — Audit Trail & Structured Logging

### Description

Ensure every permission decision produces a structured log line and a WS
event (FR19, FR20).

### Files to change

- **`internal/http/permission.go`** — Already emits structured log and WS
  event in Milestone 4. This milestone adds:
  - Append the structured decision to the per-run log file (same file used
    by `ClaudeCodeDriver` for stdout). Use a `permission_decision` JSON
    envelope distinct from Claude Code's stream-json lines.
  - Ensure `agent.permission` WS event payload includes: `run_id`,
    `tool_name`, `target_path`, `command`, `decision`, `reason`,
    `policy_rule`, `timestamp`.

- **`internal/hub/hub.go`** — No changes; existing `Broadcast` is
  sufficient. Add `"agent.permission"` to event type documentation/comments.

### Acceptance criteria

- [ ] Every permission decision is logged as structured JSON in the run log file (FR19).
- [ ] Every permission decision is broadcast as `agent.permission` WS event (FR20).
- [ ] Log lines contain all required fields: `run_id`, `tool_name`, `target_path`, `command`, `decision`, `reason`, `policy_rule`, `timestamp`.

---

## Milestone 10 — Claude Code Version Check

### Description

Check `claude --version` at startup and warn if below the minimum required
for hooks API support (NFR5).

### Files to change

- **`internal/agent/agent.go`** — In `New()` or a lazy-init path:
  - Run `claude --version`, parse the output.
  - Compare against a `MinClaudeVersion` constant.
  - Log `slog.Warn` if below minimum; do not block startup.

### Acceptance criteria

- [ ] Warning is logged when Claude Code version is below minimum.
- [ ] Startup is not blocked by version check failure.
- [ ] The minimum version is documented as a constant.
