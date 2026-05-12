---
title: 'Frontend Plan: Agent Run Summary Panel'
type: plan-frontend
status: done
lineage: agent-run-summary-panel
parent: lifecycle/requirements/agent-run-summary-panel-2.md
release: KC-Release1
---

# Frontend Plan: Agent Run Summary Panel

relates-to: [[agent-run-summary-panel]]

## Overview

Add a structured summary card with token efficiency metrics to `RunDetailModal.vue`, a full-height raw log modal, and WebSocket-driven live updates. Depends on the API and WebSocket changes from [[agent-run-summary-panel-3-be]]. The [[agent-run-summary-panel-5-test]] test plan will verify the UI behaviour.

---

## Milestone 1 — API function and types for run result

### Description

Add a TypeScript interface for the result summary and an API function to fetch it from the new backend endpoint.

### Files to change

- `web/src/types/api.ts` — add new interfaces:
  ```ts
  export interface RunResultUsage {
    input_tokens: number
    cache_creation_input_tokens: number
    cache_read_input_tokens: number
    output_tokens: number
  }

  export interface RunResult {
    subtype: string
    total_cost_usd: number
    duration_ms: number
    duration_api_ms: number
    num_turns: number
    usage: RunResultUsage
    permission_denials: unknown[]
    session_id: string
  }
  ```

- `web/src/api/agents.ts` — add:
  ```ts
  export async function getRunResult(project: string, runId: string): Promise<{ result: RunResult | null; reason?: string }>
  ```
  Calls `GET /api/p/${project}/agents/runs/${runId}/result`.

### Acceptance criteria

- Types are exported and match the backend `RunResult` struct.
- API function returns `{ result: RunResult }` or `{ result: null, reason: string }`.
- `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 2 — RunSummaryCard component

### Description

Create a self-contained component that displays the parsed run result as a compact summary card with token usage table and cache hit ratio.

### Files to change

- `web/src/components/agent/RunSummaryCard.vue` — new SFC:
  - **Props**: `result: RunResult | null`, `driverAvailable: boolean` (true for Claude Code, false for Ollama).
  - **Computed values**:
    - `formattedCost`: `$` prefix, 4 decimal places.
    - `formattedDuration`: convert `duration_ms` to `Xm Ys` format, with `(API: Xm Ys)` from `duration_api_ms`.
    - `cacheHitRatio`: `cache_read / (cache_read + cache_creation + input)`, displayed as percentage with 1 decimal place. Display "N/A" when denominator is zero.
    - `cacheQuality`: object with `label` and `color` based on ratio thresholds:
      - `>= 0.90` → `{ label: 'Excellent', color: 'green' }`
      - `>= 0.75` → `{ label: 'Good', color: 'blue' }`
      - `>= 0.50` → `{ label: 'Fair', color: 'amber' }`
      - `< 0.50` → `{ label: 'Poor', color: 'red' }`
  - **Template layout**:
    - When `result` is `null` and `driverAvailable` is `false`: show "Token metrics not available for this driver" (NFR-2).
    - When `result` is `null` and `driverAvailable` is `true`: show "Summary unavailable" (FR-1 fallback).
    - When `result` is present:
      - **Header row**: Status/subtype badge, cost, duration, turns.
      - **Token usage table**: four rows (Input, Cache Creation, Cache Read, Output) with raw token counts.
      - **Cache hit ratio**: percentage + quality label with colour-coded badge using existing CSS variables.
      - **Permission denials / errors section** (conditionally shown): render each entry from `permission_denials` if non-empty, plus display any other issues or errors. Use a collapsible `<details>` for long lists.
  - **Styling**: follow `.rdm-*` CSS patterns from `RunDetailModal.vue`. Use existing badge CSS variables (`--badge-done-bg`, `--badge-approved-bg`, `--badge-blocked-bg`, `--badge-in-progress-bg`) for cache quality colours.

### Acceptance criteria

- Card renders all fields from FR-2 (cost, duration, turns, token table).
- Cache hit ratio calculation matches the formula in FR-3.
- All four quality labels render with correct colours at each threshold.
- "N/A" displayed when denominator is zero.
- Graceful fallback messages for missing results and non-Claude drivers.
- Permission denials are displayed when present.
- `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 3 — Integrate summary card into RunDetailModal

### Description

Wire `RunSummaryCard` into the existing `RunDetailModal.vue`, fetching the result data and displaying the card for terminal runs only.

### Files to change

- `web/src/components/agent/RunDetailModal.vue`:
  1. Import `RunSummaryCard` and `getRunResult` from the API module.
  2. Add reactive state: `runResult: ref<RunResult | null>(null)`, `resultLoading: ref(false)`.
  3. In `onMounted`, after fetching the run record, check if the run status is terminal (`done`, `failed`, `killed`, `killed-timeout`). If so, call `getRunResult(project, runId)` and populate `runResult`.
  4. Add `<RunSummaryCard>` in the template between the status/exit-code row and the stderr section. Pass `result` and `driverAvailable` (derive from `run.agent_name` or a heuristic — if the agent config is not available, default to `true`).
  5. Show a loading skeleton or "Loading summary..." text while `resultLoading` is true.
  6. Do **not** render the summary card when the run status is `running` (NFR-2).
  7. For async parsing of large logs (NFR-1): the backend handles parsing, so the frontend simply awaits the API response — no client-side log parsing needed in this code path.

### Acceptance criteria

- Expanding a completed run shows the summary card with all metrics.
- Running runs do not show a summary section.
- Loading state is visible while the result API call is in flight.
- Card gracefully handles `null` result from the API.
- Existing modal layout (run ID, agent, role, timestamps, stderr, artifacts) is unchanged.
- `pnpm build` passes with no errors.

---

## Milestone 4 — Full-height raw log modal

### Description

Replace the current inline log viewing with a "View Full Log" button that opens a full-height modal (FR-4).

### Files to change

- `web/src/components/agent/RawLogModal.vue` — new SFC:
  - **Props**: `project: string`, `runId: string`.
  - On mount, fetches the log via `agentsApi.getRunLog(project, runId)`.
  - Renders the log as monospaced, pre-formatted text in a scrollable container with `min-height: 90vh`.
  - Scroll position starts at the top.
  - Dismissible via close button (top-right), Escape key, and backdrop click.
  - Focus trapping consistent with `RunDetailModal.vue`.
  - Uses `<Teleport to="body">` and `z-index: 310` (above RunDetailModal's 300).
  - Loading state while log is fetched.

- `web/src/components/agent/RunDetailModal.vue`:
  1. Import `RawLogModal`.
  2. Add local state: `showRawLog: ref(false)`.
  3. Add a "View Full Log" button at the bottom of the modal body (after artifacts section).
  4. Conditionally render `<RawLogModal>` when `showRawLog` is true. On close, set `showRawLog = false`.

### Acceptance criteria

- "View Full Log" button appears in the run detail modal.
- Clicking it opens a full-height (≥90vh) scrollable modal with the complete log in monospaced text.
- Escape key and close button dismiss the log modal without closing the run detail modal behind it.
- Scroll position starts at the top of the log.
- Log content matches the raw file content from the API.
- `pnpm build` passes.

---

## Milestone 5 — WebSocket-driven live summary display

### Description

When a run completes and the `agent.finished` or `agent.failed` WebSocket event includes the `result` payload (from [[agent-run-summary-panel-3-be]] Milestone 3), make the summary available immediately without an additional API call.

### Files to change

- `web/src/stores/agents.ts` — in `onWsEvent`:
  1. For `agent.finished` and `agent.failed` events, extract the `result` field from the payload if present.
  2. Store the result in a new reactive map: `runResults: ref<Map<string, RunResult>>(new Map())`.
  3. Add a getter: `getRunResult(runId: string): RunResult | null`.

- `web/src/components/agent/RunDetailModal.vue`:
  1. On mount, check the store's `runResults` map first. If the result is already cached (from a WS event), use it directly instead of calling the API.
  2. If not cached, fall back to the API call (existing Milestone 3 behaviour).

### Acceptance criteria

- When a run finishes while `RunDetailModal` is open for that run, the summary card appears without a page refresh or manual re-open.
- When opening `RunDetailModal` for a recently finished run (result still in store), no API call is made for the result.
- For runs that finished before the page loaded (not in store), the API call path still works.
- `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 6 — Visual polish and edge cases

### Description

Final pass to ensure visual consistency, responsive behaviour, and edge case handling.

### Files to change

- `web/src/components/agent/RunSummaryCard.vue`:
  1. Ensure token counts use locale-aware number formatting (e.g. `12,345` not `12345`).
  2. Ensure the card is visually consistent across light and dark themes (if applicable).
  3. Add `aria-label` attributes to the cache quality badge for accessibility.

- `web/src/components/agent/RawLogModal.vue`:
  1. Handle empty logs (display "No log content available").
  2. Handle fetch errors (display error message, not a blank modal).

- `web/src/components/agent/RunDetailModal.vue`:
  1. Ensure the "View Full Log" button is disabled and shows a tooltip when no log is available (run has no log file).

### Acceptance criteria

- Token counts display with thousands separators.
- Cache quality badge has an `aria-label` (e.g. "Cache efficiency: 87.3% — Good").
- Empty and error states for the raw log modal are handled gracefully.
- No console errors or warnings in any state.
- Visual appearance follows existing `.rdm-*` styling patterns.
