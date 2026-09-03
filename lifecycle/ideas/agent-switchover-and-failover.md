---
title: agent switchover and failover
type: idea
status: draft
lineage: agent-switchover-and-failover
created: "2026-08-28T00:00:00+10:00"
labels:
    - agent
    - queue
    - failover
    - providers
    - reliability
release: KC-Release6
---

## Idea

### agent switchover and failover Features  
* Manual agent switch over  
* Automated agent failover over  
* Ratelimit queue pause  
* Tool deny list queue pause [[improved-bash-allow-lists]]
* Provider outage queue pause  
* Provider availability monitoring  
* Provider failure mode monitoring  
### Provider Failure Scenarios  
1. Provider not responding  
2. provider overloaded HTTP 529   
3. Account token/quota limit reached (RateLimitQuota) HTTP 429  
### Agent and Provider Modes of Operation  
#### Single provider   
Single provider configured to use the same model for all agents or different models for different agents.  No agent switchover, on failure the queue will pause.  When an issue is detected with the provider, the queue will be paused, until work is possible again and failed jobs will be restarted.  kaos-control should keep a track of provider reachability.  
#### Multiple providers  
Multiple providers configured to use different models for different agents.  No agent switchover, on failure the queue will pause.  
#### Manual agent switchover mode  
There are 2 or more agent configurations available, when required a human decides to switch over between different agent configurations, e.g. switching from Claude to Gemini because you have run out of tokens.  On failure the queue will pause and the operator can decide to switch to a different agent configuration.  
#### Automated agent failover mode  
Each agent has a primary and secondary provider and model defined, when an agent hits a failure mode it switches to the secondary.  It continues to use the secondary until that provider fails or the CLI or GUI is triggered to fail back to primary.

## Gaps and open questions

Raised from a review of the idea against the current implementation
(2026-08-28). Each item names the code it was verified against.

### Blocker — automated failover does not currently work at all

[[automated-failover-always-rejected]]. `SwitchAgentProvider` patches `provider`
to the fallback but leaves `fallback_provider` unchanged, producing
`provider == fallback_provider`, which project-config validation rejects. The
write is refused, `tryFailover` returns false, and the queue pauses — the exact
outcome failover exists to prevent. The condition is unavoidable, because
failover only engages when `fallback_provider` is set. Nothing in this idea can
be demonstrated until that is fixed.

**But note how item 7 changes the fix.** That defect proposes either clearing or
swapping `fallback_provider` so the patched config still validates. Both assume
failover keeps rewriting `lifecycle/config.yaml`. Item 7 decides it must not:
configuration is not modified for failover, and operational state moves to
`operations.yaml` in the project root. If nothing patches the config, then
`PatchAgentProviders` is never called and the `fallback_provider must differ from
provider` rejection cannot arise — the defect is resolved as a consequence of the
design rather than by either fix it proposes.

The requirement should therefore absorb this defect rather than wait on it. The
defect [[automated-failover-always-rejected]] has been **abandoned** on that
basis (2026-09-03), and its two proposed options should not be implemented.

Its three failing integration tests carry over as acceptance criteria for this
requirement: `TestFailover_AutoSwitch_HTTP529`,
`TestFailover_AutoSwitch_RateLimitQuota` and `TestSecrets_FailoverAudit` must
pass once failover works.

### 1. Quota is per-account, but failover is per-agent

`fallback_provider` is an **agent** field (`internal/config/config.go:649`), so
failover is decided one agent at a time. A 429 / quota exhaustion is a property
of the **provider account**, shared by every agent bound to it.

With eight agents on one provider, exhausting it means eight separate job
failures, eight health probes, eight config rewrites and eight git commits —
each discovered only when that agent's next job runs and fails.

**Open question:** should provider-level exhaustion fail over every agent bound
to that provider in one action, rather than one at a time?

> The queue has been setup to be executed in that order.  When exhaustion for any agent the queue should be paused so the jobs can be executed in order.  The system should then wait until there are tokens available again and restart the job which failed and then continue processing the queue as jobs were queued.

**Decided — the switch is project-wide, not per-agent.** Which of the two
behaviours applies is chosen by the event→action configuration (see *Product
Owner Additional Thoughts*):

| Automated switchover | Behaviour on provider exhaustion |
|---|---|
| **disabled** | Pause the queue, preserving order. Wait until tokens are available, restart the job that failed, then continue processing in the order queued. |
| **enabled** | Switch **every agent** bound to that provider to its secondary in one action. Restart the job that was running; the rest of the queue continues on the secondary. |

This resolves the open question above: because a quota is a property of the
provider account rather than of one agent, failover operates on **all agents
using that provider at once**, not one at a time as each subsequent job fails.

### 2. "Track provider reachability" is not met by what exists

The *Single provider* section above requires kaos-control to track provider
reachability. The recovery prober is the only component that probes, and it
builds its target list from agents where `primary_provider != ""` — i.e. **only
agents already in failover** — returning early when that set is empty
(`internal/project/recovery_prober.go:79-99`).

In single-provider mode, nothing is ever probed. This is a new requirement, not
existing behaviour.

> Yes, this is a new requirement.

### 3. Failback has no trustworthy signal

The idea says failover continues until the CLI or GUI triggers a fail back, but
does not say what tells the operator it is safe.

Today the signal is `provider.primary_recovered`, broadcast after two
consecutive healthy probes. The probe is `GET /v1/models`, treating any
`StatusCode < 500` as healthy (`internal/agent/health_probe.go:23`). Model
listing is not quota-gated, so a fully rate-limited provider passes — and so
would a 429. **After a quota failover, recovery is declared within roughly two
minutes regardless of whether the quota has reset.**

The authoritative signal already arrives and is used for the queue pause but not
for failback: `resets_at_unix`, together with a `bucket` of `five_hour` or
`weekly`.

**Proposed:** record the expected reset time on failover, suppress the failback
signal until it passes, and probe something quota-gated (a 1-token completion)
rather than `/v1/models`.

> At this time failback will only be triggered manually.

**Why this needs a UI answer.** Making failback manual removes the risk of
switching back automatically on a bad signal, but the operator still has to judge
when the primary is usable — and the signal available today is wrong for the
quota case: `provider.primary_recovered` fires after two healthy probes of
`GET /v1/models`, which is not quota-gated, so it reports recovery within roughly
two minutes of any quota failover regardless of the actual reset.

**Resolved by the answer below.** A status button showing "Primary Agents" or
"Secondary Agents", plus a dedicated screen to inform the failback decision.

**Constraint for that screen:** it must not simply surface
`provider.primary_recovered`, or it reproduces the misleading green light in a
new place. The authoritative data already arrives on the rate-limit event —
`resets_at_unix` and a `bucket` of `five_hour` or `weekly` — so the screen should
show the expected reset time, and qualify or suppress any "recovered" indicator
until it has passed.

> When primary and secondary agents are configured, the GUI should show which is the current state, e.g. "Primary Agents" or "Secondary Agents" in a status button the GUI.  An additional screen with the content described would be great to help assist when to failback.

### 4. Failure scenarios missing from the list of three

The three scenarios above map to the three implemented kinds
(`validSwitchOnKinds`: `unreachable`, `overloaded`, `rate_limit`), with one
nuance: **`unreachable` also covers 502/503/504 gateway errors**, not only a
non-responding provider.

Not covered by the idea, and not currently failover triggers:

| Scenario | Today | Why it matters |
|---|---|---|
| Auth / credential failure | separate `queue.auth_error` path, never fails over | an expired key on the primary is exactly when the fallback is wanted |
| Model unloaded / not found | classified, not a trigger | llama.cpp swapping resident models |
| Mid-stream disconnect (`provider_disconnected`) | classified, not a trigger | common with local models; cost a 17-minute run on 2026-08-27 |
| Model cannot do tool calling | detected by preflight | see item 5 |
| Degraded but responding (very high TTFT) | undetected | a fallback 10x slower is not "working" |

Each needs an explicit decision: trigger failover, pause the queue, or fail hard.

> "Auth / credential failure" is operational, that should trigger a failover to secondary.  The other issues I consider these issues to be setup issues, e.g. when something is modified and the user should have verified they are working before setting up alot of work to be done.  

**`provider_disconnected` does not fit the "setup issue" grouping.**
Model-not-found and cannot-do-tool-calling are setup issues and are settled by
the answer above and by item 5. A mid-stream disconnect is different: it has
occurred twice on a correctly configured system with no change of any kind
between the working and failing runs.

| Run | Outcome |
|---|---|
| `97078a4c1bf40c04` | Died at turn 20 after 19 turns of real work. The provider's own log showed no error, cancel, timeout, eviction or restart, and the final request completed normally — the client received zero bytes and a connection reset. |
| `8f15fc7f0fe9afa9` | Same class of failure, 17 minutes lost. |

Today it neither triggers failover nor pauses the queue: the run simply fails and
its work is discarded.

**Decided: retry in place, and pause only if it keeps happening.**

> Retry when the provider disconnects. If it disconnects more than 3 times in
> 1 hour, pause the queue.

This supersedes the initial "pause the queue" answer below, which would have
discarded a run's completed work on the first disconnect — in
`97078a4c1bf40c04` that was nineteen turns.

**Why a retry is safe and cheap.** The chat-completions API is stateless: the
driver holds the conversation in its own memory and re-sends the entire
`messages` array every turn (`internal/agent/openai_compatible.go:141`). A retry
therefore sends a byte-identical request — the model receives exactly the
context it would have had. There is no session to lose and nothing to restore.

On a local provider the prompt cache also survives. From leia's own log, a
request arriving after a **seven-minute** idle gap still matched its slot and
reused 36,658 of 39,926 tokens:

```
checking sim = 0.918 (36658/39926) > 0.100
selected slot by LCP similarity, f_sim_best = 0.918
cached n_tokens = 36658
```

A retry moments after a disconnect has an identical prefix, so it should reuse
more still. On a cloud provider the picture is weaker but still correct: the
retry may be routed to a different upstream (OpenRouter re-evaluates routing
roughly every five minutes), which means a cold cache and full prompt
reprocessing — correct, but not free.

**Implementation notes for the requirement:**

- **Retry in place, inside the turn loop.** The current stream-error path does
  `doneCh <- scanErr; return`, which exits the run goroutine — and `messages` is
  local to it. Retrying after that point is impossible; the retry must happen
  before the return, or the conversation really is gone.
- **Retry is free only before the first token.** Once tokens have streamed, a
  retry re-bills the whole prompt and discards partial output. The run log now
  records the SSE line count at the point of failure, so this is decidable.
- **The 3-per-hour counter needs a defined scope.** Disconnects are a property
  of the provider, not of one run, so counting per provider over a rolling hour
  is the natural reading — a run that succeeds after one retry still contributes
  to the count. The counter is operational state and belongs in
  `operations.yaml`, which also means it should survive a kaos-control restart.
- **Backoff between attempts is required.** Without it, three retries can land
  within a second and pause the queue on what was really a single incident —
  spending the whole hourly budget instantly and defeating the intent of
  "3 times in 1 hour". The requirement should specify the schedule (an
  exponential backoff, e.g. 2s / 8s / 30s, is the obvious starting point) and
  state whether closely-spaced failures count once or individually toward the
  hourly threshold.

> provider_disconnected = Pause the queue.

### 5. Failover ignores model capability

A switch requires only a health probe. The tool-calling preflight is **not**
consulted anywhere in the switch path, so an agent can fail over onto a model
that silently drops the `tools` parameter — precisely the failure mode that
makes a local model unusable as an agent.

This conflicts with the project's standing rule that an LLM which cannot follow
the architecture and standards is not a suitable agent.

**Proposed:** passing the tool-calling preflight is a precondition for any
switch target, manual or automated.

> It can be assumed that the primary and secondary agents and models have already been verified before this is put into full production.

### 6. Only one fallback level

`fallback_provider` is a single field and `max_failovers_per_run` defaults to 1,
so there is no chain. The *Multiple providers* mode implies more than two.

**Open question:** is a tertiary provider in scope, and what is the ordering
rule?

> At this time a primary and secondary will be sufficient.

### 7. Failover state is written to git

Each switch rewrites `lifecycle/config.yaml` and commits it. Worth confirming as
a deliberate decision: it places runtime state under version control, adds
commit noise, and — because lifecycle files replicate through Obsidian/Unison —
allows an external sync to race a failover write.

**Open question:** should failover state live outside the committed config
(e.g. in the index or a runtime state file), with config remaining the declared
intent?

> Yes, operational state will not be committed to git.  The configuration should not be modified to support failover, the operational state should be tracked in a seperate file, e.g. operation.yaml in the lifecycle directory. How the system is currently operating will be maintained in that file.  The UI can use it to display current status and any other necessary messages for the user.

**Settled details.** The file is **`operations.yaml` in the project root**, not
under `lifecycle/` (see the answer below; item 9 also used the plural spelling,
which is canonical).

Placing it at the root resolves the replication concern properly rather than
working around it: `lifecycle/` is the tree that replicates through
Obsidian/Unison, so live operational state kept there could be overwritten by an
external sync. The root is outside that tree.

One thing still required: **it must be added to `.gitignore`.** Nothing currently
ignores it at the root any more than under `lifecycle/`, so without an explicit
entry the requirement above — operational state is not committed to git — is not
met.

> Lets move operations.yaml to the root folder of the project, OUT of lifecyle.

### 8. Manual switchover and in-flight runs

The idea does not say what happens to a run already executing when an operator
switches an agent: killed and requeued, or allowed to drain first?

> The user should be warned of the running jobs and the switchover rejected, the user can then decide how they want to handle the running and queued jobs.

**Restart semantics need defining.** Both this section and item 1 say the failed
job is restarted. Agents are not reliably idempotent — a run may already have
written artifacts, or committed them, before it failed. Run
`2073eaa29f90f088` completed its work and committed it one second before the
process died, and was still recorded as failed; re-running it would have
duplicated the work.

The requirement needs to state:

- whether "restart" means re-running the agent from the beginning, or resuming;
- what happens when the failed job already produced output — detect and skip,
  re-run regardless, or surface it to the operator;
- and whether that answer differs for a job interrupted mid-generation (nothing
  written) versus one interrupted after its files landed.

> Regarding restart semantics: this is a difficult race condition, it is unusual for a human developer to die right after a partial commit.  The cleanest way is when the agent for job has partially commited content and dies, the partially commited work should be rolled back and the whole job restarted.  So kaos-control should check the status of the partial job and if it suspects there is partial commits then it should ask the user what they want to do.  The human can then investigate.

### 9. Observability

Not mentioned, and needed to tell whether any of this is working: how often each
agent fails over, which provider caused it, how long it stayed on the fallback,
and time-to-restore. The `internal/reports` aggregation is the natural home.

> Operational state should be stored in a yaml file. e.g. operations.yaml and transitions recorded in the application log.  Saving other records into the internal/reports aggregation is a good idea.

## Product Owner Additional Thoughts

Configuration for how the switchover and failover features should be working is required.  This should include the preferred mode of operation and can provide a list of detected events and which action should be taken, e.g. pause queue or failover to secondary provider.

To make that event list complete by construction, it should be enumerated
against the failure reasons the system already classifies, so each gets an
explicit action rather than falling through to a default:

`rate_limit`, `overloaded`, `unreachable`, `auth_error`, `provider_disconnected`,
`model_not_found`, `model_unloaded`, `tools_unsupported`,
`context_window_exceeded`, `turn_token_ceiling`, `max_iterations_reached`,
`timeout`.

Actions to choose from: fail over to secondary, pause the queue, retry in place,
or fail the run outright.

When switching between the primary and secondary agents they should be done together, e.g. the backend developer agent is running a job, and the provider fails, if automated switchover is enabled, all agents are switched to the secondary, the job which was currently running restarts and the rest of the jobs are running on the secondary provider.
