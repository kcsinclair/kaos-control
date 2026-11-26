---
title: OpenRouter LLM Integration
type: idea
status: abandoned
lineage: openrouter-llm-integration
created: "2026-05-09T17:47:34+10:00"
priority: normal
parent: lifecycle/ideas/open-provider-support.md
labels:
    - feature
    - integration
    - agent
    - backend
    - open-provider-support
release: KC-Release6
---

# OpenRouter LLM Integration

Add OpenRouter as a supported LLM provider option alongside any existing providers (e.g. Anthropic direct). OpenRouter acts as a unified API gateway, giving access to a wide range of models from different vendors (OpenAI, Anthropic, Mistral, Meta, etc.) through a single API key and endpoint.

This would allow users to configure agents in `lifecycle/config.yaml` to route through OpenRouter, selecting specific models by their OpenRouter model identifier. The integration should support the OpenRouter-specific headers (e.g. `HTTP-Referer`, `X-Title`) and handle its response format, which is OpenAI-compatible.

Benefits include cost flexibility (users can pick cheaper models for lower-stakes agents), access to open-weight models, and reduced vendor lock-in. The implementation should follow the same provider abstraction pattern used for other LLM backends so that switching between providers requires only a config change.

## Superseded

Absorbed into the [[open-provider-support]] epic (workstream 1). OpenRouter is
OpenAI-compatible, so once the `openai-compatible` driver and the Provider
record exist, OpenRouter is **configuration, not code**: a provider with
OpenRouter's base URL, an API key, and its two custom headers.

What carries forward: an `extra_headers` field on the Provider record (for
`HTTP-Referer` / `X-Title`) and an OpenRouter config example/preset. No
OpenRouter-specific driver is needed.
