---
title: 'Backend Plan: Release Drill-Down Filter to Ideas and Defects'
type: plan-backend
status: approved
lineage: roadmap-release-filter-ideas-defects
parent: lifecycle/requirements/roadmap-release-filter-ideas-defects-2.md
created: "2026-05-09T00:00:00+10:00"
release: KC-Release0
assignees:
    - role: analyst
      who: agent
---

# Backend Plan: Release Drill-Down Filter to Ideas and Defects

## Overview

Per FR-2, the API endpoint `GET /p/{project}/releases/{id}/artifacts` remains **unchanged** — filtering is client-side only. This backend plan therefore has **no code changes** to the endpoint itself, but documents the decision and the verification milestone to confirm the API contract is preserved.

The backend work here is limited to confirming the existing API behaviour and ensuring nothing regresses. All substantive changes live in [[roadmap-release-filter-ideas-defects]] frontend and test plans.

## Milestone 1: Verify API Contract Preservation

**Description:** Confirm that `handleListReleaseArtifacts` in `internal/http/releases.go` continues to return all artifact types without any type filter. No code changes are made.

**Files to review (read-only):**
- `internal/http/releases.go` — `handleListReleaseArtifacts` handler (lines 319-343)
- `internal/http/server.go` — route registration (line 206)

**Acceptance criteria:**
- [ ] `handleListReleaseArtifacts` does not filter by artifact type — it returns all artifacts assigned to the release
- [ ] The response shape `{ items: [...], total: N }` is unchanged
- [ ] `total` reflects the unfiltered count (all artifact types)
- [ ] No new query parameters or type-filter logic has been introduced

## Milestone 2: Ensure No Breaking Changes to Other Consumers

**Description:** Audit other call sites that consume `listReleaseArtifacts` to confirm they still receive the full unfiltered set and are unaffected by the frontend-only filter in `ReleaseDetailModal`.

**Files to review (read-only):**
- `web/src/views/project/RoadmapView.vue` — `openDelete()` uses `listReleaseArtifacts` to count artifacts before deletion; this must remain unfiltered
- `web/src/api/releases.ts` — the API client function; confirm no type filter parameter is added

**Acceptance criteria:**
- [ ] `RoadmapView.openDelete()` still receives the full artifact count (all types) for the delete confirmation modal
- [ ] The `listReleaseArtifacts` API client function signature is unchanged
- [ ] No new optional `type` query parameter is introduced on the endpoint
