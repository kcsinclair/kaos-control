---
title: Inline Conversational Driver Provider Abstraction — Integration Test Suite
type: test
status: draft
lineage: inline-driver-provider-abstraction
parent: lifecycle/test-plans/inline-driver-provider-abstraction-5-test.md
created: "2026-08-26T13:00:00+10:00"
---

# Inline Conversational Driver Provider Abstraction — Integration Test Suite

Integration coverage for Milestone 5 of the test plan at
`lifecycle/test-plans/inline-driver-provider-abstraction-5-test.md`: driving
`POST /ideas/converse` and `POST /ideas/generate` through a stub
OpenAI-compatible provider instead of the real `claude` CLI binary. Milestones
1–4 and 6 (package unit tests in `internal/ideachat/` and `internal/config/`)
are out of scope for this artifact — they are `test-developer`'s package-unit
layer, not the integration layer, and outside this agent's write scope.

File: `tests/integration/inline_provider_completer_test.go`

None of these tests require `ANTHROPIC_API_KEY` — they exercise the
`openai-compatible` inline completer against a local `httptest.Server`, never
the `claude` CLI.

## Test infrastructure

- **`stubCompletionServer`** — a minimal `httptest`-backed stand-in for an
  OpenAI-compatible `/v1/chat/completions` endpoint. Scripted with a queue of
  canned assistant message contents (one per call, repeating the last entry
  once exhausted); records every request body + headers for assertions.
- **`newInlineProviderTestEnv`** — opens a `testproject` whose `idea-capture`
  and `docs-capture` agents are both `driver: inline` bound via
  `provider: stub-provider` to the stub server, registered as an app-level
  provider (`config.Provider`) carrying a secret `api_key` used to verify it
  never leaks back over the API. The stub server binds to a `127.0.0.1`
  loopback port by construction, so every test built on this helper already
  exercises the "local provider, no outbound network" path.

## Scenarios covered

- **`TestInlineProvider_ConverseAcceptWritesArtifact`** — drives a two-turn
  `POST /ideas/converse` conversation (clarify, then propose) through the
  stub provider and `__accept__`s the proposal. Asserts the written idea
  artifact is shaped the same way the CLI-default path produces it
  (`type: idea`, `status: draft`, `lineage: <slug>` on disk) and that
  `__accept__` makes no further LLM call (exactly 2 upstream requests total).
- **`TestInlineProvider_GenerateAllTypes`** — table test over
  `POST /ideas/generate` with `type=idea`, `type=defect`, `type=doc`, each
  through the stub provider. Asserts `target_dir` and `frontmatter.type` per
  type, that `defect` generation always forces the `defect` label, and that
  each call makes exactly one upstream request. Together with the converse
  test above, this covers all four inline template keys (`idea-capture`,
  `idea-generate`, `defect-generate`, `doc-generate`).
- **`TestInlineProvider_RequestShape_NoToolsAndMessageMapping`** — cross-checks
  the package-unit-level (M2) request-shape assertions at the integration
  layer: captured requests never carry a `tools` key, and a two-turn
  conversation's `messages` array preserves system-first ordering
  (`system, user, assistant, user`) with content mapped correctly across
  turns.
- **`TestInlineProvider_OfflineLocalStub_NoExternalNetwork`** — asserts the
  stub provider's URL is loopback-only (`127.0.0.1`/`localhost`/`::1`) and
  that a generation call completes successfully end-to-end against it,
  covering the local-model-operability capability referenced by NFR-5.
- **`TestInlineProvider_NoSecretLeak`** — asserts the provider's `api_key`
  never appears in a `POST /ideas/generate` response body nor in the
  `GET /agents` provider listing.

## Requirement AC → test mapping

- All four consumers via the OpenAI-compatible completer →
  `TestInlineProvider_ConverseAcceptWritesArtifact`,
  `TestInlineProvider_GenerateAllTypes`.
- Accept-path artifact shape matches pre-change output (FR-8) →
  `TestInlineProvider_ConverseAcceptWritesArtifact`.
- No `tools` key, correct message mapping (cross-check of M2) →
  `TestInlineProvider_RequestShape_NoToolsAndMessageMapping`.
- Local-provider offline capture/generation (NFR-5) →
  `TestInlineProvider_OfflineLocalStub_NoExternalNetwork`.
- No secret leakage (NFR-1) → `TestInlineProvider_NoSecretLeak`.

## Verification

```
go build -tags integration ./...
go vet -tags integration ./tests/...
go test -tags integration ./tests/integration/... -run TestInlineProvider -v
```

All 5 top-level tests (7 including the `GenerateAllTypes` subtests) pass.
