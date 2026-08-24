---
title: Improve Ollama Support
type: idea
status: draft
lineage: ollama-support-improvements
parent: lifecycle/ideas/open-provider-support.md
created: "2026-05-09T17:45:45+10:00"
priority: normal
labels:
    - agent
    - agent-runner
    - backend
    - enhancement
    - operability
    - open-provider-support
release: KC-Release6
---

# Improve Ollama Support

The current Ollama integration works at a basic level but lacks the observability and agent guidance needed for reliable use in the lifecycle pipeline. This idea covers a set of targeted improvements to make Ollama a first-class model provider alongside the Claude API.

Key areas to address include: structured and levelled logging for Ollama requests and responses so operators can diagnose failures; improved agent prompt templates and instructions tailored to the capabilities and limitations of locally-hosted models; and any other operability gaps (e.g. timeout handling, model availability checks, error surfacing in the UI) discovered during the improvement pass.

The goal is that a user running kaos-control entirely on a local Ollama instance should have the same quality of feedback and agent behaviour as one using the Claude API, within the constraints of the chosen model.

Feature document features/ollama-local-llms.md will need to be updated when this set of features is finalised.

## Re-scope (epic dedup)

Workstream 3 of [[open-provider-support]]. This idea **partly dissolves** now
that the native `ollama` driver is being removed in favour of one
OpenAI-compatible driver:

- **Dissolves** — Ollama-specific request/response logging. The shared driver
  owns structured logging for every provider, so there is no Ollama-specific
  logging pass to do.
- **Survives, and is still valuable** — the local-model quality work that no
  driver refactor delivers: agent prompt templates tuned to the capabilities and
  limits of small local models, model availability checks, timeout handling, and
  error surfacing in the UI. This is the difference between "a local model is
  reachable" and "a local model produces usable lifecycle artifacts".

Retitle to something provider-neutral (e.g. "Local-model operability") when this
is turned into a requirement, since it is no longer Ollama-specific.
