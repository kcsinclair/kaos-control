---
title: "Auto-Triage API Success — Startup Race Condition Regression Fix"
type: test
status: draft
lineage: auto-triage-new-ideas
parent: lifecycle/defects/auto-triage-new-ideas-api-success-race-7-defect.md
---

# Auto-Triage API Success — Startup Race Condition Regression Fix

## Overview

This artifact documents the regression fix for the defect described in
`auto-triage-new-ideas-api-success-race-7-defect.md`:
`TestTriageAPI_Success` was returning `409 Conflict` instead of `202 Accepted`
because the startup `RescanRaw` goroutine completed triage on the seeded raw
idea (in ~7 ms) before the test's 300 ms sleep ended and the POST request was
made.

## Root Cause (as documented in defect)

The original test installed a fast LLM fake via `installLLMFake`, seeded
`api-success.md` as `status: raw`, and then waited 300 ms before POSTing.
However, the startup `RescanRaw` goroutine found the raw idea, called the fast
LLM fake (completing in ~7 ms), transitioned the artifact to `draft`, and
removed the run from the `inFlight` map — all before the test's POST arrived.
The eligibility check then rejected the request with `wrong_status`, returning
`409`.

## Fix Applied

**File:** `tests/integration/triage_api_test.go` — `TestTriageAPI_Success`

The fast LLM fake was replaced with a **blocking LLM gate** that holds until
the test explicitly unblocks it:

| Change | Reason |
|--------|--------|
| `ideachat.CallLLM` replaced with a blocking func that selects on a `block` channel | Prevents any caller (startup scan or API-triggered run) from completing triage until the test controls when to release |
| LLM installed **before** `newTestEnvWithCfgYAML` | Establishes a happens-before relationship: the blocking fake is visible to the startup `RescanRaw` goroutine when it calls `ideachat.Generate` |
| `defer unblock()` + `t.Cleanup` with `unblock()` + 100 ms drain | Ensures the goroutine is released during both normal and panicked test teardown |
| `unblock()` called **after** the POST returns 202 | Allows `pollForRunStatus` to observe the run completing to `done` |

### How coalescing prevents the race

When the blocking LLM is in place, `RescanRaw` adds the run to `inFlight` and
then blocks. The test's POST arrives while the run is still in-flight. The
`Trigger` fast-path finds the existing `inFlight` entry and returns the same
`run_id` as a coalesced 202 response — without re-entering the eligibility
check. After `unblock()` the single goroutine completes, recording the run as
`done`.

## Scenarios Covered

| # | Step | Expected |
|---|------|----------|
| 1 | Blocking LLM installed; `api-success.md` seeded as `raw`; env started | `RescanRaw` finds the idea, enters `Trigger`, starts goroutine, blocks on LLM; run in `inFlight` |
| 2 | 300 ms sleep | Startup goroutine is definitely in-flight and blocking |
| 3 | `POST /api/p/testproject/ideas/api-success/triage` | `Trigger` fast-path coalesces onto startup run → `202 Accepted` with non-empty `run_id` |
| 4 | `unblock()` called | LLM returns; goroutine rewrites artifact, transitions to `draft`, records run as `done` |
| 5 | `pollForRunStatus(..., "done", 5s)` | Returns the completed run within 5 s |

## Test File

`tests/integration/triage_api_test.go` — `TestTriageAPI_Success`
