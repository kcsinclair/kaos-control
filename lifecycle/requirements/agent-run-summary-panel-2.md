---
title: Agent Run Summary Panel with Token Efficiency Metrics
type: requirement
status: planning
lineage: agent-run-summary-panel
priority: high
parent: lifecycle/ideas/agent-run-summary-panel.md
labels:
    - agent
    - agent-runner
    - frontend
    - vue
    - feature
release: KC-Release1
---

# Agent Run Summary Panel with Token Efficiency Metrics

## Problem

When an agent run completes, the UI updates the run row status but does not surface *what the agent actually did* or *how efficiently it ran*. Users must click through to the raw log and manually locate the final `type:result` JSON line to understand outcomes and costs. There is no at-a-glance indication of whether an agent used the prompt cache effectively — a poor cache hit ratio can silently multiply costs by 10×+ on long-running runs. The raw log viewer currently opens inline or in a constrained panel, making long logs difficult to scroll through.

## Goals / Non-goals

### Goals

- Display a structured, human-readable summary of completed agent runs directly in the run detail expansion area, parsed from the `type:result` log line.
- Show token usage breakdown: input tokens, cache creation tokens, cache read tokens, output tokens.
- Show run cost (`total_cost_usd`), duration, and turn count.
- Calculate and display a **cache hit ratio** metric with a colour-coded quality label.
- Provide a full-height modal for viewing the complete raw log.

### Non-goals

- Historical token usage trends, burn-rate dashboards, or cost aggregation across runs (tracked separately in `claude-token-usage.md` §1–§2).
- Rate-limit detection or queue pausing (separate concern, `claude-token-usage.md` §3).
- Modifying the agent runner or log format — consume existing data only.
- Supporting Ollama driver token metrics (Ollama does not emit the same `type:result` shape; handle gracefully with a "metrics unavailable" state).

## Detailed Requirements

### Functional

#### FR-1: Result line parsing

- When a run reaches terminal status (`done`, `failed`, `killed`, `killed-timeout`), extract the last JSON line with `"type":"result"` from the run log.
- Parse the following fields: `total_cost_usd`, `duration_ms`, `duration_api_ms`, `num_turns`, `usage.input_tokens`, `usage.cache_creation_input_tokens`, `usage.cache_read_input_tokens`, `usage.output_tokens`.
- If the result line is missing or unparseable, display a "Summary unavailable" fallback — do not error.

#### FR-2: Summary display

- Show the parsed summary in a scrollable box within the expanded run detail area (the existing `expandedRun` section in `AgentsRunsView.vue` or `RunDetailModal.vue`).
- Layout the summary as a compact card with labelled fields:
  - **Status / Result**: terminal reason (completed, interrupted, etc.)
  - **Cost**: `total_cost_usd` formatted to 4 decimal places with `$` prefix.
  - **Duration**: `duration_ms` formatted as `Xm Ys` (wall clock) with API time in parentheses.
  - **Turns**: `num_turns`.
  - **Token usage table**: four rows — Input, Cache Creation, Cache Read, Output — each showing the raw token count.

#### FR-3: Cache hit ratio metric

- Calculate: `cache_hit_ratio = cache_read / (cache_read + cache_creation + input)`.
- Display as a percentage (1 decimal place).
- Apply a quality label and corresponding colour:
  - **Excellent** (green): ≥ 90%
  - **Good** (blue): ≥ 75%
  - **Fair** (amber/yellow): ≥ 50%
  - **Poor** (red): < 50%
- Show both the percentage and the label (e.g. "87.3% — Good").
- If all three denominator fields are zero or missing, display "N/A" instead.

#### FR-4: Raw log modal

- Replace the current inline/panel log viewer with a button ("View Full Log") that opens a **full-height modal** (minimum 90vh).
- The modal must:
  - Display the complete log content as monospaced, pre-formatted text.
  - Be scrollable with the scroll position starting at the top.
  - Include a close button (top-right × and Escape key).
  - Not navigate away from the current page.
- The log content is fetched from `GET /api/p/:project/agents/:run_id/log`.

#### FR-5: WebSocket integration

- When an `agent.finished` or `agent.failed` event is received via WebSocket, the summary should be displayable immediately without requiring a page refresh or additional API call.
- The result data may be included in the WebSocket event payload, or the frontend may parse it from the accumulated `progressLines` — whichever is simpler given the existing architecture.

### Non-functional

#### NFR-1: Performance

- Result line parsing must not block the UI. For logs over 1 MB, parsing should be performed asynchronously (e.g. in a `requestIdleCallback` or after the summary card skeleton renders).

#### NFR-2: Graceful degradation

- If a run was produced by a non-Claude-Code driver (e.g. Ollama) that does not emit a `type:result` line, the summary card should display "Token metrics not available for this driver" rather than hiding the panel entirely.
- If the run is still `running`, the summary section should not appear — only show it for terminal runs.

#### NFR-3: Visual consistency

- Follow existing UI patterns in `AgentsRunsView.vue` for card styling, spacing, and colour palette.
- The cache quality label colours should use the project's existing CSS variables or Tailwind utilities where available.

## Acceptance Criteria

- [ ] Expanding a completed Claude Code agent run displays a summary card with cost, duration, turns, and per-category token counts.
- [ ] The cache hit ratio is calculated correctly and shown with the appropriate quality label and colour for each threshold band.
- [ ] Runs with no result line (missing, corrupt, or non-Claude-Code driver) show a graceful fallback message instead of errors.
- [ ] Clicking "View Full Log" opens a full-height scrollable modal containing the raw log; Escape and × close it.
- [ ] The summary appears immediately when a run finishes via WebSocket — no manual refresh required.
- [ ] Running runs do not show a summary section.
- [ ] Token counts and cost match the values in the raw `type:result` JSON line (manual spot-check against log file).

## Resolved Questions

- Should the summary card also display `permission_denials` from the result line, or is that out of scope for this feature?

> yes, that is a great idea, any errors or issues should displayed.  We should allow for conditions we may not know.

- Should the per-model breakdown (`modelUsage.<model>`) be shown when a run uses mixed models (e.g. Opus + Sonnet), or is the aggregate sufficient for v1?

> These are single model jobs aggregate works for v1

- Is the existing `RunDetailModal.vue` the preferred location for the summary, or should it live in the inline expansion row in `AgentsRunsView.vue`? (Current idea text implies the inline expansion area.)

> Lets go with the existing RunDetailModal.
