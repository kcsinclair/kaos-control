---
title: 'Test Plan: Directed Release Chain Graph'
type: plan-test
status: done
lineage: roadmap-graph-release-connections
created: "2026-05-07T00:00:00+10:00"
priority: high
parent: lifecycle/requirements/roadmap-graph-release-connections-2.md
release: May2026
---

# Test Plan: Directed Release Chain Graph

Integration and unit tests validating the directed release chain graph — backend API response structure, node ordering, edge directionality, and frontend rendering behaviour.

## Milestone 1: Backend Unit Tests — Chain Construction Logic

**Description:** Test the graph endpoint's chain-building logic in isolation: correct ordering, edge generation, and synthetic node inclusion.

**Files to change:**
- `tests/releases_graph_test.go` (new or extend existing) — test `GET /api/p/:project/releases/graph` response.

**Acceptance criteria:**
- Test: empty state — response contains only the Backlog node, no edges.
- Test: single scheduled release — Backlog → Release edge exists.
- Test: multiple scheduled releases — edges form chronological chain (Backlog → R1 → R2 → R3) with correct ordering by `start_date`.
- Test: tie-breaking — two releases with same `start_date` are ordered alphabetically.
- Test: single unscheduled release — appended after last scheduled, connected by edge.
- Test: multiple unscheduled releases — alphabetically ordered, connected by edges after last scheduled.
- Test: no scheduled releases — Backlog connects directly to first unscheduled (alphabetically).
- Test: mixed scheduled and unscheduled — full chain: Backlog → scheduled (chronological) → unscheduled (alphabetical).

## Milestone 2: Backend Unit Tests — Edge Metadata

**Description:** Validate that duration labels on timeline edges are correct and appropriately formatted.

**Files to change:**
- `tests/releases_graph_test.go` — additional test cases for edge label content.

**Acceptance criteria:**
- Test: edge between releases 7 days apart has label "1 week" (or "7 days").
- Test: edge between releases 30 days apart has label "1 month" (or "4 weeks").
- Test: edge from Backlog has no duration label.
- Test: edge between unscheduled releases has no duration label.
- Test: duration formatting uses appropriate units (days < 8, weeks < 5, months < 13, years).

## Milestone 3: Backend Unit Tests — Artifact Assignment

**Description:** Validate that artifact nodes and assignment edges are correctly attached to release nodes and the Backlog node.

**Files to change:**
- `tests/releases_graph_test.go` — test cases for artifact inclusion.

**Acceptance criteria:**
- Test: artifact with `release: "v1.0"` has an `assigned` edge from the v1.0 release node.
- Test: artifact with no release field has an `assigned` edge from the Backlog node.
- Test: artifact nodes include `id`, `title`, `type`, `status` fields.
- Test: existing `depends_on`/`blocks` edges between included artifacts are preserved.

## Milestone 4: Frontend Integration Tests — Graph Rendering

**Description:** Test that the frontend correctly renders the directed chain with proper node types, shapes, and edge arrows.

**Files to change:**
- `tests/roadmap_graph_render_test.go` or `web/src/components/graph/__tests__/` (Vitest component tests) — depending on test infrastructure.

**Acceptance criteria:**
- Test: Backlog node is rendered with distinct styling (different colour/shape from release nodes).
- Test: release nodes render in light blue.
- Test: all timeline edges have directional arrows (arrowhead at target).
- Test: edge labels display duration text for scheduled-to-scheduled edges.
- Test: no edge labels on Backlog or unscheduled edges.

## Milestone 5: Frontend Integration Tests — Click Interactions

**Description:** Validate that clicking release nodes opens the ReleaseDetailModal and clicking artifact nodes opens the artifact modal.

**Files to change:**
- `tests/roadmap_graph_interaction_test.go` or component test files.

**Acceptance criteria:**
- Test: clicking a release node triggers ReleaseDetailModal with correct release data.
- Test: clicking an artifact (idea/defect) node triggers the standard artifact modal.
- Test: clicking the Backlog node does not open a modal (no error, no navigation).
- Test: modal dismissal returns focus to the graph.

## Milestone 6: Performance Tests

**Description:** Validate rendering performance with larger datasets.

**Files to change:**
- `tests/releases_graph_perf_test.go` — benchmark test with 50 releases.

**Acceptance criteria:**
- Test: API response for 50 releases returns in < 100ms.
- Test: frontend renders 50 release nodes without perceptible delay (< 200ms from data receipt to paint, measured via performance marks or benchmark harness).
- Test: no label overlap with 20 release nodes at default viewport (visual inspection or layout metric assertion).

## Milestone 7: Edge Cases and Regression

**Description:** Cover boundary conditions and ensure no regressions in existing graph functionality.

**Files to change:**
- `tests/releases_graph_test.go` — additional edge case tests.

**Acceptance criteria:**
- Test: deleting the only scheduled release results in Backlog → unscheduled chain.
- Test: adding a new release with a `start_date` between existing releases inserts it correctly in the chain (after re-fetch).
- Test: renaming a release updates its node label in the graph.
- Test: existing main artifact graph (`/graph` endpoint) is unaffected by roadmap graph changes.
- Test: WebSocket events (`release.created`, `release.updated`, `release.deleted`) trigger graph re-render with correct chain.

## Cross-references

- [[roadmap-graph-release-connections]] backend plan defines the API response structure being tested.
- [[roadmap-graph-release-connections]] frontend plan defines the rendering behaviour being validated.
