---
title: Agent Sequencing and Concurrency Controls
type: idea
status: draft
lineage: agent-sequencing-and-concurrency
created: "2026-05-10T10:39:05+10:00"
priority: normal
labels:
    - agent
    - agent-runner
    - workflow
    - feature
release: KC-AgentHandling
---

# Agent Sequencing and Concurrency Controls

The system should enforce concurrency limits and sequencing rules on agent runs to prevent conflicting work and ensure orderly progression through the lifecycle. At minimum, only one backend-developer and one frontend-developer should be permitted to run at a time across all active lineages — launching a second instance of the same role should be blocked or queued until the first completes.

Additionally, a test-developer run on a given lineage should be gated behind the completion of both the backend-developer and frontend-developer runs for that lineage. This ensures the test-developer always targets stable, finished implementation work rather than a partially developed feature.

These rules should be enforced in the agent runner / supervisor layer, with clear error or status feedback surfaced to the UI when a requested agent run is blocked by a sequencing constraint.
