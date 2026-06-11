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
    - usability
---

## Raw Idea

## Raw Idea
When kaos-control is logging warnings or errors, show this in the GUI, e.g. {"time":"2026-05-22T10:01:54.789012+10:00","level":"WARN","msg":"test artifact has been in in-qa for over 60 minutes","path":"lifecycle/tests/agent-run-summary-panel-6-test.md","age":"14h28m43s"}

These could be logged to a file
These could be sent on MQTT
These should shown the GUI

This is a condition, show them in the GUI with Parse Errors.

## Idea

When kaos-control emits structured log entries at WARN or ERROR level (e.g. stale artifact alerts, watcher failures, indexing issues), these should be surfaced in the GUI alongside existing parse errors so operators can see the health of the system without tailing logs. A dedicated "Conditions" or "System Notices" panel — or an extension of the existing parse-error display — would show each entry with its timestamp, level, message, and any associated path or metadata fields.

On the backend, warnings and errors should optionally be written to a rotating log file and/or published to an MQTT topic, giving operators flexibility to integrate with external monitoring pipelines. The structured JSON format already emitted by the logger (slog-compatible) makes this straightforward to route to multiple sinks without changing call sites.

The GUI component should poll or receive these conditions via the existing WebSocket broadcast channel (e.g. a new `system.condition` event type), keeping the display live. Conditions should be dismissible per-session and ideally badge-counted in the navigation so they are visible even when the panel is not in focus.
