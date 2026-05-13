---
title: "Defect Fix Tests — Q4/Q8 error code JSON path correction"
type: test
status: in-qa
lineage: agent-rate-limit-queue
parent: lifecycle/defects/agent-rate-limit-queue-7-defect.md
created: "2026-05-12T00:00:00+10:00"
labels:
    - defect
    - queue
    - test
---

# Defect Fix Tests — Q4/Q8 error code JSON path correction

Fixes two integration tests in `tests/integration/queue_api_test.go` that were
reading the error `code` field from the wrong level of the JSON response.

The API returns errors as `{"error":{"code":"...","message":"..."}}`, but both
affected tests were dereferencing `data["code"]` at the top level, which always
produced an empty string and caused the assertion to fail even when the server
returned the correct 409 payload.

---

## Changes

**File**: `tests/integration/queue_api_test.go`

### Q4 `TestQueue_Enqueue_DuplicateRejected` (line ~129)

**Before**:
```go
code, _ := data2["code"].(string)
```

**After**:
```go
errData2, _ := data2["error"].(map[string]any)
code, _ := errData2["code"].(string)
```

### Q8 `TestQueue_Cancel_Running` (line ~253)

**Before**:
```go
code, _ := data["code"].(string)
```

**After**:
```go
errData, _ := data["error"].(map[string]any)
code, _ := errData["code"].(string)
```

---

## Scenarios covered

| Test | Scenario |
|------|----------|
| Q4 `TestQueue_Enqueue_DuplicateRejected` | Second enqueue of the same artifact → 409; `error.code` is `duplicate` or `already_queued` |
| Q8 `TestQueue_Cancel_Running` | Cancel a running job → 409; `error.code` is `running` or `cannot_cancel_running` |

Both tests were already structurally correct (status code assertion, request
setup, fake-claude blocking). Only the error-code extraction was wrong.

---

## Notes

No new test cases were added. This change makes two existing tests correctly
observe the API response they were designed to validate.
