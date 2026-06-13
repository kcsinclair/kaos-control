---
title: TestTriageWatcher_ReRunAfterStatusReset fails due to pollForArtifactStatus parsing bug
type: defect
status: draft
lineage: auto-triage-new-ideas
parent: lifecycle/tests/auto-triage-new-ideas-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# TestTriageWatcher_ReRunAfterStatusReset fails due to pollForArtifactStatus parsing bug

The integration test fails to detect that resetting an artifact's status back to raw triggers triage again. This is caused by incorrect parsing logic in the test helper `pollForArtifactStatus`.

## Reproduction Steps

1. Run `go test -v -tags=integration ./tests/integration/ -run="TestTriageWatcher_ReRunAfterStatusReset"` in the workspace root.
2. Observe the test timing out and failing.

## Expected Behaviour

`pollForArtifactStatus` should successfully detect when the artifact transitions to `draft` via HTTP.

## Actual Behaviour

The test helper `pollForArtifactStatus` checks `data["status"]` directly instead of looking inside the nested `"artifact"` field returned by the GET `/api/p/testproject/artifacts/...` endpoint. Because of this, the check always evaluates to false, leading to a timeout.

## Logs / Output

```
=== RUN   TestTriageWatcher_ReRunAfterStatusReset
    triage_watcher_test.go:185: initial triage did not produce draft within 5s
--- FAIL: TestTriageWatcher_ReRunAfterStatusReset (5.24s)
```
