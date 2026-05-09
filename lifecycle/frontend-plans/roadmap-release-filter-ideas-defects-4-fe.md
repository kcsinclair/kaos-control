---
title: 'Frontend Plan: Release Drill-Down Filter to Ideas and Defects'
type: plan-frontend
status: approved
lineage: roadmap-release-filter-ideas-defects
parent: lifecycle/requirements/roadmap-release-filter-ideas-defects-2.md
created: "2026-05-09T00:00:00+10:00"
release: KC-Release0
assignees:
    - role: analyst
      who: agent
---

# Frontend Plan: Release Drill-Down Filter to Ideas and Defects

## Overview

This plan implements FR-1 through FR-4 from the requirement: filter the `ReleaseDetailModal` artifact list to show only `idea` and `defect` types, and update the displayed count to match the filtered set. The API is not changed (FR-2); filtering is purely client-side. See [[roadmap-release-filter-ideas-defects]] backend plan for API contract verification and test plan for coverage.

## Milestone 1: Add Client-Side Type Filter

**Description:** Introduce a computed property in `ReleaseDetailModal.vue` that filters the raw `artifacts` ref to only include items where `artifact.type === 'idea' || artifact.type === 'defect'`. Use this filtered list in the template instead of the raw `artifacts` array.

**Files to change:**
- `web/src/components/releases/ReleaseDetailModal.vue`

**Changes:**
1. Import `computed` (already imported on line 2).
2. Add a computed property after line 59:
   ```ts
   const filteredArtifacts = computed(() =>
     artifacts.value.filter(a => a.type === 'idea' || a.type === 'defect')
   )
   ```
3. In the template, replace `artifacts` references with `filteredArtifacts`:
   - Line 115: `v-if="filteredArtifacts.length === 0"` (empty state)
   - Line 117: `v-for="artifact in filteredArtifacts"` (list rendering)

**Acceptance criteria:**
- [ ] The modal renders only `idea` and `defect` artifact cards
- [ ] No `requirement`, `plan-backend`, `plan-frontend`, `plan-test`, `test`, `prototype`, or other types appear
- [ ] The raw `artifacts` ref still holds the full API response (unfiltered) for any future use
- [ ] The grey `badge--default` CSS class is unreachable in this modal but remains in the stylesheet (per FR-4)

## Milestone 2: Update Displayed Artifact Count

**Description:** Update the section heading in the modal to show the filtered count so stakeholders see an accurate number.

**Files to change:**
- `web/src/components/releases/ReleaseDetailModal.vue`

**Changes:**
1. Update the heading on line 114 from:
   ```html
   <h4 class="artifacts-heading">Assigned Artifacts</h4>
   ```
   to:
   ```html
   <h4 class="artifacts-heading">Assigned Artifacts ({{ filteredArtifacts.length }})</h4>
   ```

**Acceptance criteria:**
- [ ] The heading displays the count of filtered artifacts (ideas + defects only)
- [ ] The count updates reactively if the underlying data changes
- [ ] The count reads "0" and the empty-state message shows when no ideas or defects are assigned (even if other artifact types exist)

## Milestone 3: Verify No Regressions in Adjacent Components

**Description:** Manually verify that the Gantt bar summary badges and backlog panel are unaffected by this change.

**Files to review (read-only):**
- `web/src/components/releases/GanttChart.vue` — summary badges use `releaseDetails` (idea_count, defect_count from the API), not the modal's filtered list
- `web/src/components/releases/BacklogPanel.vue` — independent filtering logic; not affected
- `web/src/components/releases/RoadmapGraphView.vue` — graph view; not affected by this change

**Acceptance criteria:**
- [ ] Gantt bar idea/defect count badges remain accurate and unchanged
- [ ] Backlog panel filtering works as before (per [[roadmap-backlog-panel-and-unscheduled-column]])
- [ ] Roadmap Graph view is unaffected
- [ ] The `openDelete()` flow in `RoadmapView.vue` still uses the unfiltered artifact count for its confirmation modal
