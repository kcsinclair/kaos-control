---
title: "Q4/Q8 integration tests read error code from wrong JSON path"
type: defect
status: approved
lineage: agent-rate-limit-queue
parent: lifecycle/tests/agent-rate-limit-queue-6-test.md
labels:
    - defect
assignees:
    - role: test-developer
      who: agent
---

# Q4/Q8 integration tests read error code from wrong JSON path

Two integration tests (`TestQueue_Enqueue_DuplicateRejected` and
`TestQueue_Cancel_Running`) assert the `code` field of a 409 error response by
reading `data["code"]` directly from the top-level JSON map. The API returns
errors nested under an `"error"` key — `{"error":{"code":"...","message":"..."}}` —
matching the pattern used by every other test in the suite. As a result, both
tests always observe an empty string and fail.

## Reproduction Steps

1. `go test -count=1 -tags integration -run "TestQueue_Enqueue_DuplicateRejected|TestQueue_Cancel_Running" ./tests/integration/ -timeout 60s`
2. Observe both tests fail despite the server returning the correct 409 status.

## Expected Behaviour

Both tests pass; the error `code` field is correctly read from
`data["error"].(map[string]any)["code"].(string)`.

## Actual Behaviour

Both tests fail with:

```
queue_api_test.go:131: expected duplicate/already_queued error code, got ""
--- FAIL: TestQueue_Enqueue_DuplicateRejected

queue_api_test.go:255: expected running/cannot_cancel_running error code, got ""
--- FAIL: TestQueue_Cancel_Running
```

The server does return the correct payload, e.g.:
```json
{"error":{"code":"duplicate","message":"an active job for this artifact already exists"}}
```
but the tests look for `data["code"]` instead of `data["error"]["code"]`.

## Fix Required

In `tests/integration/queue_api_test.go`:

**Q4 (`TestQueue_Enqueue_DuplicateRejected`, ~line 127–131)** — replace:
```go
code, _ := data2["code"].(string)
if code != "duplicate" && code != "already_queued" {
```
with:
```go
errData, _ := data2["error"].(map[string]any)
code, _ := errData["code"].(string)
if code != "duplicate" && code != "already_queued" {
```

**Q8 (`TestQueue_Cancel_Running`, ~line 252–255)** — replace:
```go
code, _ := data["code"].(string)
if code != "running" && code != "cannot_cancel_running" {
```
with:
```go
errData, _ := data["error"].(map[string]any)
code, _ := errData["code"].(string)
if code != "running" && code != "cannot_cancel_running" {
```

## Logs / Output

```
=== RUN   TestQueue_Enqueue_DuplicateRejected
    queue_api_test.go:131: expected duplicate/already_queued error code, got ""
--- FAIL: TestQueue_Enqueue_DuplicateRejected (0.13s)

=== RUN   TestQueue_Cancel_Running
    queue_api_test.go:255: expected running/cannot_cancel_running error code, got ""
--- FAIL: TestQueue_Cancel_Running (0.18s)

FAIL    github.com/kaos-control/kaos-control/tests/integration  20.149s
```
