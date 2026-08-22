---
title: Provider Model for Agent API Endpoints
type: idea
status: approved
lineage: provider-model-for-agents
created: "2026-08-22T12:14:51+10:00"
priority: normal
labels:
    - agent
    - agents
    - config
    - driver
    - ollama
    - ai-ml
    - backend
    - frontend
    - enhancement
    - medium-complexity
release: KC-Release6
rice_reach: 100
rice_impact: 1
rice_confidence: 75
rice_effort: 0.2
---

# Provider Model for Agent API Endpoints

Refactor the current Ollama GUI configuration into a generalised **Provider** concept. A provider has a name, a base URL, an API key, and a driver (e.g. `ollama`, `openai-compatible`, `anthropic`, `llama-cpp`). This covers both local inference endpoints (Ollama, llama.cpp) and remote cloud APIs (Anthropic, OpenAI, etc.) under a single, uniform configuration surface.

When defining an Agent, the user selects a provider from the configured list and specifies the model name to use with that provider. This decouples the agent definition from any particular API implementation and makes it straightforward to point the same agent role at a different backend by changing one field.

The open idea for llama.cpp support is subsumed by this change: adding a `llama-cpp` driver and pointing a provider at the local llama.cpp server address is sufficient to enable it, with no further special-casing required.
