---
title: Sequential Agent Execution Ordering to Prevent Race Conditions
type: idea
status: planning
lineage: sequential-agent-execution-ordering
created: "2026-05-06T18:51:21+10:00"
priority: normal
labels:
    - agent
    - agent-runner
    - architecture
    - workflow
release: KC-Release2
---

# Sequential Agent Execution Ordering to Prevent Race Conditions

On a given kaos-control instance, allowing multiple agent types to run concurrently introduces race conditions — particularly when agents write to shared lineage artifacts or depend on outputs produced by earlier agents in the same lineage. To avoid this, the system should enforce that only one agent type runs at a time per instance.

Within a lineage, agents should be sequenced in a fixed order: backend-developer first, then frontend-developer, then test-developer. This mirrors the natural dependency chain — frontend plans often reference backend contracts, and test plans cover both — so enforcing this order ensures each agent operates on a stable, complete prior output.

This could be implemented as a per-instance queue in the agent runner, where submitted agent jobs are serialised and dispatched in the prescribed order. The existing lineage lock manager may provide a foundation, but a higher-level scheduling layer would be needed to enforce cross-agent ordering beyond simple per-lineage mutual exclusion.

The Agent Runner could have an option to "Start Development" which then works through the instances in order.
