---
title: Token, cost and usage metrics for openai-compatible (OpenRouter) runs
type: requirement
status: draft
lineage: openrouter-token-metrics-extraction
created: "2026-09-04T11:20:00+10:00"
priority: high
parent: lifecycle/ideas/openrouter-token-metrics-extraction.md
labels:
    - requirement
    - driver
    - provider
    - observability
    - cost
    - agent-runner
release: KC-Release6
assignees:
    - role: product-owner
      who: agent
---

# Token, cost and usage metrics for openai-compatible (OpenRouter) runs

Parent: [[openrouter-token-metrics-extraction]] (idea). Related:
[[agent-run-summary-panel]], [[gemini-cli-stream-json]],
[[agent-usage-analytics-report]], [[open-provider-support]],
[[provider-model-for-agents]], [[agent-logging-provider-driver]],
[[local-model-operability]].

## Problem

Every agent in this project's shipped `lifecycle/config.yaml` now runs on
`driver: openai-compatible` against `provider: openrouter`. Those runs show
**"Token metrics not available for this driver."** in the run summary card, and
they are counted as `metrics_unavailable_count` in the agent usage report — so
the project's *primary* execution path is its least observable one, with no
token counts, no cost, and no per-model attribution.

The cause is a chain of three independent gaps, not a single parsing bug:

1. **The driver never asks for usage.** `OpenAICompatibleDriver.Start`
   (`internal/agent/openai_compatible.go`) posts
   `{"model","messages","tools","stream":true}` and consumes only
   `choices[].delta`. On a streaming OpenAI-compatible request, usage is
   **omitted unless `stream_options.include_usage` is requested**, and
   OpenRouter only returns `usage.cost` / `cost_details` when the request
   carries `usage: {"include": true}`. So the numbers the idea assumes are
   "typically included" are, for the requests kaos-control actually sends,
   genuinely absent from the response.
2. **The driver emits no terminal result record.** It writes
   `# event: completed` to the run log and sends a `{"type":"completed"}`
   progress event. `ParseRunResult(driver, log)` (`internal/agent/result.go`)
   dispatches `openai-compatible` to `ParseResultLine`, which scans for a
   `{"type":"result",…}` NDJSON line — there is none, so it returns
   `errNoResultLine`. `GET /api/p/:project/agents/runs/{id}/result` therefore
   answers `{"result": null, "reason": "no result line found in log"}`, and
   `agent.go`'s post-run `UpdateAgentRunMetrics` call is skipped, leaving
   `metrics_available = 0` and every metric column NULL.
3. **The frontend hard-codes a driver allow-list.** `agentHasTokenMetrics`
   (`RunDetailModal.vue`, `AgentsRunsView.vue`) returns `true` only for
   `claude-code-cli` / `claude-mediated`, so even a populated result would
   render the "not available for this driver" branch of `RunSummaryCard.vue`.

A further consequence: the tool-calling agent loop makes **one chat completion
per turn**, so per-run totals only exist if the driver accumulates usage across
turns. Nothing does that today.

This requirement closes all four gaps for the `openai-compatible` driver
(OpenRouter *and* every other provider that speaks the same wire format —
OpenAI, Ollama `/v1`, `llama-server`), reusing the existing `RunResult`,
`agent_runs` metric columns, summary card, and usage report rather than adding a
parallel metrics path.

## Goals / Non-goals

### Goals

- Have the `openai-compatible` driver **request** usage (and, on OpenRouter,
  cost) on every chat-completion request it makes.
- **Accumulate** usage across all turns of the tool-calling loop into one
  per-run total.
- Emit a **canonical terminal result record** into the run log and as a progress
  event, so the existing `RunResult` consumers (`GET …/result`, the WS finish
  payload, `UpdateAgentRunMetrics`) work unchanged.
- Populate the existing `agent_runs` metric columns and set
  `metrics_available = 1`, so [[agent-usage-analytics-report]] includes these
  runs in cost, token, throughput and cache-ratio aggregates.
- Render the full run summary card for `openai-compatible` runs (tokens, cost,
  duration, turns, cache-hit ratio) instead of the driver-unavailable message.
- Report **provider-reported** cost only; never fabricate or estimate a cost.
- Degrade cleanly and *distinguishably*: a provider that returns no usage yields
  "usage not reported by provider", not a card of zeros and not the
  driver-unsupported message.

### Non-goals

- Changing any other driver (`claude-code-cli`, `claude-mediated`, `claude-env`,
  `gemini-cli`, `codex-cli`, `shell-stub`) or the `RunResult` consumer contract.
  Their parsers and tests stay untouched.
- Maintaining a **local price catalogue for OpenRouter/OpenAI models**. Cost
  comes from the provider response or is absent (see
  [Architecture-Breaking Requirements](#architecture-breaking-requirements),
  item 5).
- Fetching remote pricing/model metadata to compute cost.
- A new metrics table, schema-version bump, or new API endpoint. The existing
  columns (`total_cost_usd`, `duration_api_ms`, `num_turns`, `input_tokens`,
  `cache_creation_tokens`, `cache_read_tokens`, `output_tokens`, `ttft_ms`,
  `metrics_available`) are sufficient.
- Capturing **reasoning/thinking tokens** as a first-class field. Same decision
  as [[gemini-cli-stream-json]] FR-6: not worth widening `RunResultUsage` for
  one driver in v1 (see Open Questions).
- Per-turn or per-tool-call metric breakdowns in the UI (run-level aggregate
  only, consistent with [[agent-run-summary-panel]]).
- Inline conversational calls (`internal/ideachat/`) — they produce no run row
  ([[inline-driver-provider-abstraction]] Resolved Question 5).
- Budget enforcement, spend alerts, or quota-based queue pausing.

## Detailed Requirements

### Architecture-Breaking Requirements

Reviewed against `lifecycle/architecture/architecture-summary.md`, the promoted
architecture ([[modular-monolith]]), the tech stack ([[go-vue]]), the ADRs, and
`standards/`. **No architecture-breaking requirement is introduced, and no new
ADR is required.** Each standing constraint from the Architecture Summary:

1. **Single self-contained binary (no cgo, no external services).** All work is
   inside `internal/agent`, `internal/index`, `internal/reports`, `internal/http`
   and `web/src`, using only `net/http` + `encoding/json` — the same building
   blocks the driver already uses. No new dependency, no cgo, no external
   datastore, no sidecar. → **Satisfied.**
2. **Local filesystem is the source of truth; the index is a rebuildable cache
   ([[index-is-a-cache]], [[adr-0003-pure-go-sqlite-index]]).** The **run log
   file on disk remains the authoritative record** of a run's metrics: the
   `agent_runs` metric columns are a derived convenience, and
   `GET …/runs/{id}/result` must keep re-deriving them from the log on demand
   (NFR-4). No schema-version bump is needed, so no cache rebuild is triggered.
   Nothing is written back into `lifecycle/` markdown. → **Satisfied.**
3. **Agents execute arbitrary tools; tool calls must be mediated
   ([[adr-0006-mediated-agent-driver-permission-model]],
   [[filesystem-sandboxing]]).** This change is read-only with respect to
   agent authority: it adds request parameters and parses response usage. The
   `ToolExecutor`/`PolicyConfig` path, `allowed_write_paths` enforcement, denial
   recording and `on_denial` behaviour are untouched. No metric is derived from
   an unmediated tool call. → **Satisfied / not applicable.**
4. **Direct-served, no trusted proxy hop
   ([[adr-0001-no-header-based-client-ip-trust]]).** No client-identity or
   header-trust surface is touched. → **Not applicable.**
5. **Cost minimisation at launch (Architecture Summary Q&A: "minimising cost at
   launch is a priority").** The obvious way to show cost for arbitrary
   OpenRouter models would be a **bundled or remotely-fetched price catalogue**.
   That is explicitly **rejected** here: a remote fetch would add an external
   runtime dependency (weakening constraint 1 and the "works with local
   providers" property of [[local-model-operability]]), and a bundled catalogue
   would silently go stale and report *wrong money*. Instead, cost is taken from
   the provider's own `usage.cost` when present and is otherwise **absent and
   labelled as such** (FR-5). Requesting usage/cost is itself free — no extra
   request, no extra tokens. → **Satisfied, with the alternative recorded as
   rejected.**
6. **Secrets hygiene ([[secrets-handling]]).** Provider API keys must not reach
   the log, the emitted result record, the index, the API, or the SPA. The
   driver's existing `mask()` wrapper around every `writeLog` call must also
   cover the new result record; the provider **name** (non-secret) may be
   recorded, the key never. → **Satisfied (enforced by NFR-2).**

**Conflicts flagged against the Architecture Summary: none.** If a later change
does want computed (rather than provider-reported) cost from a price table, that
is a new decision and MUST be raised as an ADR in
`lifecycle/architecture/decisions/` before implementation.

### Functional Requirements

#### FR-1: Request usage on every chat-completion request

- Every `/v1/chat/completions` request built by `OpenAICompatibleDriver.Start`
  MUST include `"stream_options": {"include_usage": true}` alongside the
  existing `model`, `messages`, `tools`, `stream` fields.
- When the resolved provider is an **OpenRouter gateway**, the request MUST also
  include `"usage": {"include": true}` so the response carries `cost` /
  `cost_details`. OpenRouter is detected with the **same rule already used by
  the preflight** (`internal/agent/openai_preflight.go`): `base_url` contains
  `openrouter.ai` **or** provider name is `openrouter`.
- The OpenRouter-only parameter MUST NOT be sent to other providers.
- Preflight capability probes (`VerifyToolCapability`) are unchanged — they are
  non-streaming and already read `usage.prompt_tokens`.

#### FR-2: Unknown-parameter fallback (must not break local providers)

- If a provider rejects the request with HTTP 400 **and** the error message
  references the usage parameters (e.g. contains `stream_options` or
  `include_usage` or `usage`), the driver MUST retry the *same* request once
  with those parameters removed, and continue the run.
- The fallback MUST be recorded once per run in the log
  (`# usage_params_unsupported: <provider>`), and the run then proceeds with
  `usage_source: "none"` (FR-6).
- The fallback MUST NOT consume the mid-stream disconnect retry budget
  (`providerDisconnectBackoff`) and MUST NOT be classified as
  `provider_disconnected`.

#### FR-3: Per-turn usage accumulation

- The driver MUST parse the usage object from each turn's stream. For a
  streaming response, usage arrives on a **final chunk whose `choices` array is
  empty**; the driver MUST read `usage` from any chunk that carries it, not only
  from chunks with choices.
- Run totals are the **sum over all turns** of the tool-calling loop, including
  turns that only produced tool calls.
- A turn that is retried after a mid-stream disconnect MUST contribute the usage
  of the **successful attempt only** — discarded partial attempts must not be
  double-counted.
- `num_turns` is the number of chat-completion round trips whose stream
  completed (i.e. loop turns actually executed), not the retry count.

#### FR-4: Field mapping onto `RunResult`

Map accumulated OpenAI/OpenRouter usage onto the existing `RunResult` /
`RunResultUsage` (`internal/agent/result.go`) as follows:

| `RunResult` field | Source |
|---|---|
| `Usage.InputTokens` | `Σ(prompt_tokens − prompt_tokens_details.cached_tokens)`, floored at 0 |
| `Usage.CacheReadInputTokens` | `Σ prompt_tokens_details.cached_tokens` (0 when absent) |
| `Usage.CacheCreationInputTokens` | `0` — no OpenAI-compatible equivalent; MUST NOT be inferred |
| `Usage.OutputTokens` | `Σ completion_tokens` |
| `TotalCostUSD` | `Σ usage.cost` when reported; otherwise `0` (see FR-5) |
| `NumTurns` | turn count per FR-3 |
| `DurationMs` | wall-clock ms from run start to terminal event |
| `DurationApiMs` | Σ per-turn provider request→stream-end elapsed ms |
| `Subtype` | `"success"` on clean completion; otherwise a short failure label |
| `IsError` | `false` on clean completion; `true` otherwise |
| `Result` | final assistant text on success; the classified failure text otherwise |
| `Model` | provider-reported `model` from the response when present, else the requested `run.Model` |

- Subtracting `cached_tokens` from `prompt_tokens` is required because
  OpenAI-compatible providers report cached tokens **inside** `prompt_tokens`;
  adding both would double-count and make the cache-hit ratio in
  `RunSummaryCard.vue` and `internal/reports/agent_usage.go` (denominator
  `input + cache_creation + cache_read`) wrong.
- `total_tokens` is **not** stored: it is derivable and there is no column for
  it.

#### FR-5: Cost is provider-reported or absent — never estimated

- `TotalCostUSD` is populated **only** from provider-reported cost
  (`usage.cost`, in USD).
- When no cost is reported, `TotalCostUSD` MUST be `0` and the terminal record
  MUST carry `cost_reported: false`, so the UI can distinguish "free/unreported"
  from "genuinely $0".
- The driver MUST NOT compute cost from token counts and a price table.
- `is_byok` / `cost_details.upstream_inference_cost`, when present, may be
  recorded in the terminal log record for diagnosis but MUST NOT be added to
  `TotalCostUSD`.

#### FR-6: Canonical terminal result record

- On every terminal outcome (success **and** classified failure) the driver MUST
  write exactly one canonical result line to the run log, as a single-line JSON
  object with `"type": "result"` and the `RunResult`-shaped fields of FR-4, plus
  the diagnostic fields:
  - `"driver": "openai-compatible"`,
  - `"provider": "<provider name>"` (non-secret name only),
  - `"usage_source"`: `"provider_stream"` | `"none"`,
  - `"cost_reported": true|false`.
- The same object MUST be emitted as a `ProgressEvent` so the WebSocket finish
  payload can carry the result without an extra API call
  ([[agent-run-summary-panel]] FR-5).
- `ParseRunResult(driver, log)` MUST return this record for
  `driver == "openai-compatible"`. Reusing `ParseResultLine` is acceptable
  **only because kaos-control now emits this line itself** (both ends of the
  shape are ours); the parser's doc comment MUST be updated to say the
  `type:result` line is the canonical kaos-control result shape rather than a
  Claude-only one. No existing parser behaviour may change
  ([[gemini-cli-stream-json]] NFR-1).
- The result line MUST be the last non-comment line before the log footer, so a
  backwards scan finds it in O(1) lines.
- Because the terminal record is now unconditional for this driver,
  `openai-compatible` MUST be added to `driverEmitsResultEvent` so a clean exit
  with **no** result event is classified as a truncated stream, consistent with
  the drivers already in that allow-list.

#### FR-7: Persist metrics to the run row

- On run finish, `agent.go`'s existing post-run parse MUST populate
  `index.AgentRunMetrics` from the parsed record and call
  `UpdateAgentRunMetrics`, setting `metrics_available = 1` for
  `openai-compatible` runs — the same path `claude-*` and `gemini-cli` use.
- `Model` MUST be stamped from the result record when it differs from the
  requested model (e.g. an OpenRouter upstream substitution), matching the
  existing "result reflects the actual model used" rule.
- `ttft_ms` continues to be recorded by the existing `OnTTFT` wiring — no
  change.
- When `usage_source` is `"none"`, `metrics_available` MUST stay `0` and the
  metric columns MUST stay NULL: a zero row would corrupt report averages.

#### FR-8: API result endpoint

- `GET /api/p/:project/agents/runs/{run_id}/result` MUST return the populated
  `result` object for a terminal `openai-compatible` run.
- When the provider reported no usage, it MUST return `{"result": <record>,
  …}` with `usage_source: "none"` — i.e. the outcome/duration/turns are still
  reported — rather than `{"result": null}`.
- Response shape, status codes, and the `run is still in progress` (409)
  behaviour are otherwise unchanged.

#### FR-9: Frontend — show the card, and label the degraded cases

- `agentHasTokenMetrics` (in **both** `web/src/components/agent/RunDetailModal.vue`
  and `web/src/views/project/AgentsRunsView.vue`) MUST treat
  `openai-compatible` as a metrics-capable driver. The driver id is resolved
  from the **run row** (`run.driver`, immutable at run start) in preference to
  the agent's current config, as today.
- `RunSummaryCard.vue` MUST render the full card for an `openai-compatible` run:
  outcome badge, cost, duration (wall + API), turns, the four-row token table,
  and the cache-hit ratio with its quality badge.
- Three states MUST be visually distinct:
  1. **metrics present** → full card;
  2. **driver capable but provider reported no usage** → card with
     outcome/duration/turns plus an explicit
     "Token usage not reported by this provider" note in place of the token
     table and cache ratio;
  3. **driver genuinely not metrics-capable** → the existing
     "Token metrics not available for this driver." message.
- When `cost_reported` is `false`, cost MUST render as `—` (not `$0.0000`).
- `cache_creation_input_tokens` is always 0 for this driver; the row remains
  visible with `0` for layout consistency with other drivers.

#### FR-10: Usage report attribution

- `openai-compatible` runs with metrics MUST contribute to
  [[agent-usage-analytics-report]] aggregates (`total_cost_usd`, token totals,
  `mean_output_tokens_per_second`, `cache_hit_ratio`, TTFT percentiles) and MUST
  stop being counted in `metrics_unavailable_count`.
- Per-model grouping MUST use the gateway-qualified model id as reported
  (e.g. `anthropic/claude-sonnet-5`), not a rewritten one.
- `internal/reports/pricing.go` `priceFor` MUST strip a leading `<vendor>/`
  segment before its prefix match, so gateway-qualified ids can resolve to a
  known family. An id that still matches no family MUST continue to yield **no
  input/output cost split** (`splitCost` returns `0, 0`) while the
  provider-reported total is reported unchanged — an unknown model must never
  produce invented split numbers.

### Non-Functional Requirements

#### NFR-1: No regression for other drivers or providers

- `ParseResultLine`, `ParseAgyResultLine`, and the `claude-*`, `gemini-cli`,
  `codex-cli`, `shell-stub` paths are behaviourally unchanged; their existing
  unit tests pass without modification.
- A local `openai-compatible` provider (Ollama `/v1`, `llama-server`) that
  ignores or rejects the new parameters completes its run exactly as before
  ([[local-model-operability]]).

#### NFR-2: Secret hygiene ([[secrets-handling]])

- The terminal result record, all new log lines, `error_details`, and every API
  response MUST pass through the driver's existing `mask()` / `maskSecretsInText`
  path. No `api_key`, `Authorization` header, or `extra_headers` credential may
  appear in the log, the record, the index, or the SPA.
- Only the non-secret provider **name** is recorded (as already done for
  `agent_runs.provider`, [[agent-logging-provider-driver]]).

#### NFR-3: Single-binary / stdlib only

- No new third-party dependency, no cgo, no external service, no build-tag
  change. Implementation stays within `net/http` + `encoding/json`
  ([[modular-monolith]], [[go-vue]]).

#### NFR-4: Log remains authoritative; index stays a cache

- Metrics MUST be re-derivable from the on-disk run log by
  `GET …/runs/{run_id}/result` with no dependency on the `agent_runs` row, so
  the SQLite copy stays a convenience projection ([[index-is-a-cache]]).
- No `schemaVersion` bump and no new column: the existing `agent_runs` columns
  are reused, so no cache rebuild is triggered on upgrade.

#### NFR-5: Backward compatibility with historical runs

- Runs recorded **before** this change keep `metrics_available = 0` and MUST
  render the degraded state without error — no migration, no backfill, no crash
  on NULL columns.
- Parsing a legacy `openai-compatible` log (no result line) MUST behave as
  today: a clear "no result line" reason, never a panic.

#### NFR-6: Performance and overhead

- Usage accumulation is O(1) per stream chunk and adds no additional HTTP
  request; the extra request fields add < 100 bytes per turn.
- Terminal-record parsing MUST remain a backwards line scan that stops at the
  first `type:result` line (no full-log JSON decode), so a multi-megabyte log
  still parses in well under 100 ms.

#### NFR-7: Observability of the degraded path

- When a provider returns no usage, that fact MUST be visible without reading
  the raw log: `usage_source: "none"` on the result record, a single
  `slog.Warn` (provider name only), and the FR-9 UI note.

## Acceptance Criteria

- [ ] Every `/v1/chat/completions` request from `OpenAICompatibleDriver` carries
      `stream_options.include_usage = true`; the OpenRouter `usage.include`
      parameter is present for an OpenRouter provider and absent for a
      non-OpenRouter one (asserted on the marshalled request body).
      ([[open-provider-support]])
- [ ] A provider returning HTTP 400 naming the usage parameters causes exactly
      one retry without them, the run completes, `usage_source` is `"none"`, and
      the disconnect-retry budget is untouched. ([[local-model-operability]])
- [ ] A recorded multi-turn SSE fixture (tool-call turn + final turn, usage on a
      choices-empty final chunk of each) accumulates into per-run totals; a
      retried turn is counted once. ([[open-provider-support]])
- [ ] `prompt_tokens_details.cached_tokens` maps to `cache_read_input_tokens`
      and is subtracted from `input_tokens`; `cache_creation_input_tokens` is 0;
      the cache-hit ratio computed by card and report matches the fixture by
      hand. ([[agent-run-summary-panel]])
- [ ] An OpenRouter fixture with `usage.cost` yields that exact
      `total_cost_usd` and `cost_reported: true`; a fixture without cost yields
      `total_cost_usd = 0`, `cost_reported: false`, and a `—` cost in the UI.
- [ ] Exactly one canonical `{"type":"result",…}` line is written per terminal
      run (success and failure), it is the last non-comment log line, and it is
      also emitted as a progress event.
- [ ] `ParseRunResult("openai-compatible", log)` returns the populated
      `RunResult`; `ParseResultLine` / `ParseAgyResultLine` behaviour is
      unchanged and existing driver tests pass. ([[gemini-cli-stream-json]])
- [ ] `driverEmitsResultEvent("openai-compatible")` is true, and a clean exit
      with no result event is classified as a truncated stream.
- [ ] After a real OpenRouter run, `agent_runs` has `metrics_available = 1` with
      non-NULL cost/token/turn/duration columns, and `model` reflects the
      provider-reported model. ([[agent-logging-provider-analytics]] not
      required — see [[agent-usage-analytics-report]])
- [ ] A run whose provider reported no usage leaves `metrics_available = 0` and
      the metric columns NULL (no zero rows polluting report averages).
- [ ] `GET …/runs/{run_id}/result` returns a populated result for an
      `openai-compatible` run, and a `usage_source: "none"` record (not
      `result: null`) when usage was unreported.
- [ ] The run summary card renders tokens, cost, duration, turns and cache ratio
      for an `openai-compatible` run in **both** `RunDetailModal.vue` and the
      inline expansion in `AgentsRunsView.vue`; the string "Token metrics not
      available for this driver" no longer appears for such runs.
      ([[agent-run-summary-panel]])
- [ ] The three UI states (metrics present / provider reported no usage / driver
      not capable) are individually asserted in component tests.
- [ ] The agent usage report includes OpenRouter runs in cost, token, throughput
      and cache-ratio aggregates, and their `metrics_unavailable_count`
      contribution drops to zero. ([[agent-usage-analytics-report]])
- [ ] `priceFor("anthropic/claude-sonnet-4-…")` resolves after vendor-prefix
      stripping; an unknown family still yields `splitCost = (0, 0)` while the
      provider-reported total is preserved.
- [ ] No API key, `Authorization` value, or `extra_headers` credential appears
      in the run log, the result record, the index, or any API response.
      ([[secrets-handling]])
- [ ] Metrics for a terminal run are re-derivable from the on-disk log alone;
      no `schemaVersion` bump and no new column were introduced.
      ([[index-is-a-cache]], [[adr-0003-pure-go-sqlite-index]])
- [ ] No new third-party dependency or cgo; `go vet` / `staticcheck` /
      `go test ./... -short` and the web lint/type-check/test targets are green.
      ([[modular-monolith]], [[go-vue]])
- [ ] Agent tool-call mediation, `allowed_write_paths` enforcement, and denial
      recording are unchanged.
      ([[adr-0006-mediated-agent-driver-permission-model]])
- [ ] Legacy pre-change runs render the degraded state without error.

## Resolved Questions

1. **Reasoning tokens.** OpenRouter/OpenAI report
   `completion_tokens_details.reasoning_tokens`, and reasoning-heavy models make
   this a material share of billed output. v1 folds it into `output_tokens` as
   the provider reports it and captures nothing separately (mirroring
   [[gemini-cli-stream-json]] FR-6's `thinking_tokens` decision). Do we want a
   dedicated field later — which means widening `RunResultUsage`, the API type,
   the summary card, and adding an `agent_runs` column for every driver?

> Yes roll it in for now, a dedicated field later.

2. **`cache_creation_input_tokens` for Anthropic-via-OpenRouter.** Anthropic
   models proxied through OpenRouter may expose cache-write counts under a
   vendor-specific key. v1 records 0 rather than guessing. Should we probe a
   real response and map it if a stable key exists, accepting a
   provider-specific special case in a generic driver?

> Yes.

3. **Claude-5-family list prices.** `internal/reports/pricing.go` only knows
   `claude-{opus,sonnet,haiku}-4`, while this project runs
   `anthropic/claude-{opus,sonnet}-5` — so even after vendor-prefix stripping
   (FR-10) the input/output cost **split** stays empty, though the total is
   correct. Add the 5-family prices (accepting the staleness risk the file's own
   doc comment warns about), or leave the split blank for unknown families?

> The dollar amounts are included in the final JSON output ,"cost":0.359644
> 
> Other cost data there as well.
> {"id":"gen-1788480411-QupvqjgY3QEIOKbv4XPa","object":"chat.completion.chunk","created":1788480411,"model":"anthropic/claude-sonnet-5","provider":"Claude Platform on AWS","service_tier":"default","choices":[{"index":0,"delta":{"content":"","role":"assistant"},"finish_reason":"stop","native_finish_reason":"end_turn"}],"usage":{"prompt_tokens":164082,"completion_tokens":3148,"total_tokens":167230,"cost":0.359644,"is_byok":false,"prompt_tokens_details":{"cached_tokens":0,"cache_write_tokens":0,"audio_tokens":0,"video_tokens":0},"cost_details":{"upstream_inference_cost":0.359644,"upstream_inference_prompt_cost":0.328164,"upstream_inference_completions_cost":0.03148},"completion_tokens_details":{"reasoning_tokens":2124,"image_tokens":0,"audio_tokens":0}}}

4. **BYOK / upstream cost.** When `is_byok` is true, OpenRouter's `cost` is the
   gateway fee, not the model spend (`cost_details.upstream_inference_cost`).
   Should the report show gateway cost only (v1), the sum, or both as separate
   figures?

> Show both.

5. **`stream_options` and switchover.** Should a provider that rejects the usage
   parameters (FR-2 fallback) be surfaced as a capability warning in Provider
   Settings, or is the per-run log note plus `slog.Warn` enough?
   ([[switch-provider]], [[open-provider-support]])

> per-run log note plus slog.Warn is enough

6. **Backfill.** Historical `openai-compatible` runs can never have usage (it
   was never requested), so no backfill is possible. Confirm that leaving them
   as `metrics_available = 0` — rather than deleting or annotating the rows — is
   the accepted end state.

> Is the usage data not in the final output?  If it is not available, then use metrics_available = 0
