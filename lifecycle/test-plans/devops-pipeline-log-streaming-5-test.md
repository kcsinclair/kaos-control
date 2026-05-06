---
title: "Test Plan — Pipeline Log Streaming View"
type: plan-test
status: done
lineage: devops-pipeline-log-streaming
parent: lifecycle/requirements/devops-pipeline-log-streaming-2.md
---

# Test Plan — Pipeline Log Streaming View

Integration and end-to-end tests for the pipeline log streaming split-pane feature described in [[devops-pipeline-log-streaming]]. Tests validate backend payload correctness, REST log retrieval, and frontend behaviour (split-pane layout, streaming, auto-follow, filtering, virtual scrolling, accessibility, and responsiveness).

## Milestone 1 — Backend event payload tests

### Description

Test that the WebSocket event payloads emitted by the pipeline runner have correct snake_case JSON keys, include timestamps, and that ANSI escape codes are stripped from output text. These tests exercise the code delivered in [[devops-pipeline-log-streaming]] backend plan milestones 1 and 2.

### Files to change

- `tests/devops_event_payload_test.go` (new) — integration tests that:
  - Start a minimal pipeline run (a trivial shell step).
  - Connect a WebSocket client to `/api/p/{project}/ws`.
  - Capture `pipeline.run.started`, `pipeline.step.started`, `pipeline.step.output`, `pipeline.step.completed`, `pipeline.run.completed` events.
  - Assert each event has the expected JSON keys (`run_id`, `pipeline_slug`, `step`, `step_index`, `timestamp`, `text`, `stream`, `status`, `exit_code`, `duration_seconds`).
  - Assert `timestamp` fields are valid RFC 3339.
  - Run a step that emits ANSI-coloured output and assert the `text` field contains no ANSI escape sequences.

### Acceptance criteria

- [ ] Test starts a pipeline run and receives all five event types over WebSocket.
- [ ] All payload fields use snake_case keys.
- [ ] `timestamp` fields parse as valid RFC 3339.
- [ ] ANSI escape sequences are absent from `pipeline.step.output` text.
- [ ] Tests pass with `go test ./tests/... -run TestDevOpsEventPayload`.

---

## Milestone 2 — REST completed-run log endpoint tests

### Description

Test the `GET /api/p/{project}/devops/runs/{run_id}` endpoint returns correctly structured NDJSON with event types, step boundaries, and the terminal status line. Validates [[devops-pipeline-log-streaming]] backend plan milestone 3.

### Files to change

- `tests/devops_run_log_test.go` (new) — integration tests that:
  - Start and await completion of a multi-step pipeline.
  - Fetch `GET /api/p/{project}/devops/runs/{run_id}`.
  - Assert response Content-Type is `application/x-ndjson`.
  - Parse each NDJSON line and assert every line has a `type` field.
  - Assert the sequence includes `pipeline.run.started`, at least one `pipeline.step.started`, at least one `pipeline.step.output`, at least one `pipeline.step.completed`, and `pipeline.run.completed`.
  - Assert step boundary lines include `step`, `step_index`, `timestamp`.
  - Assert the final `pipeline.run.completed` line includes `status` and `duration_seconds`.

### Acceptance criteria

- [ ] Test completes a pipeline run and fetches its log via REST.
- [ ] Every NDJSON line has a `type` field.
- [ ] Step boundary events include `step`, `step_index`, `timestamp`.
- [ ] Final event includes `status` and `duration_seconds`.
- [ ] Content-Type is `application/x-ndjson`.
- [ ] Tests pass with `go test ./tests/... -run TestDevOpsRunLog`.

---

## Milestone 3 — Frontend split-pane and log pane tests

### Description

End-to-end browser tests (or component-level tests if browser automation is not available) validating the split-pane layout, log streaming, auto-follow, step filtering, and completed-run display. Validates [[devops-pipeline-log-streaming]] frontend plan milestones 1–4.

### Files to change

- `tests/devops_log_pane_e2e_test.go` (new) or `tests/frontend/devops-log-pane.spec.ts` (new, if a JS test runner is used) — tests covering:
  1. **Split-pane renders**: navigate to pipeline detail, trigger a run, assert the log pane appears below the detail content.
  2. **Real-time streaming**: trigger a multi-step pipeline, assert log lines appear in the pane as steps execute, assert step boundary separators render with step name.
  3. **Auto-follow**: programmatically scroll the log pane up, assert the "Follow" button appears; click it, assert pane scrolls to the bottom.
  4. **Step filter**: select a specific step in the dropdown, assert only that step's lines are visible; switch back to "All steps", assert the full stream is restored.
  5. **Completed run**: after a pipeline completes, navigate away and back, select the completed run, assert the log pane shows the full log fetched via REST with auto-follow disabled.
  6. **Collapse/expand**: click the collapse toggle, assert the log pane is hidden; click expand, assert it reappears.
  7. **Keyboard navigation**: Tab to divider, press arrow keys, assert divider moves; Tab to log pane, press Escape, assert pane collapses.
  8. **Responsive layout**: set viewport to < 768 px, assert panes stack vertically.

### Acceptance criteria

- [ ] Split-pane appears when a pipeline run is active.
- [ ] Log lines stream in real time via WebSocket.
- [ ] Step boundaries are visually delineated.
- [ ] Auto-follow pauses on scroll-up; "Follow" button re-engages it.
- [ ] Step filter limits output to selected step; "All steps" restores full stream.
- [ ] Completed run log displays via REST endpoint in the same pane layout.
- [ ] Collapse/expand toggle works.
- [ ] Keyboard navigation of divider and log pane works.
- [ ] Panes stack on narrow viewports.
- [ ] No regressions to existing agent run log viewing.

---

## Milestone 4 — Performance test for large log buffers

### Description

Validate NF1: the log pane handles 10,000+ lines without perceptible scroll jank, and virtual scrolling engages correctly. This may be a focused benchmark or a test that asserts DOM element count stays bounded.

### Files to change

- `tests/devops_log_perf_test.go` (new) or `tests/frontend/devops-log-perf.spec.ts` (new) — test that:
  - Generates a pipeline run producing > 10,000 output lines (or mocks the WebSocket stream).
  - Asserts the log pane renders without error.
  - Asserts the number of DOM elements in the log container is bounded (e.g. < 200 rows rendered at once, indicating virtual scrolling is active).
  - If possible, measures scroll frame timing and asserts < 16 ms per frame.

### Acceptance criteria

- [ ] Log pane renders 10,000+ lines without error.
- [ ] Virtual scrolling limits rendered DOM elements to a bounded window (not all 10,000).
- [ ] Scroll interaction does not cause visible jank (frame budget assertion if measurable).
- [ ] Buffer correctly evicts oldest lines beyond 50,000.
