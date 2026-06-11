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
    - architecture
    - enhancement
---

## Raw Idea

## Idea
Queue Work should include multi-agent roles, e.g. backend-developer can be claude or gemini, depending on who has credits available.

## Idea

Currently, agent roles (e.g. `backend-developer`, `frontend-developer`) are tied to a single model/provider. This idea proposes that each role should support multiple configured providers (e.g. Claude, Gemini, Ollama), and the queue runner should select among them dynamically — for example, based on which provider has credits available, or by round-robin/priority ordering.

This would allow uninterrupted agent runs when one provider hits a quota or rate limit, and opens the door to cost optimisation by routing cheaper tasks to lower-cost providers. Configuration could live in `lifecycle/config.yaml` per role, with a fallback chain or availability-check mechanism rather than a hard failure.

The implementation would primarily touch the agent runner (`internal/agent/`) and the queue dispatch logic, with possible UI surface in the runs/queue views to show which provider handled a given run.
