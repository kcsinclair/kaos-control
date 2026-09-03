---
title: Agent Switchover and Failover
type: requirement
status: blocked
lineage: agent-switchover-and-failover
created: "2026-09-03T11:00:00+10:00"
priority: high
parent: lifecycle/ideas/agent-switchover-and-failover.md
labels:
    - agent
    - queue
    - failover
    - providers
    - reliability
    - backend
    - frontend
    - operability
release: KC-Release6
assignees:
    - role: analyst
      who: agent
    - role: product-owner
      who: agent
---

# Agent Switchover and Failover

Parent: [[agent-switchover-and-failover]].
Supersedes the shipped design of [[switch-provider-2]] (which rewrote
`lifecycle/config.yaml` and committed each switch to git). Absorbs the abandoned
defect [[automated-failover-always-rejected]].

---

## Problem

When an upstream AI provider fails — not responding, overloaded (HTTP 529), or
quota-exhausted (HTTP 429) — orchestrated agent runs and the global work queue
stall or fail. kaos-control must keep work moving (or pause it cleanly, in order)
according to an operator-chosen policy, and let operators move a whole set of
agents between provider configurations without editing files by hand.

Three concrete problems block this today:

1. **Automated failover does not function.** The previous design patched an
   agent's `provider` to the fallback but left `fallback_provider` unchanged,
   producing `provider == fallback_provider`, which project-config validation
   rejects. The write is refused, `tryFailover` returns false, and the queue
   pauses — the exact outcome failover exists to prevent
   ([[automated-failover-always-rejected]]).
2. **Runtime state is written to git.** Each switch rewrote
   `lifecycle/config.yaml` and committed it, placing operational state under
   version control, adding commit noise, and — because `lifecycle/` replicates
   through Obsidian/Unison — letting an external sync race a failover write.
3. **Reachability and failback signals are wrong or absent.** In single-provider
   mode nothing is probed at all; and the recovery signal
   (`provider.primary_recovered`, from two healthy `GET /v1/models` probes) is not
   quota-gated, so it reports "recovered" within ~2 minutes of a quota failover
   regardless of the real reset.

---

## Goals / Non-goals

### Goals

- Support four **modes of operation**: single provider, multiple providers,
  manual switchover, and automated failover.
- Drive all switchover/failover/pause decisions from an explicit, operator-owned
  **event → action policy** enumerated against every failure reason the system
  already classifies — no reason falls through to a default.
- Treat provider failover as a **project-wide** action: switching moves *every
  agent bound to the affected provider* to its secondary in one step, not one
  agent at a time.
- Persist all runtime/operational state in **`operations.yaml` at the project
  root** (outside `lifecycle/`, git-ignored); never modify `lifecycle/config.yaml`
  for failover.
- Track **provider reachability in every mode**, including single-provider mode.
- Make **failback manual only**, backed by a status button and a dedicated screen
  that shows the authoritative expected-reset time rather than a misleading green
  light.
- Retry **`provider_disconnected`** in place cheaply, and pause the queue only if
  it recurs beyond a bounded threshold.
- Preserve queue **ordering** on a pause; on failover, restart the interrupted
  job and continue the rest of the queue on the secondary.
- Record failover activity in the **application log** and the
  **`internal/reports`** aggregation for observability.

### Non-goals

- Automatic failback (explicitly deferred; failback is manual for this release).
- A third (tertiary) fallback level — primary + secondary only.
- Re-validating model capability in the switch path (tool-calling preflight):
  primary and secondary agents/models are assumed verified before production use.
- Implementing the underlying error classification (`extractRateLimitText`,
  `validSwitchOnKinds`) — already shipped ([[rate-limit-event-detection]]).
- Multi-provider round-robin / active load balancing across providers.
- Managing external provider accounts, billing, or quotas.

---

## Detailed Requirements

### Architecture-Breaking Requirements

Evaluated against [architecture-summary.md](../architecture/architecture-summary.md),
the recorded ADRs, and the standards.

1. **Runtime operational state moves out of committed config
   (`operations.yaml`).** — *Deliberate reversal of the shipped
   [[switch-provider-2]] design.* Live "how the system is currently operating"
   state (which agents are on primary vs secondary, expected reset times,
   disconnect counters, failover history) is written to `operations.yaml` **at the
   project root**, not to `lifecycle/config.yaml`, and is **not committed to git**.
   - *Against "Local filesystem is the source of truth"* (architecture-summary):
     **Consistent, not broken.** `lifecycle/` markdown remains authoritative for
     *artifacts*; `lifecycle/config.yaml` remains the authoritative *declared
     intent*. `operations.yaml` is a new category — authoritative *runtime state*,
     analogous in role to the rebuildable SQLite index ([[index-is-a-cache]]) but
     persisted so it survives a restart. It lives at the project root precisely so
     that Obsidian/Unison replication of `lifecycle/` cannot overwrite live state.
   - *Git hygiene:* the root `operations.yaml` is **already listed in
     `.gitignore`** (line 27), so the "not committed to git" requirement is met
     without further change. This must be asserted by a test.
   - *ADR:* no recorded ADR or standard mandates storing failover state in git, so
     this requirement is met within the recorded architecture and **no new ADR is
     mandatory.** Because it reverses the shipped behaviour of a `done`
     requirement, whether to record it as an ADR is raised under Open Questions.
2. **Single self-contained binary.** — **Satisfied.** All logic is Go stdlib +
   `gopkg.in/yaml.v3` + the embedded SPA; no external datastore, daemon, proxy, or
   cgo dependency ([[adr-0003-pure-go-sqlite-index]],
   [[adr-0004-embedded-spa-single-binary]]).
3. **Offline operation.** — **Satisfied.** A secondary may be a local provider
   (Ollama / llama.cpp), so failover can keep working with no internet.
4. **Secrets hygiene.** — **Satisfied.** `operations.yaml`, API responses, log
   lines, and the reports aggregation store only non-secret provider/model
   *names*; API keys stay in `~/.kaos-control/config.yaml` and are never written to
   the project tree or logged ([[secrets-handling]]).
5. **Mediated driver & sandboxing.** — **Satisfied.** Switching an agent's backing
   provider does not alter its `allowed_write_paths`, `bash_allowlist`, or tool
   mediation; those bind to the role and are enforced at the tool-call boundary
   regardless of the provider ([[adr-0006-mediated-agent-driver-permission-model]],
   [[filesystem-sandboxing]]).

**Conclusion:** No architectural constraint, ADR, or standard is violated. No new
ADR is required to proceed.

---

### Functional Requirements

#### FR-1: Modes of operation

The project operates in exactly one mode, derived from configuration:

| Mode | Definition | On provider failure |
|---|---|---|
| **Single provider** | One provider for all agents (same or different models). No secondary configured. | Pause the queue (order preserved); track reachability; resume when work is possible; restart the failed job. |
| **Multiple providers** | Different providers for different agents; no secondary configured. | Pause the queue for the affected provider's agents (order preserved). |
| **Manual switchover** | ≥2 provider configurations available; a human decides when to switch. | Pause on failure; operator switches all agents on the affected provider to their secondary. |
| **Automated failover** | Each agent has a primary and secondary provider/model; failover is automatic. | Switch every agent bound to the failing provider to its secondary in one action (FR-3). |

- FR-1.1: An agent's primary and secondary are declared in `lifecycle/config.yaml`
  as `{provider, model}` and `{fallback_provider, fallback_model}` respectively.
  Only a **single** fallback level is supported.
- FR-1.2: `lifecycle/config.yaml` is **never** rewritten by switchover or failover.
  It is the declared intent only.

#### FR-2: Event → action policy

- FR-2.1: The project config carries a `switchover` policy block with:
  - a top-level `automated_switchover` toggle (`enabled` / `disabled`), **disabled
    by default** (documentation must call this out prominently), and
  - an `events` map assigning exactly one action to **each** classified failure
    reason. The map is complete by construction: every reason the system already
    classifies has an explicit entry.
- FR-2.2: The available actions are: **`failover`** (switch to secondary),
  **`pause_queue`**, **`retry_in_place`**, **`fail_run`**.
- FR-2.3: Default action per reason (operator-overridable except where noted):

  | Reason | Default action | Notes |
  |---|---|---|
  | `rate_limit` (429 / quota) | `failover` if automated switchover enabled, else `pause_queue` | Provider-account scoped → project-wide (FR-3). Records `resets_at_unix` + `bucket`. |
  | `overloaded` (529) | `failover` if enabled, else `pause_queue` | |
  | `unreachable` (connect refused, 502/503/504) | `failover` if enabled, else `pause_queue` | `unreachable` covers gateway 5xx, not only a dead provider. |
  | `auth_error` | `failover` (operational failure — expired key is exactly when the secondary is wanted) | If automated switchover disabled, `pause_queue`. |
  | `provider_disconnected` | `retry_in_place`, then `pause_queue` on threshold (FR-6) | |
  | `model_not_found` | `fail_run` | Setup issue; assumed verified before production. |
  | `model_unloaded` | `fail_run` | Setup issue. |
  | `tools_unsupported` | `fail_run` | Setup issue; capability assumed pre-verified. |
  | `context_window_exceeded` | `fail_run` | Run-level; a different provider will not help. |
  | `turn_token_ceiling` | `fail_run` | Run-level limit. |
  | `max_iterations_reached` | `fail_run` | Run-level limit. |
  | `timeout` | `fail_run` | Run-level limit. |

- FR-2.4: A reason with no explicit config entry uses the FR-2.3 default; the
  effective policy (including defaults) is inspectable via the API (FR-8).

#### FR-3: Project-wide failover (automated switchover enabled)

Quota/overload/unreachable failures are properties of the **provider account**,
shared by every agent bound to it. Therefore:

- FR-3.1: When a failover-triggering reason occurs for a provider `P`, **all
  agents whose active provider is `P`** move to their configured secondary in a
  **single** action — not one-at-a-time as each subsequent job fails.
- FR-3.2: The job that was running when the failure occurred is **restarted**
  (subject to FR-7). The remaining queued jobs continue on the secondary,
  preserving queue order.
- FR-3.3: Failover records, per affected agent, in `operations.yaml`: the run that
  triggered it, agent name, primary `{provider, model}`, active `{provider,
  model}`, `switched_at`, `reason`, and (for rate-limit) `resets_at_unix` +
  `bucket`.
- FR-3.4: If any agent bound to `P` has no secondary configured, that agent cannot
  fail over; its jobs `pause_queue` while other agents proceed on their
  secondaries. This partial state must be visible in the status API and UI.
- FR-3.5: Failover is capped at **one** level (primary → secondary). If the
  secondary also fails, the affected jobs `pause_queue` (no third target exists).

#### FR-4: Automated switchover disabled — pause the queue in order

- FR-4.1: On a would-be-failover reason, pause the queue preserving order, wait
  until work is possible again (reachability / reset time), **restart the job that
  failed**, then continue processing in the order queued.

#### FR-5: Reachability tracking in all modes (including single provider)

- FR-5.1: kaos-control tracks provider reachability in **every** mode. This is a
  new requirement: the recovery prober currently builds its target list only from
  agents already in failover (`primary_provider != ""`) and returns early when
  empty (`internal/project/recovery_prober.go:79-99`), so single-provider mode is
  never probed.
- FR-5.2: Reachability state (last-probed, healthy/unhealthy, since) is written to
  `operations.yaml` and surfaced via the status API/UI.

#### FR-6: `provider_disconnected` — retry in place, bounded pause

A mid-stream disconnect on a correctly configured system (observed twice, e.g.
runs `97078a4c1bf40c04` and `8f15fc7f0fe9afa9`) must not silently discard a run's
completed work.

- FR-6.1: **Retry in place, inside the turn loop, before the run goroutine
  returns.** The chat-completions API is stateless — the driver re-sends the full
  `messages` array each turn (`internal/agent/openai_compatible.go:141`) — so a
  retry sends a byte-identical request and loses no session. The current
  stream-error path (`doneCh <- scanErr; return`) exits the goroutine and discards
  the local `messages`; the retry must happen before that return.
- FR-6.2: **Backoff is required** between attempts (exponential, starting
  **2s / 8s / 30s**) so a single incident cannot trip the threshold instantly.
- FR-6.3: **Threshold pause.** If the provider disconnects **more than 3 times in a
  rolling 1 hour**, the queue is paused. Each disconnect event counts individually
  toward the threshold (backoff prevents a single incident from spending the budget
  at once); a run that succeeds after a retry still contributes its disconnect(s)
  to the count.
- FR-6.4: The counter is **per provider**, over a rolling hour, persisted in
  `operations.yaml`, and **survives a kaos-control restart**.
- FR-6.5: The run log records the SSE line count at the point of failure so that a
  retry after tokens have already streamed is distinguishable from a pre-first-token
  retry (the latter is free; the former re-bills the prompt).

#### FR-7: Restart semantics and the partial-commit race

An agent may complete and commit work moments before its process dies (observed:
run `2073eaa29f90f088`). Blindly re-running would duplicate work.

- FR-7.1: On restart, kaos-control **checks whether the failed job produced partial
  commits**.
- FR-7.2: If **no** partial commit is detected, the job is re-run from the
  beginning (clean restart).
- FR-7.3: If a partial commit **is suspected**, kaos-control does **not** auto-run
  and does **not** auto-rollback. It **surfaces the job to the operator** for a
  decision (the human can investigate). The recommended resolution is to roll back
  the partially committed work and restart the whole job, but the choice is the
  operator's.

#### FR-8: Manual switchover, failback, and status API

- FR-8.1: Manual switchover moves all agents on a chosen provider to their
  secondary (and the inverse: failback to primary). It is the operator equivalent
  of FR-3, applied on demand.
- FR-8.2: **In-flight runs block a manual switch.** If any run is executing when an
  operator requests a switch, the operator is **warned of the running jobs and the
  switchover is rejected**; the operator then decides how to handle the running and
  queued jobs before retrying.
- FR-8.3: **Failback is manual only** for this release. There is no automatic
  return to primary.
- FR-8.4: The backend exposes authenticated routes (mutations require `devops` or
  `product-owner`) to: read switchover/reachability/policy status, manually switch
  all agents on a provider to secondary, and manually fail back to primary. The
  status response reflects `operations.yaml` (current active vs primary per agent,
  reason, `switched_at`, `resets_at_unix`, `bucket`, reachability, and any partial
  pause from FR-3.4).

#### FR-9: Failback decision support (UI)

- FR-9.1: When primary and secondary are configured, the GUI shows a **status
  button** indicating the current state — e.g. **"Primary Agents"** or **"Secondary
  Agents"**.
- FR-9.2: A **dedicated failback screen** presents the information needed to judge
  when to fail back. It **must show the expected reset time** derived from the
  authoritative rate-limit data (`resets_at_unix` and `bucket` of `five_hour` or
  `weekly`).
- FR-9.3: The screen **must not** simply surface `provider.primary_recovered`. Any
  "recovered" indicator is **qualified or suppressed until the recorded reset time
  has passed**, because `provider.primary_recovered` today fires after two healthy
  `GET /v1/models` probes, which are not quota-gated and so report recovery within
  ~2 minutes of a quota failover regardless of the real reset.

#### FR-10: Observability

- FR-10.1: Every switchover/failover/failback/pause/retry transition is written to
  `operations.yaml` (current state) and recorded in the **application log**.
- FR-10.2: The `internal/reports` aggregation records, per agent/provider: failover
  count, which provider caused each failover, how long agents stayed on the
  secondary, and time-to-restore.

---

### Non-Functional Requirements

- **NFR-1 — Secrets:** `operations.yaml`, API payloads, logs, and reports contain
  only non-secret provider/model names; keys are masked/absent ([[secrets-handling]]).
- **NFR-2 — Atomic writes:** `operations.yaml` writes are atomic (temp file +
  rename) to survive a crash mid-write; the file is git-ignored (already true).
- **NFR-3 — Ordering:** a queue pause preserves the queued order; on resume the
  failed job is restarted first, then processing continues in order.
- **NFR-4 — Realtime:** switchover/failover/pause decisions and their WS broadcasts
  complete within **500 ms** of the triggering event so queued work resumes
  promptly.
- **NFR-5 — Restart durability:** disconnect counters, active-vs-primary state, and
  recorded reset times survive a kaos-control restart (they live in
  `operations.yaml`).
- **NFR-6 — Bounded failover:** at most one failover level; no cyclic switching
  between two broken providers.

---

## Acceptance Criteria

- [ ] The three integration tests carried over from
      [[automated-failover-always-rejected]] pass once failover works:
      `TestFailover_AutoSwitch_HTTP529`, `TestFailover_AutoSwitch_RateLimitQuota`,
      and `TestSecrets_FailoverAudit`.
- [ ] Automated failover no longer produces `provider == fallback_provider`;
      `lifecycle/config.yaml` is **not** modified and **no git commit** is created
      by any switchover, failover, or failback.
- [ ] All runtime state is written to **`operations.yaml` at the project root**;
      a test asserts the file is git-ignored and never appears in `git status`
      after a failover.
- [ ] A simulated 529 / 429 on provider `P` with automated switchover **enabled**
      moves **every agent bound to `P`** to its secondary in one action, restarts
      the interrupted job, and continues the rest of the queue on the secondary in
      order.
- [ ] With automated switchover **disabled**, the same failure pauses the queue in
      order, waits, then restarts the failed job and continues in queued order.
- [ ] An `auth_error` on the primary triggers failover to the secondary (when
      automated switchover is enabled).
- [ ] The event→action policy has an explicit entry for every classified reason
      (`rate_limit`, `overloaded`, `unreachable`, `auth_error`,
      `provider_disconnected`, `model_not_found`, `model_unloaded`,
      `tools_unsupported`, `context_window_exceeded`, `turn_token_ceiling`,
      `max_iterations_reached`, `timeout`); the effective policy is inspectable via
      the API.
- [ ] `provider_disconnected` retries in place with 2s/8s/30s backoff; the 4th
      disconnect within a rolling hour (per provider) pauses the queue; the counter
      survives a restart.
- [ ] Provider reachability is tracked and surfaced in **single-provider** mode
      (not only when an agent is already in failover).
- [ ] A manual switch requested while a run is executing is **rejected with a
      warning** listing the running jobs.
- [ ] On restart, a job with a suspected partial commit is **surfaced to the
      operator** rather than auto-rerun or auto-rolled-back; a job with no partial
      commit is cleanly restarted.
- [ ] The GUI status button shows **"Primary Agents"** / **"Secondary Agents"**,
      and the failback screen shows the expected **reset time** and does **not**
      show an unqualified "recovered" state before that time passes.
- [ ] Failover activity appears in the application log and in the `internal/reports`
      aggregation (count, causing provider, time on secondary, time-to-restore).
- [ ] Lineage: `parent: lifecycle/ideas/agent-switchover-and-failover.md`; related
      artifacts linked — [[switch-provider-2]], [[automated-failover-always-rejected]],
      [[rate-limit-event-detection]], [[improved-bash-allow-lists]],
      [[secrets-handling]].

---

## Open Questions

1. **Closely-spaced disconnect counting (FR-6.3).** The default is that each
   disconnect event counts individually toward the rolling-hour threshold, with
   backoff preventing a single incident from tripling instantly. Should tightly
   clustered disconnects (e.g. within the backoff window) instead collapse to a
   single occurrence? A crisp default is set; confirm or override.

> Yes, a backoff window is a good idea.

2. **Retry after first token (FR-6.5).** A pre-first-token retry is free
   (byte-identical resend); a post-first-token retry re-bills the whole prompt and
   discards partial output. Should a post-first-token disconnect still auto-retry,
   or immediately count toward the pause threshold without retrying?

> a post-first-token disconnect should auto-retry

3. **ADR for `operations.yaml`.** This requirement reverses the shipped
   [[switch-provider-2]] approach of writing failover state into
   `lifecycle/config.yaml` and git. No recorded ADR/standard mandates the old
   behaviour, so an ADR is not *required* — but should the "operational state lives
   in a git-ignored root `operations.yaml`, config is declared intent only"
   decision be recorded as an ADR under `lifecycle/architecture/decisions/` given it
   reverses a `done` requirement?
