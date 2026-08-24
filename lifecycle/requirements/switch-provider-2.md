---
title: Dynamic Provider Switching and Failover
type: requirement
status: draft
lineage: switch-provider
created: "2026-08-25T08:25:25+10:00"
priority: high
parent: lifecycle/ideas/switch-provider.md
labels:
    - agent
    - provider
    - config
    - agent-runner
    - reliability
    - backend
    - frontend
    - feature
    - operability
    - ai-ml
    - open-provider-support
release: KC-Release6
assignees:
    - role: product-owner
      who: agent
---

# Dynamic Provider Switching and Failover

Parent: [[switch-provider]], Workstream 2 of [[open-provider-support]].  
Depends on: [[provider-model-for-agents]] (Provider entity & agent `{provider, model}` abstraction).

---

## Problem

When an upstream AI provider becomes unavailable — due to HTTP 529 Overloaded errors, API outages, or extended quota exhaustion — orchestrated agent runs and global work queue jobs fail or are stalled in long pause states.

Currently, resolving this requires an operator to:
1. Manually open and edit `lifecycle/config.yaml` on disk.
2. Update the `driver`, `model`, or `provider` fields for each affected agent.
3. Wait for the configuration watcher to reload or restart the server.
4. Manually re-trigger the stalled jobs or restart the queue.
5. Manually reverse the configuration once the primary provider recovers.

Without automated failover and discoverable UI switching controls, upstream provider instability directly disrupts autonomous agent pipelines and requires manual operator intervention.

---

## Goals / Non-goals

### Goals

- **Automated Provider Failover:** Automatically switch an agent's active provider to a configured fallback provider when persistent provider-level failure signals (HTTP 529 Overloaded, HTTP 429 quota exhaustion with long reset, or connection outages) are detected by the runner.
- **Immediate Queue Retries:** Re-enqueue rate-limited or overloaded queue jobs immediately on the fallback provider rather than pausing the global queue for 30–60 minutes.
- **Atomic Config Rewrite & Hot-Reload:** Programmatically rewrite the affected agent declarations in `lifecycle/config.yaml` and hot-reload the project config (`project.ReloadConfig()`) without server restarts.
- **Reversible & Auditable Failover State:** Preserve the original `primary_provider` and `primary_model` in the agent configuration, record the switch in the event feed and git commit log, and provide a one-click "Restore Primary" mechanism.
- **Manual Switching Affordance:** Provide REST APIs and frontend UI controls (Agent Panel, Global Header, and Queue View) to manually switch an agent's provider or batch-switch all agents from one provider to another.
- **Primary Provider Health Probing:** Periodically probe the health of inactive primary providers and notify operators when the primary provider is healthy and ready to be restored.
- **Strict Secrets & Sandboxing Compliance:** Ensure all provider switching operations strictly adhere to [[standards/secrets-handling]], keeping credentials out of `lifecycle/config.yaml`, logs, and markdown artifacts.

### Non-goals

- Implementing the underlying error detection logic for HTTP 529 and rate limits. (This is already built and shipped in `internal/agent/agent.go` via `extractRateLimitText` and [[rate-limit-event-detection]]).
- Implementing the OpenAI-compatible agent driver or tool-calling loop. (This is specified in [[open-provider-support-2]]).
- Implementing the Provider CRUD settings and `/v1/models` discovery. (This is specified in [[provider-model-for-agents]]).
- Managing external AI provider accounts, billing, or load balancing across multiple active providers simultaneously (multi-provider round-robin).

---

## Detailed Requirements

### Architecture-Breaking Requirements

Review against `lifecycle/architecture/architecture-summary.md` and recorded architectural standards:

1. **Single self-contained binary:**
   - *Requirement:* Dynamic provider switching, config rewriting, and health probing must be implemented entirely within the Go backend using standard library packages (`net/http`, `gopkg.in/yaml.v3`) and embedded Vue SPA assets.
   - *Evaluation:* **Satisfied.** No external daemons, proxies, or native CGO dependencies are introduced.
2. **Local filesystem is the source of truth:**
   - *Requirement:* All agent configuration changes must be written directly to `lifecycle/config.yaml` on disk, followed by synchronous hot-reloading and optional git commits.
   - *Evaluation:* **Satisfied.** Disk remains authoritative; the index cache is updated via existing watchers and synchronous reloads ([[standards/index-is-a-cache]]).
3. **Offline operation capability:**
   - *Requirement:* Failover to locally-hosted providers (e.g. Ollama or llama.cpp at `http://localhost:11434` or `http://localhost:7442`) must operate without internet access when cloud providers go offline.
   - *Evaluation:* **Satisfied.** Fallback destinations can be local or remote.
4. **Direct-served, no trusted proxy hop & Secrets hygiene:**
   - *Requirement:* Provider switching mutates only provider reference names (e.g. `provider: "gemini-local"`) in `lifecycle/config.yaml`. Secret API keys remain securely in `~/.kaos-control/config.yaml` and are never written to project artifacts or logged ([[standards/secrets-handling]]).
   - *Evaluation:* **Satisfied.** Non-secret identifiers only are written to disk.
5. **Agent tool mediation and sandboxing:**
   - *Requirement:* Switching an agent's provider must not alter or bypass the agent's configured `allowed_write_paths`, `bash_allowlist`, or tool mediation policies ([[decisions/adr-0006-mediated-agent-driver-permission-model]], [[standards/filesystem-sandboxing]]).
   - *Evaluation:* **Satisfied.** Tool execution rules remain enforced on the agent role regardless of the backing provider.

**Conclusion:** No architectural constraints are violated. No new ADR is required.

---

### Functional Requirements

#### FR-1: Fallback Configuration Model

- `AgentConfig` in `internal/config/config.go` is extended with:
  - `fallback_provider` (string, **optional**): Name of a registered Provider in app config to use when the active provider fails.
  - `fallback_model` (string, **optional**): Model identifier to use with the fallback provider (e.g. `gemini-2.5-flash`, `qwen3-coder:30b`).
  - `primary_provider` (string, **optional**): Set automatically when operating in a failover state to preserve the original provider name.
  - `primary_model` (string, **optional**): Set automatically when operating in a failover state to preserve the original model name.
- `ProjectConfig` gains an optional `provider_failover` policy block:
  ```yaml
  provider_failover:
    enabled: true                  # default: true
    auto_switch: true              # default: true; if false, surfaces UI alert without auto-rewrite
    switch_on_kinds:               # which error kinds trigger failover
      - "overloaded"               # HTTP 529 / overloaded_error
      - "rate_limit"               # HTTP 429 / quota exhaustion
      - "unreachable"              # network connection refused / 5xx
    max_failovers_per_run: 1       # bounds cascading failovers
    probe_interval_seconds: 60     # frequency to probe primary provider recovery
  ```
- `config.ValidateProject` validates:
  - When `fallback_provider` is specified, it must not be identical to `provider`.
  - When `fallback_provider` is specified, `fallback_model` must not be empty.
  - `primary_provider` and `primary_model` are accepted as valid YAML fields but are managed by the failover engine.

#### FR-2: Failure Trigger & Automatic Failover Engine

- The failover engine hooks into `internal/agent/agent.go` and `internal/queue/dispatcher.go`:
  - When an agent run terminates with a failure or emits an event matching `extractRateLimitText` (`RateLimitKindOverloaded` or `RateLimitKindRateLimit`), or fails during HTTP connect/preflight:
  - The engine checks if `provider_failover.enabled` and `provider_failover.auto_switch` are `true`, and the agent has a configured `fallback_provider`.
  - **Pre-switch Health Check:** Before applying the switch, the engine performs a fast probe (using `GET /api/providers/{fallback_provider}/health` logic) to ensure the fallback provider is reachable.
  - If reachable:
    1. **State Transition:** 
       - Sets `primary_provider = agent.provider` and `primary_model = agent.model`.
       - Sets `agent.provider = agent.fallback_provider` and `agent.model = agent.fallback_model`.
    2. **Persistence & Hot-Reload:** Calls the config mutation path (FR-3) to write `lifecycle/config.yaml` and reloads runtime state via `project.ReloadConfig()`.
    3. **Queue Re-enqueue:** Re-enqueues the failed queue job at the head of the queue (`attempts` incremented, but without applying the long pause delay).
    4. **Event Broadcast:** Emits WebSocket event `provider.switched` and inserts a project feed record `EventType: "provider_switched"`.
  - If the fallback provider is unreachable or no fallback is defined:
    - Falls back to existing behavior: pauses the queue according to `queue.rate_limit` pause rules and emits a warning event.

#### FR-3: Config Mutation, Hot-Reload & Git Commit

- A safe YAML AST/node-based updater in `internal/config/` (or dedicated project config helper) rewrites `lifecycle/config.yaml` preserving existing comments, structure, and indentation.
- The update atomically writes `lifecycle/config.yaml` and synchronously executes `project.ReloadConfig()` to refresh the active `agent.Manager` roster.
- If the project is a git repo (`project.Git != nil`), commits the config change with author identity set to system/agent:
  - Auto-failover commit message: `failover(agent): <agent_name> <primary_provider> → <fallback_provider> (reason: <reason>)`
  - Revert commit message: `restore(agent): <agent_name> restored to <primary_provider>`
- WebSocket event `config.reloaded` is broadcast across the hub so all connected clients observe the updated agent configuration immediately.

#### FR-4: Reversible & Auditable Failover State Tracking

- An agent is considered **In Failover** when its `primary_provider` field is non-empty.
- The system maintains an in-memory and event-backed failover log:
  - `run_id`: The run that triggered the failover.
  - `agent`: Name of the agent.
  - `primary_provider` / `primary_model`: The original settings.
  - `active_provider` / `active_model`: The currently operating settings.
  - `switched_at`: Timestamp of the switch.
  - `reason`: Failure reason (e.g. `529 Overloaded`, `Rate limit exceeded (resets 8pm)`, `Manual`).
- Restoring an agent swaps `provider` back to `primary_provider`, `model` back to `primary_model`, clears `primary_provider` and `primary_model`, writes config, and commits.

#### FR-5: Provider Switching REST API (`/api/p/{project}/provider-switch`)

The server exposes authenticated routes for inspecting and controlling provider state (requiring `devops` or `product-owner` role for mutations):

- `GET /api/p/{project}/provider-switch/status`
  - Returns failover status across all agents in the project:
    ```json
    {
      "failover_active": true,
      "agents": [
        {
          "agent": "requirements-analyst",
          "is_failover": true,
          "primary_provider": "anthropic-cloud",
          "primary_model": "claude-3-7-sonnet",
          "active_provider": "gemini-cloud",
          "active_model": "gemini-2.5-flash",
          "switched_at": "2026-08-25T08:30:00Z",
          "reason": "HTTP 529 Overloaded",
          "primary_healthy": true
        }
      ]
    }
    ```
- `POST /api/p/{project}/agents/{name}/switch-provider`
  - Manually switches an individual agent to a specified `{provider, model}`.
  - Payload: `{"provider": "gemini-cloud", "model": "gemini-2.5-flash", "reason": "Manual operator switch"}`.
  - Updates `lifecycle/config.yaml`, hot-reloads, and broadcasts `provider.switched`.
- `POST /api/p/{project}/agents/{name}/restore-provider`
  - Restores a single agent from its `primary_provider` / `primary_model`.
  - Clears failover state, rewrites config, hot-reloads, and broadcasts `provider.restored`.
- `POST /api/p/{project}/provider-switch/switch-all`
  - Batch-switches all agents currently targeting `from_provider` to `to_provider` (with optional default model mapping).
  - Useful during widespread provider outages.
- `POST /api/p/{project}/provider-switch/restore-all`
  - Restores all agents in the project currently in failover state back to their primary providers.

#### FR-6: Primary Provider Health Probing & Recovery Alerts

- When any agent in a project has `primary_provider` set (failover active), the backend runs a background recovery probe every `probe_interval_seconds` (default 60s) targeting the primary provider's health endpoint.
- Once the primary provider answers with healthy HTTP 200 / valid `/v1/models` for 2 consecutive probes:
  - Broadcasts WebSocket event `provider.primary_recovered` with `{provider: primary_provider, project: project}`.
  - Generates a notification feed event: `"Primary provider <name> has recovered and is ready to be restored."`
- Health probing stops automatically when all agents are restored to primary.

#### FR-7: Frontend Provider Status & Switching Affordances

- **App Header Alert Badge:**
  - When one or more agents are operating in failover state, an amber "Failover Active" badge appears in the top navigation bar.
  - Clicking the badge opens the Provider Failover drawer/modal.
- **Provider Failover Modal / Drawer:**
  - Summarises all agents currently in failover mode with their active vs primary providers, trigger reason, and time elapsed.
  - Shows live health indicator for the primary provider (e.g. Green "Recovered", Red "Still Unavailable").
  - Provides a one-click **"Restore All Primary Providers"** action button.
- **Agent Panel (`AgentPanel.vue` & `AgentConfigForm.vue`):**
  - Displays a dedicated status badge when an agent is on fallback: `"Active: Gemini 2.5 (Fallback for Claude Sonnet)"`.
  - Renders a **"Restore Primary"** button directly on the agent card when `primary_provider` is active.
  - Includes a **"Switch Provider"** action in the agent menu allowing immediate manual target change.
  - Form editor includes fields to configure `Fallback Provider` and `Fallback Model` alongside primary settings.
- **Queue View Banner (`QueueView.vue`):**
  - If a job fails due to rate-limit/529 and failover is disabled or unavailable, the queue pause banner displays a **"Switch Provider & Resume"** button that opens the switch modal for the failed agent.

---

### Non-Functional Requirements

#### NFR-1: Secret Hygiene & Standard Compliance

- Adheres strictly to [[standards/secrets-handling]]:
  - `lifecycle/config.yaml` stores only provider names (non-secret slugs).
  - API keys are NEVER written to project configuration, commit messages, event logs, or markdown artifacts.
  - Provider health probes and switch API responses mask credentials as `"***"`.

#### NFR-2: Safe Atomic Config Writes & Git Integrity

- All writes to `lifecycle/config.yaml` MUST be atomic (written to temporary file and renamed) to prevent file corruption.
- Git commits created by the switch engine must use the system/agent bot identity and cleanly record the transition without interfering with working tree changes on other files.

#### NFR-3: Realtime Event Propagation & Latency

- Failover decisions, config updates, and event broadcasts (`provider.switched`, `provider.restored`, `config.reloaded`) must complete within **500 ms** from the triggering failure event, allowing queued jobs to resume immediately.

#### NFR-4: Prevention of Infinite Cascading Failovers

- To prevent cyclical switching loops between broken providers, an individual agent run or queue job is capped at `max_failovers_per_run` (default 1) failovers. If the fallback provider also fails, the job transitions to `failed` and the queue pauses normally.

---

## Acceptance Criteria

- [ ] `AgentConfig` supports `fallback_provider`, `fallback_model`, `primary_provider`, and `primary_model` in `lifecycle/config.yaml`.
- [ ] `ProjectConfig` parses and validates `provider_failover` policy options (`enabled`, `auto_switch`, `switch_on_kinds`, `max_failovers_per_run`, `probe_interval_seconds`).
- [ ] Injecting a simulated HTTP 529 Overloaded or rate-limit failure on an agent with a configured fallback provider triggers automatic failover:
  - `lifecycle/config.yaml` is updated with `provider = fallback_provider`, `model = fallback_model`, and `primary_provider`/`primary_model` recorded.
  - `project.ReloadConfig()` is executed and `config.reloaded` WS event is emitted.
  - A git commit is created documenting the failover.
  - The stalled queue job is re-enqueued at the head and runs successfully on the fallback provider without a 30-minute pause delay.
- [ ] An agent with no `fallback_provider` configured continues to pause the queue normally according to existing rate-limit rules.
- [ ] `POST /api/p/{project}/agents/{name}/switch-provider` manually updates the agent's provider in `lifecycle/config.yaml` and reloads config live.
- [ ] `POST /api/p/{project}/agents/{name}/restore-provider` restores the original primary provider and model, clearing `primary_provider` and `primary_model`.
- [ ] `POST /api/p/{project}/provider-switch/switch-all` and `POST /api/p/{project}/provider-switch/restore-all` perform atomic batch switches across multiple agents.
- [ ] Background recovery probe detects when an inactive primary provider becomes healthy again and emits `provider.primary_recovered`.
- [ ] Frontend header displays an alert badge when any agent is operating in failover mode.
- [ ] Frontend Provider Failover modal displays all agents in failover state with primary provider health status and a "Restore All" button.
- [ ] Agent panel displays fallback badges, quick "Switch Provider" modal, and "Restore Primary" action.
- [ ] Provider API keys never appear in `lifecycle/config.yaml`, git commit logs, or REST API payloads.
- [ ] Integration tests verify automated 529 failover, manual switching API, batch restore, and queue retry behavior.
- [ ] Lineage artifacts correctly link via `parent: lifecycle/ideas/switch-provider.md` and reference related artifacts [[open-provider-support]], [[provider-model-for-agents]], [[open-provider-support-2]], and [[rate-limit-event-detection]].

---

## Resolved Questions

1. **Auto-switch vs User-Prompt Default:** `auto_switch: true` is chosen as default for 529 / overload errors because 529 is a transient server outage where autonomous pipelines should continue running. Operators can disable this globally or per-project via `provider_failover.auto_switch: false` if strict model consistency is required.

> auto_switch should be disabled by default, but the documentation and feature information should ensure it is mentioned.

2. **Batch Switching Granularity:** Batch switching (`switch-all`) targets all agents sharing a given `from_provider` rather than forcing every agent in the project to the same model, allowing role-specific model specializations (e.g. fast models for analysts, coder models for developers) to be preserved across providers.

> Switching should be between templates, there should be pre-defined templates which the user has setup which use the combination of providers for the agents, the user then switches between hybrid, claude, gemini, local-ai, etc.
