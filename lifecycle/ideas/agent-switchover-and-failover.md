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

### 8. Manual switchover and in-flight runs

The idea does not say what happens to a run already executing when an operator
switches an agent: killed and requeued, or allowed to drain first?

> The user should be warned of the running jobs and the switchover rejected, the user can then decide how they want to handle the running and queued jobs.

### 9. Observability

Not mentioned, and needed to tell whether any of this is working: how often each
agent fails over, which provider caused it, how long it stayed on the fallback,
and time-to-restore. The `internal/reports` aggregation is the natural home.

> Operational state should be stored in a yaml file. e.g. operations.yaml and transitions recorded in the application log.  Saving other records into the internal/reports aggregation is a good idea.

## Product Owner Additional Thoughts

Configuration for how the switchover and failover features should be working is required.  This should include the preferred mode of operation and can provide a list of detected events and which action should be taken, e.g. pause queue or failover to secondary provider.

When switching between the primary and secondary agents they should be done together, e.g. the backend developer agent is running a job, and the provider fails, if automated switchover is enabled, all agents are switched to the secondary, the job which was currently running restarts and the rest of the jobs are running on the secondary provider.
