---
title: 'Test Plan: Release Drill-Down Filter to Ideas and Defects'
type: plan-test
status: done
lineage: roadmap-release-filter-ideas-defects
parent: lifecycle/requirements/roadmap-release-filter-ideas-defects-2.md
created: "2026-05-09T00:00:00+10:00"
release: KC-Release0
assignees:
    - role: analyst
      who: agent
---

# Test Plan: Release Drill-Down Filter to Ideas and Defects

## Overview

This plan covers automated test coverage for the [[roadmap-release-filter-ideas-defects]] feature. Tests are written in Vitest with `@vue/test-utils` (happy-dom), following the patterns established in `tests/web/GanttChart.unscheduled.test.ts`. No backend test changes are needed since the API is unchanged (see [[roadmap-release-filter-ideas-defects]] backend plan).

## Milestone 1: Unit Test — Filtered Artifact List

**Description:** Create a new test file that mounts `ReleaseDetailModal` with a mock API returning a mix of artifact types (idea, defect, requirement, plan-backend, plan-frontend, plan-test, test, prototype) and asserts that only `idea` and `defect` cards are rendered.

**Files to create:**
- `tests/web/ReleaseDetailModal.filter.test.ts`

**Test cases:**
1. **Only ideas and defects are rendered** — provide 8 artifacts (one of each type listed above); assert exactly 2 artifact cards are rendered (the idea and defect).
2. **Empty state when no ideas or defects exist** — provide artifacts of types `requirement`, `plan-backend`, `test`; assert the "No artifacts assigned" message is shown.
3. **All artifacts are ideas/defects** — provide 3 ideas and 2 defects; assert all 5 are rendered.

**Acceptance criteria:**
- [ ] Test file mounts `ReleaseDetailModal` using `@vue/test-utils` with mocked `releasesApi.getRelease` and `releasesApi.listReleaseArtifacts`
- [ ] Test 1 passes: only idea and defect cards appear; other types are absent
- [ ] Test 2 passes: empty state shown when only non-idea/defect artifacts exist
- [ ] Test 3 passes: all artifacts shown when all are ideas or defects

## Milestone 2: Unit Test — Filtered Count in Heading

**Description:** Verify that the artifact count displayed in the section heading matches the filtered set (ideas + defects only), not the total API response.

**Files to change:**
- `tests/web/ReleaseDetailModal.filter.test.ts` (add test cases to the file from Milestone 1)

**Test cases:**
1. **Count reflects filtered set** — provide 5 total artifacts (2 ideas, 1 defect, 2 plans); assert the heading contains "(3)" (not "(5)").
2. **Count is zero when no ideas/defects** — provide 3 non-idea/defect artifacts; assert the heading contains "(0)".

**Acceptance criteria:**
- [ ] Test 1 passes: heading shows filtered count
- [ ] Test 2 passes: heading shows zero when only implementation artifacts are assigned

## Milestone 3: Regression Test — Gantt Badge Counts Unaffected

**Description:** Add a test (or extend existing `GanttChart.unscheduled.test.ts`) confirming that the Gantt bar summary badges still display the correct idea and defect counts from `releaseDetails`, independent of the modal filter.

**Files to change:**
- `tests/web/GanttChart.unscheduled.test.ts` (add test case) OR create `tests/web/GanttChart.badgeCounts.test.ts`

**Test cases:**
1. **Gantt bar badges use releaseDetails counts** — provide a release with `idea_count: 3` and `defect_count: 2` in `releaseDetails`; assert the badges display "3" and "2" regardless of what the modal would filter.

**Acceptance criteria:**
- [ ] Badge counts on Gantt bars are driven by `releaseDetails.idea_count` and `releaseDetails.defect_count`, not by any filtered artifact list
- [ ] Existing Gantt chart tests continue to pass

## Milestone 4: Regression Test — API Returns All Types

**Description:** Add an integration test confirming that `GET /p/{project}/releases/{id}/artifacts` returns all artifact types (no server-side filtering was introduced).

**Files to change:**
- `tests/integration/releases_test.go` (add test case)

**Test cases:**
1. **API returns all artifact types** — assign artifacts of types `idea`, `defect`, `requirement`, `plan-backend` to a release; call the endpoint; assert all 4 are returned in the response.

**Acceptance criteria:**
- [ ] The API response includes all artifact types assigned to the release
- [ ] The `total` field matches the full unfiltered count
- [ ] Existing integration tests continue to pass
