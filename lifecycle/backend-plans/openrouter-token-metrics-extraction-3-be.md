---
title: "Token, cost and usage metrics for openai-compatible (OpenRouter) runs"
type: plan-backend
status: in-development
lineage: openrouter-token-metrics-extraction
parent: lifecycle/requirements/openrouter-token-metrics-extraction-2.md
created: "2026-09-04T12:00:00Z"
---

# Backend Plan: OpenRouter Token Metrics Extraction

## Milestone 1: Data Structures and Request Modifications

**Description:** Define usage-related structs in `openai_compatible.go` to parse provider responses. Modify `OpenAICompatibleDriver.Start` to include `"stream_options": {"include_usage": true}` in chat completion requests, and add `"usage": {"include": true}` when the provider is OpenRouter. Wire up the fallback retry logic when a provider rejects the usage parameters with HTTP 400.

**Files to change:**
- `internal/agent/openai_compatible.go`

**Acceptance Criteria:**
- `openAIUsage` and `openAICostDetails` structs are defined to match OpenAI/OpenRouter stream usage responses.
- `openAIStreamChunk` includes a `Usage` field.
- Requests built by `OpenAICompatibleDriver.Start` unconditionally include `"stream_options": {"include_usage": true}`.
- If the provider base URL contains `openrouter.ai` or the name is `openrouter`, the request also includes `"usage": {"include": true}`.
- A 400 response complaining about usage parameters (message containing `stream_options`, `include_usage`, or `usage`) triggers exactly one retry without those fields, writes `# usage_params_unsupported: <provider>` to the log, flags `usage_source = "none"`, and does not consume the disconnect-retry budget.

## Milestone 2: Per-Turn Usage Accumulation

**Description:** Accumulate usage metrics and API duration across all turns of the tool-calling loop. Calculate accurate input (deducting cached tokens) and output tokens, and sum provider-reported cost (if any).

**Files to change:**
- `internal/agent/openai_compatible.go`

**Acceptance Criteria:**
- Turn usage is correctly accumulated: `prompt_tokens`, `cached_tokens` (from `prompt_tokens_details`), `completion_tokens`, and `cost`.
- If a turn is retried due to a mid-stream disconnect, only the successful attempt's usage is added to the totals.
- Turn API duration (time from request sent to stream end) is accumulated into a total `duration_api_ms`.
- Accumulated `input_tokens` is floored at 0 after subtracting `cached_tokens`. `cache_creation_input_tokens` remains 0.

## Milestone 3: Canonical Result Record Emission

**Description:** Emit a canonical terminal result record at the end of the `openai-compatible` run so it populates `RunResult` just like Claude streams. Update `ParseRunResult` configuration to recognise it.

**Files to change:**
- `internal/agent/openai_compatible.go`
- `internal/agent/agent.go`

**Acceptance Criteria:**
- On loop exit (success or max iterations/error), the driver logs exactly one `{"type":"result", ...}` JSON line as the final non-comment line.
- The record populates fields matching `RunResult` plus diagnostic fields: `driver: "openai-compatible"`, non-secret `provider` name, `usage_source` ("provider_stream" or "none"), and `cost_reported` (true/false).
- `TotalCostUSD` is strictly the accumulated `cost`; it is `0` if `cost_reported` is false.
- The same JSON payload is emitted as a `ProgressEvent` via the progress channel.
- `driverEmitsResultEvent` in `agent.go` is updated to return `true` for `"openai-compatible"`, enabling truncated-stream detection.
- `ParseRunResult` correctly parses this emitted line without modification to the `ParseResultLine` function logic itself (though doc comments should be updated).

## Milestone 4: Pricing Fallback Adjustment

**Description:** Ensure that the reports package gracefully handles OpenRouter gateway-qualified model IDs (e.g. `anthropic/claude-sonnet-5`). Update the pricing lookup to strip vendor prefixes before matching against known families.

**Files to change:**
- `internal/reports/pricing.go`

**Acceptance Criteria:**
- `priceFor` strips leading `<vendor>/` segments (everything up to the first slash) before checking family prefixes.
- `splitCost` returns `(0, 0)` for unknown families but preserves the API-reported total cost unchanged.

## Milestone 5: Backend Tests

**Description:** Add/update unit tests to verify the new openai-compatible usage extraction, fallback, and result-emission logic.

**Files to change:**
- `internal/agent/openai_compatible_test.go`
- `internal/reports/pricing_test.go`

**Acceptance Criteria:**
- Test confirming `"stream_options"` (and OpenRouter `"usage"`) is injected correctly.
- Test confirming fallback retry logic operates without consuming disconnect budgets on a 400 rejection.
- Test confirming turn-over-turn accumulation and mid-stream disconnect handling.
- Test confirming the terminal result event matches `RunResult` shape and `ParseRunResult` successfully parses it.
- Existing tests for other drivers continue to pass unchanged.
- Test confirming `priceFor` handles vendor-prefixed model IDs correctly.