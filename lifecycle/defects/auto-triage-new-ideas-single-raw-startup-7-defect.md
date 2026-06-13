---
title: TestTriageStartup_SingleRawIdea fails due to pollForArtifactStatus parsing bug
type: defect
status: approved
lineage: auto-triage-new-ideas
parent: lifecycle/tests/auto-triage-new-ideas-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# TestTriageStartup_SingleRawIdea fails due to pollForArtifactStatus parsing bug

The integration test fails to detect that a raw idea enqueued at startup has successfully transitioned to `draft`. This occurs because the test helper `pollForArtifactStatus` parses the response incorrectly.

## Reproduction Steps

1. Run `go test -v -tags=integration ./tests/integration/ -run="TestTriageStartup_SingleRawIdea"` in the workspace root.
2. Observe the test timing out and failing with the message:
   `startup-triaged idea not draft within 5s; current fm: map[lineage:foo priority:normal status:draft title:Foo Idea type:idea]`

## Expected Behaviour

`pollForArtifactStatus` should successfully detect when the artifact status transitions to `draft` via HTTP.

## Actual Behaviour

Although the artifact is successfully updated to `draft` on disk (as shown in the error output's `current fm`), the helper function `pollForArtifactStatus` queries the GET `/api/p/testproject/artifacts/...` endpoint and checks `data["status"]` directly. However, the API returns the status nested under the `"artifact"` field (i.e. `data["artifact"].(map[string]any)["status"]`). As a result, `data["status"]` is always `nil`, causing the poll loop to timeout and fail.

## Logs / Output

```
=== RUN   TestTriageStartup_SingleRawIdea
    triage_startup_test.go:30: startup-triaged idea not draft within 5s; current fm: map[lineage:foo priority:normal status:draft title:Foo Idea type:idea]
--- FAIL: TestTriageStartup_SingleRawIdea (5.22s)
```
