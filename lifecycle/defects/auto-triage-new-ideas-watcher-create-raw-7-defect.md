---
title: TestTriageWatcher_CreateRawIdea_TriageRuns fails due to pollForArtifactStatus parsing bug
type: defect
status: approved
lineage: auto-triage-new-ideas
parent: lifecycle/tests/auto-triage-new-ideas-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# TestTriageWatcher_CreateRawIdea_TriageRuns fails due to pollForArtifactStatus parsing bug

The integration test fails to detect that a newly created raw idea has been successfully triaged to `draft` by the file watcher. This is caused by incorrect parsing logic in the test helper `pollForArtifactStatus`.

## Reproduction Steps

1. Run `go test -v -tags=integration ./tests/integration/ -run="TestTriageWatcher_CreateRawIdea_TriageRuns"` in the workspace root.
2. Observe the test timing out and failing.

## Expected Behaviour

`pollForArtifactStatus` should successfully detect when the created artifact transitions to `draft` via HTTP.

## Actual Behaviour

The test helper `pollForArtifactStatus` checks `data["status"]` directly instead of looking inside the nested `"artifact"` field returned by the GET `/api/p/testproject/artifacts/...` endpoint. Because of this, the check always evaluates to false, leading to a timeout despite the file successfully transitioning on disk.

## Logs / Output

```
=== RUN   TestTriageWatcher_CreateRawIdea_TriageRuns
    triage_watcher_test.go:30: artifact not triaged to draft within 5s; current fm: map[lineage:alpha priority:normal status:draft title:Alpha Idea type:idea]
--- FAIL: TestTriageWatcher_CreateRawIdea_TriageRuns (5.24s)
```
