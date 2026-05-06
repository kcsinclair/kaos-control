---
title: "Tests — Pipeline Log Streaming View"
type: test
status: draft
lineage: devops-pipeline-log-streaming
parent: lifecycle/test-plans/devops-pipeline-log-streaming-5-test.md
---

# Tests — Pipeline Log Streaming View

Integration and component-level tests for the pipeline log streaming split-pane feature. Tests cover the four milestones defined in the test plan.

## Test files

| File | Kind | Milestone |
|------|------|-----------|
| `tests/integration/devops_event_payload_test.go` | Go integration | 1 |
| `tests/integration/devops_run_log_test.go` | Go integration | 2 |
| `tests/web/PipelineLogPane.test.ts` | Vitest component | 3 & 4 |

## Milestone 1 — Backend event payload (`devops_event_payload_test.go`)

Run with: `go test ./tests/... -tags integration -run TestDevOpsEventPayload`

### Scenarios covered

- **All five event types received** — `TestDevOpsEventPayload_AllFiveTypes`: starts a two-step pipeline and asserts `pipeline.run.started`, `pipeline.step.started`, `pipeline.step.output`, `pipeline.step.completed`, and `pipeline.run.completed` are all delivered over the Hub in-process channel.
- **`pipeline.run.started` snake_case keys** — `TestDevOpsEventPayload_RunStartedKeys`: asserts `run_id`, `pipeline_slug` (not the legacy `pipeline` key), and `project` are present and non-empty.
- **`pipeline.step.started` keys + RFC 3339 timestamp** — `TestDevOpsEventPayload_StepStartedKeys`: asserts `run_id`, `pipeline_slug`, `step`, `step_index`, and a valid RFC 3339 `timestamp`.
- **`pipeline.step.output` keys + RFC 3339 timestamp** — `TestDevOpsEventPayload_StepOutputKeys`: asserts all output fields including `text`, `stream` (`stdout` or `stderr`), and a valid RFC 3339 `timestamp`.
- **`pipeline.step.completed` keys** — `TestDevOpsEventPayload_StepCompletedKeys`: asserts `run_id`, `pipeline_slug`, `step`, `step_index`, `status`, `exit_code`, `duration_seconds`.
- **`pipeline.run.completed` keys** — `TestDevOpsEventPayload_RunCompletedKeys`: asserts `run_id`, `pipeline_slug`, `project`, `status`, `duration_seconds`.
- **RFC 3339 across all events** — `TestDevOpsEventPayload_TimestampsAreRFC3339`: iterates all received events and validates every `timestamp` field that is present.
- **ANSI stripping** — `TestDevOpsEventPayload_ANSIStripped`: pipeline step uses `printf '\033[31mred text\033[0m\n'`; asserts the `text` field in `pipeline.step.output` contains no ESC characters.
- **Consistent run_id** — `TestDevOpsEventPayload_RunIDConsistentAcrossEvents`: captures the `run_id` from the trigger API response and verifies every pipeline event in the subsequent stream carries the same value.

## Milestone 2 — REST NDJSON log endpoint (`devops_run_log_test.go`)

Run with: `go test ./tests/... -tags integration -run TestDevOpsRunLog`

Uses `LogStore.ReadLogNDJSON` output, where each line is a flat JSON object with `type` merged from event payload fields (`time` and `event_type` log-store fields are not forwarded).

### Scenarios covered

- **Content-Type** — `TestDevOpsRunLog_ContentTypeIsNDJSON`: response header is `application/x-ndjson`.
- **Every line has `type`** — `TestDevOpsRunLog_EveryLineHasTypeField`: parses all NDJSON lines and fails on any with an absent or empty `type`.
- **All five event types** — `TestDevOpsRunLog_AllEventTypesPresent`: asserts each expected event type appears at least once.
- **Step boundary fields** — `TestDevOpsRunLog_StepBoundaryFields`: `pipeline.step.started` lines have `step`, `step_index`, RFC 3339 `timestamp`; `pipeline.step.completed` lines have `step`, `step_index`, `status`, `exit_code`, `duration_seconds`.
- **Final event fields** — `TestDevOpsRunLog_FinalEventHasStatusAndDuration`: last line is `pipeline.run.completed` with `status` and `duration_seconds`.
- **First event is `run.started`** — `TestDevOpsRunLog_FirstEventIsRunStarted`.
- **`pipeline_slug` field** — `TestDevOpsRunLog_PipelineSlugField`: `pipeline_slug` is present in run-level events; legacy `pipeline` key is absent.
- **Multi-step ordering** — `TestDevOpsRunLog_MultiStepSequenceOrder`: three-step pipeline; verifies run.started → (step.started → step.output* → step.completed) × 3 → run.completed with correct step names in order.

## Milestone 3 — Frontend split-pane and log pane (`PipelineLogPane.test.ts`)

Run with: `cd tests/web && pnpm vitest run PipelineLogPane`

Tests use Vitest + `@vue/test-utils` under happy-dom. `useVirtualScroll` is mocked to avoid `ResizeObserver` dependency. Scroll-based auto-follow tests manipulate DOM properties directly because happy-dom does not compute layout.

### SplitPane scenarios

- Top and bottom slot containers exist in the rendered DOM.
- Divider is present with `role="separator"`.
- Component starts in expanded state (`collapsed = false`).
- `collapsePane()` sets `collapsed = true` and applies `flex-basis: 0px` to the bottom pane.
- `expandPane()` after collapse restores `collapsed = false`.
- Toggle button click cycles between collapsed/expanded.
- `ArrowUp` on divider reduces `ratio`; `ArrowDown` increases it.

### PipelineLogPane scenarios

- Shows "Waiting for output" hint when `lines = []`.
- Renders `.log-row--output` elements for output lines.
- Renders `.log-row--step-start` and `.log-row--step-end` separator rows.
- Renders `.log-row--run-end` terminal row when `runCompleted = true`.
- Displays pipeline name in `.log-pane__title`.
- Step filter dropdown shown whenever any step names are present (component shows it for ≥ 1 step).
- Selecting a step hides other steps' output rows; selecting `__all__` restores them.
- Follow button absent when `autoFollow = true` (default).
- Follow button absent when `runCompleted = true`.
- Follow button appears after a scroll-up event (distFromBottom > 8 px).
- Clicking Follow button re-engages auto-follow and hides the button.
- `Escape` keydown on `.log-pane` emits the `collapse` event.
- `ArrowDown` / `ArrowUp` keys do not throw under happy-dom.
- `.log-pane` has `tabindex="0"` for keyboard focus.

## Milestone 4 — Virtual scrolling performance (`PipelineLogPane.test.ts`)

Extends the same test file with large-buffer assertions.

### Scenarios covered

- **Virtual mode activates above 10,000 lines**: `.log-pane__spacer` element is present when `lines.length > 10_000`.
- **No spacer for small line counts**: normal-flow rows are used when `lines.length` is small.
- **Rendered row count bounded**: in virtual mode, the number of `.log-row` elements inside `.log-pane__spacer` is < 200 (mock capped at 100 visible items).
- **50,000-line smoke test**: mounting with 50,000 lines does not throw.
