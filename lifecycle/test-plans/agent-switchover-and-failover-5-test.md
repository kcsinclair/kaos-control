---
title: Agent Switchover and Failover — Test Plan
type: plan-test
status: draft
lineage: agent-switchover-and-failover
parent: lifecycle/requirements/agent-switchover-and-failover-2.md
created: "2026-09-03T12:00:00+10:00"
---

# Agent Switchover and Failover — Test Plan

Verifies [[agent-switchover-and-failover]] against the backend and frontend
plans — see [[agent-switchover-and-failover]] for the shared lineage. Existing
targets to extend live under `tests/integration/`
(`failover_auto_test.go`, `failover_recovery_test.go`, `failover_secrets_test.go`,
`failover_helpers_test.go`, `provider_switch_api_test.go`), plus package unit
tests. The three named tests carried over from
[[automated-failover-always-rejected]] — `TestFailover_AutoSwitch_HTTP529`,
`TestFailover_AutoSwitch_RateLimitQuota`, `TestSecrets_FailoverAudit` — must pass
once failover works. Reuse the fake-upstream provider harness in
`failover_helpers_test.go` to simulate 429/529/5xx/disconnect responses.

---

## Milestone 1 — `operations.yaml` store & git-isolation

**Description.** Prove runtime state lives only in the git-ignored root
`operations.yaml`, is atomic, secret-free, and restart-durable.

**Files to change.**
- `internal/project/operations_test.go` (new) — round-trip, atomic-write
  (interrupt between temp + rename), restart durability.
- `tests/integration/failover_secrets_test.go` — extend: after a failover assert
  `operations.yaml` exists at project root, is absent from `git status`, and
  contains no configured API key (`TestSecrets_FailoverAudit`).

**Acceptance criteria.**
- Store round-trips losslessly; an interrupted write never truncates the file.
- After a failover: `lifecycle/config.yaml` unchanged, no new git commit,
  `operations.yaml` never in `git status`, no secret in the file.

## Milestone 2 — Automated project-wide failover

**Description.** A 529/429 on provider P with `automated_switchover` enabled moves
every agent on P to its secondary in one action, restarts the interrupted job,
and drains the rest of the queue on the secondary in order; failover never yields
`provider == fallback_provider`.

**Files to change.**
- `tests/integration/failover_auto_test.go` — `TestFailover_AutoSwitch_HTTP529`,
  `TestFailover_AutoSwitch_RateLimitQuota` (multi-agent-on-P, order-preserved
  drain, interrupted-job restart); `auth_error` triggers failover when enabled.
- `internal/queue/dispatcher_failover_test.go` — action-dispatch unit coverage
  (project-wide switch, one-level cap, secondary-also-fails → pause).

**Acceptance criteria.**
- Every agent on P switches in one action; interrupted job restarts; queue
  continues on the secondary in order.
- `auth_error` fails over when automated switchover is enabled.
- A secondary failure pauses (no third target); no config write, no commit.

## Milestone 3 — Disabled mode & queue ordering

**Description.** With `automated_switchover` disabled, a would-be-failover reason
pauses the queue in order, waits, restarts the failed job first, then continues
in queued order (FR-4, NFR-3).

**Files to change.**
- `tests/integration/failover_auto_test.go` (or a new `failover_paused_test.go`)
  — disabled-mode pause/resume with ordering assertions.

**Acceptance criteria.**
- Order is preserved across the pause; the failed job is restarted first on
  resume; remaining jobs follow in queued order.

## Milestone 4 — Event→action policy completeness & inspection

**Description.** Every classified reason has an explicit effective action
(defaults applied), and the effective policy is inspectable via the API.

**Files to change.**
- `internal/config/config_test.go` — `EffectiveSwitchoverPolicy()` covers every
  reason (`rate_limit`, `overloaded`, `unreachable`, `auth_error`,
  `provider_disconnected`, `model_not_found`, `model_unloaded`,
  `tools_unsupported`, `context_window_exceeded`, `turn_token_ceiling`,
  `max_iterations_reached`, `timeout`); overrides win; invalid verbs/reasons
  rejected; `automated_switchover` disabled by default.
- `tests/integration/provider_switch_api_test.go` — the policy route returns an
  action for every reason including defaults.

**Acceptance criteria.**
- No reason falls through to an implicit default; the API exposes the resolved
  action for each.

## Milestone 5 — `provider_disconnected` retry, backoff, threshold, durability

**Description.** Retry-in-place with 2s/8s/30s backoff; backoff-window collapse
(Resolved Q1); post-first-token disconnect still retries (Resolved Q2); the 4th
distinct disconnect within a rolling hour (per provider) pauses the queue; the
counter survives restart.

**Files to change.**
- `internal/agent/openai_compatible_test.go` — retry-in-place resends identical
  `messages`, backoff timing (injected clock), SSE-line-count recorded,
  pre/post-first-token handling.
- `internal/project/operations_test.go` — rolling-hour counter, backoff-window
  collapse, reload-after-restart.
- `tests/integration/` (new `failover_disconnect_test.go`) — 4th disconnect in an
  hour pauses the queue; counter intact after a simulated restart.

**Acceptance criteria.**
- Single incident retries and completes without losing prior turns; clustered
  disconnects collapse to one; 4th-in-hour pauses; counter durable.

## Milestone 6 — Reachability in all modes & quota-gated recovery

**Description.** Single-provider mode is probed and its reachability surfaced;
`primary_recovered` is not announced before the recorded reset time after a
quota failover.

**Files to change.**
- `internal/project/recovery_prober_test.go` — probe set includes all
  in-use providers (not only failed-over primaries); quota gating suppresses
  recovery until `resets_at_unix`.
- `tests/integration/failover_recovery_test.go` — status API shows reachability
  in single-provider mode; no premature `primary_recovered` after a quota
  failover.

**Acceptance criteria.**
- Single-provider reachability appears in the status API; recovery is suppressed
  until the reset time passes despite healthy `/v1/models` probes.

## Milestone 7 — Restart semantics & partial-commit surfacing

**Description.** No partial commit → clean restart; suspected partial commit →
surfaced to the operator, neither auto-rerun nor auto-rolled-back.

**Files to change.**
- `internal/git/` unit test — partial-commit detection helper.
- `internal/queue/dispatcher_test.go` — clean restart vs operator-decision
  branch.
- `tests/integration/` — a job with a suspected partial commit is surfaced (not
  rerun/rolled back); a clean job restarts.

**Acceptance criteria.**
- Clean job restarts; suspected-partial job is surfaced with a pending decision
  in the status API.

## Milestone 8 — Manual switch/failback API & guards

**Description.** Manual project-wide switch and manual failback work off
`operations.yaml`; a manual switch during an executing run is rejected with a
warning listing the running jobs; mutations require `devops`/`product-owner`.

**Files to change.**
- `internal/http/provider_switch_test.go` — status built from operations;
  in-flight-run rejection lists running jobs; role enforcement.
- `tests/integration/provider_switch_api_test.go` — end-to-end manual switch/
  failback and the rejection path.

**Acceptance criteria.**
- Manual switch during a run is rejected with the running jobs named; failback is
  manual only; unauthorized callers are rejected.

## Milestone 9 — Observability & reports aggregation

**Description.** Transitions appear in the application log (secret-free) and the
reports aggregation reports failover count, causing provider, time on secondary,
and time-to-restore.

**Files to change.**
- `internal/reports/failover_report_test.go` (new) — aggregation correctness.
- `tests/integration/` — a failover appears in the log and the reports API with
  no secret material.

**Acceptance criteria.**
- Log lines and reports carry the required metrics and no secrets.

## Milestone 10 — Frontend unit tests

**Description.** Cover the new UI behaviour with vitest.

**Files to change.**
- `web/src/stores/__tests__/providerSwitch.spec.ts` — mode/side getters,
  reachability in single-provider mode, policy load.
- `web/src/views/project/__tests__/FailbackView.spec.ts` (new) — reset-time
  display; "recovered" suppressed until reset passes.
- `web/src/components/layout/__tests__/AppHeader.spec.ts` — status button text
  (Primary/Secondary/partial) and hidden-without-secondary.
- `web/src/components/provider/__tests__/ProviderFailoverModal.spec.ts` /
  `SwitchProviderModal` — running-jobs rejection warning.

**Acceptance criteria.**
- Status button text is correct per state; failback screen shows reset time and
  suppresses unqualified recovery; manual-switch warning lists running jobs.

---

## Coverage artifact

- On merge, add/update a `type: test` artifact under `lifecycle/tests/`
  describing what the new integration tests in `tests/integration/` cover, per the
  repository's test-artifact convention. See [[agent-switchover-and-failover]]
  backend and frontend plans for the behaviour under test.

## Traceability (requirement acceptance criteria → milestones)

- Carried-over tests pass → M2, M6, M1.
- No config write / no commit; state in `operations.yaml`, git-ignored → M1, M2.
- 529/429 project-wide failover with restart + ordered drain → M2.
- Disabled-mode ordered pause + restart → M3.
- `auth_error` failover → M2.
- Policy has explicit entry per reason, inspectable → M4.
- Disconnect retry/backoff/threshold/durability → M5.
- Reachability in single-provider mode → M6.
- Manual switch rejected while running → M8.
- Partial-commit surfaced, not auto-run/rolled-back → M7.
- Status button + failback reset time, no premature recovery → M10, M6.
- Failover in log + reports → M9.
