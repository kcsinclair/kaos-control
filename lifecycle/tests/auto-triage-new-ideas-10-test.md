---
title: "Auto-Triage Rapid Writes — pollForArtifactStatus Fix"
type: test
status: draft
lineage: auto-triage-new-ideas
parent: lifecycle/defects/auto-triage-new-ideas-watcher-rapid-writes-7-defect.md
---

# Auto-Triage Rapid Writes — pollForArtifactStatus Fix

## Overview

This artifact documents the regression fix for the defect described in
`auto-triage-new-ideas-watcher-rapid-writes-7-defect.md`:
`TestTriageWatcher_RapidWrites_OneRun` was timing out because the test helper
`pollForArtifactStatus` read `data["status"]` directly from the top-level JSON
response instead of drilling into the nested `data["artifact"]["status"]` field
that the `GET /api/p/{project}/artifacts/{path}` endpoint actually returns.

## Root Cause (as documented in defect)

The `GET /api/p/testproject/artifacts/<path>` endpoint wraps the artifact object
under an `"artifact"` key:

```json
{ "artifact": { "status": "draft", ... } }
```

The original `pollForArtifactStatus` looked up `data["status"]` at the top
level, which is always absent, so the comparison with `"draft"` always returned
false and the poll loop ran until timeout.

## Fix Applied

**File:** `tests/integration/triage_helpers_test.go` — `pollForArtifactStatus`

The response body is now unwrapped through the `"artifact"` key before the
status comparison:

```go
data := readJSON(t, resp)
if art, ok := data["artifact"].(map[string]any); ok {
    if status, _ := art["status"].(string); status == want {
        return true
    }
}
```

This is a two-level type assertion: first asserting the outer `"artifact"` field
is a `map[string]any`, then reading `"status"` from it. If either assertion
fails (e.g. unexpected response shape) the loop continues polling rather than
panicking, preserving the eventual-consistency polling contract.

## Scenarios Covered

| # | Test | Scenario |
|---|------|----------|
| 1 | `TestTriageWatcher_RapidWrites_OneRun` | Two writes within the 150 ms debounce window → watcher coalesces → exactly one triage run → artifact transitions to `draft` |

The fix to `pollForArtifactStatus` also benefits every other test that calls it:

| Test | How it uses `pollForArtifactStatus` |
|------|-------------------------------------|
| `TestTriageWatcher_CreateRawIdea_TriageRuns` | Waits for `draft` after initial triage |
| `TestTriageWatcher_ReRunAfterStatusReset` | Waits for `draft` after each of the two triage passes |

## Test Files

- `tests/integration/triage_watcher_test.go` — `TestTriageWatcher_RapidWrites_OneRun`
- `tests/integration/triage_helpers_test.go` — `pollForArtifactStatus` (fix location)
