---
title: Queue work should include multi-agent roles
type: idea
status: draft
lineage: multi-agent-roles
priority: normal
labels:
    - agent
    - agent-runner
    - queue
    - feature
    - enhancement
---

## Raw Idea

## Idea
Queue Work should include multi-agent roles, e.g. backend-developer can be claude or gemini, depending on who has credits available.

## Idea

When queuing work, each role (e.g. `backend-developer`, `frontend-developer`) should support multiple configured AI providers rather than being bound to a single one. The system selects the provider to use at dispatch time based on runtime criteria such as available credits, quota limits, or cost ceilings.

The initial use case is choosing between Claude and Gemini for a given role: if one provider has exhausted its credits or is rate-limited, the queue dispatches to the other. The configuration should allow an ordered preference list per role, with automatic fallback rather than a hard failure.

This makes the agent runner more resilient and cost-aware, and lays the groundwork for future strategies such as round-robin, cost-minimisation, or capability-based routing across providers.
