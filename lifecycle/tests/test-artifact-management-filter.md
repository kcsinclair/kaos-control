---
title: Test Artifact Filter API — Integration Tests
type: test
status: draft
lineage: test-artifact-management
parent: lifecycle/test-plans/test-artifact-management-5-test.md
---

# Test Artifact Filter API — Integration Tests

Integration tests covering **Milestone 1** of the test-artifact-management feature: verifying that the artifact listing API correctly filters by `type=test` and meets the 200 ms performance requirement.

## Test file

`tests/integration/test_artifact_filter_test.go`

## Scenarios covered

| Test function | Scenario |
|---|---|
| `TestTestArtifactFilter_TypeOnly` | `GET /artifacts?type=test` returns only test artifacts; ticket and idea types are excluded. Fixtures include 3 test + 1 ticket + 1 idea. |
| `TestTestArtifactFilter_TypeAndStatus` | `GET /artifacts?type=test&status=approved` returns only approved test artifacts; draft tests and non-test types are excluded. |
| `TestTestArtifactFilter_TotalAccuracy` | The `total` field in the filtered response matches the length of the `items` array (badge count accuracy). |
| `TestTestArtifactFilter_EmptyProject` | Filtering by `type=test` on a project with only non-test artifacts returns `{items: [], total: 0}`. |
| `TestTestArtifactFilter_Performance` | `GET /artifacts?type=test` responds within 200 ms for a project seeded with 500 test artifacts (NF1). |
| `TestTestArtifactFilter_Unauthenticated` | The endpoint returns 401 when no session cookie is present. |

## Fixtures

- Mixed seeds: `lifecycle/tests/tf-test-a.md` (test/approved), `tf-test-b.md` (test/approved), `tf-test-c.md` (test/draft), `lifecycle/requirements/tf-ticket-2.md` (ticket/approved), `lifecycle/ideas/tf-idea.md` (idea/draft).
- Performance seeds: 500 × `lifecycle/tests/perf-test-NNNN.md` (test/approved).
