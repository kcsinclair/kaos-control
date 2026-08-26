---
title: Inline Conversational Driver Provider Abstraction — Test Plan
type: plan-test
status: done
lineage: inline-driver-provider-abstraction
parent: lifecycle/requirements/inline-driver-provider-abstraction-2.md
created: "2026-08-25T14:30:00+10:00"
release: KC-Release6
labels:
    - driver
    - provider
    - inline
    - ideachat
    - test
---

# Test Plan: Inline Conversational Driver Provider Abstraction

Parent requirement: [[inline-driver-provider-abstraction]]
(`lifecycle/requirements/inline-driver-provider-abstraction-2.md`). Companion
plans: [[inline-driver-provider-abstraction]] backend (`-3-be`) and frontend
(`-4-fe`).

## Strategy

Two layers:

1. **Package unit tests** in `internal/ideachat/` — the completer contract,
   dispatcher selection, the OpenAI completer wire format (via `httptest`), and
   secret/error hygiene. These are the natural home for the abstraction and keep
   the async agent driver untouched. They satisfy NFR-2 (existing fakes keep
   working) and are `-short`-safe (no network, no `claude` binary).
2. **Integration tests** in `tests/` with artifacts under `lifecycle/tests/` —
   end-to-end `POST /ideas/converse` and `POST /ideas/generate` driven through a
   stub OpenAI-compatible provider (`httptest`), asserting all four inline
   template keys work through both completers and that no secret leaks over the
   API. Integration tests reuse the project's `testEnv` (admin auto-login) and
   full-URL devops helpers per existing conventions.

The Claude CLI completer path is asserted at the **argument-construction** level
(no real `claude` binary is invoked in CI): the test verifies the exact `exec`
args + `buildPrompt` output rather than shelling out.

Each milestone below maps to requirement Acceptance Criteria (AC) and FR/NFR
numbers.

---

## Milestone 1 — Completer contract & dispatcher selection (unit)

**Description.** Prove the interface, the reassignable `CallLLM`, and dispatcher
routing by `ModelConfig` (FR-1, FR-2, FR-4; AC: interface/dispatch,
CLI-default).

**Files to change.**
- `internal/ideachat/completer_test.go` (new).
- Reuse `generate_test.go:stubCallLLM` pattern to prove reassignment still
  bypasses the dispatcher.

**Test cases.**
- Reassigning `CallLLM` to a fake short-circuits the dispatcher (existing fakes
  unchanged) — regression guard for NFR-2.
- `dispatchComplete` with `ModelConfig.Provider == nil` selects the Claude CLI
  completer; with `Provider.Driver == "openai-compatible"` selects the OpenAI
  completer; with an unknown driver returns a named error (no panic).
- Golden check: for a given `SystemPrompt` + message history, `buildPrompt`
  output and the assembled `claude` arg vector
  (`--dangerously-skip-permissions -p <prompt> --model <model>`) are byte-equal
  to the pre-change implementation (a frozen expected string in the test).

**Acceptance criteria.**
- [ ] Dispatcher selection table (nil / openai-compatible / unknown) verified.
- [ ] Reassigned `CallLLM` bypasses the dispatcher (NFR-2).
- [ ] Frozen golden asserts CLI arg vector + prompt are unchanged (FR-2, FR-8).

## Milestone 2 — OpenAI completer wire format (unit, httptest)

**Description.** Assert the request body, headers, message mapping, and return
value of the OpenAI-compatible inline completer (FR-3, FR-5; AC: POST shape,
auth/extra-headers/no-tools, system+role mapping).

**Files to change.**
- `internal/ideachat/openai_completer_test.go` (new) — an `httptest.Server`
  capturing the request.

**Test cases.**
- POST path is `<base_url>/v1/chat/completions`; body carries `model` from
  `ModelConfig.Model` and a `messages` array.
- Message mapping: non-empty `SystemPrompt` becomes the **first** message with
  `role:"system"`; each `LLMMessage` maps in order with its `role`
  (`user`/`assistant`) preserved.
- `max_tokens` present **only** when `ModelConfig.MaxTokens > 0`; absent when 0.
- Body contains **no** `tools` key.
- `Content-Type: application/json` always set; `Authorization: Bearer <key>`
  present iff `api_key != ""` (two cases); each `extra_headers` entry applied.
- Return value equals `choices[0].message.content` (trimmed), including the
  array-of-parts content shape if implemented.

**Acceptance criteria.**
- [ ] All request-shape assertions pass against the httptest capture.
- [ ] Auth-header presence toggles exactly on `api_key` emptiness.
- [ ] No `tools` key is ever sent (AC: OpenAI path sends no tools array).
- [ ] Assistant text is returned verbatim (trimmed).

## Milestone 3 — Failure behaviour & secret hygiene (unit)

**Description.** Bounded, non-panicking, non-retrying failures and no credential
leakage (FR-7, NFR-1; AC: failure surfaces as error, no secret leak).

**Files to change.**
- `internal/ideachat/openai_completer_test.go` (extend).

**Test cases.**
- Non-2xx (e.g. 401/429/500), malformed JSON, and empty `choices` each return a
  non-nil `error`, no panic.
- A hung endpoint with a cancelled/deadline `context` returns promptly with a
  context-bounded error; the completer performs **no** retry (assert single
  request received by the stub).
- The returned error string contains neither the `api_key` value nor any
  `extra_headers` secret value.
- The Claude CLI completer's missing-binary error message is unchanged (assert
  wording via a forced `exec` failure or an injected runner seam).

**Acceptance criteria.**
- [ ] Each failure mode yields a returned error, never a panic or unbounded
      retry; exactly one upstream request is made per call.
- [ ] No credential material appears in any error string (NFR-1).
- [ ] CLI missing-binary error wording is unchanged (FR-7).

## Milestone 4 — Config validation (unit)

**Description.** FR-6 validation rules (AC: reject unregistered provider; reject
provider-without-model; no-provider validates as today).

**Files to change.**
- `internal/config/config_test.go` (extend) and, if `ValidateAgentProviders`
  lives at the project seam, `internal/project/*_test.go`.

**Test cases.**
- `inline` agent with `provider` set + empty `model` → rejected, error names the
  agent.
- `inline` agent with `provider` not present in the supplied app provider list →
  rejected, error names the agent and the missing provider.
- `inline` agent with **no** `provider` → validates (no new mandatory field),
  matching current behaviour; the shipped `lifecycle/config.yaml` still loads.
- Existing app/project configs load and validate without edits (NFR-4).

**Acceptance criteria.**
- [ ] Both rejection cases produce agent-naming errors.
- [ ] No-provider inline agents and the shipped config validate unchanged.

## Milestone 5 — End-to-end via stub provider, all four keys (integration)

**Description.** Drive both endpoints through a stub OpenAI-compatible provider
and through the CLI-default path, covering `idea-capture` (converse),
`idea-generate`, `defect-generate`, `doc-generate` (FR-8; AC: all four consumers
via both completers, offline local-provider capability).

**Files to change.**
- `tests/integration/inline_provider_completer_test.go` (new) — spin an
  `httptest` provider returning canned `propose`/`clarify` JSON; register it as
  an app provider; bind an inline agent to it in a temp project config; exercise
  `POST /ideas/converse` (multi-turn to a proposal + `__accept__`) and
  `POST /ideas/generate` for `idea`/`defect`/`doc` types.
- `lifecycle/tests/inline-driver-provider-abstraction-test.md` (new artifact,
  `type: test`) describing what this integration coverage asserts.

**Test cases.**
- Converse to a `propose` state and `__accept__` writes an idea artifact whose
  shape matches the CLI-default output (FR-8).
- `idea-generate`, `defect-generate`, `doc-generate` each return a valid preview
  proposal through the provider-backed completer; `defect` still forces the
  `defect` label.
- The stub captures a request with **no** `tools` key and the expected
  `messages` mapping (cross-checks M2 at the integration layer).
- "Offline" surrogate: point the inline agent at a `localhost` stub with **no**
  outbound network and confirm capture/generation complete (NFR-5,
  [[local-model-operability]]).
- Secret boundary: the converse/generate JSON responses contain no `api_key`;
  any provider listing via API remains masked.

**Acceptance criteria.**
- [ ] All four template keys succeed through the OpenAI-compatible completer.
- [ ] Accept-path artifact shape matches pre-change output (FR-8).
- [ ] Stub-captured request has no `tools` key and correct message mapping.
- [ ] Local-only stub run succeeds with no external network (NFR-5).
- [ ] No `api_key` appears in any endpoint response (NFR-1).

## Milestone 6 — Regression & suite gates

**Description.** Guarantee no regression and green gates (NFR-2, NFR-3).

**Test cases.**
- Full `internal/ideachat` package test suite passes unchanged (including the
  reassigned-`CallLLM` fakes in `generate_test.go`).
- `make test-unit` (`go test ./... -short`) and `make lint` (go vet +
  staticcheck) are green; no new third-party dependency appears in `go.mod`.

**Acceptance criteria.**
- [ ] Pre-existing `internal/ideachat` tests pass unchanged (NFR-2).
- [ ] `make test-unit` and `make lint` green; `go.mod` unchanged except no new
      deps (NFR-3).

---

## Requirement AC → milestone coverage

- Interface + dispatch + reassignable `CallLLM` → M1.
- CLI completer default, identical invocation → M1, M5.
- OpenAI completer POST/return → M2, M5.
- Auth iff key, extra_headers, no tools → M2, M5.
- System→`system`, role order preserved → M2.
- Validation (unregistered / provider-without-model / no-provider) → M4.
- All four consumers via both completers → M5.
- Bounded failure, no panic/retry; CLI error unchanged → M3.
- No secret leakage → M3, M5.
- No new dep/cgo; existing tests pass → M6.
- Local-provider offline capture/generation → M5.
