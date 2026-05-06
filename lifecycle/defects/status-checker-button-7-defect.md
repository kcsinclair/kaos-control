---
title: "POST /status-check/advance response schema mismatch — outcome always empty"
type: defect
status: in-development
lineage: status-checker-button
parent: lifecycle/tests/status-checker-button-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# POST /status-check/advance response schema mismatch — outcome always empty

The `handleStatusCheckAdvance` handler at `internal/http/status_check.go` returns a
response schema that does not match the contract defined in the test plan. The handler
uses `ok` (bool), `advanced_to` (string), and `error` (string) fields, but all tests
expect `outcome` (`"advanced"` | `"skipped"` | `"error"`), `new_status` (string), and
`reason` (string).

Because the `outcome` field is always missing from responses, every assertion that checks
for `outcome == "advanced"` fails, and every assertion checking for `outcome == "error"` or
`"skipped"` also fails. Six integration tests are broken by this single mismatch.

## Reproduction Steps

1. `go test -tags integration ./tests/integration/ -run "TestAdvance_Single|TestAdvance_MultipleSequential|TestAdvance_PermissionDenied|TestAdvance_WebSocketEvent|TestStatusCheckE2E_FullFlow|TestStatusCheckE2E_ConcurrentAdvance" -v -timeout 120s`
2. Observe that every test that asserts a specific `outcome` value fails immediately.

## Expected Behaviour

`POST /api/p/{project}/status-check/advance` response body:

```json
{
  "results": [
    {
      "path": "lifecycle/ideas/foo.md",
      "outcome": "advanced",
      "new_status": "clarifying"
    },
    {
      "path": "lifecycle/ideas/bar.md",
      "outcome": "skipped"
    },
    {
      "path": "lifecycle/ideas/baz.md",
      "outcome": "error",
      "reason": "requires role with permission to transition \"draft\" → \"clarifying\""
    }
  ]
}
```

Each result entry uses:
- `outcome` — `"advanced"`, `"skipped"`, or `"error"` (never `ok`)
- `new_status` — the status the artifact was advanced to (only when `outcome == "advanced"`)
- `reason` — human-readable message (when `outcome` is `"skipped"` or `"error"`)

## Actual Behaviour

The handler defines a local `advanceResult` struct with:
- `Ok bool json:"ok"`
- `AdvancedTo string json:"advanced_to,omitempty"`
- `Error string json:"error,omitempty"`

All six fields `outcome`, `new_status`, and `reason` are absent from every response.
Tests that decode `outcome` get an empty string; the struct comment in the handler even
documents the old format (`{"path": "...", "advanced_to": "planning", "ok": true}`).

## Logs / Output

```
--- FAIL: TestAdvance_Single (0.12s)
    status_check_test.go:379: outcome: want "advanced", got ""
    status_check_test.go:382: new_status: want "clarifying", got ""

--- FAIL: TestAdvance_MultipleSequential (0.12s)
    status_check_test.go:459: idea outcome: want advanced, got ""
    status_check_test.go:467: req outcome: want advanced, got ""

--- FAIL: TestAdvance_PermissionDenied (0.11s)
    status_check_test.go:519: expected outcome 'error' or 'skipped', got ""
    status_check_test.go:522: reason should be non-empty when permission is denied

--- FAIL: TestAdvance_WebSocketEvent (0.12s)
    status_check_test.go:623: advance did not succeed; outcome: [{Path:lifecycle/ideas/sc-adv-ws.md Outcome: NewStatus: Reason:}]

--- FAIL: TestStatusCheckE2E_FullFlow (0.16s)
    status_check_e2e_test.go:71: unexpected outcome "" for path "lifecycle/ideas/e2e-full.md"

--- FAIL: TestStatusCheckE2E_ConcurrentAdvance (0.12s)
    status_check_e2e_test.go:166: expected exactly 1 'advanced' outcome across both concurrent calls; got 0
```

## Fix guidance

In `internal/http/status_check.go`, replace the local `advanceResult` struct and its
population logic in `handleStatusCheckAdvance`:

- Rename `ok bool / advanced_to string / error string` → `outcome string / new_status string / reason string`
- Map success cases to `outcome: "advanced"` with `new_status` set
- Map no-staleness / already-current cases to `outcome: "skipped"`
- Map permission-denied and error cases to `outcome: "error"` with `reason` set
- Update the handler's doc comment to reflect the correct response shape
