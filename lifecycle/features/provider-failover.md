---
title: Provider switching and failover
type: feature
status: approved
lineage: feature-provider-failover
created: "2026-08-26T16:31:00+10:00"
summary: Automated failover to a fallback provider on upstream overload/rate-limit/outage, plus manual switch, restore, and template controls.
function: Agents
labels:
    - feature
    - agent
    - provider
    - reliability
    - open-provider-support
related_to:
    - lifecycle/ideas/switch-provider.md
    - lifecycle/requirements/switch-provider-2.md
parent: lifecycle/docs/open-provider-support-7-doc.md
---

# Provider switching and failover

When an upstream provider is overloaded (HTTP 529), rate-limited (HTTP
429), or unreachable, agents can move to a configured fallback provider
without an operator hand-editing config or restarting the server.

## What it does

- **Per-agent fallback.** `fallback_provider` / `fallback_model` on an
  agent; the original `provider` / `model` is preserved automatically as
  `primary_provider` / `primary_model` for later restore.
- **Automated failover.** Opt-in per project
  (`provider_failover.auto_switch`). The queue dispatcher switches an
  affected agent and immediately re-enqueues its job on a matching failure
  kind (`overloaded`, `rate_limit`, `unreachable`), instead of pausing the
  whole queue for the usual long backoff. Bounded by
  `max_failovers_per_run` to prevent cascades.
- **Atomic, auditable switch.** Config is rewritten and hot-reloaded
  (`project.ReloadConfig()`, no restart), and every switch is recorded in
  the event feed and git commit log.
- **Recovery probing.** A background prober re-checks each switched-away
  primary provider on an interval and surfaces when it is healthy again —
  restore is still an explicit action.
- **Manual controls.** Per-agent switch/restore, and project-wide
  switch-all / restore-all, via REST API and UI (fallback badge + Restore
  Primary button on the Agents panel; header-level batch controls).
- **Provider templates.** Named presets that apply a provider/model
  binding across a project's agents in one action.

Reachable at **Agents** panel (per-agent) and project settings
(switch-all/restore-all); API under `/api/p/{project}/provider-switch/*`
and `/api/p/{project}/agents/{name}/switch-provider`. Full reference:
[open-provider-support](../../docs/open-provider-support.md). Depends on
[[provider-management]].
