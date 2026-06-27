---
title: "Tests — Inherit Priority and Release Through Lineage"
type: test
status: approved
lineage: inherit-priority-and-release
parent: lifecycle/test-plans/inherit-priority-and-release-5-test.md
---

# Tests — Inherit Priority and Release Through Lineage

## Overview

Integration and unit tests verifying that `priority` and `release` are inherited
from a child's `parent` at creation time across all three server-side paths.
Tests map directly to the requirement's Acceptance Criteria and
FR-1…FR-11 / NFR-1…NFR-5.

## Test Files

| File | Layer | Milestone(s) |
|------|-------|--------------|
| `internal/artifact/inherit_test.go` | Unit | M1 — `ApplyInheritedFields` helper |
| `internal/ideachat/generate_test.go` | Unit | M3 — `ideachat.Generate` with stubbed LLM |
| `tests/integration/inherit_priority_release_test.go` | Integration | M2, M3 (handler), M4, M5, M6 |

---

## Milestone 1 — Unit tests: `ApplyInheritedFields`

**File:** `internal/artifact/inherit_test.go`

Table-driven tests via `TestApplyInheritedFields`:

| Sub-test | What is verified |
|----------|-----------------|
| empty child priority inherits from parent | FR-2: empty child Priority ← parent Priority |
| empty child release inherits from parent | FR-3: empty child Release ← parent Release |
| non-empty child priority wins over parent (FR-4) | Child value always wins |
| non-empty child release wins over parent (FR-4) | Child value always wins |
| empty child and empty parent leave fields empty | No fabricated defaults |
| title, type, lineage, labels, assignees are not inherited | Non-priority/release fields untouched |

---

## Milestone 2 — Manual creation path (`POST /artifacts`)

**File:** `tests/integration/inherit_priority_release_test.go`

| Test | Acceptance Criterion |
|------|---------------------|
| `TestInherit_Create_InheritsPriorityFromParent` | Child inherits parent `priority: high` |
| `TestInherit_Create_InheritsReleaseFromParent` | Child inherits parent `release: KC-Release4` |
| `TestInherit_Create_ExplicitValuesWin` | Explicit priority/release override parent (FR-4 / NFR-1) |
| `TestInherit_Create_ParentWithNoFields` | Parent with no priority/release → child has none |
| `TestInherit_Create_DanglingParent` | Non-existent parent → 201, no failure (FR-5 / NFR-4) |
| `TestInherit_Create_NFR2_InheritedEqualsExplicit` | Inherited values equal explicitly-supplied values (NFR-2) |
| `TestInherit_Create_OnlyDirectParentInherited` | Grandparent values not inherited (NFR-3, no recursive walk) |

---

## Milestone 3 — Agent / LLM generation path

**Unit tests (stubbed LLM):** `internal/ideachat/generate_test.go`

Stubs `ideachat.CallLLM` with a deterministic `propose` response so the
frontmatter is testable without a live API key.

| Test | Acceptance Criterion |
|------|---------------------|
| `TestGenerate_InheritsPriorityFromSource` | `SourcePriority: "high"` → frontmatter `priority: high` |
| `TestGenerate_InheritsReleaseFromSource` | `SourceRelease: "KC-Release4"` → frontmatter `release: KC-Release4` |
| `TestGenerate_ParentlessDefaultsPriorityNormal` | No source → `priority: normal`, no `release` key |
| `TestGenerate_ParentWithNoPriorityFallsToNormal` | Empty `SourcePriority` → `priority: normal` (FR-6) |
| `TestGenerate_PreviewFrontmatterReflectsInheritedValues` | Preview response carries inherited values before persist |

**Integration handler tests (require `ANTHROPIC_API_KEY`):** `tests/integration/inherit_priority_release_test.go`

Skipped automatically when `ANTHROPIC_API_KEY` is not set.

| Test | Acceptance Criterion |
|------|---------------------|
| `TestInherit_Generate_InheritsPriority` | Handler: source with priority → preview has priority |
| `TestInherit_Generate_InheritsRelease` | Handler: source with release → preview has release |
| `TestInherit_Generate_ParentlessDefaultPriority` | No source_path → `priority: normal`, no release |
| `TestInherit_Generate_EmptySourcePriorityFallsToNormal` | Source with no priority → `priority: normal` |

---

## Milestone 4 — Workflow rejection path

**File:** `tests/integration/inherit_priority_release_test.go`

| Test | Acceptance Criterion |
|------|---------------------|
| `TestInherit_Rejection_InheritsPriorityAndRelease` | Rejection artifact inherits source's priority + release (FR-7) |
| `TestInherit_Rejection_SourceWithNoFields` | Source with no fields → rejection artifact has none |

---

## Milestone 5 — Override isolation, validation, and no-migration

**File:** `tests/integration/inherit_priority_release_test.go`

| Test | Acceptance Criterion |
|------|---------------------|
| `TestInherit_Override_PatchPriorityIsolatedToChild` | PATCH child priority → only child changed, parent byte-unchanged (FR-9) |
| `TestInherit_Override_PatchReleaseIsolatedToChild` | PATCH child release → only child changed, parent byte-unchanged (FR-9) |
| `TestInherit_Create_InheritedReleaseSkipsValidation` | Inherited release not in release list → 201 accepted (FR-11) |
| `TestInherit_Override_PatchReleaseStillValidatesUnknownRelease` | PATCH with unknown release → 422 (existing validation preserved) |
| `TestInherit_NoMigration` | Server startup does not modify pre-existing files on disk |

---

## Milestone 6 — Cross-path consistency

**File:** `tests/integration/inherit_priority_release_test.go`

| Test | Acceptance Criterion |
|------|---------------------|
| `TestInherit_CrossPath_ManualAndRejectionConsistency` | Manual and rejection paths yield identical priority/release for same parent (NFR-5 / FR-8); LLM path added when `ANTHROPIC_API_KEY` is set |
