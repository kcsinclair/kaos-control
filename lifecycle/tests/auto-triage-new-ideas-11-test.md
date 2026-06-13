---
title: "Auto-Triage MultipleRawWithCap — pollForArtifactStatus Startup Fix"
type: test
status: draft
lineage: auto-triage-new-ideas
parent: lifecycle/defects/auto-triage-new-ideas-multiple-raw-cap-startup-7-defect.md
---

# Auto-Triage MultipleRawWithCap — pollForArtifactStatus Startup Fix

## Overview

This artifact documents the regression fix for the defect described in
`auto-triage-new-ideas-multiple-raw-cap-startup-7-defect.md`:
`TestTriageStartup_MultipleRawWithCap` was timing out because the test helper
`pollForArtifactStatus` read `data["status"]` directly from the top-level JSON
response instead of drilling into the nested `data["artifact"]["status"]` field
that the `GET /api/p/{project}/artifacts/{path}` endpoint returns.

## Root Cause (as documented in defect)

The `GET /api/p/testproject/artifacts/<path>` endpoint wraps the artifact under
an `"artifact"` key:

```json
{ "artifact": { "status": "draft", ... } }
```

The original `pollForArtifactStatus` looked up `data["status"]` at the top
level, which is always absent, so the comparison with `"draft"` always returned
false. The disk file had already transitioned to `draft` (as shown by
`current fm: map[... status:draft ...]` in the failure log) but the HTTP poll
never detected it, causing the test to time out.

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

This fix is shared with the watcher rapid-writes defect
(`auto-triage-new-ideas-watcher-rapid-writes-7-defect.md`) documented in
`auto-triage-new-ideas-10-test.md`. Both tests failed for the same underlying
reason; fixing `pollForArtifactStatus` once resolves both.

## Scenarios Covered

| # | Test | Scenario |
|---|------|----------|
| 1 | `TestTriageStartup_MultipleRawWithCap` | Three raw ideas seeded before project open; startup `RescanRaw` queues all three; `MaxConcurrent=2` caps goroutines; all three eventually transition to `draft` within a shared 10 s deadline |

### Test walkthrough

| Step | What happens |
|------|-------------|
| Seed `bar1.md`, `bar2.md`, `bar3.md` as `raw` | Pre-existing ideas present before `project.Open` |
| `newTriageTestEnvWithSeeds` starts the project | `RescanRaw` picks up all three; semaphore limits to 2 concurrent runs |
| Three LLM fake responses installed (one per slug) | Each run returns a valid `propose` JSON, transitions to `draft` |
| `pollForArtifactStatus` polls each path sequentially | Returns `true` once `data["artifact"]["status"] == "draft"` |
| Shared 10 s deadline across all three polls | Ensures even the third queued run completes before timeout |

## Test Files

- `tests/integration/triage_startup_test.go` — `TestTriageStartup_MultipleRawWithCap`
- `tests/integration/triage_helpers_test.go` — `pollForArtifactStatus` (fix location)
