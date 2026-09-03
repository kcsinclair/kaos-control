---
title: Inline conversational driver provider abstraction
type: feature
status: approved
lineage: feature-inline-conversational-provider-abstraction
created: "2026-08-27T10:00:00+10:00"
summary: A Completer interface lets the inline Idea Capture chat and idea/defect/doc generation flows target any registered Provider (Claude CLI default, or any openai-compatible endpoint), instead of being hard-coupled to the claude CLI binary.
function: Agents
labels:
    - feature
    - agent
    - provider
    - driver
    - ideachat
    - open-provider-support
related_to:
    - lifecycle/ideas/inline-driver-provider-abstraction.md
    - lifecycle/requirements/inline-driver-provider-abstraction-2.md
    - lifecycle/backend-plans/inline-driver-provider-abstraction-3-be.md
    - lifecycle/frontend-plans/inline-driver-provider-abstraction-4-fe.md
    - lifecycle/test-plans/inline-driver-provider-abstraction-5-test.md
    - lifecycle/tests/inline-driver-provider-abstraction-test.md
    - lifecycle/ideas/open-provider-support.md
    - lifecycle/features/provider-management.md
parent: docs/inline-driver-provider-abstraction.md
---

# Inline conversational driver provider abstraction

kaos-control has two independent LLM execution paths: the async agent path
(`internal/agent/`), already brought under the Provider abstraction by
[[provider-management]], and the inline conversational/generation path
(`internal/ideachat/`) that powers the Idea Capture chat and single-shot
idea/defect/doc generation. This feature brings the inline path to parity by
routing it through the same `{provider, model}` config shape, without adding
tool-calling or joining the async run pipeline.

## What it does

- **`Completer` interface.** `internal/ideachat` defines `Completer`
  (`Complete(ctx, cfg, messages) (string, error)`); the package-level
  `CallLLM` becomes a dispatcher that selects a completer from
  `ModelConfig.Provider` and delegates. Reassigning `CallLLM` in tests still
  bypasses the dispatcher, so pre-existing test fakes were unaffected.
- **Claude CLI completer (default, unchanged behaviour).** An inline agent
  with no `provider` field — the shipped default for every inline agent —
  continues to shell out to `claude --dangerously-skip-permissions -p
  <prompt> --model <model>` exactly as before, byte-identical prompt
  construction included.
- **OpenAI-compatible inline completer.** A small, standalone, non-streaming,
  tool-free client against `<base_url>/v1/chat/completions`, reusing the
  same app-level `Provider` records (`base_url`, `api_key`, `extra_headers`)
  the async `openai-compatible` driver already manages. Maps the system
  prompt to a `system` message, preserves history role order, includes
  `max_tokens` only when set, and never sends a `tools` array.
- **Provider resolution from agent config.** `resolveIdeaCaptureConfig`
  (`internal/http/idea_chat.go`) resolves the owning inline agent's
  `provider` field against the app's provider snapshot, feeding all four
  inline template keys (`idea-capture`, `idea-generate`, `defect-generate`,
  `doc-generate`) through the same mechanism.
- **Config validation.** `config.ValidateAgentProviders` rejects an inline
  agent whose `provider` isn't registered in app config, or that has
  `provider` set with no `model`, each error naming the offending agent; an
  inline agent with no `provider` validates exactly as before.
- **Secret hygiene.** `api_key`/`extra_headers` never appear in prompts,
  artifacts, logs, or error text — errors are scrubbed of the key value, and
  `ModelConfig` is never marshalled or returned by any API response.
- **Offline capability.** An inline agent bound to a local provider (Ollama
  `/v1`, a local `llama-server`) completes conversational capture and
  generation with zero internet connectivity — not possible with the
  CLI-only path.

Deliberately unchanged: no tool-calling or file-writing from the inline
path (the server, not the model, writes artifacts); no new UI (provider
selection is config-only); no inline run recording or provider failover in
v1; no new third-party dependency.

Configured via `provider:` on the `idea-capture` / `docs-capture` agents in
`lifecycle/config.yaml`, referencing a `Provider` registered in
`~/.kaos-control/config.yaml`. Full reference:
[inline-driver-provider-abstraction](../../docs/inline-driver-provider-abstraction.md).
See also [[provider-management]] and [[local-llm-operability]].
