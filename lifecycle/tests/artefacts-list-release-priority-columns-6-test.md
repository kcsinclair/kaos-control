---
title: 'Tests: Artefacts List Release & Priority Columns'
type: test
status: approved
lineage: artefacts-list-release-priority-columns
parent: lifecycle/test-plans/artefacts-list-release-priority-columns-5-test.md
---

# Tests: Artefacts List Release & Priority Columns

## Summary

Integration and component-level tests for the Priority and Release columns in the artifact list view, covering API filter behaviour, column rendering, sort order, filter interaction, and accessibility.

---

## Test files

| File | Milestones covered |
|---|---|
| `tests/integration/artifact_release_priority_filter_test.go` | M1 TC3–TC6 |
| `tests/web/ArtifactListView.priority-column.test.ts` | M2 |
| `tests/web/ArtifactListView.release-column.test.ts` | M3 |
| `tests/web/ArtifactListView.prioritySort.test.ts` | M4 |
| `tests/web/ArtifactListView.releaseSort.test.ts` | M5 |
| `tests/web/ArtifactListView.releaseFilter.test.ts` | M6 |
| `tests/web/ArtifactListView.responsive.test.ts` | M7 |

> **M1 TC1 and TC2** (release exact-match and `__unassigned__` filter) are already covered by
> `tests/integration/releases_filter_test.go` (`TestReleaseFilter_ByReleaseName` and
> `TestReleaseFilter_Unassigned`).

---

## Scenarios covered

### Milestone 1 — Backend API filter tests (`artifact_release_priority_filter_test.go`)

- `TestReleaseFilter_Composition` — `?release=v-cmp-1&status=draft` returns exactly the artifact that satisfies both conditions; artifacts matching only one condition are excluded.
- `TestReleaseFilter_NoMatch` — `?release=nonexistent-release-xyz` returns `items: []` and `total: 0`.
- `TestPriority_InListResponse` — Artifacts seeded with `priority: high`, `priority: critical`, `priority: low`, and no priority. The list response `frontmatter.priority` field reflects the correct value (empty string for unset).
- `TestRelease_InListResponse` — Artifacts seeded with `release: v-rrl-1`, `release: v-rrl-2`, and no release. The list response `frontmatter.release` field reflects the correct value.

All integration tests use `//go:build integration`, seed their own isolated test environments via `newTestEnv`, and are idempotent.

### Milestone 2 — Priority column display (`ArtifactListView.priority-column.test.ts`)

- TC1: `<span class="priority-pill priority-high">high</span>` is rendered for an artifact with `priority: high`.
- TC2: `<span class="muted">—</span>` is rendered in `.cell-priority` when no priority is set; no `.priority-pill` is present.
- TC3: Each of `critical`, `high`, `normal`, `low` renders its own `priority-{value}` CSS class.
- TC4: The Priority column header index is strictly greater than Status and strictly less than Release in the `<thead>` `<th>` sequence.

### Milestone 3 — Release column display (`ArtifactListView.release-column.test.ts`)

- TC1: `.cell-release` cell text equals the release value (e.g. "v1.0") when set.
- TC2: `.cell-release` cell text equals "—" when the release field is absent.
- TC3: The Release column header index is strictly greater than the Priority column header.

### Milestone 4 — Priority sort (`ArtifactListView.prioritySort.test.ts`)

- TC1: First click (ascending) — row order: `''` (0), `low` (1), `normal` (2), `high` (3), `critical` (4).
- TC2: Second click (descending) — row order: `critical`, `high`, `normal`, `low`, `''`.
- TC3: Third click resets sort (`aria-sort` absent on all headers); original insertion order is restored.
- TC4: Sort is non-alphabetical — index positions of `low`, `normal`, `high`, `critical` are in severity order, not A–Z order.

### Milestone 5 — Release sort (`ArtifactListView.releaseSort.test.ts`)

- TC1: Ascending — `''` (empty string maps to `—` in cell), `alpha`, `v1.0`, `v2.0`.
- TC2: Descending — `v2.0`, `v1.0`, `alpha`, `''`.
- TC3: Artifact with no release always sorts first in ascending and last in descending (empty string `''` via `?? ''` mapping).
- TC4: `alpha` and `Alpha` are treated as equivalent (localeCompare `sensitivity: 'base'`) and always sort adjacently, both before `beta`.

### Milestone 6 — Release filter interaction (`ArtifactListView.releaseFilter.test.ts`)

- TC1: Dropdown options include "All releases", each name in `releasesStore.releases`, and "Unassigned".
- TC2: Selecting a named release calls `listArtifacts` with `filter.release === 'v1.0'`.
- TC3: Selecting "Unassigned" calls `listArtifacts` with `filter.release === '__unassigned__'`.
- TC4: Selecting "All releases" calls `listArtifacts` with `filter.release` absent/empty.
- TC5: Combining status=draft and release=v1.0 filters passes both in the same `listArtifacts` call.
- TC6: Text search combined with a release filter passes both `filter.q` and `filter.release`.
- TC7: Changing the release dropdown clears the active sort indicator (`aria-sort` removed).
- TC8: Clicking the Reset button returns `#release-filter` value to `""` (All releases).
- TC9: When the store is empty (`items: []`), the `.state-msg` "No artifacts found" message is visible.

### Milestone 7 — Responsive layout / accessibility (`ArtifactListView.responsive.test.ts`)

- Structural: All 8 column headers (Path, Stage, Status, Priority, Release, Type, Created, Modified) are present in `<thead>`.
- Structural: Every `<tbody>` row has the same number of `<td>` elements as there are `<th>` columns.
- Structural: `#release-filter` is rendered inside `.filter-bar`.
- TC4 (accessibility): `<label for="release-filter">` exists and associates with `<select id="release-filter">`.
- TC4 (accessibility): `#release-filter` does not have `tabindex="-1"`.
- TC4 (accessibility): Keyboard-driven `setValue` + `change` event on the dropdown is accepted.
- TC4 (accessibility): Priority and Release `<th>` elements have `tabindex="0"` for keyboard activation.

> TC1–TC3 (viewport rendering at 1280/1024/768 px) are deferred to browser-based E2E tests; they cannot be meaningfully exercised in vitest's happy-dom environment.

---

## How to run

```sh
# Integration tests
go test -tags integration ./tests/integration/... -run 'TestReleaseFilter_Composition|TestReleaseFilter_NoMatch|TestPriority_InListResponse|TestRelease_InListResponse'

# Web component tests
cd tests/web && pnpm vitest run ArtifactListView.priority-column ArtifactListView.release-column ArtifactListView.prioritySort ArtifactListView.releaseSort ArtifactListView.releaseFilter ArtifactListView.responsive
```
