---
title: "2D Graph Layout Selector — Backend Plan"
type: plan-backend
status: draft
lineage: 2d-graph-layout-selector
parent: lifecycle/requirements/2d-graph-layout-selector-2.md
---

# 2D Graph Layout Selector — Backend Plan

## Summary

This feature is entirely client-side. The backend graph endpoint (`GET /api/p/:project/graph`) already returns the full node/edge dataset; layout algorithm selection, animation, and session persistence all happen in the browser. No backend changes are required.

## Milestones

### Milestone 1: Confirm No Backend Changes Required

**Description:** Verify that the existing graph API response provides all data needed for every supported layout algorithm (fcose, breadthfirst, concentric, circle, dagre). Specifically confirm that node and edge payloads include no layout-specific fields that would need to vary by algorithm.

**Files to review (no changes):**
- `internal/http/graph.go` — graph endpoint handler
- `internal/index/graph.go` — SQLite graph query (nodes + edges)

**Acceptance Criteria:**
- [ ] Confirmed: the graph API returns topology only (nodes with metadata, edges with source/target/type) — no layout coordinates or algorithm hints.
- [ ] Confirmed: no new query parameters or response fields are needed for the frontend to switch layouts client-side.
- [ ] No code changes committed in this plan.

---

## Cross-links

- [[2d-graph-layout-selector]] frontend plan handles all UI and Cytoscape integration.
- [[2d-graph-layout-selector]] test plan covers E2E verification of layout switching.
