---
title: 'Tests: Release Drill-Down Filter to Ideas and Defects'
type: test
status: approved
lineage: roadmap-release-filter-ideas-defects
parent: lifecycle/test-plans/roadmap-release-filter-ideas-defects-5-test.md
---

# Tests: Release Drill-Down Filter to Ideas and Defects

Automated test coverage for the client-side idea/defect filter in `ReleaseDetailModal` and the confirmation that the backend API remains unfiltered.

---

## Test Files

### `tests/web/ReleaseDetailModal.filter.test.ts` *(new)*

Vitest + `@vue/test-utils` (happy-dom) component tests for `ReleaseDetailModal.vue`.

Mocking: `releasesApi.getRelease` and `releasesApi.listReleaseArtifacts` are
mocked via `vi.mock`; `vue-router` is stubbed so `router.push` never throws.

**Milestone 1 — Filtered artifact list**

- **Only idea and defect cards from 8 types** — mounts modal with one artifact
  of each type (idea, defect, requirement, plan-backend, plan-frontend, plan-test,
  test, prototype); asserts exactly 2 `.artifact-card` elements are rendered and
  only the `idea` and `defect` type badges appear.
- **Empty state when no ideas or defects** — provides requirement, plan-backend,
  and test artifacts only; asserts zero cards and the "No artifacts assigned."
  message is visible.
- **All cards when every artifact is idea or defect** — provides 3 ideas and 2
  defects; asserts all 5 `.artifact-card` elements render.

**Milestone 2 — Filtered count in heading**

- **Heading reflects filtered count (3)** — 5 total artifacts (2 ideas + 1 defect
  + 2 plans); heading must contain "(3)" and must not contain "(5)".
- **Heading shows (0) for non-idea/defect only** — 3 non-idea/defect artifacts;
  heading must contain "(0)".

---

### `tests/web/GanttChart.unscheduled.test.ts` *(extended)*

Three new test cases appended to the existing `GanttChart` test file under the
heading "Milestone 3 — Regression: Gantt badge counts driven by releaseDetails".

- **Scheduled bar badges use `releaseDetails` counts** — `releaseDetails` with
  `idea_count: 3, defect_count: 2`; badge text must contain "3 ideas" and
  "2 defects".
- **Badge shows ideas-only count** — `releaseDetails` with `idea_count: 7,
  defect_count: 0`; badge text must contain "7 ideas".
- **No badge when counts are zero** — `releaseDetails` with both counts zero;
  `.bar-badge` must not exist on the bar.

These tests confirm the badge on Gantt bars is driven exclusively by
`releaseDetails.idea_count` and `releaseDetails.defect_count` — independent of
any filtering that `ReleaseDetailModal` applies to the artifact list.

---

### `tests/integration/releases_test.go` *(extended)*

One new Go integration test (`TestReleases_ListArtifactsReturnsAllTypes`) added
to the existing releases integration test file.

Seeds four artifacts — one each of type `idea`, `defect`, `requirement`, and
`plan-backend` — all assigned to the same release name. Calls
`GET /api/p/testproject/releases/{id}/artifacts` and asserts:

- Exactly 4 items are returned (no server-side filtering by type).
- The `total` field in the response equals 4.
- All four types (`idea`, `defect`, `requirement`, `plan-backend`) appear in
  the `type` field of the returned items.
