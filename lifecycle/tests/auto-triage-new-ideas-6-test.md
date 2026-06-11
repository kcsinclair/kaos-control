---
title: "Auto-Triage Raw Ideas — Integration Test Suite"
type: test
status: draft
lineage: auto-triage-new-ideas
parent: lifecycle/test-plans/auto-triage-new-ideas-5-test.md
---

# Auto-Triage Raw Ideas — Integration Test Suite

## Overview

This artifact documents the automated test suite for the auto-triage-new-ideas
feature. Tests are written in Go and split between internal unit tests (in
`internal/triage/`) and integration tests (in `tests/integration/`).

The LLM is replaced by a deterministic in-process fake (`ideachat.CallLLM` is a
package-level var; tests swap it and restore via `t.Cleanup`). Integration tests
run under the `//go:build integration` tag.

---

## Milestone 2 — Eligibility filter tests

**Files:**
- `internal/triage/eligibility_test.go` (additions to existing file)

| # | Test | Scenario |
|---|------|----------|
| 1 | `TestEligible_NotInIdeasDir` | Path outside ideas dir → `not_in_ideas_dir` |
| 2 | `TestEligible_NestedPath` | Nested under ideas → `not_in_ideas_dir` |
| 3 | `TestEligible_WrongType` | `type: defect`, `status: raw` → `wrong_type` |
| 4 | `TestEligible_WrongStatus` | `status: draft` → `wrong_status` |
| 5 | `TestEligible_WrongStatus_Clarifying` | `status: clarifying` → `wrong_status` |
| 6 | `TestEligible_OK` | `type: idea`, `status: raw` → eligible |
| 7 | `TestEligible_NotIndexed` | Path not in index → `not_indexed` |
| 8 | `TestEligible_CaseSensitivity` | `Status: Raw` (capital R) → `wrong_status` |

---

## Milestone 3 — Body rewrite and frontmatter mutation tests

**File:** `internal/triage/run_test.go`

| # | Test | Scenario |
|---|------|----------|
| 1 | `TestRewriteBody_FreshTriage` | First triage: wraps original in `## Raw Idea`, adds `## Idea` |
| 2 | `TestRewriteBody_ReRun` | Re-run: `## Raw Idea` unchanged, `## Idea` replaced |
| 3 | `TestRewriteBody_NoH1` | No H1 in original: output starts with `## Raw Idea` |
| 4 | `TestRewriteBody_AgentBodyWithH1` | Agent H1 stripped from `## Idea` section |
| 5 | `TestMergeAndFilterLabels_MergeAndDedup` | Labels merged, deduped, vocab-filtered |
| 6 | `TestPriority_AbsentDefaultsToNormal` | No priority → defaults to "normal" |
| 7 | `TestPriority_PresentIsPreserved` | Existing `high` priority preserved |
| 8 | `TestMarshalArtifact_KnownFieldsPreserved` | release, parent, lineage, assignees survive round-trip |
| 9 | (failure case covered in Milestone 8 integration tests) | Empty body → run failed |
| 10 | `TestRewriteBody_TitlePreservedInOutput` | Original H1 preserved; agent H1 not in `## Idea` |

---

## Milestone 4 — Concurrency and lock tests

**File:** `internal/triage/triage_test.go` (additions to existing file)

| # | Test | Scenario |
|---|------|----------|
| 1 | `TestTrigger_Dedup` | Coalesce same path → one run |
| 2 | `TestTrigger_SemaphoreCap` | Cap of 2: third trigger blocks |
| 3 | `TestStop_WaitsForInFlight` | Stop drains in-flight goroutines |
| 4 | `TestTrigger_ErrLocked` | Pre-locked lineage → `ErrLocked` |
| 5 | `TestTrigger_LockReleasedOnFailure` | Failure → lock released |
| 6 | `TestTrigger_LockReleasedOnPanic` | Panic → recovery, cleanup runs |

---

## Milestone 5 — Watcher → triage integration tests

**File:** `tests/integration/triage_watcher_test.go`

| # | Test | Scenario |
|---|------|----------|
| 1 | `TestTriageWatcher_CreateRawIdea_TriageRuns` | Create raw idea → triaged to draft within 5s |
| 2 | `TestTriageWatcher_CreateDraftIdea_NoTriage` | Draft idea → no agent_runs row |
| 3 | `TestTriageWatcher_CreateRawDefect_NoTriage` | Raw defect → no triage |
| 4 | `TestTriageWatcher_ModifyDraftIdea_NoTriage` | Modify draft → no new run |
| 5 | `TestTriageWatcher_RapidWrites_OneRun` | Two writes within debounce → exactly one run |
| 6 | `TestTriageWatcher_ReRunAfterStatusReset` | Reset to raw → re-triage, `## Raw Idea` preserved |

---

## Milestone 6 — Startup re-scan tests

**File:** `tests/integration/triage_startup_test.go`

| # | Test | Scenario |
|---|------|----------|
| 1 | `TestTriageStartup_SingleRawIdea` | Pre-existing raw idea triaged within 5s of Open |
| 2 | `TestTriageStartup_EmptyIdeasDir` | Empty ideas dir → no agent_runs rows |
| 3 | `TestTriageStartup_MultipleRawWithCap` | Three raw ideas all eventually triaged (MaxConcurrent=2) |
| 4 | `TestTriageStartup_AlreadyDraftNoRuns` | Pre-existing draft artifacts → no new runs |

---

## Milestone 7 — REST endpoint tests

**File:** `tests/integration/triage_api_test.go`

| # | Test | Scenario |
|---|------|----------|
| 1 | `TestTriageAPI_Unauthenticated` | No session → 401 or 403 |
| 2 | `TestTriageAPI_WrongRole` | `dev@test.local` (no analyst/reviewer) → 403 |
| 3 | `TestTriageAPI_UnknownSlug` | Unknown slug → 404 |
| 4 | `TestTriageAPI_AlreadyDraft` | Draft artifact → 409 `reason: wrong_status` |
| 5 | `TestTriageAPI_WrongType` | Raw defect under ideas → 409 `reason: wrong_type` |
| 6 | `TestTriageAPI_Success` | Raw idea → 202 with run_id, run completes |
| 7 | `TestTriageAPI_InFlightCoalesce` | Two rapid POSTs → same run_id, one DB row |
| 8 | `TestTriageAPI_LockedLineage` | Pre-locked lineage → 409 `error: locked` |

---

## Milestone 8 — Failure and observability tests

**File:** `tests/integration/triage_failure_test.go`

| # | Test | Scenario |
|---|------|----------|
| 1 | `TestTriageFailure_MalformedJSON` | Malformed LLM response → failed run, artifact unchanged |
| 2 | `TestTriageFailure_ActionClarify` | `action: clarify` response → failed run |
| 3 | `TestTriageFailure_EmptyBody` | Empty body in propose → failed run, artifact unchanged |
| 4 | `TestTriageFailure_SandboxViolation_UnitCheck` | Path traversal → `ErrIneligible{not_in_ideas_dir}` |
| 5 | `TestTriageFailure_NoRetry` | Failure → exactly one run row after 3s wait |
| 6 | `TestTriageFailure_LogLineContents` | Failure emits warn with `path`, `lineage`, `reason` fields |

---

## LLM injection mechanism

`ideachat.CallLLM` was converted from a named function to a package-level
`var` pointing to the real implementation. Tests call `installLLMFake(t,
scripted)` (defined in `tests/integration/triage_helpers_test.go`) before
creating the test environment. Cleanup restores the original implementation
via `t.Cleanup` (LIFO order ensures goroutines finish before restoration).
