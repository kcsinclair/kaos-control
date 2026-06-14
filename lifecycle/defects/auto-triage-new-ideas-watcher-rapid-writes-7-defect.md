---
title: TestTriageWatcher_RapidWrites_OneRun fails due to pollForArtifactStatus parsing bug
type: defect
status: done
lineage: auto-triage-new-ideas
parent: lifecycle/tests/auto-triage-new-ideas-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# TestTriageWatcher_RapidWrites_OneRun fails due to pollForArtifactStatus parsing bug

The integration test fails to detect that two rapid writes within the debounce window correctly trigger exactly one triage run and transition the status to `draft`. This is caused by incorrect parsing logic in the test helper `pollForArtifactStatus`.

## Reproduction Steps

1. Run `go test -v -tags=integration ./tests/integration/ -run="TestTriageWatcher_RapidWrites_OneRun"` in the workspace root.
2. Observe the test timing out and failing.

## Expected Behaviour

`pollForArtifactStatus` should successfully detect when the artifact transitions to `draft` via HTTP.

## Actual Behaviour

The test helper `pollForArtifactStatus` checks `data["status"]` directly instead of looking inside the nested `"artifact"` field returned by the GET `/api/p/testproject/artifacts/...` endpoint. Because of this, the check always evaluates to false, leading to a timeout.

## Logs / Output

```
=== RUN   TestTriageWatcher_RapidWrites_OneRun
    triage_watcher_test.go:159: artifact not triaged to draft within 5s
--- FAIL: TestTriageWatcher_RapidWrites_OneRun (5.27s)
```
