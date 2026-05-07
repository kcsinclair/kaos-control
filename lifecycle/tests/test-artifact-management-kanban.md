---
title: Kanban Board Test Visibility — Integration Tests
type: test
status: approved
lineage: test-artifact-management
parent: lifecycle/test-plans/test-artifact-management-5-test.md
---

# Kanban Board Test Visibility — Integration Tests

Integration tests covering **Milestone 4** of the test-artifact-management feature: verifying that the artifact listing API does not implicitly exclude test artifacts and that the kanban config endpoint is unaffected.

## Test file

`tests/integration/test_artifact_kanban_test.go`

## Scenarios covered

| Test function | Scenario |
|---|---|
| `TestKanbanVisibility_UnfilteredIncludesTests` | `GET /artifacts` (no type filter) includes `type=test` artifacts. A mix of 2 test + 1 ticket + 1 idea is seeded; all 4 must appear in the response. |
| `TestKanbanVisibility_ConfigStructureUnchanged` | `GET /config/kanban` returns the expected structure (columns, uncategorised, card_fields) unchanged. Regression guard for the kanban config endpoint. |
| `TestKanbanVisibility_TypeFilterExcludesTests` | `GET /artifacts?type=ticket` does NOT include test artifacts. Only the seeded ticket must appear. |
| `TestKanbanVisibility_AllTestArtifactsInUnfilteredCount` | The unfiltered `total` equals the sum of per-type totals (test + ticket + idea), confirming no implicit exclusion at the query layer. |

## Fixtures

- `lifecycle/tests/kb-test-1.md` (test/approved)
- `lifecycle/tests/kb-test-2.md` (test/draft)
- `lifecycle/requirements/kb-ticket-2.md` (ticket/approved)
- `lifecycle/ideas/kb-idea.md` (idea/draft)
