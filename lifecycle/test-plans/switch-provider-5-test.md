---
title: "Test Plan — Dynamic Provider Switching, Automated Failover & UI Controls"
type: plan-test
status: draft
lineage: switch-provider
parent: lifecycle/requirements/switch-provider-2.md
created: "2026-08-25T08:50:00+10:00"
priority: high
labels:
    - agent
    - provider
    - test
    - reliability
    - open-provider-support
release: KC-Release6
---

# Test Plan — Dynamic Provider Switching, Automated Failover & UI Controls

This test plan defines the automated and manual verification suite for [[switch-provider-2]], validating the backend implementation [[switch-provider-3-be]] and frontend implementation [[switch-provider-4-fe]]. It covers fallback configuration parsing and validation, AST-based YAML config mutations, automated 529 failover engine triggers, head-of-queue immediate retry, REST switching endpoints (`/api/p/{project}/provider-switch`), template preset switching, background recovery probing, secret hygiene enforcement, and Vue SPA component interactions.

## Scope Boundary

- **In scope:**
  - Go unit tests for schema validation, AST node patcher, failover state machine, and health prober.
  - Go integration tests in `tests/integration/` verifying automated failover execution, queue retries, REST switching APIs, template application, and recovery alerts.
  - Secret masking and credential containment assertions conforming to [[standards/secrets-handling]].
  - Frontend Vitest suite for `useProviderSwitchStore`, `AppHeader.vue`, `ProviderFailoverModal.vue`, `AgentPanelRow.vue`, `AgentConfigForm.vue`, and `QueuePauseBanner.vue`.
  - Manual end-to-end smoke verification script.

- **Out of scope:**
  - OpenAI-compatible tool loop streaming parser tests (tested in [[open-provider-support-5-test]]).
  - Provider CRUD in app config and `/v1/models` discovery (tested in [[provider-model-for-agents-5-test]]).

## Architecture & Standards Conformance

- **Single self-contained binary:** Tests run in-process using Go `testing` and `vitest` without external supervisor daemons or CGO.
- **Index is a cache & Disk is authoritative:** Tests verify that all config modifications persist directly to `lifecycle/config.yaml` on disk and survive server restarts ([[standards/index-is-a-cache]]).
- **Secrets hygiene:** Explicit assertions guarantee zero API keys or authentication credentials leak into `lifecycle/config.yaml`, git commits, feed events, or API payloads ([[standards/secrets-handling]]).
- **Tool mediation:** Tests verify agent execution permissions and `allowed_write_paths` remain strictly enforced during and after provider switching ([[decisions/adr-0006-mediated-agent-driver-permission-model]]).

## Cross-References

- [[switch-provider-2]] — Authoritative requirement artifact.
- [[switch-provider-3-be]] — Backend implementation plan.
- [[switch-provider-4-fe]] — Frontend implementation plan.
- [[provider-model-for-agents-5-test]] — Provider model and `/api/providers` test plan.
- [[agent-rate-limit-queue-5-test]] — Work queue and rate limit test plan.

---

## Suite 1 — Go Unit Tests

### 1.1 Configuration & Policy Validation (`internal/config/`)

| # | Test Name | Description |
|---|---|---|
| C1 | `TestAgentConfig_FallbackFields` | Unmarshals agent YAML with `fallback_provider` and `fallback_model`; asserts fields populated. |
| C2 | `TestProjectConfig_ValidateFailover` | Validates `fallback_provider != provider` and `fallback_model != ""`; rejects invalid configurations. |
| C3 | `TestProjectConfig_EffectiveFailoverDefaults` | Asserts default failover configuration: `Enabled: true`, `AutoSwitch: false`, `MaxFailoversPerRun: 1`, `ProbeIntervalSeconds: 60`. |
| C4 | `TestProjectConfig_ProviderTemplates` | Parses and validates `provider_templates` with unique names and valid agent mappings. |

### 1.2 AST-based Config Mutation (`internal/config/patch_provider_test.go`)

| # | Test Name | Description |
|---|---|---|
| P1 | `TestPatchAgentProviders_SingleAgent` | Updates agent `provider` and `model` in `lifecycle/config.yaml`; asserts comments and other agent fields remain intact. |
| P2 | `TestPatchAgentProviders_FailoverState` | Inserts `primary_provider` and `primary_model` nodes; verifies clean round-trip. |
| P3 | `TestPatchAgentProviders_RestoreState` | Deletes `primary_provider` and `primary_model` nodes cleanly without leaving orphaned empty lines. |
| P4 | `TestPatchAgentProviders_Batch` | Applies multiple agent patches in a single atomic disk write. |

### 1.3 Failover Engine & Dispatcher (`internal/queue/dispatcher_failover_test.go`)

| # | Test Name | Description |
|---|---|---|
| D1 | `TestDispatcher_FailoverAutoSwitch_529` | Simulated 529 error with `auto_switch: true` triggers failover switch, re-enqueues job at head, and leaves queue unpaused. |
| D2 | `TestDispatcher_FailoverAutoSwitchDisabled` | Simulated 529 error with `auto_switch: false` (default) pauses queue and emits `queue.paused`. |
| D3 | `TestDispatcher_FailoverUnhealthyFallback` | Fallback health probe fails -> dispatcher does not switch; pauses queue normally. |
| D4 | `TestDispatcher_MaxFailoversPerRun` | Forces cascading failure past `max_failovers_per_run`; confirms job fails and queue pauses. |

### 1.4 Background Recovery Prober (`internal/project/recovery_prober_test.go`)

| # | Test Name | Description |
|---|---|---|
| R1 | `TestRecoveryProber_IdleWhenNoFailover` | Prober makes 0 HTTP requests when no project agents have `primary_provider` set. |
| R2 | `TestRecoveryProber_TwoConsecutiveHealthy` | When 2 consecutive probes succeed, prober emits `provider.primary_recovered` WS event and feed record. |
| R3 | `TestRecoveryProber_ResetOnFailure` | Unhealthy probe resets consecutive success counter to 0. |

---

## Suite 2 — Go Integration Tests (`tests/integration/`)

Each test sets up a complete test environment using `newTestEnvWithCfgYAML` and mock HTTP servers for upstream providers.

### 2.1 Automated Failover & Queue Retry (`tests/integration/failover_auto_test.go`)

| # | Test Name | Description |
|---|---|---|
| FA1 | `TestFailover_AutoSwitch_HTTP529` | Enqueues job for agent with Anthropic primary and Gemini fallback. Fake driver emits 529 overload. Asserts: (1) `lifecycle/config.yaml` updated with `primary_provider: anthropic-cloud` and `provider: gemini-cloud`, (2) git commit created, (3) `provider.switched` WS event broadcast, (4) stalled job immediately retries and finishes on Gemini fallback without queue pause. |
| FA2 | `TestFailover_AutoSwitch_RateLimitQuota` | Emits HTTP 429 quota exhaustion. Asserts auto-switch occurs and head-of-queue retry executes immediately. |
| FA3 | `TestFailover_Disabled_PausesQueue` | Agent with `fallback_provider` but `auto_switch: false`. Asserts queue transitions to paused with reset time banner. |

### 2.2 Provider Switching REST API (`tests/integration/provider_switch_api_test.go`)

| # | Test Name | Description |
|---|---|---|
| SA1 | `TestProviderSwitchAPI_GetStatus` | Calls `GET /api/p/{project}/provider-switch/status`. Asserts failover status, agent list, and active vs primary targets. |
| SA2 | `TestProviderSwitchAPI_ManualSwitch` | Calls `POST /api/p/{project}/agents/{name}/switch-provider`. Asserts disk updated, `config.reloaded` WS emitted, git commit created. |
| SA3 | `TestProviderSwitchAPI_RestoreAgent` | Calls `POST /api/p/{project}/agents/{name}/restore-provider`. Asserts primary restored, `primary_provider` cleared, git commit created. |
| SA4 | `TestProviderSwitchAPI_RestoreAll` | Puts 3 agents in failover; calls `POST /api/p/{project}/provider-switch/restore-all`. Asserts all 3 restored atomically. |
| SA5 | `TestProviderSwitchAPI_ApplyTemplate` | Calls `POST /api/p/{project}/provider-templates/apply` with `template: "local-ai"`. Asserts all mapped agents updated. |
| SA6 | `TestProviderSwitchAPI_RoleAuth` | Asserts non-admin/non-devops user receives `403 Forbidden` on mutation endpoints. |

### 2.3 Recovery Probing & Alerts (`tests/integration/failover_recovery_test.go`)

| # | Test Name | Description |
|---|---|---|
| FR1 | `TestRecovery_ProbeAndAlert` | Puts agent in failover. Mock primary server returns 200 OK. Asserts `provider.primary_recovered` received over WebSocket. |

### 2.4 Secrets Hygiene Audit (`tests/integration/failover_secrets_test.go`)

| # | Test Name | Description |
|---|---|---|
| FS1 | `TestSecrets_FailoverAudit` | Inspects `lifecycle/config.yaml`, git commit log, WS event payloads, and REST responses; asserts zero plaintext API keys are present. |

---

## Suite 3 — Frontend Vitest Component Tests

### 3.1 Store Tests (`web/src/stores/__tests__/providerSwitch.spec.ts`)

| # | Test Name | Description |
|---|---|---|
| ST1 | `fetches initial failover status and templates` | Verifies `fetchStatus` and `fetchTemplates` populate store state. |
| ST2 | `reacts to provider.switched WS event` | Updates matching agent in store and marks `failover_active = true`. |
| ST3 | `reacts to provider.restored WS event` | Clears failover state on matching agent. |
| ST4 | `reacts to provider.primary_recovered WS event` | Marks primary provider as recovered and sets `hasRecoveredPrimary = true`. |

### 3.2 UI Component Tests

| # | File | Test Name | Description |
|---|---|---|---|
| UI1 | `AppHeader.spec.ts` | `renders failover badge when active` | Amber alert badge renders with agent count; clicking emits open event. |
| UI2 | `AppHeader.spec.ts` | `renders green indicator on recovery` | Green pulsing indicator appears when primary provider recovers. |
| UI3 | `ProviderFailoverModal.spec.ts` | `renders active failovers list` | Displays agent name, fallback target, primary target, trigger reason, and health badge. |
| UI4 | `ProviderFailoverModal.spec.ts` | `restore all calls store action` | Click on "Restore All" triggers `restoreAll()` and disables button during flight. |
| UI5 | `AgentPanelRow.spec.ts` | `renders fallback badge and restore button` | Agent in failover shows amber badge and direct "Restore Primary" button. |
| UI6 | `AgentConfigForm.spec.ts` | `renders fallback fields` | Fallback provider and model form inputs render and save properly. |
| UI7 | `QueuePauseBanner.spec.ts` | `renders Switch Provider & Resume button` | Button renders when queue paused on rate limit; click opens switch modal. |

---

## Suite 4 — Manual Smoke Verification Script

Run after automated suites pass:

1. Start kaos-control with test project containing 2 agents (`requirements-analyst` and `backend-developer`).
2. Configure `requirements-analyst` with Primary: Anthropic Claude, Fallback: Gemini Flash.
3. Simulate upstream Anthropic outage (or trigger mock 529 via test endpoint).
4. Verify:
   - Amber "Failover Active (1)" badge appears in header navigation.
   - Click badge to open `ProviderFailoverModal`. Verify agent shows active on Gemini Flash with reason "HTTP 529 Overloaded".
   - Open `AgentPanelRow.vue` for `requirements-analyst`. Verify amber fallback badge and "Restore Primary" button.
5. In `ProviderFailoverModal`, click "Apply Preset Template" $\to$ "Local AI".
   - Verify both agents switch to local Ollama/llama.cpp targets in real time.
6. Bring Anthropic mock server back online.
   - Verify header badge pulses green with "Primary provider recovered".
7. Click "Restore All Primary Providers".
   - Verify all agents return to their primary configurations, header badge dismisses, and git commit history documents all transitions cleanly.

---

## Acceptance Criteria Checklist

- [ ] All Go unit tests in Suite 1 pass.
- [ ] All Go integration tests in Suite 2 pass.
- [ ] All Vitest tests in Suite 3 pass.
- [ ] Manual smoke script in Suite 4 executes cleanly.
- [ ] Zero API keys appear in markdown artifacts, git commit logs, or API payloads.
