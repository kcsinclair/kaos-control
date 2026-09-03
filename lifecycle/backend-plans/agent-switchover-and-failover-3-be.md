---
title: Agent Switchover and Failover — Backend Plan
type: plan-backend
status: draft
lineage: agent-switchover-and-failover
parent: lifecycle/requirements/agent-switchover-and-failover-2.md
created: "2026-09-03T12:00:00+10:00"
---

# Agent Switchover and Failover — Backend Plan

Implements the backend for [[agent-switchover-and-failover]]. Pairs with the
frontend plan (status button, failback screen, policy inspector) and the test
plan — see [[agent-switchover-and-failover]] for the shared lineage. Related
prior work: [[switch-provider-2]] (superseded), [[automated-failover-always-rejected]]
(absorbed), [[rate-limit-event-detection]] (error classification, reused),
[[secrets-handling]] (NFR-1).

Conforms to the recorded architecture: single self-contained Go binary, no new
external datastore (`operations.yaml` is a plain YAML file read/written with
`gopkg.in/yaml.v3`), mediated-driver permission model unchanged
([[adr-0006-mediated-agent-driver-permission-model]]), secrets never leave
config ([[secrets-handling]]). The requirement's Architecture-Breaking analysis
(§ "Architecture-Breaking Requirements") concludes **no new ADR is required**;
Resolved Question 3 confirms this. No deviation is introduced by this plan.

**Central design shift.** Today the switch engine patches
`lifecycle/config.yaml` and git-commits every switch
(`internal/project/provider_switch.go:64,79,290-297`). This plan makes
`lifecycle/config.yaml` **declared intent only** and moves all live state to a
git-ignored `operations.yaml` at the project root. The *effective* provider for
an agent becomes `operations override, falling back to config`. This is also
what fixes [[automated-failover-always-rejected]]: because we no longer write
`provider = fallback_provider` into config, project-config validation
(`config.go:1294-1298`, "fallback_provider must differ from provider") can never
reject a failover.

---

## Milestone 1 — `operations.yaml` runtime-state store

**Description.** Introduce an authoritative, git-ignored runtime-state store at
the project root. It holds, per project: per-agent active-vs-primary state
(`agent`, primary `{provider, model}`, active `{provider, model}`, `switched_at`,
`reason`, `resets_at_unix`, `bucket`), provider reachability
(`{provider: {healthy, last_probed_at, since}}`), per-provider disconnect events
(timestamps, for the rolling-hour counter), and a failover history log. Writes
are atomic (temp file + `os.Rename`, NFR-2). The store loads on project open and
survives restart (NFR-5). It carries **no secret material** — provider/model
names only (NFR-1). `operations.yaml` is already in `.gitignore` (line 27); this
milestone adds nothing to git.

**Files to change.**
- `internal/project/operations.go` (new) — `Operations` struct + `Load`, atomic
  `Save`, and accessors (`AgentState`, `SetAgentState`, `ClearAgentState`,
  `Reachability`, `SetReachability`, `RecordDisconnect`, `DisconnectCountSince`,
  `AppendHistory`). `Save` writes `<root>/operations.yaml.tmp` then renames.
- `internal/project/operations_test.go` (new) — unit tests (round-trip,
  atomic-write, restart durability).
- `internal/project/project.go` — hold `*Operations` on `Project`, load it in
  `Open`, expose `p.Operations()`.
- `internal/config/config.go` — add a helper resolving an agent's **effective**
  active `{provider, model}` = operations override else config
  (`FindAgentConfig` stays the declared-intent source).

**Acceptance criteria.**
- Round-tripping an `Operations` value through `Save`/`Load` is lossless.
- A crash simulated between temp-write and rename leaves the previous
  `operations.yaml` intact and parseable (no truncated file).
- After `Save`, `operations.yaml` contains only provider/model **names** — a
  test greps the file for any configured API key and finds none (NFR-1).
- `operations.yaml` never appears in `git status` after any store write.

## Milestone 2 — Switch/restore/failover write to `operations.yaml`, not config or git

**Description.** Rewrite the switch engine so switchover, failover, failback,
and template-apply mutate `operations.yaml` (per-agent active state + history)
and **never** call `PatchAgentProviders`, `commitConfigChange`, or write
`lifecycle/config.yaml`. Stashing the "primary" is no longer a config patch —
primary is simply the config-declared `{provider, model}`, and the active
override lives in operations state. This removes the
`provider == fallback_provider` failure mode entirely (FR-1.2, FR-3.1). Broadcast
and feed events are preserved; the git-commit call is deleted.

**Files to change.**
- `internal/project/provider_switch.go` — rewrite `SwitchAgentProvider`,
  `RestoreAgentProvider`, `SwitchAllAgentProviders`, `RestoreAllAgentProviders`,
  `ApplyProviderTemplate` to update the operations store; **delete**
  `commitConfigChange` and its call sites and the `PatchAgentProviders` calls in
  this file. Keep `insertFeedEvent` + hub broadcasts.
- `internal/config/config.go` — the `fallback_provider != provider` validation
  (`config.go:1294-1298`) stays (declared intent is still validated); no failover
  path writes config, so it can no longer be tripped by a switch.
- Anywhere the run dispatcher/driver reads an agent's provider for a run, route
  through the effective-provider resolver (Milestone 1) so an active override is
  honoured.

**Acceptance criteria.**
- A failover produces **no** change to `lifecycle/config.yaml` and **no** git
  commit (asserted by the carried-over `TestSecrets_FailoverAudit` and a
  `git status`/log assertion, per Acceptance Criteria).
- Automated failover never yields `provider == fallback_provider`;
  `TestFailover_AutoSwitch_HTTP529` and `TestFailover_AutoSwitch_RateLimitQuota`
  pass.
- A run dispatched for a failed-over agent uses the secondary `{provider, model}`
  from operations state.

## Milestone 3 — Event → action policy (complete by construction)

**Description.** Add a `switchover` policy block to the project config (FR-2):
a top-level `automated_switchover` toggle (**disabled by default**, FR-2.1) and
an `events` map of reason → action. Provide an effective-policy resolver that
fills every classified reason with the FR-2.3 default when unset, so the map is
**complete by construction** (every reason from `rate_limit` … `timeout` has an
entry). Actions: `failover`, `pause_queue`, `retry_in_place`, `fail_run`
(FR-2.2). Reconcile with the existing `ProviderFailoverConfig`
(`config.go:851-869`): `automated_switchover` supersedes `auto_switch`, and the
reason→action map supersedes `switch_on_kinds`; keep back-compat mapping so
existing configs still load.

**Files to change.**
- `internal/config/config.go` — add `Switchover` struct (`AutomatedSwitchover
  *bool`, `Events map[string]string`); `EffectiveSwitchoverPolicy()` returning
  the fully-defaulted map; validation that every configured action is one of the
  four verbs and every configured reason is a known reason. Map legacy
  `ProviderFailover` fields onto the new block.
- `internal/agent/policy_defaults.go` (or new `switchover_defaults.go`) — the
  canonical reason list and FR-2.3 default action table.
- `internal/queue/dispatcher.go` — extend the `FailoverPolicy` snapshot
  (`dispatcher.go:116-124`) so the dispatcher resolves an **action** per reason,
  not just a boolean auto-switch.

**Acceptance criteria.**
- `EffectiveSwitchoverPolicy()` returns an explicit action for every reason in
  the FR-2.3 table, with configured entries overriding defaults.
- `automated_switchover` defaults to disabled; a config that sets no `switchover`
  block behaves exactly as FR-2.3 "else" columns (would-be failovers become
  `pause_queue`).
- Invalid action/reason strings are rejected at config load with a clear error.

## Milestone 4 — Project-wide failover & the disabled-mode pause path

**Description.** Wire the dispatcher to the policy. On a failure classified with
reason R for provider P:
- Look up the effective action. For `failover` (automated switchover enabled):
  switch **every agent whose effective active provider is P** to its secondary in
  a **single** action (FR-3.1) via `SwitchAllAgentProviders`, record per-agent
  failover detail incl. `resets_at_unix`+`bucket` for rate-limit (FR-3.3),
  **restart the interrupted job** and continue the remaining queue on the
  secondary in order (FR-3.2, NFR-3). Agents bound to P with **no** secondary
  cannot fail over — their jobs `pause_queue` while others proceed; this partial
  state is recorded and surfaced (FR-3.4). Failover is capped at one level; if
  the secondary also fails, affected jobs `pause_queue` (FR-3.5, NFR-6).
- For `pause_queue` (incl. automated switchover disabled, FR-4): pause preserving
  order, wait until work is possible (reachability/reset), **restart the failed
  job first**, then continue in queued order.
- For `fail_run`: fail the run, no switch, no pause.

**Files to change.**
- `internal/queue/dispatcher.go` — replace the boolean auto-switch branch with
  action dispatch; add project-wide switch (call
  `Project.SwitchAllAgentProviders` for provider P); restart-interrupted-job
  logic; partial-pause bookkeeping; one-level cap enforced via active-vs-primary
  state (an agent already failed over that fails again → `pause_queue`).
- `internal/project/provider_switch.go` — `SwitchAllAgentProviders` gains a
  "by provider P → each agent's own secondary" variant (agents may differ in
  secondary), recording per-agent `resets_at_unix`/`bucket`/`reason`.
- `internal/agent/errors.go` — ensure the classifier surfaces `resets_at_unix`
  + `bucket` for rate-limit to the dispatcher (reuse
  [[rate-limit-event-detection]]).

**Acceptance criteria.**
- A simulated 529/429 on P with automated switchover **enabled** moves every
  agent on P to its secondary in one action, restarts the interrupted job, and
  drains the rest of the queue on the secondary in order.
- With automated switchover **disabled**, the same failure pauses in order,
  waits, restarts the failed job, then continues in queued order.
- An agent on P with no secondary has its jobs paused while other agents proceed;
  the partial pause is present in the status API (Milestone 8).
- A secondary failure after failover pauses (no third target); no cyclic
  switching (NFR-6).

## Milestone 5 — `provider_disconnected`: retry-in-place, backoff, bounded pause

**Description.** Handle mid-stream disconnects inside the turn loop **before**
the run goroutine returns, so completed session context is not discarded (FR-6.1).
Because the chat-completions request re-sends the full `messages` array each turn
(`openai_compatible.go:141`), a retry is a byte-identical resend. Add exponential
backoff 2s/8s/30s between attempts (FR-6.2). Record the SSE line count at failure
so a post-first-token retry is distinguishable from a free pre-first-token retry
(FR-6.5); per Resolved Question 2, a **post-first-token disconnect still
auto-retries**. Maintain a **per-provider** disconnect counter over a rolling
1 hour, persisted in `operations.yaml` and surviving restart (FR-6.4, NFR-5);
**more than 3 in the hour** pauses the queue (FR-6.3). Per Resolved Question 1,
disconnects inside the active backoff window **collapse to a single occurrence**
so one incident cannot trip the threshold instantly.

**Files to change.**
- `internal/agent/openai_compatible.go` — at the stream-error path
  (`doneCh <- scanErr; return`, ~line 501-523) retry in-loop with backoff before
  returning; count SSE lines emitted this turn and log them; emit a
  `provider_disconnected` event carrying the provider name and pre/post-first-token
  flag.
- `internal/project/operations.go` — `RecordDisconnect(provider, at)` with
  backoff-window collapse, `DisconnectCountSince(provider, since)`.
- `internal/queue/dispatcher.go` — on `provider_disconnected`, apply
  `retry_in_place` then `pause_queue` once the rolling-hour count exceeds 3.

**Acceptance criteria.**
- A single disconnect mid-run retries in place (2s/8s/30s) and, on success,
  completes the run without losing prior turns.
- Two disconnects inside one backoff window count as one occurrence.
- The 4th distinct disconnect for a provider within a rolling hour pauses the
  queue; the counter is reloaded intact after a simulated restart.
- The run log records the SSE line count at failure.

## Milestone 6 — Reachability tracking in all modes (incl. single provider)

**Description.** The recovery prober currently builds its target list only from
agents already in failover and returns early when empty
(`recovery_prober.go:84-99`), so single-provider mode is never probed (FR-5.1).
Change it to probe **every configured provider in use by any agent**, in every
mode, writing reachability (`healthy`, `last_probed_at`, `since`) to
`operations.yaml` (FR-5.2) and surfacing it via the status API. The "recovered"
signal must be **quota-gated**: do not announce recovery for a provider that
failed over on rate-limit until the recorded `resets_at_unix` has passed
(FR-9.3 backend half) — two healthy `GET /v1/models` probes alone are
insufficient.

**Files to change.**
- `internal/project/recovery_prober.go` — build the probe set from all providers
  bound to agents (not just `PrimaryProvider != ""`); write reachability to the
  operations store each cycle; gate `provider.primary_recovered` on
  `resets_at_unix` for rate-limit failovers.
- `internal/project/operations.go` — reachability accessors (Milestone 1).

**Acceptance criteria.**
- In single-provider mode the provider is probed and its reachability appears in
  the status API (previously never probed).
- After a quota failover, `primary_recovered` is **not** broadcast before the
  recorded reset time, even with healthy `/v1/models` probes.

## Milestone 7 — Restart semantics & the partial-commit race

**Description.** On restart of an interrupted job, check whether it produced
partial commits (FR-7.1). No partial commit → clean re-run from the beginning
(FR-7.2). Partial commit suspected → do **not** auto-run and do **not**
auto-rollback; **surface the job to the operator** for a decision (FR-7.3),
recording the suspicion in operations state and broadcasting so the UI can prompt.

**Files to change.**
- `internal/git/` — add a helper to detect commits attributable to the failed
  run since it started (e.g. by run-id trailer / author / time window).
- `internal/queue/dispatcher.go` — branch restart on the partial-commit check;
  emit an operator-decision state instead of re-running when a partial commit is
  suspected.
- `internal/project/operations.go` — record the "awaiting operator decision"
  flag on the affected job/agent.

**Acceptance criteria.**
- A job with no partial commit is cleanly restarted.
- A job with a suspected partial commit is surfaced to the operator, neither
  auto-rerun nor auto-rolled-back; the pending decision is visible in the status
  API.

## Milestone 8 — Manual switch/failback API, status & policy inspection

**Description.** Update the HTTP surface (FR-8). The status response is built
from `operations.yaml`: current active vs primary per agent, `reason`,
`switched_at`, `resets_at_unix`, `bucket`, reachability (all modes), and any
FR-3.4 partial pause (FR-8.4). Add manual project-wide switch (all agents on a
chosen provider → their secondary) and manual failback to primary (FR-8.1), plus
a route exposing the **effective** event→action policy incl. defaults (FR-2.4).
A manual switch requested while **any run is executing** is **rejected with a
warning listing the running jobs** (FR-8.2). Failback is manual only (FR-8.3).
All mutating routes require `devops` or `product-owner` (FR-8.4).

**Files to change.**
- `internal/http/provider_switch.go` — rebuild `handleGetFailoverStatus`
  (`:57-90`) from the operations store (add reachability, reason, switched_at,
  reset fields, partial-pause); add `handleGetSwitchoverPolicy`; guard
  `handleSwitchAllProviders`/manual switch on in-flight runs (reject + list
  running jobs); keep restore/failback handlers pointing at operations state.
- `internal/http/server.go` — register the policy route and any new routes under
  `/p/{project}/...`; enforce `RequireRole(devops, product-owner)` on mutations.
- `internal/http/provider_switch_test.go` — update handler tests.

**Acceptance criteria.**
- The status response reflects `operations.yaml` (active vs primary, reason,
  switched_at, resets_at_unix, bucket, reachability, partial pause).
- The effective policy (with defaults) is retrievable via the API and lists an
  action for every classified reason.
- A manual switch while a run is executing returns a rejection naming the running
  jobs; no switch occurs.
- Mutating routes reject callers lacking `devops`/`product-owner`.

## Milestone 9 — Observability: application log + `internal/reports` aggregation

**Description.** Every switchover/failover/failback/pause/retry transition is
written to `operations.yaml` (state) **and** the application log (FR-10.1). Extend
`internal/reports` to aggregate, per agent/provider: failover count, which
provider caused each failover, time spent on the secondary, and time-to-restore
(FR-10.2).

**Files to change.**
- `internal/project/provider_switch.go`, `internal/queue/dispatcher.go`,
  `internal/agent/openai_compatible.go` — structured `slog` lines at each
  transition (provider/model **names** only, NFR-1).
- `internal/reports/failover_report.go` (new) + wire into the reports API
  (`internal/reports/agent_usage.go` patterns) — failover-history aggregation
  sourced from operations history / feed events.

**Acceptance criteria.**
- Each transition appears in the application log with no secret material.
- The reports aggregation reports failover count, causing provider, time on
  secondary, and time-to-restore per agent/provider.

---

## Cross-cutting acceptance (from the requirement)

- Carried-over tests pass: `TestFailover_AutoSwitch_HTTP529`,
  `TestFailover_AutoSwitch_RateLimitQuota`, `TestSecrets_FailoverAudit`.
- No `lifecycle/config.yaml` write and no git commit on any switch/failover/failback.
- All runtime state in git-ignored root `operations.yaml`, atomic writes, secrets
  absent. See [[agent-switchover-and-failover]] frontend and test plans.
