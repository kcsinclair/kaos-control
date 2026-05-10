---
title: 'Backend Plan: 3D Map Node Labels Toggle'
type: plan-backend
status: done
lineage: 3d-map-node-labels-toggle
parent: lifecycle/requirements/3d-map-node-labels-toggle-2.md
---

## Summary

This feature requires **no backend changes**. The 3D node label toggles are entirely client-side: toggle state lives in the Pinia graph store (session-scoped reactive refs), and label rendering uses the existing `textSprite()` helper in `ForceGraph3D.vue` with Three.js canvas textures. No new API endpoints, no schema changes, no server-side persistence.

This plan exists to satisfy the required three-plan gate (`plan-backend`, `plan-frontend`, `plan-test`) and to explicitly document the backend non-involvement.

## Milestone 1: Confirm No Backend Surface Area

### Description

Verify that the existing `/api/projects/:slug/graph` endpoint already returns `title`, `lineage`, `type`, and `slug` fields on every `GraphNode`. These are the only data fields the frontend needs for label rendering.

### Files to change

None.

### Acceptance criteria

- [ ] The `GraphNode` response from `GET /api/projects/:slug/graph` includes `title`, `lineage`, `type`, and `slug` for every node — confirmed by inspecting the existing handler at `internal/http/graph.go` and the `GraphNode` struct.
- [ ] No new fields, endpoints, or query parameters are required.

## Milestone 2: Confirm No Persistence Requirements

### Description

Per the requirement, toggle state is session-scoped (reactive ref in the Pinia store). There is no localStorage or server-side persistence. Confirm no config or database changes are needed.

### Files to change

None.

### Acceptance criteria

- [ ] No changes to `internal/config/`, `internal/index/`, or any Go package.
- [ ] No new REST or WebSocket messages.

## Cross-references

- Frontend implementation: [[3d-map-node-labels-toggle]] (frontend plan covers all UI and rendering work)
- Test coverage: [[3d-map-node-labels-toggle]] (test plan covers integration and visual verification)
