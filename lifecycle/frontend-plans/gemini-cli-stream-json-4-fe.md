---
title: Frontend plan — render agy stream-json progress & summary for gemini-cli runs
type: plan-frontend
status: done
lineage: gemini-cli-stream-json
created: "2026-08-25T14:00:00+10:00"
parent: lifecycle/requirements/gemini-cli-stream-json-2.md
release: KC-Release6
---

# Frontend plan — agy stream-json progress & summary

Implements the frontend of [[gemini-cli-stream-json]] (requirement
`gemini-cli-stream-json-2`). Companion plans: backend `-3-be`, test `-5-test`.

## Architecture conformance

Confirmed against `lifecycle/architecture/` (go-vue tech stack, embedded SPA
[[adr-0004-embedded-spa-single-binary]]): all work is TypeScript/Vue in
`web/src`, no new dependency, no build/embedding change. No deviation, no ADR.

## Context / key facts

- **`RunSummaryCard.vue`** already renders from the `RunResult`
  (`web/src/types/api.ts:650`) using `is_error`, `subtype`, `total_cost_usd`,
  `duration_ms`, `num_turns`, and `usage.*`. Because the backend (plan `-3-be`)
  fills the **same** `RunResult` struct for agy runs, the summary card needs
  **no field changes**: turns/duration/tokens populate, cache-creation shows 0,
  and `total_cost_usd:0` already renders `$0.0000` without error. This plan's
  summary-card work is therefore **verification**, not new code.
- **Live progress** is normalized in `web/src/stores/agents.ts` by the
  `progressText` function, which switches on `ev.type`. agy events are
  discriminated by **`ev.event`** (`init` / `step_update` / `result`) with the
  payload nested under a same-named key — none of which the current switch
  understands, so today they would fall through to `JSON.stringify(ev)`. This is
  the real frontend gap (FR-5: "`step_update` events appear as streaming
  progress in the Agents view").
- The backend forwards the parsed agy object verbatim as the progress event's
  `event` payload (plan `-3-be`, M5), so the store receives agy's native shape.

## Milestone 1 — Normalize agy progress events in the store (FR-5)

**Description.** Extend `progressText` in `web/src/stores/agents.ts` with an
agy branch that keys off `ev.event`:

- `event === 'init'` → a concise session-start line (e.g. `▸ session started`),
  optionally noting `init.cwd`.
- `event === 'step_update'` → render the human-readable content: prefer
  `step_update.text_delta` when present and non-empty (trimmed), else a compact
  status line from `step_update.step_type` / `step_update.state` so tool/thinking
  steps without text still show progress. This is what makes the Agents view
  stream incrementally.
- `event === 'result'` → a terminal line from `result.status` (e.g.
  `▸ result: success`), mirroring the existing Claude `result` handling.

Place the branch so it does not disturb the existing Ollama/Claude/openai
handling (those key off `ev.type`, which agy events lack, so the branches are
naturally disjoint). Keep the `JSON.stringify` fallback for genuinely unknown
shapes (NFR-2 tolerance on the client side).

**Files to change.**
- `web/src/stores/agents.ts` — `progressText` (agy `event` branch).
- `web/src/types/api.ts` — add optional agy progress-payload typings if needed
  to keep `progressText` typed (e.g. an `AgyProgressEvent` shape); no change to
  `RunResult`.

**Acceptance criteria.**
- A `step_update` with a non-empty `text_delta` renders that text as a progress
  line in the Agents view (no `JSON.stringify` blob).
- A `step_update` without `text_delta` still renders a meaningful status line.
- `init` and `result` events render concise lines consistent with existing
  driver formatting.
- No regression to Ollama/Claude/openai progress rendering.

## Milestone 2 — Verify the run summary card for a gemini-cli run

**Description.** Confirm (and only adjust if a gap surfaces) that
`RunSummaryCard.vue` and `RunDetailModal.vue` render correctly from an agy-filled
`RunResult`: `is_error` drives the outcome badge (SUCCESS → not-error),
`num_turns`, `duration_ms`, and `usage.{input,output,cache_read}_tokens`
populate, `cache_creation_input_tokens` shows 0, and cost renders as `$0.0000`.
The cache-quality ratio computation (`RunSummaryCard.vue:49`) divides by
`cache_read + cache_creation + input`; verify it does not `NaN` when creation is
0 (it won't, since input is non-zero) — add a guard only if a zero-denominator
edge is observed.

**Files to change.**
- None expected. If verification reveals a gap (e.g. a `NaN`/undefined guard),
  patch `web/src/components/agent/RunSummaryCard.vue` minimally.

**Acceptance criteria.**
- The summary card renders for a real `gemini-cli` run: turns, duration, and
  token usage populated; cost shown as zero; outcome badge reflects `is_error`.
- A non-success agy run shows the error outcome and the failure reason surfaces
  in the detail modal / failure banner as it does for other drivers.

## Out of scope

No changes to reports views, cost charts (agy cost is always 0 by FR-6), or any
non-`gemini-cli` driver rendering.
