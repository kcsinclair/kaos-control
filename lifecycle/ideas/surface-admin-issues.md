---
title: Surface Admin Issues in GUI
type: idea
status: draft
lineage: surface-admin-issues
priority: normal
labels:
    - observability
    - frontend
    - backend
    - reliability
    - websocket
    - feature
---

## Raw Idea

## Raw Idea
When kaos-control is logging warnings or errors, show this in the GUI, e.g. {"time":"2026-05-22T10:01:54.789012+10:00","level":"WARN","msg":"test artifact has been in in-qa for over 60 minutes","path":"lifecycle/tests/agent-run-summary-panel-6-test.md","age":"14h28m43s"}

These could be logged to a file
These could be sent on MQTT
These should shown the GUI

This is a condition, show them in the GUI with Parse Errors.

## Idea

When kaos-control emits structured log entries at WARN or ERROR level, these should be surfaced in the GUI so operators can see problems without tailing server logs. The existing parse-error display in the UI is the natural home for these: backend log alerts should appear alongside parse errors as a unified "conditions" panel.

The backend should capture WARN/ERROR log lines (e.g. stale artifact warnings, indexing failures) and forward them to connected clients over the existing WebSocket connection as a new event type. Optionally, alerts could also be written to a rotating log file or published on an MQTT topic for external consumers.

The frontend should render these alerts in a deduplicated, dismissible list, showing the log level, message, timestamp, and any structured fields (e.g. `path`, `age`). This gives operators immediate visibility into degraded conditions — such as a test artifact stuck in `in-qa` for hours — without leaving the UI.
