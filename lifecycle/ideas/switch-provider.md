---
title: Switch provider function
type: idea
status: draft
lineage: switch-provider
priority: high
labels:
    - agent
    - provider
    - config
    - agent-runner
    - reliability
    - backend
    - feature
    - operability
    - ai-ml
release: KC-Release6
---

## Raw Idea

## Raw Idea
Switch provider function, which essentially changes the config.yaml to switch between Claude and Gemini for example, when Claude goes offline with "API Error: 529 Overloaded."

## Idea

A switch-provider function that dynamically updates `config.yaml` to change the active AI provider (e.g. from Claude to Gemini) when the current provider becomes unavailable. The primary trigger is a provider-side failure such as HTTP 529 Overloaded, enabling agent runs to continue without manual YAML editing or operator intervention.

The feature would involve detecting provider-level failure codes in the agent runner, then either automatically or via a user-triggered UI action rewriting the relevant driver/model fields in the project config and reloading them at runtime. A visible status indicator with a "switch provider" affordance would make the current provider state discoverable and the switch operable without leaving the UI.

Provider switching must be reversible and auditable: the previous configuration should be preserved or the change logged so operators can restore the original provider once it recovers. The implementation spans the agent driver abstraction, the config loader hot-reload path, and optionally a lightweight provider health-check mechanism.
