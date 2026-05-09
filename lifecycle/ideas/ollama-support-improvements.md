---
title: Improve Ollama Support
type: idea
status: draft
lineage: ollama-support-improvements
created: "2026-05-09T17:45:45+10:00"
priority: normal
labels:
    - agent
    - agent-runner
    - backend
    - enhancement
    - operability
---

# Improve Ollama Support

The current Ollama integration works at a basic level but lacks the observability and agent guidance needed for reliable use in the lifecycle pipeline. This idea covers a set of targeted improvements to make Ollama a first-class model provider alongside the Claude API.

Key areas to address include: structured and levelled logging for Ollama requests and responses so operators can diagnose failures; improved agent prompt templates and instructions tailored to the capabilities and limitations of locally-hosted models; and any other operability gaps (e.g. timeout handling, model availability checks, error surfacing in the UI) discovered during the improvement pass.

The goal is that a user running kaos-control entirely on a local Ollama instance should have the same quality of feedback and agent behaviour as one using the Claude API, within the constraints of the chosen model.
