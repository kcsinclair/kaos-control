---
title: "Token, cost and usage metrics for openai-compatible (OpenRouter) runs"
type: plan-frontend
status: draft
lineage: openrouter-token-metrics-extraction
parent: lifecycle/requirements/openrouter-token-metrics-extraction-2.md
created: "2026-09-04T12:00:00Z"
---

# Frontend Plan: OpenRouter Token Metrics Extraction

## Milestone 1: Enable Metrics Display for openai-compatible Driver

**Description:** Update the frontend driver allow-lists to treat `openai-compatible` as metrics-capable, and enhance the summary card to handle the new degraded/unreported state.

**Files to change:**
- `web/src/components/agent/RunDetailModal.vue`
- `web/src/views/project/AgentsRunsView.vue`
- `web/src/components/agent/RunSummaryCard.vue`

**Acceptance Criteria:**
- `agentHasTokenMetrics` functions in `RunDetailModal.vue` and `AgentsRunsView.vue` return `true` for `openai-compatible` (resolved from `run.driver`).
- `RunSummaryCard.vue` displays the full metrics card (outcome badge, cost, duration, turns, token table, cache-hit ratio) for `openai-compatible` runs.
- The UI differentiates three states clearly:
  1. **Metrics present**: Full card is rendered.
  2. **Driver capable but usage not reported**: (Detected via `result.usage_source == "none"`). Displays outcome/duration/turns, and a note reading "Token usage not reported by this provider" instead of the token table.
  3. **Driver genuinely not metrics-capable**: Displays the legacy "Token metrics not available for this driver." message.

## Milestone 2: Cost Formatting Refinement

**Description:** Ensure cost rendering correctly distinguishes between "$0.00" reported by the provider and "unreported". 

**Files to change:**
- `web/src/components/agent/RunSummaryCard.vue`

**Acceptance Criteria:**
- If `cost_reported` is `false`, the cost UI renders as `—` rather than `$0.0000`.
- The `cache_creation_input_tokens` row remains visible with a value of `0` for layout consistency, rather than being hidden.
- The string "Token metrics not available for this driver" never appears for `openai-compatible` runs.

## Milestone 3: Frontend Component Tests

**Description:** Add Vue component tests asserting the new rendering states of `RunSummaryCard.vue`.

**Files to change:**
- `web/src/components/agent/__tests__/RunSummaryCard.spec.ts`

**Acceptance Criteria:**
- A test asserts full metric display when all usage/cost data is provided.
- A test asserts the "Token usage not reported by this provider" fallback state when `usage_source: "none"` is passed.
- A test asserts the cost renders as `—` when `cost_reported: false`.