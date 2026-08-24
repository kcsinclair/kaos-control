---
title: llama.cpp Agent Driver for Local Models
type: idea
status: abandoned
lineage: llama-cpp-driver
created: "2026-08-11T16:42:45+10:00"
priority: normal
parent: lifecycle/ideas/open-provider-support.md
labels:
    - driver
    - agent
    - agent-runner
    - backend
    - go
    - portability
    - feature
    - open-provider-support
release: KC-Release6
---

# llama.cpp Agent Driver for Local Models

Add a driver that integrates llama.cpp with the agent runner, enabling locally-hosted models to participate in the same agent-mode workflow as cloud-backed providers. The driver should expose the llama.cpp server API (OpenAI-compatible endpoint) and map agent tool calls and responses to the existing driver interface so any configured agent can run against a local model without code changes.

The implementation must support agent mode specifically — multi-turn conversations with tool use — rather than simple completion. This requires handling the tool-call round-trip that llama.cpp exposes via its OpenAI-compatible `/v1/chat/completions` endpoint with tool definitions.

Testing and example configurations are to be developed in `~/Code/agent-benchmark`, covering at minimum: driver initialisation, a single tool-call round-trip, and a multi-step agent run. Examples should document the required llama.cpp server flags and model recommendations for reliable agent-mode behaviour.

## Superseded

Absorbed into the [[open-provider-support]] epic (workstream 1) — as this
lineage's own requirement anticipated, and as [[provider-model-for-agents]]
states ("The open idea for llama.cpp support is subsumed by this change").
llama.cpp speaks the same OpenAI-compatible `/v1/chat/completions` endpoint as
every other target, so it needs no dedicated driver.

**Its requirement is being generalised, not discarded.**
`requirements/open-provider-support-2.md` is the most developed specification of the
genuinely hard part — the bounded multi-turn tool-call loop, sandbox and
`allowed_write_paths` scoping for tool execution, and the ProgressEvent/TTFT
contract. That content becomes the driver requirement under the epic lineage,
with llama.cpp retained as a **verification target** (local `llama-server`,
plus the `~/Code/agent-benchmark` examples) rather than the subject.
