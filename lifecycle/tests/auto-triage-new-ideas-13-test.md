---
title: "Auto-Triage Watcher CreateRaw — pollForArtifactStatus Regression Test"
type: test
status: draft
lineage: auto-triage-new-ideas
parent: lifecycle/defects/auto-triage-new-ideas-watcher-create-raw-7-defect.md
---

# Auto-Triage Watcher CreateRaw — pollForArtifactStatus Regression Test

## Overview

This artifact documents the regression tests for the defect described in
`auto-triage-new-ideas-watcher-create-raw-7-defect.md`: `TestTriageWatcher_CreateRawIdea_TriageRuns`
timed out because `pollForArtifactStatus` was checking `data["status"]` at the
top level of the GET `/api/p/:project/artifacts/*path` response instead of the
correct nested path `data["artifact"]["status"]`.

The fix was already applied to `pollForArtifactStatus` in
`tests/integration/triage_helpers_test.go` before these regression tests were
written. The regression tests exist to catch any future reversion to the
broken parsing logic.

## Root Cause (as documented in defect)

The GET `/api/p/:project/artifacts/*path` endpoint returns:

```json
{
  "artifact": { "status": "draft", "title": "...", ... },
  "body": "...",
  "body_html": "...",
  "file_sha": "..."
}
```

The broken `pollForArtifactStatus` checked `data["status"]` (top level), which
is always nil. Even after triage completed and the artifact transitioned to
`draft` on disk, the poll loop could never match, so the test timed out. The
fix reads `data["artifact"]["status"]` instead.

## Scenarios Covered

| # | Test | Scenario |
|---|------|----------|
| 1 | `TestArtifactAPIResponse_StatusNestedInArtifactKey` | GET artifact response does NOT have `status` at top level; status is inside `data["artifact"]` |
| 2 | `TestPollForArtifactStatus_ReadsNestedArtifactField` | `pollForArtifactStatus` correctly detects a pre-existing `draft` status by reading `data["artifact"]["status"]`; times out under the broken top-level implementation |

## Test File

`tests/integration/triage_watcher_regression_test.go`
