---
title: Provider management
type: feature
status: approved
lineage: feature-provider-management
created: "2026-08-26T16:30:00+10:00"
summary: A first-class Provider entity plus an openai-compatible driver that gives any OpenAI-compatible endpoint — local or cloud — the full tool-calling agent loop.
function: Agents
labels:
    - feature
    - agent
    - provider
    - driver
    - open-provider-support
related_to:
    - lifecycle/ideas/open-provider-support.md
    - lifecycle/requirements/open-provider-support-2.md
    - lifecycle/ideas/provider-model-for-agents.md
    - lifecycle/requirements/provider-model-for-agents-2.md
parent: lifecycle/docs/open-provider-support-7-doc.md
---

# Provider management

Any endpoint speaking the OpenAI `/v1/chat/completions` wire format — local
(Ollama, llama.cpp), gateway (OpenRouter), or direct cloud (OpenAI, Groq,
Together, Azure) — is configured once as a **Provider** and reused by any
number of agents.

## What it does

- **Provider entity.** App-wide `{name, base_url, driver, api_key,
  extra_headers}` record, shared across all projects; CRUD via
  `/api/providers`, with live health probing and `/v1/models` discovery.
- **Agents as `{provider, model}` pairs.** An agent references a provider by
  name and a model id instead of duplicating connection details.
- **`openai-compatible` driver.** Full agentic loop against any registered
  provider — tool-calling (`read_file`, `write_file`, `list_dir`, `grep`,
  sandboxed to `allowed_write_paths`), streamed progress, TTFT, per-run
  logging, and cancellation, matching every other driver's contract.
- **Tool-capability preflight.** Before running, the driver detects a
  server that silently drops tool definitions (the most dangerous failure —
  it looks like a normal, confident answer) and hard-fails the run rather
  than writing a possibly-fabricated artifact.
- **Native tool-call recovery.** Recovers tool calls some models emit in
  their own text syntax instead of a parsed `tool_calls` field, logged
  separately from natively-parsed calls.
- **Provider Settings UI.** Add, edit, test, and delete providers; browse
  discovered models and their advertised tool support.
- **Secret hygiene.** `api_key` is masked in every API response and log;
  never written to run logs or `ProgressEvent` output.

Replaces the legacy `ollama_instances` config block, the native `ollama`
driver (single-shot, no tool use), and the `OllamaSettingsView` page —
migrated automatically on startup. See [[ollama-local-llms]] for the
retired predecessor.

Reachable at **Settings → Providers**; API under `/api/providers`. Full
reference: [open-provider-support](../../docs/open-provider-support.md).
See also [[agent-orchestration]] and [[provider-failover]].
