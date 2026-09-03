---
title: Local-model operability
type: feature
status: approved
lineage: feature-local-llm-operability
created: "2026-08-27T13:00:00+10:00"
summary: Local-model-tuned prompt fallbacks, model availability preflight, warmup/lazy-load UI feedback, and a structured error taxonomy so a local inference server gets the same legible feedback as a frontier cloud API.
function: Agents
labels:
    - feature
    - agent
    - provider
    - local-models
    - open-provider-support
related_to:
    - lifecycle/ideas/local-model-operability.md
    - lifecycle/requirements/local-model-operability-2.md
    - lifecycle/ideas/open-provider-support.md
    - lifecycle/features/provider-management.md
    - lifecycle/features/ollama-local-llms.md
parent: lifecycle/docs/local-model-operability-7-doc.md
---

# Local-model operability

Once the `openai-compatible` driver makes a local inference server
*reachable* (see [[provider-management]]), a separate problem remains: small
local models (8B–30B, served via llama.cpp or Ollama) need different prompt
shapes, fail in different ways, and hide different states than frontier
cloud models. This feature closes that gap.

## What it does

- **Local-model-tuned prompt fallbacks.** `LocalModelPromptDefaults`
  (`internal/agent/prompt_defaults.go`) supplies a compact, single-step,
  frontmatter-few-shot prompt per standard agent role whenever an agent's
  `prompt_templates` has no entry for its active role — no configuration
  needed to opt in, and per-agent overrides still work exactly as before.
- **Model availability preflight.** Before acquiring a lineage lock,
  `CheckModelAvailability` probes the provider's `/v1/models` within a
  3-second timeout. A missing model or unreachable endpoint fails the run
  immediately (`model_not_found` / `endpoint_unreachable`) — no lock held,
  no artifact touched, no wasted run.
- **Warmup / lazy-load detection.** When a local server is loading
  multi-gigabyte weights (a 30–120s pause is common on first request), the
  driver surfaces a live `warming_up` state at 5s and applies a dedicated,
  configurable load timeout (default 60s) distinct from the run's overall
  `timeout_minutes`. The watchdog is satisfied by content, reasoning tokens,
  or a tool-call delta alike, so reasoning models aren't killed mid-think.
- **Live warmup UI.** An animated "Warming up model weights..." badge in
  `AgentRunningBanner.vue` and `AgentsRunsView.vue`, and a warmup line in
  `RunDetailModal.vue`'s turn timeline.
- **Structured error taxonomy.** Ten `failure_reason` codes
  (`tools_unsupported`, `model_not_found`, `model_unloaded`,
  `endpoint_unreachable`, `provider_disconnected`, `context_window_exceeded`,
  `turn_token_ceiling`, `max_iterations_reached`, `auth_error`, `timeout`),
  each with backend-generated remediation steps, secret-masked and attached
  to the run row and `agent.failed` broadcast.
- **Rich failure UI.** `RunFailureBanner.vue` renders a dedicated heading,
  explanation, and numbered remediation per code, with links to Provider
  Settings and Agent Config; `RunDetailModal.vue` adds a collapsible
  diagnostic panel; `AgentsRunsView.vue` shows a failure badge in the run
  table.

Replaces the historical Ollama-specific documentation in
[[ollama-local-llms]] (retired alongside the native `ollama` driver) —
this feature is provider-agnostic and applies to any `openai-compatible`
local endpoint.

Full reference: [local-model-operability](../../docs/local-model-operability.md).
See also [[provider-management]] and [[provider-failover]].
