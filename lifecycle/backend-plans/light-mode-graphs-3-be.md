---
title: Backend Plan — Light Mode Colour Scheme for Graphs
type: plan-backend
status: approved
lineage: light-mode-graphs
priority: medium
parent: lifecycle/requirements/light-mode-graphs-2.md
release: May2026
---

# Backend Plan — Light Mode Colour Scheme for Graphs

This feature is entirely frontend. The Go backend does not serve, store, or transform any colour/theme information for the graph renderers — colours are resolved client-side from `graphConstants.ts` and the Pinia theme store. No backend API changes, database schema changes, or configuration changes are required.

## Milestone 1 — Confirm No Backend Work Required

### Description

Verify that the graph colour pipeline is fully client-side and that no REST or WebSocket endpoints reference theme or colour data.

### Files to change

None.

### Acceptance criteria

- [ ] Grep of `internal/http/` confirms no handler returns colour or theme data.
- [ ] No changes to Go source are needed for this feature.
- [ ] Related: [[light-mode-graphs]]
