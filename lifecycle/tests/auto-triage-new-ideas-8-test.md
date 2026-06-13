---
title: "Auto-Triage Locked Lineage — Defect Regression Test"
type: test
status: draft
lineage: auto-triage-new-ideas
parent: lifecycle/defects/auto-triage-new-ideas-api-locked-cooldown-7-defect.md
---

# Auto-Triage Locked Lineage — Defect Regression Test

## Overview

This artifact documents the regression fix for the defect described in
`auto-triage-new-ideas-api-locked-cooldown-7-defect.md`: `TestTriageAPI_LockedLineage`
was returning `202 Accepted` instead of `409 Conflict` because a zombie cooldown
entry in the triage manager's `inFlight` map caused the Trigger fast-path to
coalesce onto the already-failed startup run.

## Root Cause (as documented in defect)

The original test seeded `locked-idea.md` as `status: raw`. On startup,
`RescanRaw` immediately triggered triage. With no LLM fake installed the run
failed within milliseconds, releasing the lock but leaving a 5-second zombie
entry in the `inFlight` map. The test then acquired the lock (succeeding
because the triage run had already released it), rewrote the file to raw, and
POSTed — but the Trigger fast-path hit the zombie before ever checking the
lock, returning the zombie's run_id with HTTP 202.

## Fix Applied

**File:** `tests/integration/triage_api_test.go` — `TestTriageAPI_LockedLineage`

The test was rewritten with three changes:

| Change | Reason |
|--------|--------|
| Seed `locked-idea.md` as `draft` (not `raw`) | Startup `RescanRaw` skips draft artifacts; no zombie is ever created |
| Acquire the lineage lock *before* rewriting the file to `raw` | The watcher-triggered `Trigger` sees the lock immediately and returns `ErrLocked` without spawning a goroutine or zombie entry |
| Replace `time.Sleep(300ms)` with `pollForArtifactStatus(..., "raw", 3s)` | Eliminates the timing gap on slow CI hosts where 300 ms is insufficient for watcher debounce + re-index to complete |

## Scenario Covered

| # | Step | Expected |
|---|------|----------|
| 1 | Seed `locked-idea.md` as `draft`; start env | Startup scan finds no raw ideas; triage manager `inFlight` map is empty |
| 2 | Sleep 300 ms | Startup scan has definitely completed |
| 3 | `Locks.Acquire("locked-idea", "test-holder", "agent")` | Succeeds — no competing holder |
| 4 | `writeRawIdea(...)` | File written as `raw`; watcher debounce starts |
| 5 | `pollForArtifactStatus(..., "raw", 3s)` | Returns `true` once watcher has re-indexed the file |
| 6 | Watcher goroutine calls `Trigger` | Lock held by "test-holder" → `ErrLocked` → no goroutine spawned → no zombie |
| 7 | `POST /api/p/testproject/ideas/locked-idea/triage` | `Trigger`: `inFlight` empty → eligible → lock held → `ErrLocked` → HTTP 409 `{"error":"locked"}` |

## Test File

`tests/integration/triage_api_test.go` — `TestTriageAPI_LockedLineage`
