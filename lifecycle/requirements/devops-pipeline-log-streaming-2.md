---
title: DevOps Pipeline Log Streaming View
type: requirement
status: approved
lineage: devops-pipeline-log-streaming
created: "2026-05-06T00:00:00+10:00"
priority: normal
parent: lifecycle/ideas/devops-pipeline-log-streaming.md
labels:
    - feature
    - frontend
    - agent-runner
    - usability
assignees:
    - role: product-owner
      who: agent
---

# DevOps Pipeline Log Streaming View

## Problem

When a DevOps pipeline is running, users must open a modal (`LogViewer.vue`) to see log output. This obscures the pipeline configuration and other page context, forces a context switch, and does not provide a persistent, always-visible log stream during execution. Users cannot monitor pipeline progress while simultaneously reviewing the pipeline definition, step list, or other artifacts.

## Goals / Non-goals

### Goals

1. Provide a persistent, split-pane log panel that streams pipeline output in real time while the user retains visibility of the pipeline detail view above.
2. Auto-follow new log output by default, with the ability to scroll back without losing the current read position.
3. Reuse and extend the existing WebSocket event infrastructure (`pipeline.step.output`, `pipeline.run.started`, `pipeline.run.completed`) — no new transport mechanism.
4. Support viewing logs across all steps of a running pipeline in a single unified stream, with clear step boundaries.
5. Maintain consistency with the agent run log viewing experience where practical.

### Non-goals

- **Terminal emulation**: no ANSI colour parsing, interactive input, or PTY features.
- **Log persistence UI**: historical log retrieval and search across past runs is out of scope; this requirement covers live and just-completed runs only.
- **Pipeline editing**: the top pane is read-only context; inline editing of pipeline YAML is not part of this feature.
- **Multi-pipeline view**: only one pipeline run's logs are displayed at a time.

## Detailed Requirements

### Functional

#### F1 — Split-pane layout

- When a pipeline run is active (or the user selects a recent run), the pipeline detail page must split into two vertical panes:
  - **Top pane**: existing pipeline detail content (configuration, step list, status badges).
  - **Bottom pane**: scrolling log output panel.
- The divider between panes must be draggable so the user can resize the split. The position must persist for the duration of the session (not required to persist across page reloads).
- A collapse/expand toggle must allow the user to hide the log pane entirely and restore the full-height detail view.

#### F2 — Real-time log streaming

- The log pane must subscribe to the existing WebSocket hub and render `pipeline.step.output` events as they arrive, appending each line to the visible log.
- When a `pipeline.run.started` event is received for the viewed pipeline, the log pane must clear any previous content and begin streaming.
- When `pipeline.step.started` is received, a visual separator (step name and timestamp) must be inserted into the log stream.
- When `pipeline.run.completed` is received, a terminal status line (success/failure + duration) must be appended and auto-follow must stop.

#### F3 — Auto-follow with scroll lock

- By default the log pane must auto-scroll to keep the most recent output visible.
- When the user scrolls upward (away from the bottom), auto-follow must pause and a visible indicator (e.g. "Follow" button or "New output below" badge) must appear.
- Clicking the indicator or scrolling back to the bottom must re-engage auto-follow.
- Auto-follow state must not interfere with the user's scroll position in the top pane.

#### F4 — Step filtering

- The log pane must include a control to filter output by individual pipeline step.
- "All steps" must be the default. Selecting a specific step must show only that step's output lines.
- Changing the filter must not discard buffered lines — switching back to "All steps" must restore the full stream.

#### F5 — Log pane for completed runs

- When the user views a pipeline run that has already completed, the log pane must fetch the full log via the existing REST endpoint and display it in the same split-pane layout.
- Auto-follow must be disabled for completed runs (the user starts at the top).

### Non-functional

#### NF1 — Performance

- The log pane must handle at least 10,000 buffered lines without perceptible scroll jank (< 16 ms frame budget at 60 fps).
- If the buffer exceeds 10,000 lines, the oldest lines may be evicted from the DOM (virtual scrolling) while remaining available via scroll-back up to the full buffer limit of 50,000 lines.

#### NF2 — Accessibility

- The log pane must be keyboard-navigable (Tab to focus, arrow keys to scroll, Escape to collapse).
- The split-pane divider must be operable via keyboard (arrow keys when focused).

#### NF3 — Responsiveness

- On viewports narrower than 768 px, the split pane must stack vertically with the log pane below, taking the full width. The draggable divider must remain functional.

## Acceptance Criteria

- [ ] Pipeline detail page shows a bottom log pane when a run is active or selected.
- [ ] Log output streams in real time via the existing WebSocket hub — no polling.
- [ ] Step boundaries are visually delineated in the log stream.
- [ ] Auto-follow scrolls to newest output; scrolling up pauses auto-follow; a "Follow" affordance re-engages it.
- [ ] Pane divider is draggable to resize top/bottom split.
- [ ] Collapse/expand toggle hides and restores the log pane.
- [ ] Step filter control limits displayed output to a single step or all steps.
- [ ] Completed pipeline runs display their full log in the same pane layout.
- [ ] 10,000-line log renders without scroll jank.
- [ ] Log pane is keyboard-navigable (focus, scroll, collapse).
- [ ] Layout stacks correctly on viewports < 768 px wide.
- [ ] No regressions to existing agent run log viewing ([[devops-pipelines]]).

## Resolved Questions

1. Should the step filter also be available as clickable step labels in the top pane (click a step → filter log to that step), or is a standalone dropdown sufficient?

> standalone dropdown works.

2. Is there a maximum log retention duration for in-memory buffers when the user leaves the pipeline page and returns — should the buffer survive navigation, or is clearing on route change acceptable?

> Clearing on route change is acceptable.

3. Should ANSI escape codes be stripped server-side (current behaviour) or should basic colour support be added as a future enhancement?

> Strip ANSI server side.
