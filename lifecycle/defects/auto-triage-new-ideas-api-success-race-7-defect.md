---
title: TestTriageAPI_Success fails due to race condition with startup rescan
type: defect
status: approved
lineage: auto-triage-new-ideas
parent: lifecycle/tests/auto-triage-new-ideas-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# TestTriageAPI_Success fails due to race condition with startup rescan

The integration test `TestTriageAPI_Success` fails because the seeded raw idea is immediately triaged by the background startup rescan before the test can make its POST API request.

## Reproduction Steps

1. Run `go test -v -tags=integration ./tests/integration/triage_api_test.go ./tests/integration/triage_helpers_test.go ./tests/integration/helpers_test.go -run="TestTriageAPI_Success"` in the workspace root.
2. Observe the test failing because the POST API returns `409` instead of `202`.

## Expected Behaviour

The POST API call to `/api/p/testproject/ideas/api-success/triage` should trigger triage and return `202 StatusAccepted`.

## Actual Behaviour

The test seeds `api-success.md` as `status: raw` and starts the test environment. The project's startup rescan goroutine (`RescanRaw`) immediately processes the raw idea and triages it to `draft` (taking only ~7ms).
By the time the test wakes up from its `300ms` sleep and sends the POST request, the artifact's status is already `draft`. The API endpoint rejects the request as ineligible (wrong status), returning `409 wrong_status`.

## Logs / Output

```
=== RUN   TestTriageAPI_Success
2026/06/13 12:08:29 INFO triage started path=lifecycle/ideas/api-success.md lineage=api-success run_id=a903d313-5bb3-4b9b-8abd-8bf56de0bcb3 trigger=startup
2026/06/13 12:08:29 INFO triage completed path=lifecycle/ideas/api-success.md lineage=api-success run_id=a903d313-5bb3-4b9b-8abd-8bf56de0bcb3 duration_ms=7
2026/06/13 12:08:29 INFO http method=POST path=/api/p/testproject/ideas/api-success/triage status=409 bytes=49 duration=1.030791ms
    triage_api_test.go:128: expected status 202, got 409: {"error":"not_eligible","reason":"wrong_status"}
--- FAIL: TestTriageAPI_Success (0.50s)
```
