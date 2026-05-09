---
title: OpenRouter LLM Integration
type: idea
status: draft
lineage: openrouter-llm-integration
created: "2026-05-09T17:47:34+10:00"
priority: normal
labels:
    - feature
    - integration
    - agent
    - backend
---

# OpenRouter LLM Integration

Add OpenRouter as a supported LLM provider option alongside any existing providers (e.g. Anthropic direct). OpenRouter acts as a unified API gateway, giving access to a wide range of models from different vendors (OpenAI, Anthropic, Mistral, Meta, etc.) through a single API key and endpoint.

This would allow users to configure agents in `lifecycle/config.yaml` to route through OpenRouter, selecting specific models by their OpenRouter model identifier. The integration should support the OpenRouter-specific headers (e.g. `HTTP-Referer`, `X-Title`) and handle its response format, which is OpenAI-compatible.

Benefits include cost flexibility (users can pick cheaper models for lower-stakes agents), access to open-weight models, and reduced vendor lock-in. The implementation should follow the same provider abstraction pattern used for other LLM backends so that switching between providers requires only a config change.
