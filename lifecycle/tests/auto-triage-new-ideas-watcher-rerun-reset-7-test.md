---
title: "Auto-Triage Raw Ideas — Watcher Rerun After Status Reset"
type: test
status: in-qa
lineage: auto-triage-new-ideas
parent: lifecycle/defects/auto-triage-new-ideas-watcher-rerun-reset-7-defect.md
---

# Auto-Triage Raw Ideas — Watcher Rerun After Status Reset

## Overview

This artifact documents integration tests for the auto-triage functionality that verifies a triaged artifact can be reset to raw status and re-triaged properly. The tests ensure that:

1. A raw idea gets triaged to draft status
2. When the status is reset back to raw, it triggers re-triage
3. The `## Raw Idea` section is preserved during re-triage
4. The `pollForArtifactStatus` helper correctly parses the API response structure

## Test Coverage

This test suite covers:
- Initial triage of a raw idea artifact
- Resetting an artifact's status back to raw 
- Re-triage triggering and completion
- Verification that the original `## Raw Idea` section is preserved
- Correct behavior of the `pollForArtifactStatus` helper function

The tests specifically validate the fix for the defect where `pollForArtifactStatus` was incorrectly checking `data["status"]` directly instead of looking inside the nested `"artifact"` field returned by the API.

## Test Files

- `tests/integration/triage_watcher_test.go` - Contains the main test implementation
- `tests/integration/triage_helpers_test.go` - Contains the helper functions including `pollForArtifactStatus`

The existing `TestTriageWatcher_ReRunAfterStatusReset` test in `triage_watcher_test.go` already covers this functionality, and it has been verified to pass.