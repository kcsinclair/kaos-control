---
title: "Backend Plan — Pipeline Log Streaming Endpoint Enhancements"
type: plan-backend
status: in-development
lineage: devops-pipeline-log-streaming
parent: lifecycle/requirements/devops-pipeline-log-streaming-2.md
---

# Backend Plan — Pipeline Log Streaming Endpoint Enhancements

This plan covers the backend changes required to support the split-pane pipeline log streaming view described in [[devops-pipeline-log-streaming]]. The backend already emits WebSocket events (`pipeline.step.output`, `pipeline.run.started`, `pipeline.step.started`, `pipeline.step.completed`, `pipeline.run.completed`) and serves a REST endpoint for fetching completed run logs. The changes here are minor — ensuring payload consistency, ANSI stripping, and the REST log endpoint returns data in a shape the frontend can render with step boundaries.

## Milestone 1 — Normalise WebSocket event payload keys

### Description

The devops store on the frontend destructures payload fields as `pipeline_slug`, `step_index`, etc. The backend `events.go` structs use Go-convention field names (`Pipeline`, `StepIndex`). Verify and ensure JSON serialisation tags produce snake_case keys that match the frontend's expectations. Add a `timestamp` field (ISO 8601) to `StepOutputPayload` so the frontend can display timestamps on step-boundary separators without relying on client-side clock.

### Files to change

- `internal/devops/events.go` — audit and fix JSON tags on all payload structs; add `Timestamp` field to `StepOutputPayload` and `StepStartedPayload`.
- `internal/devops/runner.go` — populate `Timestamp` when constructing payloads.

### Acceptance criteria

- [ ] All five event payload structs serialise with snake_case JSON keys matching: `run_id`, `pipeline_slug`, `project`, `step`, `step_index`, `text`, `stream`, `status`, `exit_code`, `duration_seconds`, `timestamp`.
- [ ] `StepOutputPayload` and `StepStartedPayload` include a `timestamp` field in RFC 3339 format.
- [ ] Existing WebSocket consumers (devops store, LogViewer) are unaffected — field renaming is additive or already correct.
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 2 — Ensure ANSI escape codes are stripped server-side

### Description

Per the resolved question in the requirement, ANSI escape codes must be stripped server-side. Verify the pipeline runner strips ANSI from `StepOutputPayload.Text` before broadcasting. If not already handled, add a stripping step using a lightweight regex or byte-scanner.

### Files to change

- `internal/devops/runner.go` — add ANSI strip call on captured stdout/stderr text before constructing `StepOutputPayload`.
- `internal/devops/ansi.go` (new, if needed) — small utility function `StripANSI(s string) string`.

### Acceptance criteria

- [ ] `pipeline.step.output` events never contain ANSI escape sequences (`\x1b[...m`, `\x1b[...K`, etc.).
- [ ] A unit test in `internal/devops/ansi_test.go` confirms stripping of common ANSI sequences (colour, cursor movement, erase).
- [ ] `go build ./...` and `go vet ./...` pass.

---

## Milestone 3 — Enrich the completed-run REST log endpoint

### Description

`GET /api/p/{project}/devops/runs/{run_id}` currently returns NDJSON lines. Ensure each line includes the event `type` field and all fields the frontend needs to reconstruct step boundaries and the terminal status line (step name, timestamp, status, duration). The [[devops-pipeline-log-streaming]] frontend plan relies on this endpoint to render completed runs in the same split-pane layout.

### Files to change

- `internal/http/devops.go` — update the `GET .../runs/{run_id}` handler to include `type` in each NDJSON line if not already present.
- `internal/devops/runner.go` — if the run log is built from stored events, ensure `StepStartedPayload`, `StepCompletedPayload`, and `RunCompletedPayload` are all persisted alongside output lines.

### Acceptance criteria

- [ ] `GET /api/p/{project}/devops/runs/{run_id}` returns NDJSON where every line has a `type` field (`pipeline.step.output`, `pipeline.step.started`, `pipeline.step.completed`, `pipeline.run.started`, `pipeline.run.completed`).
- [ ] Step boundary events include `step`, `step_index`, and `timestamp`.
- [ ] `pipeline.run.completed` line includes `status` and `duration_seconds`.
- [ ] Response Content-Type remains `application/x-ndjson`.
- [ ] `go build ./...` and `go vet ./...` pass.
