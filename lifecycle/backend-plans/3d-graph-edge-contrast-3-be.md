---
title: "Backend Plan: Improve Edge Line Contrast in 3D Graph"
type: plan-backend
status: done
lineage: 3d-graph-edge-contrast
parent: lifecycle/requirements/3d-graph-edge-contrast-2.md
---

# Backend Plan: Improve Edge Line Contrast in 3D Graph

## Summary

This requirement is scoped exclusively to the frontend (`ForceGraph3D.vue` and `graphConstants.ts`) per NG2. No backend Go code changes are required. The backend already serves edge data with `kind` fields that the frontend uses to determine styling — no new data or API surface is needed.

## Milestone 1: Confirm No Backend Impact

### Description

Verify that the existing REST and WebSocket APIs already provide all edge metadata (`kind`, `source`, `target`) needed by the frontend to apply the new styling. No schema changes, no new fields.

### Files to Change

None.

### Acceptance Criteria

- [ ] Confirmed: `/api/graph` response includes `kind` on every edge — already the case.
- [ ] Confirmed: no new edge kinds are introduced by this requirement.
- [ ] No Go code changes committed for this lineage.

## Cross-references

- [[3d-graph-edge-contrast]] — frontend plan carries implementation.
