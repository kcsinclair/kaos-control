---
title: "Backend Plan — Move Running Agents Indicator to Menu Bar"
type: plan-backend
status: approved
lineage: agents-indicator-in-menu-bar
parent: lifecycle/requirements/agents-indicator-in-menu-bar-2.md
---

# Backend Plan — Move Running Agents Indicator to Menu Bar

## Summary

No backend changes are required for this feature. The existing backend infrastructure already provides everything the frontend needs:

- **WebSocket events** (`agent.started`, `agent.progress`, `agent.finished`, `agent.failed`) are broadcast per-project via the hub and consumed by the frontend's `useAgentsStore`.
- **REST endpoint** `GET /api/p/:project/agents/runs` supports listing runs with optional `?status=running` filter, used for initial hydration on page load.
- **Agent run lifecycle** (start, progress, finish/fail) is managed by `internal/agent/supervisor.go` and events are already emitted on the project WebSocket channel.

The frontend plan ([[agents-indicator-in-menu-bar]]-4-fe) handles the entire UI relocation from the floating pill to the header. No new API endpoints, WebSocket event types, or data model changes are needed.

## Milestone 1: Confirm no backend work needed

### Description
Verify that the existing WebSocket events and REST API provide all data required by the frontend indicator (active run count, project scoping).

### Files to review (no changes)
- `internal/http/agents.go` — run list handler, already supports `?status=running`
- `internal/agent/supervisor.go` — emits `agent.started` / `agent.finished` / `agent.failed` events
- `internal/hub/hub.go` — per-project WebSocket broadcast

### Acceptance criteria
- [ ] The `agent.started` and `agent.finished`/`agent.failed` events contain `run_id` — confirmed in existing code.
- [ ] The REST list endpoint supports filtering by status — confirmed via `?status` query parameter.
- [ ] No new endpoints or event types are required for the header indicator to function.
