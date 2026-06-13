---
title: TestTriageAPI_LockedLineage fails due to startup rescan lock and 5s cooldown
type: defect
status: in-development
lineage: auto-triage-new-ideas
parent: lifecycle/tests/auto-triage-new-ideas-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# TestTriageAPI_LockedLineage fails due to startup rescan lock and 5s cooldown

The integration test `TestTriageAPI_LockedLineage` fails because the seeded raw idea is processed by the background startup rescan. The background run immediately fails (since the LLM is not mocked for this test) and enters the `5s` failure cooldown, which makes the subsequent API POST call coalesce (returning `202`) instead of failing with `409 locked`.

## Reproduction Steps

1. Run `go test -v -tags=integration ./tests/integration/triage_api_test.go ./tests/integration/triage_helpers_test.go ./tests/integration/helpers_test.go -run="TestTriageAPI_LockedLineage"` in the workspace root.
2. Observe the test failing because the POST API returns `202` instead of `409`.

## Expected Behaviour

The POST API call to `/api/p/testproject/ideas/locked-idea/triage` should return `409 StatusConflict` with `error: locked` since the lineage lock was pre-acquired by `"test-holder"`.

## Actual Behaviour

The test seeds `locked-idea.md` as `status: raw` and starts the test environment. The project's startup rescan goroutine (`RescanRaw`) immediately triggers triage on the raw idea. The triage run starts and fails quickly (since `ideachat.CallLLM` is not mocked/faked in this test and fails or is mocked to fail).
When it fails, the run releases the lock but enters a `5s` failure cooldown.
Meanwhile, the test sleeps for `300ms` and then successfully acquires the lineage lock as `"test-holder"` (since the background run has already failed and released the lock).
However, when the test makes the POST request, the `5s` cooldown is still active. The `Trigger` method coalesces the request onto the zombie in-flight failed run, returning `202 Accepted` instead of failing with `409 locked`.

## Logs / Output

```
=== RUN   TestTriageAPI_LockedLineage
2026/06/13 12:08:30 INFO triage started path=lifecycle/ideas/locked-idea.md lineage=locked-idea run_id=b73610f0-9bd2-4793-9ce3-6465c28f866c trigger=startup
2026/06/13 12:08:30 WARN triage failed path=lifecycle/ideas/locked-idea.md lineage=locked-idea run_id=b73610f0-9bd2-4793-9ce3-6465c28f866c reason="ideachat.Generate: Generate: LLM call failed: deliberate failure"
2026/06/13 12:08:34 INFO http method=POST path=/api/p/testproject/ideas/locked-idea/triage status=202 bytes=50
    triage_api_test.go:236: expected status 409, got 202: {"run_id":"b73610f0-9bd2-4793-9ce3-6465c28f866c"}
--- FAIL: TestTriageAPI_LockedLineage (3.70s)
```
