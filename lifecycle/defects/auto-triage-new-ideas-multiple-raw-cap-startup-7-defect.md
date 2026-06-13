---
title: TestTriageStartup_MultipleRawWithCap fails due to pollForArtifactStatus parsing bug
type: defect
status: approved
lineage: auto-triage-new-ideas
parent: lifecycle/tests/auto-triage-new-ideas-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# TestTriageStartup_MultipleRawWithCap fails due to pollForArtifactStatus parsing bug

The integration test fails to detect that multiple raw ideas enqueued at startup have transitioned to `draft` under concurrency constraints. This is caused by the incorrect parsing logic in the test helper `pollForArtifactStatus`.

## Reproduction Steps

1. Run `go test -v -tags=integration ./tests/integration/ -run="TestTriageStartup_MultipleRawWithCap"` in the workspace root.
2. Observe the test timing out and failing.

## Expected Behaviour

`pollForArtifactStatus` should successfully detect when each of the multiple artifacts transitions to `draft` via HTTP.

## Actual Behaviour

The test helper `pollForArtifactStatus` checks `data["status"]` directly instead of looking inside the nested `"artifact"` field returned by the GET `/api/p/testproject/artifacts/...` endpoint. Because of this, the check always evaluates to false, leading to a timeout.

## Logs / Output

```
=== RUN   TestTriageStartup_MultipleRawWithCap
    triage_startup_test.go:95: idea lifecycle/ideas/bar1.md not triaged to draft within timeout; current fm: map[lineage:bar1 priority:normal status:draft title:Bar One type:idea]
    triage_startup_test.go:91: timeout waiting for triage of lifecycle/ideas/bar2.md
--- FAIL: TestTriageStartup_MultipleRawWithCap (10.22s)
```
