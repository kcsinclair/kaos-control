---
title: "Backend Plan: Auto-Refresh Editor Content on External Disk Change"
type: plan-backend
status: in-development
lineage: editor-live-refresh-on-disk-change
parent: lifecycle/requirements/editor-live-refresh-on-disk-change-2.md
created: "2026-04-29"
---

# Backend Plan: Auto-Refresh Editor Content on External Disk Change

## Overview

The backend already emits `file.changed` WebSocket events when the fsnotify watcher detects disk modifications (see `internal/watcher/watcher.go`). The existing REST endpoint `GET /api/projects/:project/artifacts/*path` returns the full artifact detail including `body`, `body_html`, and `file_sha`. **No new backend endpoints or event types are required.** This plan documents the verification that the current backend surface is sufficient and identifies one minor hardening change.

## Milestone 1: Verify existing `file.changed` event payload is sufficient

### Description

Confirm that the `file.changed` WebSocket event includes the relative artifact path in its payload, which the frontend needs to match against the currently-open artifact. The current broadcast in `internal/watcher/watcher.go` already sends `{"path": "<relPath>"}`.

### Files to review (no changes expected)

- `internal/watcher/watcher.go` — confirm `file.changed` broadcast payload shape.
- `internal/hub/hub.go` — confirm broadcast does not filter or transform the event.
- `internal/http/ws.go` — confirm the WebSocket handler forwards hub events unmodified to connected clients.

### Acceptance criteria

- [ ] The `file.changed` event payload contains a `path` field with the lifecycle-relative path (e.g. `lifecycle/requirements/foo-2.md`).
- [ ] Events are broadcast to all connected WebSocket clients for the project.
- [ ] No backend code changes are needed for this milestone — this is a verification step.

## Milestone 2: Ensure rapid successive writes produce distinct events

### Description

When an agent or external tool writes the same file multiple times in quick succession, the backend's 150 ms fsnotify debounce (`internal/watcher/watcher.go`) coalesces filesystem events. Verify that at least one `file.changed` event is emitted per coalesced burst so the frontend's own debounce (300-500 ms, handled in [[editor-live-refresh-on-disk-change]]-4-fe) can trigger a re-fetch.

### Files to review (no changes expected)

- `internal/watcher/watcher.go` — confirm the debounce timer resets on each fsnotify event and fires once after 150 ms of quiet, emitting exactly one `file.changed`.

### Acceptance criteria

- [ ] After N rapid writes to the same file within 150 ms, exactly one `file.changed` event is broadcast.
- [ ] After a quiet period > 150 ms following the last write, a subsequent write produces a new event.
- [ ] Existing integration test `TestExternalEditUpdateExisting` in `tests/integration/external_edit_test.go` passes without modification.

## Milestone 3: Harden `GET /artifacts/*` for conditional-fetch friendliness

### Description

The frontend will re-fetch the artifact on every `file.changed` event (when the editor is in read mode). To keep this lightweight, verify the existing endpoint returns the `file_sha` field that the frontend can use to detect no-op refreshes (content unchanged). No new caching headers are required for v1, but the response must include `file_sha` consistently.

### Files to review

- `internal/http/artifacts.go` — confirm `GET` handler always populates `file_sha` in the response JSON.
- `internal/artifact/parser.go` — confirm SHA computation is deterministic for identical content.

### Acceptance criteria

- [ ] `GET /api/projects/:project/artifacts/*path` response includes `file_sha` as a non-empty string.
- [ ] Two successive GETs of an unchanged file return the same `file_sha`.
- [ ] After modifying the file on disk and re-fetching, `file_sha` differs from the previous value.
- [ ] Existing integration test `TestExternalEditPickedUp` continues to pass.

## Dependencies

- This plan has no blocking dependencies. The backend surface is already sufficient.
- The [[editor-live-refresh-on-disk-change]]-4-fe frontend plan depends on the `file.changed` event shape and artifact API response verified here.
- The [[editor-live-refresh-on-disk-change]]-5-test test plan will add integration tests that exercise the end-to-end flow.
