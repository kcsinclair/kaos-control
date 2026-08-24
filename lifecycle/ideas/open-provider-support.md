---
title: Open Provider Support
type: idea
status: draft
lineage: open-provider-support
created: "2026-08-24T18:36:36+10:00"
priority: normal
labels:
    - provider
    - driver
    - agent-runner
    - ai-ml
    - architecture
    - config
    - feature
    - high-complexity
---

# Open Provider Support

This idea encompasses all existing threads around local-AI (Ollama, local models), OpenRouter, and other open or third-party LLM provider support. The goal is to make kaos-control provider-agnostic so that agent runs are not locked to a single upstream API.

The current driver/model two-tuple may need to become a provider/driver/model three-tuple, or alternatively the OpenAI-compatible API driver could carry an explicit provider configuration field rather than embedding provider identity in the driver name. This architectural decision should be captured in an ADR once the approach is settled.

This lineage (open-provider-support) will serve as the parent for related requirements and plans covering provider configuration, credential management per-provider, model listing/discovery from each provider, and any UI changes needed to expose provider selection in the agent configuration workflow.
