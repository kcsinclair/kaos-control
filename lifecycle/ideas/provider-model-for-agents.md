---
title: Provider Model for Agent API Endpoints
type: idea
status: done
lineage: provider-model-for-agents
parent: lifecycle/ideas/open-provider-support.md
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
    - open-provider-support
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

## Role in the epic

This is the **architecture spine** of [[open-provider-support]] workstream 1 —
the Provider entity itself. It settles the epic's open shape question: the
driver moves *onto* the provider, so a Provider is
`{name, base_url, api_key, driver, extra_headers}` and an Agent is a clean
`{provider, model}` pair (not a provider/driver/model three-tuple).

Scope confirmed alongside it:

- The native `ollama` driver is **removed outright** — Ollama is reached via its
  OpenAI-compatible endpoint like any other provider. `ollama_instances`
  generalises to `providers`, `/ollama/instances` becomes the provider API, and
  `OllamaSettingsView` becomes provider settings. The currently-registered
  instance (`Loki`) carries over as a provider record.
- `extra_headers` on the record is what makes [[openrouter-llm-integration]]
  pure configuration.
- The driver this selects must implement the tool-calling agent loop, not just
  chat completion — see [[llama-cpp-driver]]'s requirement.
