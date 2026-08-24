---
title: Inline Driver Provider Abstraction
type: idea
status: approved
lineage: inline-driver-provider-abstraction
created: "2026-08-25T08:28:45+10:00"
priority: normal
labels:
    - driver
    - provider
    - agent
    - open-provider-support
    - architecture
    - backend
release: KC-Release6
parent: lifecycle/ideas/local-model-operability.md
---

# Inline Driver Provider Abstraction

The inline driver is currently hard-coupled to Claude, making it impossible to switch or configure alternative LLM providers without code changes. The driver layer needs to be broken up so that provider selection becomes a runtime or configuration concern rather than a compile-time dependency.

The refactoring should introduce a provider interface or abstraction that the inline driver delegates to, with Claude as the default implementation. Additional providers (e.g. OpenAI, Ollama, local models) should be pluggable by implementing the same interface, enabling the operator to select a provider via config without modifying the core agent runner.

This change improves portability and long-term maintainability of the agent infrastructure, and is a prerequisite for any multi-provider or local-model operability work.
