---
title: Local-Model Operability
type: idea
status: approved
lineage: local-model-operability
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

# Local-Model Operability

Workstream 3 of [[open-provider-support]]. Once the OpenAI-compatible driver
lands, a local model is *reachable* — this covers what it takes for a local
model to produce **usable lifecycle artifacts**, which is a different problem
and one no driver refactor solves.

The goal: a user running kaos-control entirely against a local endpoint should
get the same quality of feedback and agent behaviour as one using a frontier
cloud API, within the constraints of the chosen model.

## What this still covers

Verified against [[open-provider-support-2]] and
[[provider-model-for-agents]]'s requirement — these are the gaps neither
addresses:

- **Agent prompt templates tuned to local models.** The current templates are
  written for frontier models. Small local models need shorter context, more
  explicit structure, fewer simultaneous instructions, and worked examples.
  This is the highest-value item: the benchmark evidence in
  [[open-provider-support-2]] shows capability ranging 68/68 to 0/68 on the same
  task, and prompt shape is one of the few levers that moves it.
- **Model availability checks.** Distinct from the *capability* preflight
  (FR-5b, which detects whether tools are supported): this is whether the named
  model is present and loaded on the endpoint at all — llama.cpp lazy-loads and
  reports `status: unloaded`, and a first request can therefore stall for
  minutes with no feedback.
- **Error surfacing in the UI.** Driver-level failure behaviour is specified
  (NFR-3), but making those failures legible in the interface — including the
  hard-fail from FR-5b — is not.

## What has dissolved

Originally scoped as "Improve Ollama Support"; the following no longer belong
here now that the native `ollama` driver is being removed in favour of one
OpenAI-compatible driver:

- **Ollama-specific request/response logging** — the shared driver owns
  structured logging for every provider (FR-7).
- **Timeout handling** — covered by FR-8, which bounds runs by the agent's
  `timeout_minutes` so an unreachable or hung endpoint fails within that bound.

## Notes

`lifecycle/features/ollama-local-llms.md` will need updating — and likely
renaming to match — once this work and the driver replacement land, since the
capability it documents will no longer be Ollama-specific.
