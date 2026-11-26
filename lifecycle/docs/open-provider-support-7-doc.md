---
title: 'This idea is the EPIC for comprehensive support of AI providers and the models they offer, there are many related ideas '
type: doc
status: done
lineage: open-provider-support
created: "2026-08-26T16:03:14+10:00"
priority: normal
parent: lifecycle/ideas/open-provider-support.md
release: KC-Release6
output: docs/open-provider-support.md
---

This idea is the EPIC for comprehensive support of AI providers and the models they offer, there are many related ideas and information to summarise into the documentation for this.

## Produced

Documentation written to `docs/open-provider-support.md`, covering the two
shipped workstreams of the epic ([[provider-model-for-agents]] and
[[open-provider-support-2]] driver requirement — done; [[switch-provider]]
— done):

- The Provider entity (`{name, base_url, driver, api_key, extra_headers}`),
  its config shape, and validation rules
- Provider management REST API (`/api/providers/*`) and the Provider
  Settings UI
- Agents as `{provider, model}` pairs, with a config example
- The `openai-compatible` driver: request construction, the sandboxed
  tool-calling loop (`read_file`/`write_file`/`list_dir`/`grep`), native
  tool-call recovery for models that emit calls in their own text syntax,
  the tool-capability preflight (silent-drop detection and hard-fail
  behaviour), streaming/TTFT/logging/cancellation, and secret hygiene
- Migration from the legacy `ollama_instances` block and the now-removed
  native `ollama` driver / `OllamaSettingsView`
- Dynamic provider switching and failover: per-agent fallback config,
  automated failover policy (`provider_failover.*`), the recovery prober,
  manual switch/restore/template REST API and UI
- A troubleshooting section (preflight failures, iteration-cap runs,
  provider-delete-in-use, config validation errors, no automatic failback)
- A "Related and in-progress work" section pointing to the two workstreams
  not yet shipped ([[inline-driver-provider-abstraction]],
  [[agent-logging-provider-driver]]) and the pending
  [[reporting-open-provider-gemini-stream]] reporting work, so the doc does
  not overclaim

Feature records under `lifecycle/features/`:

- Added `provider-management.md` (Provider entity + `openai-compatible`
  driver)
- Added `provider-failover.md` (dynamic switching + automated failover)
- Updated `ollama-local-llms.md` — marked superseded, since the native
  `ollama` driver, `/api/ollama/instances`, and `OllamaSettingsView` it
  described are removed
- Updated `agent-orchestration.md`'s driver list, which was stale (still
  named the removed `Ollama` driver, didn't mention `claude-env` or
  `openai-compatible`)
