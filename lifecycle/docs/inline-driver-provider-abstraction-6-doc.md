---
title: Inline Driver Provider Abstraction — Documentation
type: doc
status: done
lineage: inline-driver-provider-abstraction
created: "2026-08-27T14:00:00+10:00"
priority: normal
parent: lifecycle/ideas/inline-driver-provider-abstraction.md
release: KC-Release6
output: docs/inline-driver-provider-abstraction.md
---

Produce documentation and a feature record for the inline driver provider
abstraction: breaking the inline conversational/generation path's
compile-time coupling to the Claude CLI, per
[[inline-driver-provider-abstraction]] (idea),
[[inline-driver-provider-abstraction-2]] (requirement), and the backend
(`-3-be`), frontend (`-4-fe`), and test (`-5-test`) plans under the same
lineage.

## Produced

Documentation written to `docs/inline-driver-provider-abstraction.md`,
verified against the shipped implementation in `internal/ideachat/completer.go`,
`internal/ideachat/claude_cli.go`, `internal/ideachat/openai_completer.go`,
`internal/ideachat/llm.go`, `internal/http/idea_chat.go`
(`resolveIdeaCaptureConfig` / `resolveAgentProvider`), and
`internal/config/config.go` (`ValidateAgentProviders`):

- The two independent LLM execution paths (async agent vs. inline) and how
  this feature brings the inline path to provider parity without merging
  the two engines or adding tool-calling to the inline path
- The `Completer` interface, the reassignable `CallLLM` dispatcher, and how
  `selectCompleter` chooses between the Claude CLI completer (default) and
  the OpenAI-compatible completer based on `ModelConfig.Provider`
- Why `ModelConfig` can safely carry a full `config.ProviderConfig`
  (including `api_key`) — it is never marshalled or returned by any HTTP
  handler
- The Claude CLI completer's byte-identical-behaviour guarantee for
  existing, unconfigured inline agents
- The OpenAI-compatible completer's request/response shape (system-first
  message mapping, `max_tokens` only when set, no `tools` key ever sent),
  and its bounded, non-retrying failure handling
- Provider resolution via `resolveIdeaCaptureConfig`, covering all four
  inline template keys (`idea-capture`, `idea-generate`, `defect-generate`,
  `doc-generate`) through one function
- Why the inline path deliberately stays tool-free
  ([[adr-0006-mediated-agent-driver-permission-model]] scoping)
- Configuration: default (no-provider) behaviour, binding an inline agent to
  a registered Provider, local/offline capture and generation, and the
  config validation rules `ValidateAgentProviders` enforces
- Secret hygiene (masking, scrubbed error text, no serialisation surface)
- An explicit "what did not change" section (no new UI, no tool-calling, no
  run recording, no automatic failover, no new dependency) so the doc does
  not overclaim beyond the requirement's recorded non-goals
- A troubleshooting section covering CLI-binary errors, provider-backed
  failures, config validation errors, and the tool-calling non-goal

Feature record added: `lifecycle/features/inline-conversational-provider-abstraction.md`
(`function: Agents`), covering the `Completer` interface, both
implementations, provider resolution, config validation, secret hygiene,
and offline capability, and cross-linking to [[provider-management]] and
[[local-llm-operability]].

Also cross-linked from `docs/open-provider-support.md`'s existing
"Related and in-progress work" entry for
[[inline-driver-provider-abstraction]] (no edit needed there — that entry
already correctly described the work as in progress; it should be updated
to **Done** by whichever agent next revises that document, since this
agent's brief was scoped to this lineage only).
