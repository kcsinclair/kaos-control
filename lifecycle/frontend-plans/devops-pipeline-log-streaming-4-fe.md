---
title: "Frontend Plan — Pipeline Log Streaming Split-Pane View"
type: plan-frontend
status: approved
lineage: devops-pipeline-log-streaming
parent: lifecycle/requirements/devops-pipeline-log-streaming-2.md
---

# Frontend Plan — Pipeline Log Streaming Split-Pane View

This plan implements the split-pane log streaming view for the pipeline detail page as specified in [[devops-pipeline-log-streaming]]. It replaces the modal-based `LogViewer.vue` workflow with a persistent, resizable bottom pane that streams pipeline output in real time.

## Milestone 1 — Resizable split-pane layout component

### Description

Create a generic `SplitPane.vue` component providing a vertical two-pane layout with a draggable divider. The component accepts a default split ratio, emits resize events, and supports collapse/expand of the bottom pane via a toggle button. The divider must be keyboard-operable (arrow keys when focused). On viewports < 768 px, the panes stack vertically at full width with the divider still functional.

### Files to change

- `web/src/components/common/SplitPane.vue` (new) — renders top/bottom slots separated by a draggable divider; manages split ratio state; handles pointer events for drag, keyboard events for accessibility, and responsive stacking via CSS media query.
- `web/src/components/common/SplitPane.css` (or scoped styles) — divider styling, cursor, focus ring, responsive breakpoint at 768 px.

### Acceptance criteria

- [ ] `SplitPane.vue` renders two slots (top, bottom) separated by a horizontal divider.
- [ ] Dragging the divider resizes the panes; pointer-up commits the ratio.
- [ ] Split ratio persists in component state for the session (not across reloads).
- [ ] Collapse/expand toggle hides the bottom pane and restores it.
- [ ] Divider is focusable via Tab and resizable via Up/Down arrow keys.
- [ ] On viewports < 768 px, panes stack vertically at full width.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 2 — Log pane component with auto-follow and virtual scrolling

### Description

Create `PipelineLogPane.vue`, the bottom-pane content that displays log lines. It must:
- Accept log lines as a reactive array prop (or via a composable).
- Auto-scroll to the bottom on new lines by default.
- Detect user scroll-up and pause auto-follow, showing a "Follow" button.
- Clicking "Follow" or scrolling to the bottom re-engages auto-follow.
- Implement virtual scrolling when the line count exceeds 10,000 to maintain < 16 ms frame budget. Buffer up to 50,000 lines; evict oldest beyond that.
- Render step-boundary separators (step name + timestamp) distinctly from output lines.
- Render a terminal status line (success/failure + duration) when the run completes.
- Be keyboard-navigable: Tab to focus, arrow keys to scroll, Escape to collapse (emits event to parent SplitPane).

### Files to change

- `web/src/components/devops/PipelineLogPane.vue` (new) — the log display component.
- `web/src/composables/useVirtualScroll.ts` (new) — virtual scrolling logic: computes visible window from scroll offset and container height, renders only visible items plus an overscan buffer, maintains a spacer for total height.

### Acceptance criteria

- [ ] Log lines render correctly with monospace font.
- [ ] Auto-follow keeps newest output visible; scrolling up pauses auto-follow.
- [ ] A "Follow" button/badge appears when auto-follow is paused; clicking it re-engages.
- [ ] Auto-follow stops when `pipeline.run.completed` is received.
- [ ] Step boundaries show step name and timestamp with distinct visual styling (e.g. coloured background bar).
- [ ] Completed-run terminal line shows status (success/failure) and duration.
- [ ] Virtual scrolling activates above 10,000 lines; DOM contains only visible rows + overscan.
- [ ] Buffer evicts oldest lines beyond 50,000 total.
- [ ] Keyboard navigation works: Tab focuses the pane, arrows scroll, Escape emits collapse.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 3 — Step filter dropdown

### Description

Add a step filter control to the log pane header. It lists "All steps" (default) plus each step that has appeared in the current run. Selecting a step filters the displayed log to only that step's output. Switching back to "All steps" restores the full stream. The filter must not discard buffered lines — it operates as a view filter over the full buffer.

### Files to change

- `web/src/components/devops/PipelineLogPane.vue` — add a `<select>` dropdown in the pane header; add a computed property that filters the log buffer by the selected step; feed the filtered list to the virtual scroller.

### Acceptance criteria

- [ ] Dropdown lists "All steps" and each pipeline step that has emitted output.
- [ ] Selecting a step shows only that step's output lines (plus its boundary separator).
- [ ] Switching back to "All steps" shows the full stream including all step boundaries.
- [ ] Buffered lines are preserved regardless of filter changes.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 4 — Integrate into pipeline detail page with WebSocket streaming

### Description

Wire `SplitPane.vue` and `PipelineLogPane.vue` into the pipeline detail page. The top pane contains the existing pipeline detail content. The bottom pane appears when a pipeline run is active or the user selects a recent completed run. Connect the log pane to the WebSocket hub via `useWebSocket` composable, subscribing to `pipeline.step.output`, `pipeline.run.started`, `pipeline.step.started`, `pipeline.step.completed`, and `pipeline.run.completed` for the currently viewed pipeline. For completed runs, fetch the full log via the REST endpoint (`GET /api/p/{project}/devops/runs/{run_id}`) and render it in the same pane — auto-follow disabled, user starts at the top.

The [[devops-pipeline-log-streaming]] backend plan ensures the REST endpoint and WebSocket payloads provide the fields this milestone consumes.

### Files to change

- `web/src/views/DevOpsPipelineDetailView.vue` (or equivalent pipeline detail view) — wrap existing content in `SplitPane` top slot; place `PipelineLogPane` in the bottom slot; manage show/hide state based on active run or user selection.
- `web/src/stores/devops.ts` — extend the store (or add a composable) to buffer log lines for the viewed pipeline, handle WebSocket events, and expose the reactive line array to the log pane. Increase per-step output cap from 1,000 to 50,000 to support the requirement.
- `web/src/api/devops.ts` — ensure `fetchRunLog()` parses NDJSON into the same line-object format used by the WebSocket stream so the log pane can render both sources identically.

### Acceptance criteria

- [ ] Pipeline detail page shows the split-pane layout when a run is active.
- [ ] WebSocket events for the viewed pipeline stream into the log pane in real time.
- [ ] `pipeline.run.started` clears previous log content and begins a new stream.
- [ ] Selecting a completed run fetches its log via REST and displays it in the same pane, auto-follow disabled.
- [ ] Log pane is hidden when no run is active or selected; collapse toggle works.
- [ ] No regressions to the existing agent run log viewing experience.
- [ ] Log buffer cleared on route change (navigating away from pipeline detail).
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.
