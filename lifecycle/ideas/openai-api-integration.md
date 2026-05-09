---
title: Add Support for OpenAI-Compatible LLM API
type: idea
status: draft
lineage: openai-api-integration
created: "2026-05-09T17:46:36+10:00"
priority: normal
labels:
    - feature
    - integration
    - agent
    - backend
---

# Add Support for OpenAI-Compatible LLM API

The agent runner currently targets a specific LLM provider. Adding support for the OpenAI-compatible REST API would allow kaos-control to integrate with virtually any modern LLM — including OpenAI, Azure OpenAI, Mistral, Together AI, Groq, Ollama (via its OpenAI-compatible endpoint), and any other provider that exposes the `/v1/chat/completions` interface.

The implementation should introduce an `openai` driver alongside the existing `ClaudeCodeDriver`, configurable per-agent in `lifecycle/config.yaml`. At minimum the driver should support model selection, system prompt injection, and streaming responses so that agent output can be broadcast over the existing WebSocket hub in real time.

This unlocks significant flexibility for teams that cannot or do not want to use Claude — enabling self-hosted, cost-optimised, or air-gapped deployments — and positions the tool as genuinely LLM-agnostic rather than tied to a single provider.
