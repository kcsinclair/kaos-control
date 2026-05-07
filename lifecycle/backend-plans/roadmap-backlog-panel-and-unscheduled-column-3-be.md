---
title: Backend Plan — Roadmap Backlog Panel and Unscheduled Column
type: plan-backend
status: draft
lineage: roadmap-backlog-panel-and-unscheduled-column
priority: high
parent: lifecycle/requirements/roadmap-backlog-panel-and-unscheduled-column-2.md
release: May2026
---

# Backend Plan — Roadmap Backlog Panel and Unscheduled Column

This feature is **frontend-dominant**. The requirement explicitly states (N4) that no backend API changes are needed for artifact querying — the frontend already has access to artifact data via existing endpoints. The Backlog panel sources its data from `GET /p/:project/artifacts` with client-side filtering (FR5.1), and the Unscheduled column is a purely visual rearrangement of data already served by `GET /p/:project/releases`.

Cross-references: [[roadmap-backlog-panel-and-unscheduled-column]] (frontend plan for all Gantt/Backlog UI changes), [[roadmap-backlog-panel-and-unscheduled-column]] (test plan for integration tests).

---

## Milestone 1 — Verify existing API coverage for Backlog data needs

### Description

Confirm that the existing `GET /p/:project/artifacts` endpoint returns all fields the Backlog panel requires (title, type, status, lineage, release) and that the `release` query parameter supports filtering for artifacts with no release assignment. No code changes are expected — this milestone is a verification gate.

### Files to review (no changes expected)

- `internal/http/artifacts.go` — `handleListArtifacts`: confirm `ArtifactRow` JSON response includes `frontmatter.release` and that the endpoint can return all artifacts without pagination limits when needed (or that the frontend can paginate).
- `internal/index/index.go` — `ArtifactRow` struct: confirm `FM.Release` is serialised in the JSON `frontmatter` field.

### Acceptance criteria

- [ ] `GET /p/:project/artifacts` returns `frontmatter.release` for every artifact row, including when the field is empty/null.
- [ ] The endpoint supports fetching up to 500 artifacts via `limit` parameter (NFR2 requirement).
- [ ] No backend code changes are required — if this milestone reveals a gap, escalate to the frontend and test plans.

---

## Milestone 2 — Ensure WebSocket artifact.indexed events carry release field context

### Description

Verify that when an artifact's `release` field is changed (via `PUT /artifacts/*path`), the `artifact.indexed` WebSocket event is broadcast so the Backlog panel can react. The event payload only needs `path` and `action` — the frontend will re-fetch artifact data. No code changes are expected if the existing write path already broadcasts after frontmatter updates.

### Files to review (no changes expected)

- `internal/http/write.go` — `handleUpdateArtifact`: confirm it broadcasts `artifact.indexed` with action `updated` after any frontmatter change including `release`.
- `internal/hub/hub.go` — `Broadcast`: confirm event delivery is non-blocking and handles concurrent clients.

### Acceptance criteria

- [ ] Updating an artifact's `release` frontmatter field via `PUT /artifacts/*path` triggers an `artifact.indexed` WebSocket event with action `updated`.
- [ ] Clearing an artifact's `release` field (setting it to empty/null) also triggers the event.
- [ ] No new event types or payload fields are needed — the frontend uses the event as a signal to re-fetch.

---

## Summary

No backend code changes are required for this feature. Both milestones are verification-only gates that confirm existing API and WebSocket behaviour meets the frontend plan's assumptions. If any gap is discovered during verification, the finding should be documented and the frontend plan updated accordingly.
