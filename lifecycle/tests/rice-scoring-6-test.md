---
title: "Test Suite — RICE Scoring Integration Tests"
type: test
status: in-qa
lineage: rice-scoring
parent: lifecycle/test-plans/rice-scoring-5-test.md
---

# Test Suite — RICE Scoring Integration Tests

Backend integration tests exercising the RICE feature over HTTP + index +
disk, per the `tests/` scope defined in
[[rice-scoring-5-test]]. Backend Go unit tests
(`internal/artifact/rice_test.go`, `internal/artifact/artifact_test.go`,
`internal/index/rice_test.go`) and frontend Vitest coverage
(`web/src/lib/__tests__/rice.spec.ts`,
`web/src/components/artifact/__tests__/RiceEditor.spec.ts`) are out of scope
for this artifact — they belong to the backend-developer and
frontend-developer milestones respectively and were not written here.

## Test files

- `tests/integration/rice_fidelity_test.go` — Milestone 2: frontmatter
  round-trip and backward compatibility via the API.
- `tests/integration/rice_sort_test.go` — Milestone 3: index back-fill and
  grouped RICE sort via the API.
- `tests/integration/rice_patch_test.go` — Milestone 4: the
  `PATCH .../rice` endpoint.
- `tests/integration/rice_parity_test.go` — Milestone 5: Go-side
  single-source-of-truth checks.
- `tests/fixtures/rice_fixtures.json` — shared component-tuple fixtures for
  the Go/TS parity check (requirement §22); consumed here by
  `TestRiceParity_GoFormulaMatchesFixtures`. A companion TS spec consuming
  the same file is expected from the frontend-developer role to close the
  cross-language half of Milestone 5.

## Scenarios covered

### Milestone 2 — Frontmatter round-trip & backward compatibility

| Test | Scenario |
|------|----------|
| `TestRiceFidelity_NoFieldsUnaffected` | A pre-existing idea/defect with no RICE fields indexes with `rice_score` absent and the file untouched on disk (§21). |
| `TestRiceFidelity_PresentZeroIsNotUnset` | A present `rice_reach: 0` (all four components valid) is returned in the API frontmatter and yields `rice_score: 0`, not N/A (§4, §8). |
| `TestRicePatchByteForByte` | After `PATCH .../rice`, every non-RICE line of the file is unchanged and in the same order; exactly the four RICE lines are added (§23). |
| `TestRicePatchClearRemovesLine` | Clearing one component via `null` removes its YAML line entirely (no `0`/`""` residue) while the other three remain, and the score reverts to N/A (§20). |

### Milestone 3 — Index column & grouped sort

| Test | Scenario |
|------|----------|
| `TestRiceScore_BackfilledOnInitialScan` | Artifacts present on disk before server start get `rice_score` back-filled (or null) by the initial full scan (§24). |
| `TestRiceSort_AscGroupsNullsLast` | `sort=rice:asc` returns scored rows in ascending order with all N/A rows grouped after them (§12). |
| `TestRiceSort_DescGroupsNullsLast` | `sort=rice:desc` returns scored rows in descending order with all N/A rows still grouped after them (§12). |
| `TestRiceSort_PerformanceOnLargeList` | Sorting 500 artifacts by `rice:desc` responds within a 2s budget and preserves correct ordering/grouping (§25). |

### Milestone 4 — `PATCH .../rice` endpoint

| Test | Scenario |
|------|----------|
| `TestRicePatch_HappyPath` | Valid PATCH returns 200 with the re-indexed row including `rice_score`; GET reflects the same value (§15). |
| `TestRicePatch_WebSocketEvent` | A successful PATCH broadcasts `artifact.indexed` with `action: "updated"` for the patched path (§15). |
| `TestRicePatch_InvalidInput` | Table-driven: non-numeric value → 400; negative reach/impact, confidence outside 0–100, effort ≤ 0 → 422 with a field-level error and an unchanged file (§19). |
| `TestRicePatch_ForeignLineageLock` | A lineage locked by another user → 423 with `error.code: "locked"`. |
| `TestRicePatch_Unauthenticated` | No session → 401. |
| `TestRicePatch_UnauthorizedRole` | A role outside `RolesPriorityEditors` (product-owner, analyst) → 403. |
| `TestRicePatch_Idempotent` | Two identical PATCH bodies in a row leave the file byte-identical after the second call. |

### Milestone 5 — Cross-tier parity

| Test | Scenario |
|------|----------|
| `TestRiceParity_GoFormulaMatchesFixtures` | Every case in `tests/fixtures/rice_fixtures.json` reproduces its `expected_score`/N/A verdict via `internal/artifact.RiceScore` — the Go half of §22. |
| `TestRiceParity_ListAndDetailEditorsAgree` | Identical components PATCHed through the shared endpoint from two different artifacts produce identical `rice_score` and byte-identical RICE frontmatter lines — the entry-point-independence half of §18. |

> **Scope note**: Milestone 5's TS-vs-fixtures parity check
> (`web/src/lib/__tests__/rice.spec.ts`) and Milestone 1 and 6's coverage
> (`internal/artifact/rice_test.go`, `web/src/components/artifact/__tests__/RiceEditor.spec.ts`,
> `web/src/views/project/__tests__/ArtifactListView.spec.ts`) fall outside this
> role's `tests/` write scope; Milestone 1 and the backend half of Milestone 2/3
> were already present in `internal/` when this suite was written.

## Known pre-existing issue found while running the full suite

Running the complete `tests/integration` suite (not just the new RICE tests)
surfaced 3 failing tests unrelated to this change and reproducible on the
pre-RICE-test commit:

- `TestReleases_ListArtifactsForRelease`
- `TestReleases_ListArtifactsReturnsAllTypes`
- `TestReleaseUnscheduled_ArtifactAssignment`

All three fail with `sql: expected 12 destination arguments in Scan, not 13`
from `GET /releases/{id}/artifacts`, i.e. a `SELECT`/`Scan` column-count
mismatch in a release-artifacts index query that was not updated for the new
`rice_score` column added by the RICE backend milestone. This is a backend
regression outside this role's `internal/` write scope — flagging for a
backend-developer/qa follow-up rather than fixing here.

## Run commands

```sh
# All RICE integration tests
go test ./tests/... -tags integration -run "TestRice"

# Individual files
go test ./tests/... -tags integration -run "TestRiceFidelity"
go test ./tests/... -tags integration -run "TestRiceSort|TestRiceScore_Backfilled"
go test ./tests/... -tags integration -run "TestRicePatch"
go test ./tests/... -tags integration -run "TestRiceParity"

# Run all integration tests
go test ./tests/... -tags integration
```
