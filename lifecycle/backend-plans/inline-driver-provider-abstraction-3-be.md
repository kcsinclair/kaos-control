---
title: Inline Conversational Driver Provider Abstraction — Backend Plan
type: plan-backend
status: draft
lineage: inline-driver-provider-abstraction
parent: lifecycle/requirements/inline-driver-provider-abstraction-2.md
created: "2026-08-25T14:30:00+10:00"
release: KC-Release6
labels:
    - driver
    - provider
    - inline
    - ideachat
    - backend
---

# Backend Plan: Inline Conversational Driver Provider Abstraction

Parent requirement: [[inline-driver-provider-abstraction]]
(`lifecycle/requirements/inline-driver-provider-abstraction-2.md`). Companion
plans: [[inline-driver-provider-abstraction]] frontend (`-4-fe`) and test
(`-5-test`). Related lineages: [[provider-model-for-agents]],
[[open-provider-support]], [[agent-logging-provider-driver]],
[[local-model-operability]], [[switch-provider]].

## Architecture conformance

Confirmed against `lifecycle/architecture/architecture-summary.md`, the promoted
[[modular-monolith]] architecture, the [[go-vue]] stack, the ADRs, and the
standards. The requirement's own **Architecture-Breaking Requirements** section
(§Detailed Requirements) already establishes there is **no** architecture-breaking
requirement and **no new ADR** is needed. This plan holds to that:

- **Single self-contained binary / stdlib-only** — the completer interface, the
  dispatcher, and the OpenAI-compatible inline client use only `net/http` +
  `encoding/json`; no new module, no cgo. (NFR-3)
- **Mediated tool calls ([[adr-0006-mediated-agent-driver-permission-model]])** —
  the inline path issues **no** model-driven tool calls; it returns one completion
  string that server Go code parses and writes. The plan preserves this: the
  OpenAI completer sends **no** `tools` array. The requirement's recorded
  forward-constraint (any future model-issued tool call MUST route through the
  mediated driver) is respected by keeping the completer tool-free.
- **Secrets handling ([[secrets-handling]])** — `api_key` / `extra_headers` are
  carried only on the internal, never-serialised `ModelConfig`; the OpenAI
  completer emits them solely as request headers; error strings are scrubbed.

If, during implementation, any milestone is found to require deviating from the
recorded architecture/stack/standards, stop and raise a new ADR under
`lifecycle/architecture/decisions/` rather than deviating silently. None is
anticipated.

## Design summary

`internal/ideachat.CallLLM` is a package-level `var` (today `= callLLMImpl`,
the Claude CLI shell-out). **Every** inline caller funnels through it —
`converse.go` (`Converse`, and the slug-retry in `resolveSlug`) and
`generate.go` (`Generate`). Replacing `callLLMImpl` with a **dispatcher** that
selects a `Completer` therefore brings all four inline agents (`idea-capture`,
`idea-generate`, `defect-generate`, `doc-generate`) under provider selection
with no change to caller code, and reassigning `CallLLM` in tests still
short-circuits the dispatcher (NFR-2).

Provider identity reaches the dispatcher on `ModelConfig`, which is **internal
only** (never returned by any HTTP handler), so carrying `base_url`/`api_key`/
`extra_headers` on it does not create a serialised secret surface (NFR-1).
`internal/http/idea_chat.go:resolveIdeaCaptureConfig` — the single builder of
`ModelConfig` used by both the converse and generate endpoints — is the one
place that reads the agent's `provider` and looks it up in
`project.Project.Providers` (the app-level snapshot already present on the
project).

Resolved-question decisions taken as design inputs: **non-streaming** OpenAI
completer (RQ-1); **standalone** small inline client rather than reusing the
async agent-loop client (RQ-2 — the async client in
`internal/agent/openai_compatible.go` is welded to streaming, tool-calling,
preflight and recovery; a tool-free blocking client is smaller and cleaner than
carving a shared subset out of it); **empty `provider` = Claude CLI default**,
no sentinel (RQ-3); **single attempt**, no failover (RQ-4); **no run recording**
(RQ-5).

---

## Milestone 1 — Completer interface + Claude CLI completer + dispatcher

**Description.** Introduce the `Completer` abstraction and refactor the existing
Claude CLI shell-out into the default implementation, with `CallLLM` becoming a
dispatcher. Behaviour for the empty-`provider` path must be byte-identical to
today (FR-1, FR-2, FR-8).

**Files to change.**
- `internal/ideachat/llm.go` — define `type Completer interface { Complete(ctx
  context.Context, cfg ModelConfig, messages []LLMMessage) (string, error) }`.
  Rename the body of `callLLMImpl` into a `claudeCLICompleter` type whose
  `Complete` method holds the current `buildPrompt` + `exec.CommandContext(...,
  "claude", ...)` logic verbatim. Repoint `var CallLLM = dispatchComplete` where
  `dispatchComplete` selects a completer from `cfg` and delegates. `buildPrompt`
  is unchanged.
- (Optional split for clarity) `internal/ideachat/completer.go` — new file
  holding the interface + `dispatchComplete`; keep the CLI implementation in
  `llm.go` or a `claude_cli.go`. Author's discretion; no behavioural difference.

**Acceptance criteria.**
- [ ] A `Completer` interface exists with the exact contract of today's
      `CallLLM` signature.
- [ ] `claudeCLICompleter.Complete` produces the **same** `claude
      --dangerously-skip-permissions -p <prompt> --model <model>` invocation and
      the same `buildPrompt` output as the pre-change `callLLMImpl` (verified by
      the test plan's golden-argument check).
- [ ] `CallLLM` remains a reassignable package `var`; existing tests that set
      `CallLLM = func(...)` (`generate_test.go:stubCallLLM`) compile and pass
      unchanged, bypassing the dispatcher (NFR-2).
- [ ] With no provider information on `ModelConfig`, `dispatchComplete` selects
      the Claude CLI completer.

## Milestone 2 — Carry provider identity on `ModelConfig` (no secret leak)

**Description.** Extend the internal `ModelConfig` so the dispatcher can pick and
configure a completer, without exposing secrets on any serialised surface
(FR-4, NFR-1).

**Files to change.**
- `internal/ideachat/llm.go` — add an optional provider reference to
  `ModelConfig`, e.g. `Provider *config.ProviderConfig` (reusing the existing
  app record type per FR-3; `internal/config` does not import `internal/ideachat`,
  so no import cycle is introduced). `ModelConfig` keeps **no** struct tags and
  is never marshalled to JSON/YAML anywhere (assert this in the test plan).
  `MaxTokens` already exists and is retained.

**Acceptance criteria.**
- [ ] `ModelConfig` carries enough to identify + configure the OpenAI completer
      (`base_url`, `api_key`, `extra_headers`, provider `driver`) via the
      embedded `*config.ProviderConfig`; a nil pointer means "CLI default".
- [ ] `ModelConfig` (and any type embedding it) is not referenced by any
      `writeJSON`/response payload; a grep-style assertion in the test plan
      confirms no serialisation path exists.
- [ ] No import cycle: `go build ./...` succeeds with `internal/ideachat`
      importing `internal/config`.

## Milestone 3 — OpenAI-compatible inline completer

**Description.** Add a plain, tool-free, **non-streaming**
`/v1/chat/completions` completer against a `config.ProviderConfig` (FR-3, FR-5,
FR-7). Standalone small client (RQ-2 decision above).

**Files to change.**
- `internal/ideachat/openai_completer.go` (new) — `type openAICompleter struct {
  provider config.ProviderConfig; client *http.Client }` implementing
  `Completer`. `Complete`:
  - Build `messages`: when `cfg.SystemPrompt != ""` prepend a `{"role":"system"}`
    entry, then map each `LLMMessage` to `{"role":<role>,"content":<content>}`
    preserving order and role (`user`/`assistant`) (FR-5).
  - Build body `{"model": cfg.Model, "messages": [...]}`; add `"max_tokens"`
    **only** when `cfg.MaxTokens > 0`; send **no** `"tools"` key; non-streaming
    (`stream` omitted/false).
  - `POST` to `strings.TrimRight(provider.BaseURL,"/") + "/v1/chat/completions"`
    with `context` from the caller. Headers: `Content-Type: application/json`;
    `Authorization: Bearer <api_key>` **iff** `api_key != ""`; apply each
    `extra_headers` entry.
  - On 2xx, decode `choices[0].message.content` and return it (trimmed to match
    the CLI completer's `TrimSpace` contract). Concatenate content if the shape
    is an array of parts.
  - On non-2xx, malformed JSON, empty choices, or transport/context error:
    return a wrapped `error` (never panic, never retry). Scrub any credential
    material from the error text (do not echo request headers/body).

**Acceptance criteria.**
- [ ] Against an `httptest` server, a `provider`+`model` `ModelConfig` produces a
      POST to `<base_url>/v1/chat/completions` carrying `model`, the mapped
      `messages` (system first when present, roles preserved in order), and
      `max_tokens` only when set; the body contains **no** `tools` key.
- [ ] `Authorization: Bearer` header present iff `api_key` non-empty; each
      `extra_headers` entry applied; `Content-Type: application/json` set.
- [ ] Returns the assistant text from `choices[0].message.content`.
- [ ] Non-2xx, malformed JSON, empty-choices, and unreachable/hung endpoints each
      surface as a returned `error` bounded by the request context — no panic,
      no unbounded retry (FR-7).
- [ ] The error string contains no `api_key` or `extra_headers` credential
      material (NFR-1).

## Milestone 4 — Dispatcher selection + resolver wiring

**Description.** Make `dispatchComplete` pick the completer from `ModelConfig`,
and teach the single `ModelConfig` builder to populate provider identity from
the agent config (FR-4).

**Files to change.**
- `internal/ideachat/llm.go` (or `completer.go`) — `dispatchComplete`: if
  `cfg.Provider == nil` → `claudeCLICompleter`; else switch on
  `cfg.Provider.Driver`: `"openai-compatible"` → `openAICompleter`; unknown
  driver → a clear returned error naming the driver (defensive; validation in
  M5 should prevent it reaching here).
- `internal/http/idea_chat.go` — extend `resolveIdeaCaptureConfig(p, templateKey)`:
  after resolving the owning agent, if `agent.Provider != ""` look it up in
  `p.Providers` (the app-level snapshot on `project.Project`); on hit, set
  `ModelConfig.Provider = &prov`. Empty `agent.Provider` leaves it nil (CLI
  default). This one function feeds **both** the converse endpoint
  (`handleIdeaConverse`) and the generate endpoint (`handleIdeaGenerate` via
  `idea_generate.go`), so all four template keys resolve through it. `MaxTokens`
  stays `0` in v1 (no agent field populates it), so the OpenAI path omits it and
  behaviour is preserved (documented, not a regression).

**Acceptance criteria.**
- [ ] An inline agent with **no** `provider` yields `ModelConfig.Provider == nil`
      and routes to the CLI completer (FR-8).
- [ ] An inline agent with `provider: <name>` (driver `openai-compatible`)
      resolves `p.Providers[<name>]` onto `ModelConfig.Provider` and routes to
      the OpenAI completer.
- [ ] The same resolution serves `idea-capture` (converse) and the three
      single-shot keys (`idea-generate`, `defect-generate`, `doc-generate`).
- [ ] An unknown/unregistered provider driver reaching the dispatcher returns a
      named error rather than panicking (belt-and-braces behind M5).

## Milestone 5 — Config validation for inline provider bindings (FR-6)

**Description.** Reject misconfigured inline agents with agent-naming errors,
consistent with the async-path rule in [[provider-model-for-agents]].

**Design note (seam).** `config.validateProject(*Project)` has **no** access to
the app-level provider list, and today performs **no** "provider registered"
check for any driver. Two distinct rules:
1. *Intra-project* (no app config needed): an `inline` agent with `provider` set
   MUST have a non-empty `model`. Add this to `validateProject` alongside the
   existing per-driver checks.
2. *Cross-config* (needs app providers): an `inline` agent whose `provider` is
   **not** present in app config is rejected. Implement as a new exported
   `config.ValidateAgentProviders(proj *Project, providers []ProviderConfig)
   error` invoked at the seam where both are available — `project.Open`
   (which already receives `OpenOptions.Providers`) and/or server bootstrap.
   The error names the offending agent and the missing provider.

**Files to change.**
- `internal/config/config.go` — in `validateProject`, extend the existing
  `else if a.Provider != ""` branch (and/or add an `a.Driver == "inline"` case)
  to enforce rule 1. Add `ValidateAgentProviders` for rule 2.
- `internal/project/project.go` — call `ValidateAgentProviders(cfg, opts.Providers)`
  in `Open` after config load, surfacing the error so a misconfigured project
  fails fast (matching how async provider agents are expected to fail).
- (If server bootstrap is the chosen seam instead) `cmd/kaos-control/*` or
  `internal/http/server.go` — call it there; pick one seam and document it.

**Acceptance criteria.**
- [ ] An `inline` agent with `provider` set and empty `model` is rejected with an
      error naming the agent.
- [ ] An `inline` agent whose `provider` is absent from app config is rejected
      with an error naming the agent and the missing provider.
- [ ] An `inline` agent with **no** `provider` validates exactly as today — no
      new mandatory field, no `model` requirement change (FR-6, NFR-4).
- [ ] Existing project/app configs load and validate without edits (NFR-4);
      the shipped `lifecycle/config.yaml` (inline agents, no `provider`) is
      unaffected.

## Milestone 6 — Regression & secret-hygiene hardening

**Description.** Prove no behavioural change for existing agents and no secret
leakage across logs/errors (FR-8, NFR-1, NFR-2).

**Files to change.**
- `internal/ideachat/*` — ensure any new `slog`/error paths scrub credentials;
  no `api_key` in log lines (mirror the async driver's masking discipline).
- No consumer-signature changes: `Converse`, `Generate`, `resolveSlug`,
  `handleIdeaConverse`, `handleIdeaGenerate` keep their current signatures and
  produced-artifact shapes.

**Acceptance criteria.**
- [ ] All pre-existing `internal/ideachat` tests pass unchanged (NFR-2).
- [ ] With the shipped config, the produced idea/defect/doc artifact shape is
      identical to pre-change output (FR-8) — asserted by the test plan.
- [ ] No `api_key`/`extra_headers` value appears in any log line, prompt,
      conversation history, generated artifact, or API response (NFR-1) — the
      OpenAI path reuses the provider masking already applied at the HTTP
      provider surface (`internal/http/providers.go:maskedProviders`).
- [ ] `make lint` (go vet + staticcheck) and `make test-unit` are green.

---

## Out of scope (from requirement Non-goals / resolved questions)

- Tool-calling / file-writing from the inline path; adding inline agents to the
  `Manager` driver map or async run pipeline; streaming (RQ-1); shared client
  extraction beyond a standalone small client (RQ-2); provider failover (RQ-4,
  see [[switch-provider]]); run/attribution recording (RQ-5, see
  [[agent-logging-provider-driver]]); any provider-selection UI (see the
  companion frontend plan, which is deliberately a no-net-new-UI plan).

## Verification map (requirement Acceptance Criteria → milestones)

- Completer interface + reassignable `CallLLM` → M1.
- CLI completer default, byte-identical invocation → M1, M4, M6.
- OpenAI completer POST shape / return → M3.
- `Authorization` iff key, `extra_headers`, no `tools` → M3.
- System→`system`, role order preserved → M3.
- Validation (unregistered provider; provider-without-model; no-provider as
  today) → M5.
- All four consumers via both completers → M4 (+ test plan `-5-test`).
- Failure behaviour bounded, no panic/retry; CLI error unchanged → M3, M1.
- No secret leakage → M2, M6.
- No new dependency/cgo; existing tests pass → M1–M6, NFR-3/NFR-2.
- Local-provider offline capture/generation → M3, M4 (validated in `-5-test`).
