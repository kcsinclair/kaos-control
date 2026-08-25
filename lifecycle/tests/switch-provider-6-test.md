---
title: "Dynamic Provider Switching, Automated Failover & UI Controls — Go Integration Test Suite"
type: test
status: draft
lineage: switch-provider
parent: lifecycle/test-plans/switch-provider-5-test.md
created: "2026-08-25T11:35:00+10:00"
---

# Dynamic Provider Switching, Automated Failover & UI Controls — Go Integration Test Suite

Implements Suite 2 ("Go Integration Tests") of [[switch-provider-5-test]] against the
real HTTP server + queue dispatcher + git-backed `lifecycle/config.yaml`, mirroring
the production wiring in `cmd/kaos-control/main.go`. Frontend Vitest (Suite 3) and
the manual smoke script (Suite 4) are out of scope for this artifact — a test-developer
agent's write scope is `tests/**` (Go) and this `lifecycle/tests/` record.

## Test files

| File | Covers |
|---|---|
| `tests/integration/failover_helpers_test.go` | Shared `failoverTestEnv`: project + HTTP server + queue dispatcher wired with `FailoverPolicy`/`AgentFailoverInfo`/`ProbeProviderHealth`/`SwitchAgentProvider` (same shape as production), a toggleable `mockProvider` (fake `/v1/models` upstream), a project-WebSocket event buffer (`connectProjectWS`), and a `config.LoadProject`-backed structured config reader (`loadConfig`/`findFailoverAgentConfig`) for assertions that don't depend on YAML text formatting. |
| `tests/integration/failover_auto_test.go` | §2.1 — FA1, FA2, FA3. |
| `tests/integration/provider_switch_api_test.go` | §2.2 — SA1–SA6. |
| `tests/integration/failover_recovery_test.go` | §2.3 — FR1. |
| `tests/integration/failover_secrets_test.go` | §2.4 — FS1. |

## Scenarios covered

- **FA1** `TestFailover_AutoSwitch_HTTP529` — HTTP 529 (`overloaded_error`) drives
  automated failover: config write, git commit, `provider.switched` WS broadcast,
  head-of-queue retry to completion without a pause.
- **FA2** `TestFailover_AutoSwitch_RateLimitQuota` — HTTP 429 (`rate_limit_error`)
  quota exhaustion drives the same automated-failover path.
- **FA3** `TestFailover_Disabled_PausesQueue` — with `auto_switch` left at its
  default (false), a transient failure falls through to the standard rate-limit
  pause instead of switching provider. **Passing.**
- **SA1–SA6** `TestProviderSwitchAPI_*` — `GET .../provider-switch/status`,
  manual `switch-provider`/`restore-provider`, `restore-all` (3 agents
  atomically), `provider-templates/apply`, and a `403` role-auth check for a
  user without `product-owner`/`devops`. **All passing.**
- **FR1** `TestRecovery_ProbeAndAlert` — background `RecoveryProber` (1s probe
  interval) detects a primary provider's mock endpoint flip from 503 to 200 and
  broadcasts `provider.primary_recovered` after two consecutive healthy probes.
  **Passing.**
- **FS1** `TestSecrets_FailoverAudit` — drives a full automated-failover cycle
  with app-level providers carrying distinct `api_key` values, then scans
  `lifecycle/config.yaml`, the git commit log, buffered WS payloads, and the
  `provider-switch/status` REST response for either key.

## Known issue: FA1, FA2, FS1 currently fail — backend defect, not a test defect

`TestFailover_AutoSwitch_HTTP529`, `TestFailover_AutoSwitch_RateLimitQuota`, and
`TestSecrets_FailoverAudit` are implemented exactly per the plan and correctly
exercise the described behaviour, but **fail against the current backend**
because automated failover never actually engages. Every automated-failover
attempt fails with:

```
queue: automated provider switch failed; falling back to standard pause
err="... fallback_provider must differ from provider \"gemini-cloud\""
```

**Root cause**: `internal/config/config.go:1269` rejects a config where an
agent's `fallback_provider` equals its active `provider`. `Project.SwitchAgentProvider`
(`internal/project/provider_switch.go:50-62`) patches `provider`/`model` (and,
on first failover, stashes `primary_provider`/`primary_model`) but never
touches `fallback_provider`/`fallback_model`. Automated failover
(`internal/queue/dispatcher.go`'s `tryFailover`) always switches an agent to
*its own configured* `info.FallbackProvider` — so the moment the switch lands,
`provider == fallback_provider` and the very next config write/reload (inside
the same call, via `PatchAgentProviders`'s validate-before-write) is rejected.
The dispatcher then falls back to a standard rate-limit pause, silently
defeating the feature. A manual `switch-provider` call to an agent's own
`fallback_provider` value hits the identical error (verified directly; worked
around in `TestProviderSwitchAPI_ManualSwitch` by switching agent-a to a third
provider instead, since that test only needs to exercise the manual-switch
mechanics, not this specific value).

This was never caught by the existing unit tests
(`internal/project/project_switch_test.go`,
`internal/queue/dispatcher_failover_test.go`) because both switch to a
provider *other than* the agent's own `fallback_provider`
(`internal/project/project_switch_test.go:29` sets `fallback_provider:
local-ollama` but every `SwitchAgentProvider` call in that file targets
`gemini-cloud`) — a case automated failover never actually produces.

Per this agent's role scope (`tests/**`, `lifecycle/tests/`, and
`lifecycle/architecture/decisions/` for a proposed ADR only), a fix to
`internal/project/provider_switch.go` / `internal/config/config.go` is out of
scope here. These three tests are left in place, written to the spec, as the
regression check for whichever change resolves the defect (e.g. clearing or
rotating `fallback_provider`/`fallback_model` as part of the failover patch).

## Acceptance criteria status

- [x] Suite 2 Go integration tests implemented for all of FA1–FA3, SA1–SA6, FR1, FS1.
- [ ] All Suite 2 tests pass — **8/11 pass**; FA1, FA2, FS1 fail pending the
      backend fix described above.
- [ ] Zero API keys in artifacts/git/API payloads — verified by FS1's assertions
      for the parts of the flow that do complete; the full assertion set
      (including the post-completion REST/WS checks) cannot run to completion
      until FS1 itself unblocks.
