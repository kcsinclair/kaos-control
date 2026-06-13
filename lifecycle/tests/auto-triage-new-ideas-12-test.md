---
title: "Auto-Triage SingleRawIdea Startup — pollForArtifactStatus Fix"
type: test
status: draft
lineage: auto-triage-new-ideas
parent: lifecycle/defects/auto-triage-new-ideas-single-raw-startup-7-defect.md
---

# Auto-Triage SingleRawIdea Startup — pollForArtifactStatus Fix

## Overview

This artifact documents the regression fix for the defect described in
`auto-triage-new-ideas-single-raw-startup-7-defect.md`:
`TestTriageStartup_SingleRawIdea` was timing out because the test helper
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
false. The disk file had already transitioned to `draft` (confirmed by the error
output showing `current fm: map[lineage:foo priority:normal status:draft title:Foo Idea type:idea]`)
but the HTTP poll never detected it, causing the 5 s timeout.

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

This fix is shared with the watcher rapid-writes defect
(`auto-triage-new-ideas-watcher-rapid-writes-7-defect.md`, documented in
`auto-triage-new-ideas-10-test.md`) and the multiple-raw-cap-startup defect
(`auto-triage-new-ideas-multiple-raw-cap-startup-7-defect.md`, documented in
`auto-triage-new-ideas-11-test.md`). All three tests failed for the same
underlying reason; fixing `pollForArtifactStatus` once resolves all three.

## Scenarios Covered

| # | Test | Scenario |
|---|------|----------|
| 1 | `TestTriageStartup_SingleRawIdea` | Single raw idea seeded before project open; startup `RescanRaw` triages it within 5 s; `pollForArtifactStatus` detects `draft` via the nested `data["artifact"]["status"]` field |

### Test walkthrough

| Step | What happens |
|------|-------------|
| LLM fake installed with single `propose` response for slug `foo` | Deterministic; no real API calls |
| `foo.md` seeded as `type: idea`, `status: raw` before `newTriageTestEnvWithSeeds` | File exists on disk before `project.Open` |
| `newTriageTestEnvWithSeeds` starts the project | `RescanRaw` finds `foo.md`, enqueues triage |
| Triage goroutine calls LLM fake | Returns valid `propose` JSON; artifact rewritten with `status: draft` |
| `pollForArtifactStatus` polls `GET /api/p/testproject/artifacts/lifecycle/ideas/foo.md` | Unwraps `data["artifact"]["status"]`; returns `true` once `"draft"` is observed |
| Test passes within the 5 s deadline | Verified end-to-end: startup scan → triage → HTTP status visible via API |

## Test Files

- `tests/integration/triage_startup_test.go` — `TestTriageStartup_SingleRawIdea`
- `tests/integration/triage_helpers_test.go` — `pollForArtifactStatus` (fix location)
