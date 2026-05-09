---
title: 'Release Drill-Down: Filter to Ideas and Defects Only'
type: requirement
status: blocked
lineage: roadmap-release-filter-ideas-defects
created: "2026-05-09T00:00:00+10:00"
priority: high
parent: lifecycle/ideas/roadmap-release-filter-ideas-defects.md
labels:
    - frontend
    - roadmaps
    - enhancement
    - usability
release: KC-Release0
assignees:
    - role: analyst
      who: agent
    - role: product-owner
      who: agent
---

# Release Drill-Down: Filter to Ideas and Defects Only

## Problem

When a user clicks a release bar on the Roadmap Gantt chart, the `ReleaseDetailModal` displays every artifact assigned to that release — including requirements, backend plans, frontend plans, test plans, tests, and other implementation-level types. These internal artifacts are noise at the roadmap level of abstraction; stakeholders care about **what** user-facing work and known issues are targeted for a release, not **how** those items are being built.

Currently `listReleaseArtifacts` (`GET /p/{project}/releases/{id}/artifacts`) returns all artifact types unfiltered, and `ReleaseDetailModal` renders them all with a generic grey badge fallback for non-idea/defect types.

## Goals / Non-goals

### Goals

- G1: The release drill-down panel shows **only** `idea` and `defect` artifacts, providing a clean roadmap-level summary.
- G2: The total count displayed in the panel header reflects the filtered set (ideas + defects only).
- G3: The existing summary badges on Gantt bars (idea count, defect count) remain accurate and unchanged.

### Non-goals

- NG1: This requirement does not add user-configurable type filters or toggles — the filter is hardcoded to ideas and defects.
- NG2: This requirement does not change the Roadmap Graph view (`RoadmapGraphView`), only the Gantt release drill-down modal.
- NG3: This requirement does not alter the backlog panel filtering logic.

## Detailed Requirements

### Functional

- **FR-1**: When `ReleaseDetailModal` loads artifacts for a release, only artifacts with `type` equal to `idea` or `defect` shall be displayed.
- **FR-2**: The filtering shall be applied **client-side** in `ReleaseDetailModal.vue` after the API response is received. The API endpoint (`listReleaseArtifacts`) shall remain unchanged so other consumers are unaffected.
- **FR-3**: The artifact count shown in the modal header/footer (if any) shall reflect the filtered count (ideas + defects only).
- **FR-4**: The badge styling already in place for `idea` (purple) and `defect` (red) types shall continue to apply. The grey `badge--default` fallback becomes unreachable in this modal but need not be removed.

### Non-functional

- **NFR-1**: The filter logic shall be a simple computed property or inline filter — no new components, utilities, or API changes required.
- **NFR-2**: No performance impact; the artifact list per release is small (typically <50 items).

## Acceptance Criteria

- [ ] Clicking a release bar on the Roadmap Gantt chart opens the detail modal showing **only** `idea` and `defect` artifacts
- [ ] No `requirement`, `plan-backend`, `plan-frontend`, `plan-test`, `test`, `prototype`, or other non-idea/defect types appear in the modal
- [ ] The item count in the modal header matches the number of displayed (filtered) artifacts
- [ ] Gantt bar summary badges (idea count, defect count) are unaffected
- [ ] The backlog panel in [[roadmap-backlog-panel-and-unscheduled-column]] continues to work as before
- [ ] The Roadmap Graph view is unaffected
- [ ] The `GET /p/{project}/releases/{id}/artifacts` API continues to return all artifact types (no breaking change)

## Open Questions

- **OQ-1**: Should the modal indicate that other artifact types exist but are hidden (e.g., "3 other artifacts not shown"), or silently omit them? _Recommendation: silently omit — the roadmap view is intentionally high-level._
- **OQ-2**: In the future, should this filter be configurable per-project in `lifecycle/config.yaml`? _Parked for a future enhancement if requested._
