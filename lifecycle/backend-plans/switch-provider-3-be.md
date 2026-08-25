---
title: "Backend Plan — Dynamic Provider Switching, Automated Failover & Recovery"
type: plan-backend
status: done
lineage: switch-provider
parent: lifecycle/requirements/switch-provider-2.md
created: "2026-08-25T08:50:00+10:00"
priority: high
labels:
    - agent
    - provider
    - config
    - agent-runner
    - reliability
    - backend
    - open-provider-support
release: KC-Release6
---

# Backend Plan — Dynamic Provider Switching, Automated Failover & Recovery

This plan implements the backend architecture for [[switch-provider-2]] (Workstream 2 of [[open-provider-support]]). It builds upon the Provider entity foundation introduced in [[provider-model-for-agents-3-be]] to deliver automated runtime provider failover, immediate queue retries without long pause delays, atomic AST-based configuration rewrites of `lifecycle/config.yaml`, git-audited switch and restore actions, provider template presets, a comprehensive REST API under `/api/p/{project}/provider-switch`, and background health probing for inactive primary providers with real-time recovery event broadcasts.

## Scope Boundary

- **In scope:**
  - `AgentConfig` extensions (`fallback_provider`, `fallback_model`, `primary_provider`, `primary_model`) in `internal/config`.
  - `ProjectConfig` extensions (`provider_failover` policy block and `provider_templates` catalog) in `internal/config`.
  - AST-based atomic YAML mutation helper in `internal/config` using `gopkg.in/yaml.v3` to preserve comments, indentation, and formatting.
  - Failover engine in `internal/agent` and `internal/queue/dispatcher.go` with fast pre-switch health probing and immediate head-of-queue retry.
  - Reversible failover state logging and tracking in `internal/project`.
  - Provider switching REST API routes (`GET /api/p/{project}/provider-switch/status`, `POST /api/p/{project}/agents/{name}/switch-provider`, `POST /api/p/{project}/agents/{name}/restore-provider`, `POST /api/p/{project}/provider-switch/switch-all`, `POST /api/p/{project}/provider-switch/restore-all`, `POST /api/p/{project}/provider-switch/apply-template`).
  - Periodic background health prober in `internal/project` monitoring primary provider reachability during active failover.
  - Realtime WebSocket events (`provider.switched`, `provider.restored`, `provider.primary_recovered`, `config.reloaded`) and feed notifications.
  - Strict compliance with [[standards/secrets-handling]]: zero API keys written to disk, commits, logs, or API payloads.

- **Out of scope:**
  - Detection logic for HTTP 529, 429, and quota resets (already handled in `internal/agent/agent.go` and `internal/queue/dispatcher.go` via [[rate-limit-event-detection]]).
  - OpenAI-compatible tool loop and SSE stream parsing (specified in [[open-provider-support-3-be]]).
  - Provider CRUD in app config and `/v1/models` discovery (specified in [[provider-model-for-agents-3-be]]).
  - Frontend UI components, stores, and modal views (specified in [[switch-provider-4-fe]]).
  - Integration test suite definitions (specified in [[switch-provider-5-test]]).

## Architecture & Standards Conformance

- **Single self-contained binary:** Pure Go stdlib (`net/http`, `sync`, `time`, `context`), `gopkg.in/yaml.v3`, and existing internal packages. No CGO or external supervisor daemons.
- **Local filesystem is source of truth:** All configuration changes persist directly to `lifecycle/config.yaml` on disk atomically (`.tmp` + rename), followed by synchronous `project.ReloadConfig()` and index cache reconciliation ([[standards/index-is-a-cache]]).
- **Secrets hygiene standard:** Provider switching only mutates provider slug names. Secrets remain securely in `~/.kaos-control/config.yaml` and are never written to project files, commit messages, or event payloads ([[standards/secrets-handling]]).
- **Filesystem sandboxing & tool mediation:** Switching an agent's provider does not alter or widen `allowed_write_paths` or bypass driver mediation policies ([[decisions/adr-0006-mediated-agent-driver-permission-model]], [[standards/filesystem-sandboxing]]).
- **Offline / Local provider support:** Supports seamless failover to local endpoints (e.g. Ollama `:11434` or llama.cpp `:7442`) during cloud outages.

## Cross-References

- [[switch-provider-2]] — Authoritative requirement artifact.
- [[switch-provider-4-fe]] — Frontend implementation plan.
- [[switch-provider-5-test]] — Test plan.
- [[provider-model-for-agents-3-be]] — Provider entity model and `/api/providers` API.
- [[open-provider-support-3-be]] — OpenAI-compatible driver execution loop.
- [[rate-limit-event-detection]] — Upstream rate-limit and 529 overload event detection.

---

## Milestone 1 — Fallback Configuration Model, Policy & Provider Templates

### Description

Extend `AgentConfig` and `Project` in `internal/config/config.go` to support fallback provider definitions, failover policies, and provider templates. Implement validation in `validateProject`.

### Schema & Types

```go
// ProviderFailoverConfig defines project-level provider failover behaviour.
type ProviderFailoverConfig struct {
    Enabled              *bool    `yaml:"enabled,omitempty" json:"enabled"`                             // default: true
    AutoSwitch           *bool    `yaml:"auto_switch,omitempty" json:"auto_switch"`                     // default: false (per resolved question 1)
    SwitchOnKinds        []string `yaml:"switch_on_kinds,omitempty" json:"switch_on_kinds"`             // ["overloaded", "rate_limit", "unreachable"]
    MaxFailoversPerRun   int      `yaml:"max_failovers_per_run,omitempty" json:"max_failovers_per_run"` // default: 1
    ProbeIntervalSeconds int      `yaml:"probe_interval_seconds,omitempty" json:"probe_interval_seconds"` // default: 60
}

// ProviderTemplateAgentBinding defines an agent's provider and model in a template.
type ProviderTemplateAgentBinding struct {
    Provider string `yaml:"provider" json:"provider"`
    Model    string `yaml:"model" json:"model"`
}

// ProviderTemplate defines a named multi-agent provider configuration preset.
type ProviderTemplate struct {
    Name        string                                  `yaml:"name" json:"name"`
    Description string                                  `yaml:"description,omitempty" json:"description,omitempty"`
    Agents      map[string]ProviderTemplateAgentBinding `yaml:"agents" json:"agents"` // agent name -> {provider, model}
}
```

`AgentConfig` gains:
- `FallbackProvider string `yaml:"fallback_provider,omitempty" json:"fallback_provider,omitempty"`
- `FallbackModel string `yaml:"fallback_model,omitempty" json:"fallback_model,omitempty"`
- `PrimaryProvider string `yaml:"primary_provider,omitempty" json:"primary_provider,omitempty"`
- `PrimaryModel string `yaml:"primary_model,omitempty" json:"primary_model,omitempty"`

`Project` gains:
- `ProviderFailover ProviderFailoverConfig `yaml:"provider_failover,omitempty" json:"provider_failover,omitempty"`
- `ProviderTemplates []ProviderTemplate `yaml:"provider_templates,omitempty" json:"provider_templates,omitempty"`

### Files to change

- **Edit** `internal/config/config.go`:
  - Add `ProviderFailoverConfig`, `ProviderTemplateAgentBinding`, `ProviderTemplate` structs.
  - Update `AgentConfig` and `agentConfigRaw` with fallback and primary fields.
  - Update `AgentConfig.UnmarshalYAML` to parse fallback/primary fields.
  - Update `Project` struct with `ProviderFailover` and `ProviderTemplates`.
  - Update `validateProject`:
    - If `agent.FallbackProvider != ""`:
      - Verify `agent.FallbackProvider != agent.Provider` (cannot failover to same provider).
      - Verify `agent.FallbackModel != ""` (fallback model required when fallback provider set).
    - Validate `provider_failover.switch_on_kinds` values against allowed enum (`overloaded`, `rate_limit`, `unreachable`).
    - Validate `provider_templates`: unique template names, non-empty agent mappings.
  - Add helper `(p *Project) EffectiveFailoverConfig() ProviderFailoverConfig`:
    - Returns defaults: `Enabled: true`, `AutoSwitch: false`, `SwitchOnKinds: ["overloaded", "rate_limit", "unreachable"]`, `MaxFailoversPerRun: 1`, `ProbeIntervalSeconds: 60`.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit tests in `internal/config/config_test.go`:
  - `AgentConfig` correctly unmarshals `fallback_provider`, `fallback_model`, `primary_provider`, `primary_model`.
  - `validateProject` rejects `fallback_provider == provider`.
  - `validateProject` rejects `fallback_provider` with empty `fallback_model`.
  - `EffectiveFailoverConfig` returns `AutoSwitch: false` by default and applies configured overrides.
  - `ProviderTemplates` parse and validate correctly.

---

## Milestone 2 — Safe AST-based Config Mutation & Git Commit Engine

### Description

Implement an atomic YAML AST-based agent configuration mutator that modifies `lifecycle/config.yaml` in place while preserving comments, YAML formatting, and unmodeled sections. Wire into `internal/project` with automated Git commits using the system/agent bot identity.

### Files to change

- **New** `internal/config/patch_provider.go`:
  - `type AgentProviderPatch struct { AgentName string; Provider string; Model string; PrimaryProvider *string; PrimaryModel *string }`
  - `func PatchAgentProviders(projectRoot string, patches []AgentProviderPatch) error`:
    - Reads `lifecycle/config.yaml`.
    - Unmarshals into `yaml.Node` root document AST.
    - Locates `agents` sequence node.
    - For each matching agent mapping node:
      - Upserts / updates `provider` scalar node.
      - Upserts / updates `model` scalar node.
      - If `PrimaryProvider != nil`:
        - If `*PrimaryProvider != ""`, upserts `primary_provider` scalar node.
        - If `*PrimaryProvider == ""`, deletes `primary_provider` key-value nodes.
      - If `PrimaryModel != nil`:
        - If `*PrimaryModel != ""`, upserts `primary_model` scalar node.
        - If `*PrimaryModel == ""`, deletes `primary_model` key-value nodes.
    - Validates patched document against `config.LoadProject` using a temporary file before writing.
    - Writes atomically to `lifecycle/config.yaml` via temporary file and `os.Rename`.
- **Edit** `internal/project/project.go`:
  - `func (p *Project) SwitchAgentProvider(agentName, newProvider, newModel, reason string, isFailover bool) error`:
    - Finds agent in `p.Config().Agents`.
    - Constructs patch:
      - If `isFailover && agent.PrimaryProvider == ""`: sets `PrimaryProvider = &agent.Provider`, `PrimaryModel = &agent.Model`.
      - Sets `Provider = newProvider`, `Model = newModel`.
    - Calls `config.PatchAgentProviders(p.Entry.Path, []config.AgentProviderPatch{...})`.
    - Executes `p.ReloadConfig()`.
    - If `p.Git != nil`:
      - Creates commit with author `kaos-control bot <bot@kaos-control.local>`:
        - Failover commit: `failover(agent): <agentName> <primary> -> <fallback> (reason: <reason>)`
        - Manual switch commit: `switch(agent): <agentName> -> <newProvider>/<newModel> (reason: <reason>)`
    - Inserts event feed record: `EventType: "provider_switched"` with metadata.
    - Broadcasts `hub.Event{Type: "provider.switched", Payload: map[string]any{...}}`.
  - `func (p *Project) RestoreAgentProvider(agentName string) error`:
    - Finds agent in `p.Config().Agents`.
    - Requires `agent.PrimaryProvider != ""` (error if not in failover).
    - Constructs patch setting `Provider = agent.PrimaryProvider`, `Model = agent.PrimaryModel`, `PrimaryProvider = ""` (delete), `PrimaryModel = ""` (delete).
    - Calls `config.PatchAgentProviders`.
    - Executes `p.ReloadConfig()`.
    - If `p.Git != nil`:
      - Creates commit: `restore(agent): <agentName> restored to <primaryProvider>`.
    - Inserts event feed record: `EventType: "provider_restored"`.
    - Broadcasts `hub.Event{Type: "provider.restored", Payload: map[string]any{...}}`.
  - `func (p *Project) ApplyProviderTemplate(templateName string) error`:
    - Finds template in `p.Config().ProviderTemplates`.
    - Assembles patches for all mapped agents.
    - Applies `config.PatchAgentProviders`, reloads config, commits `template(provider): applied <templateName>`, and broadcasts `config.reloaded`.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit tests in `internal/config/patch_provider_test.go`:
  - Modifying an agent preserves comments and formatting across the file.
  - Adding `primary_provider` and removing `primary_provider` round-trips cleanly.
  - Multi-agent batch patching applies atomically in a single write.
- Unit tests in `internal/project/project_switch_test.go`:
  - `SwitchAgentProvider` updates disk, reloads config, and generates git commit.
  - `RestoreAgentProvider` reverts to primary and clears failover fields.
  - `ApplyProviderTemplate` batch-switches mapped agents atomically.

---

## Milestone 3 — Failover Engine, Pre-switch Health Check & Queue Re-enqueue

### Description

Hook into the agent execution loop in `internal/agent` and the dispatcher loop in `internal/queue/dispatcher.go` to intercept provider-level failure signals (HTTP 529 Overload, HTTP 429 quota exhaustion, network unreachable), perform a fast fallback reachability precheck, execute automatic failover if enabled, and immediately re-enqueue stalled queue jobs at the head of the queue.

### Files to change

- **Edit** `internal/agent/agent.go`:
  - Attach `FailoverAttempt int` counter to `RunHandle` and `Run` context to bound cascades (`max_failovers_per_run`).
  - Classify stream and termination errors into `FailureKind`:
    - `RateLimitKindOverloaded` (HTTP 529 / `overloaded_error`)
    - `RateLimitKindRateLimit` (HTTP 429 / quota reset)
    - `FailureKindUnreachable` (connection refused / 5xx gateway errors)
- **Edit** `internal/queue/dispatcher.go`:
  - Update `runResult` to carry `failureKind`, `agentName`, `project`, `failoverAttempts`.
  - In `Dispatcher.processNext`:
    - On run failure matching `runResult.kind == "rate_limit"` or `"overloaded"` or `"unreachable"`:
      - Retrieve `project.Project` from lookup.
      - Check project's `EffectiveFailoverConfig()`:
        - If `Enabled == true` and `AutoSwitch == true`:
          - Check if agent has configured `FallbackProvider` and `job.Attempts <= cfg.MaxFailoversPerRun`:
            - Execute pre-switch health probe: query fallback provider's health endpoint (timeout: 2s).
            - If fallback is healthy:
              - Call `project.SwitchAgentProvider(agentName, fallbackProvider, fallbackModel, reason, true)`.
              - Mark current job failed with reason `failover_triggered`.
              - Immediately re-enqueue failed job at head of queue (`Position = store.MinPosition() - 1`, `Attempts = job.Attempts + 1`, `State = StatePending`).
              - Broadcast `queue.added` with reason `failover_retry`.
              - **Do NOT pause the queue.** Return from failure handler, allowing dispatcher to immediately pick up the re-enqueued job on the next tick.
      - If `AutoSwitch == false` or no fallback provider or fallback unhealthy:
        - Fall back to standard `handleRateLimit` (pauses queue, broadcasts `queue.paused`, sets `paused_until`).
- **Edit** `internal/agent/runner.go`:
  - When running an agent with `Driver == "openai-compatible"`, wrap HTTP errors (e.g. 529, 503, connection refused) into typed `RunError` with appropriate failure kind.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit tests in `internal/queue/dispatcher_failover_test.go`:
  - Simulated 529 error with `auto_switch: true` and healthy fallback triggers `SwitchAgentProvider`, re-enqueues job at head, and does not pause queue.
  - Simulated 529 error with `auto_switch: false` pauses queue and emits `queue.paused`.
  - Agent with unhealthy fallback falls back to queue pause.
  - Cascading failover stops when `max_failovers_per_run` is exceeded.

---

## Milestone 4 — Provider Switching REST API (`/api/p/{project}/provider-switch`)

### Description

Mount authenticated HTTP endpoints under `/api/p/{project}/provider-switch` and `/api/p/{project}/agents/{name}/` to inspect failover state, execute manual switches, restore primary configurations, and apply multi-agent provider templates.

### API Endpoints

| Method | Path | Handler | Roles | Description |
|---|---|---|---|---|
| `GET` | `/api/p/{project}/provider-switch/status` | `handleGetFailoverStatus` | Any authenticated user | Returns project-wide failover status and agent details |
| `POST` | `/api/p/{project}/agents/{name}/switch-provider` | `handleAgentSwitchProvider` | `devops`, `product-owner`, `admin` | Manually switches an individual agent |
| `POST` | `/api/p/{project}/agents/{name}/restore-provider` | `handleAgentRestoreProvider` | `devops`, `product-owner`, `admin` | Restores an agent to primary provider/model |
| `POST` | `/api/p/{project}/provider-switch/switch-all` | `handleSwitchAllProviders` | `devops`, `product-owner`, `admin` | Batch-switches agents from `from_provider` to `to_provider` |
| `POST` | `/api/p/{project}/provider-switch/restore-all` | `handleRestoreAllProviders` | `devops`, `product-owner`, `admin` | Restores all agents currently in failover state |
| `GET` | `/api/p/{project}/provider-templates` | `handleListProviderTemplates` | Any authenticated user | Lists configured provider presets |
| `POST` | `/api/p/{project}/provider-templates/apply` | `handleApplyProviderTemplate` | `devops`, `product-owner`, `admin` | Applies a named provider template across project agents |

### Response Payloads

```jsonc
// GET /api/p/{project}/provider-switch/status (200 OK)
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
      "fallback_provider": "gemini-cloud",
      "fallback_model": "gemini-2.5-flash",
      "switched_at": "2026-08-25T08:30:00Z",
      "reason": "HTTP 529 Overloaded",
      "primary_healthy": true
    }
  ]
}

// POST /api/p/{project}/agents/{name}/switch-provider
// Request: { "provider": "gemini-cloud", "model": "gemini-2.5-flash", "reason": "Manual operator switch" }
// Response 200 OK: { "ok": true, "agent": "requirements-analyst", "provider": "gemini-cloud", "model": "gemini-2.5-flash" }

// POST /api/p/{project}/provider-templates/apply
// Request: { "template": "local-ai" }
// Response 200 OK: { "ok": true, "template": "local-ai", "updated_agents": 4 }
```

### Files to change

- **New** `internal/http/provider_switch.go`:
  - Implement the 7 route handlers with role validation (`requireProjectRole` or `requireAnyProjectRole`).
  - Add request payload validation: non-empty provider/model, verify provider exists in app config.
- **Edit** `internal/http/server.go`:
  - Mount `/provider-switch` and `/provider-templates` routes under project router `/api/p/{project}`.
- **Edit** `internal/http/agents.go`:
  - Update `agentSummary` struct to include `fallback_provider`, `fallback_model`, `primary_provider`, `primary_model`, `is_failover`.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit tests in `internal/http/provider_switch_test.go`:
  - `GET /provider-switch/status` accurately identifies agents in failover mode.
  - `POST /agents/{name}/switch-provider` mutates config, reloads project, emits WS event.
  - `POST /agents/{name}/restore-provider` reverts primary and clears failover flags.
  - `POST /provider-switch/restore-all` restores all agents in failover in a single transaction.
  - `POST /provider-templates/apply` applies mapped agents from template.
  - Role-gating enforces permissions (viewer gets 403 Forbidden).

---

## Milestone 5 — Primary Provider Health Prober & Realtime Recovery Engine

### Description

Implement a background goroutine in `internal/project` that activates whenever at least one agent is operating in failover mode (`primary_provider != ""`). Every `probe_interval_seconds` (default 60s), the prober tests the primary provider's `/health` endpoint. When 2 consecutive probes succeed, it emits `provider.primary_recovered` WebSocket events and notification feed records.

### Files to change

- **New** `internal/project/recovery_prober.go`:
  - `type RecoveryProber struct { ... }`
  - `func (rp *RecoveryProber) Start(ctx context.Context)`
  - `func (rp *RecoveryProber) probe(ctx context.Context)`:
    - Scans `p.Config().Agents` for any non-empty `PrimaryProvider`.
    - If none found: enters idle sleep (no network traffic when all agents are on primary).
    - If found: gathers set of unique `PrimaryProvider` names.
    - For each primary provider:
      - Dispatches fast HTTP probe to `<base_url>/v1/models` (or `/health`) with 3s timeout.
      - Tracks consecutive healthy count in `consecutiveSuccess[providerName]`.
      - When count reaches 2:
        - Broadcasts `hub.Event{Type: "provider.primary_recovered", Payload: map[string]any{"provider": providerName, "project": p.Entry.Name}}`.
        - Inserts notification feed record: `EventType: "primary_recovered"`, message: `"Primary provider <name> has recovered and is ready to be restored."`.
      - On failure: resets `consecutiveSuccess[providerName] = 0`.
- **Edit** `internal/project/project.go`:
  - Instantiate `RecoveryProber` during `Open()`.
  - Launch prober in `StartRecoveryProber(ctx context.Context)`.
  - Shut down cleanly in `Close()`.
- **Edit** `cmd/kaos-control/main.go`:
  - Invoke `p.StartRecoveryProber(ctx)` alongside other project background tasks.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit tests in `internal/project/recovery_prober_test.go`:
  - Prober remains idle when no agents are in failover.
  - When an agent enters failover, prober executes every probe interval.
  - 2 consecutive healthy responses trigger `provider.primary_recovered` WS event and feed record.
  - Transient failure resets success counter.
  - Prober exits cleanly on context cancellation.

---

## Verification (End-to-End)

1. `make lint` clean (`go vet` + `staticcheck`).
2. `make test-unit` clean (all unit tests in `internal/config`, `internal/agent`, `internal/queue`, `internal/http`, `internal/project`).
3. `make test-integration` clean (executing full failover integration test suite from [[switch-provider-5-test]]).
4. Secrets audit: Verify that `lifecycle/config.yaml`, git log output, and HTTP responses never contain unmasked API keys.

## Risk Notes

- **Cascading Failover Loops:** If fallback provider also fails (e.g. 529), bounding with `max_failovers_per_run: 1` prevents infinite switching ping-pong; subsequent failures cleanly transition to queue pause.
- **Concurrent Config Writes:** All YAML AST updates must acquire project-level locks and write atomically via `.tmp` file and rename to prevent file corruption.
- **Probe Load on Recovering Providers:** Probing is strictly throttled to `probe_interval_seconds` (default 60s) with 3s timeouts, preventing thundering-herd effects on recovering upstream servers.
