---
title: "Automated provider failover can never succeed — the switch writes a config that fails its own validation"
type: defect
status: draft
lineage: automated-failover-always-rejected
created: "2026-08-28T12:20:00+10:00"
priority: high
labels:
    - defect
    - queue
    - failover
    - providers
    - config
---

## Summary

Automated provider failover is non-functional. Every attempt is rejected by
config validation, so the dispatcher falls through to the standard rate-limit
pause — the exact outcome failover exists to avoid.

An agent that can fail over is configured with two different providers:

```yaml
provider: anthropic-cloud
fallback_provider: gemini-cloud
```

`Project.SwitchAgentProvider` ([internal/project/provider_switch.go:40](../../internal/project/provider_switch.go#L40))
patches `provider` to the fallback but leaves `fallback_provider` untouched, so
the resulting config reads:

```yaml
provider: gemini-cloud
fallback_provider: gemini-cloud      # now identical
```

which violates the rule at
[internal/config/config.go:1296](../../internal/config/config.go#L1296):

> `project config: agent %q fallback_provider must differ from provider %q`

`PatchAgentProviders` validates before writing, refuses, and returns an error.
`tryFailover` ([internal/queue/dispatcher.go:471](../../internal/queue/dispatcher.go#L471))
treats that as a failed switch and returns false, which routes the job into
`handleRateLimit` — marking it failed and pausing the queue.

## Evidence

```
ERROR queue: automated provider switch failed; falling back to standard pause
  job_id=67723064c27b4446 agent=requirements-analyst
  err="patching agent provider: patched .../lifecycle/config.yaml would not parse
       as a project config, not writing: project config: agent
       \"requirements-analyst\" fallback_provider must differ from provider
       \"gemini-cloud\""
WARN  queue: rate-limit text not parsed; using fallback pause kind=overloaded pause=5m0s
INFO  queue: paused due to rate limit paused_until=2026-08-28T12:26:34+10:00
```

The failure is deterministic and independent of the trigger kind — it reproduces
identically for `overloaded` (HTTP 529) and for `rate_limit` / quota.

## Impact

The condition is unavoidable: failover only engages when `fallback_provider` is
set, and that is precisely the configuration that makes the post-switch config
invalid. So the feature never works. A rate-limited or overloaded provider pauses
the queue for `overload_pause` (5 min) or `fallback_pause` (30 min) even when a
healthy fallback is configured and has passed its health probe.

The health probe runs and succeeds before the switch is attempted, so the
failure happens at the last step, after the operator-visible signals suggest
failover is proceeding.

## Detected by

These three tests are correct and should not be changed — they fail because the
product is broken:

- `TestFailover_AutoSwitch_HTTP529` — "queue should not pause on automated failover, but it did"
- `TestFailover_AutoSwitch_RateLimitQuota` — same assertion
- `TestSecrets_FailoverAudit` — times out waiting for the requeued job, which never runs

## Proposed fix

The patch must leave a config that validates. Two candidates; the choice is a
product decision:

1. **Clear `fallback_provider`** on failover. The primary is already stashed in
   `primary_provider` / `primary_model` for restore, so nothing is lost. Simple,
   and means one failover per agent until restored.
2. **Swap** — set `fallback_provider` to the stashed primary. Keeps a fallback
   available and allows failing back automatically, but can oscillate between two
   unhealthy providers unless `max_failovers_per_run` is respected.

Whichever is chosen, `SwitchAgentProvider` should validate the resulting config
itself and surface a clear error, rather than relying on `PatchAgentProviders`
rejecting the write.
