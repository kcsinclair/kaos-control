---
title: "Token, cost and usage metrics for openai-compatible (OpenRouter) runs"
type: plan-test
status: draft
lineage: openrouter-token-metrics-extraction
parent: lifecycle/requirements/openrouter-token-metrics-extraction-2.md
created: "2026-09-04T12:00:00Z"
---

# Test Plan: OpenRouter Token Metrics Extraction

## Milestone 1: Pre-Execution Environment & Fixture Setup

**Description:** Prepare test fixtures simulating OpenRouter and non-OpenRouter providers, including valid usage chunks, cost metrics, and 400 rejection scenarios.

**Files to change:**
- `internal/agent/openai_compatible_test.go`
- `internal/agent/testdata/`

**Acceptance Criteria:**
- Multi-turn SSE mock responses exist with usage objects in the final, empty-choices chunk of each turn.
- A specific fixture models an OpenRouter stream with `usage.cost` included.
- A specific fixture models a local LLM rejecting `stream_options` with a HTTP 400.
- A mid-stream disconnect fixture simulates dropped connections.

## Milestone 2: Driver Request & Loop Validation

**Description:** Execute the `openai-compatible` driver against the test fixtures to ensure the new request shapes, loop accumulation, and terminal emission logic functions end-to-end.

**Files to change:**
- `internal/agent/openai_compatible_test.go`

**Acceptance Criteria:**
- Test executes against an OpenRouter mock, verifying the request body contains `stream_options` and `usage`.
- Test verifies the loop correctly accumulates tokens (deducting cached prompt tokens) and cost, and outputs a terminal `type:result` record matching expected totals.
- Test verifies that a simulated disconnect only counts tokens from the successful retry attempt.
- Test verifies the 400 rejection fixture correctly triggers a single retry (without usage parameters) and records `usage_source = "none"`.
- `driverEmitsResultEvent` is asserted to return true for `openai-compatible`.

## Milestone 3: End-to-End Metrics Persistence

**Description:** Validate that `ParseRunResult` correctly interprets the terminal record, persists metrics into `agent_runs`, and that those metrics surface in the API and reports.

**Files to change:**
- `internal/agent/agent_test.go`
- `internal/http/api_agent_runs_test.go`
- `internal/reports/agent_usage_test.go`

**Acceptance Criteria:**
- `agent.go`'s post-run parsing stores a populated `AgentRunMetrics` in the index and updates `metrics_available = 1`.
- The GET result endpoint returns the full `RunResult` object for an `openai-compatible` run.
- When `usage_source` is `"none"`, the API returns the result with `usage_source: "none"` instead of `result: null`, and `metrics_available` remains `0` in the database.
- Agent Usage report correctly includes the dummy OpenRouter runs in its cost/token aggregates, confirming vendor-prefix stripping in pricing logic.

## Milestone 4: Frontend Component Assertions

**Description:** Execute frontend unit tests ensuring the UI correctly responds to the new metrics capabilities.

**Files to change:**
- `web/src/components/agent/__tests__/RunSummaryCard.spec.ts`

**Acceptance Criteria:**
- Frontend unit tests pass for the three distinct states: fully populated metrics, degraded (usage unreported), and unsupported driver.
- The `cost` string renders as `—` (not zero) when `cost_reported` is false.