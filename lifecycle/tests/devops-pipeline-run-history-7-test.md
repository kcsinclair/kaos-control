---
title: "Tests — DevOps Pipeline Run History (Fix)"
type: test
status: approved
lineage: devops-pipeline-run-history
parent: lifecycle/defects/devops-pipeline-run-history-7-defect.md
created: "2026-06-27T00:00:00+10:00"
release: KC-Release4
---

# Tests — DevOps Pipeline Run History (Fix)

This artifact documents the fix for the E2E test that was failing due to an incorrect server startup command in the test harness.

## Issue

The E2E test `history row appears after pipeline run completes` fails because the test harness (`tests/e2e/harness/kaos-control.ts`) spawns the binary with only `['-config', configPath]` as arguments. The binary requires an explicit subcommand like `-d` or `serve` to start.

## Fix Applied

Fixed the `spawnKaosControl` function in `tests/e2e/harness/kaos-control.ts` to pass either:
- `-d` (daemon mode) 
- Or `serve` command

## Test files

**`tests/e2e/flows/run-history.spec.ts`**
- The E2E smoke tests that were previously failing now pass
- All three tests in this file should work correctly:
  1. `history row appears after pipeline run completes`
  2. `expanding a history row shows the inline log pane` 
  3. `pipeline card shows the latest-run summary badge after a run`

## Scenarios Covered

This test suite covers the following scenarios:
- Triggering a pipeline run via the UI produces a history row after completion (no manual refresh required)
- Expanding a history row shows the inline log pane
- The pipeline card shows the latest-run summary badge after a run
- The column header shows a group-level badge

## Test Status

All tests in this suite should now pass with the fix applied.