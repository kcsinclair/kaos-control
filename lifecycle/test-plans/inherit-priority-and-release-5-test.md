---
title: "Test Plan — Inherit Priority and Release Through Lineage"
type: plan-test
status: draft
lineage: inherit-priority-and-release
parent: lifecycle/requirements/inherit-priority-and-release-2.md
---

# Test Plan — Inherit Priority and Release Through Lineage

## Overview

Verify that `priority` and `release` are inherited from a child's `parent` at
creation across all three server-side paths, that explicit values win, that the
helper is the single enforcement point, that overrides stay isolated, and that
unresolvable parents and existing on-disk files are unaffected. Tests map
directly to the requirement's Acceptance Criteria and FR-1…FR-11 / NFR-1…NFR-5.

Test layers:
- **Go unit tests** for the pure helper `artifact.ApplyInheritedFields` (backend Milestone 1).
- **Go handler/integration tests** for the three creation paths and override isolation, using the existing `testEnv` harness (admin auto-login; see [[inherit-priority-and-release]] backend plan for handler locations).
- **No frontend test is required for an inheritance indicator** — it is descoped (Resolved Question 4). Add a lightweight check that create-request payloads omit unset `priority`/`release`.

---

## Milestone 1 — Unit tests for the shared helper

### Description

Table-driven tests for `artifact.ApplyInheritedFields(child *Frontmatter, parent Frontmatter)`
covering FR-2, FR-3, FR-4 and the "no fabricated default" rule.

### Files to change

- `internal/artifact/inherit_test.go` (new).

### Acceptance criteria

- [ ] Empty child priority + parent priority `high` → child priority `high`.
- [ ] Empty child release + parent release `KC-Release4` → child release `KC-Release4`.
- [ ] Non-empty child priority `low` + parent priority `high` → child stays `low` (FR-4).
- [ ] Non-empty child release + parent release → child release unchanged (FR-4).
- [ ] Empty child + empty parent for each field → child field stays empty (no default).
- [ ] Fields other than `Priority`/`Release` (title, type, lineage, labels, assignees) are untouched.

---

## Milestone 2 — Manual creation path (`POST /artifacts`)

### Description

Handler tests through the real route: create a parent artifact with
`priority: high, release: KC-Release4`, then `POST /artifacts` with that parent.

### Files to change

- `tests/` (or `internal/http/write_test.go`) — new cases under the existing create-artifact test harness.

### Acceptance criteria

- [ ] Create with `parent`, no `priority` → response/file has child `priority` == parent's.
- [ ] Create with `parent`, no `release` → child `release` == parent's.
- [ ] Create with explicit `priority`/`release` differing from parent → supplied values preserved (FR-4 / NFR-1).
- [ ] Parent with no `priority`/`release` → child has none (no fabricated default).
- [ ] **Dangling `parent`** (points at a non-existent file) → `201 Created`, no inheritance, request does not fail (FR-5 / NFR-4).
- [ ] Created file is byte-identical to the same artifact created with the values supplied explicitly (NFR-2).
- [ ] Exactly one parent lookup occurs; no recursive lineage walk (NFR-3) — assert via a parent whose own parent has different values and confirm only the direct parent's values are inherited.

---

## Milestone 3 — Agent / LLM generation path

### Description

Test `ideachat.Generate` and `handleIdeaGenerate` with a stubbed/faked LLM
response so the proposed frontmatter is deterministic, exercising FR-6 and
Resolved Question 3.

### Files to change

- `internal/ideachat/generate_test.go` — extend with `SourcePriority`/`SourceRelease` cases.
- `internal/http/idea_generate_test.go` (or `tests/`) — handler test resolving the source from the index.

### Acceptance criteria

- [ ] Generation with a parent carrying `priority: high` → result frontmatter `priority: high` (not `normal`).
- [ ] Generation with a parent carrying a `release` → result frontmatter includes that `release`.
- [ ] Parentless generation → `priority: normal`, no `release` key (Resolved Q3).
- [ ] Parent present but with no `priority` → falls back to `normal` (FR-6).
- [ ] The generate **preview** response (not just the eventual persisted file) reflects the inherited values.

---

## Milestone 4 — Workflow rejection path

### Description

Drive a transition that produces a rejection artifact via `writeRejectionArtifact`
from a source carrying `priority`/`release`, and confirm inheritance (FR-7) plus
the defect case (Resolved Question 2).

### Files to change

- `internal/http/transition_test.go` (or `tests/`) — rejection-artifact inheritance cases.

### Acceptance criteria

- [ ] Rejection artifact inherits the source's `priority` and `release` alongside the already-copied `title`/`type`/`lineage`/`parent`.
- [ ] A defect created with a `parent` (artifact under test) inherits that parent's `priority`/`release`.
- [ ] Source with no `priority`/`release` → rejection artifact has none.

---

## Milestone 5 — Override isolation, validation, and no-migration

### Description

Cover FR-9, FR-11, NFR-1 and the no-migration non-goal.

### Files to change

- `tests/` — override-isolation and no-migration cases (may reuse existing priority/release PATCH tests from [[artefact-priority-inline-edit]] / [[inline-release-display-edit]]).

### Acceptance criteria

- [ ] After inheritance, `PATCH .../priority` on the child changes only the child file; the parent and any sibling files are byte-unchanged on disk (FR-9).
- [ ] After inheritance, `PATCH .../release` on the child changes only the child file.
- [ ] Inherited `release` is accepted at creation **without** a release-list validation call (FR-11), even when that release name is not in the current project release list.
- [ ] Override via `PATCH .../release` with an invalid release still returns `422` (existing behaviour preserved).
- [ ] Running the change against a fixture tree of pre-existing artifacts modifies none of them (no migration).

---

## Milestone 6 — Single-enforcement-point and consistency assertion

### Description

Assert FR-8 / NFR-5 structurally: the same `(parent, unset child field)` produces
the same inherited value across all three paths. Use one shared parent fixture and
exercise manual, agent, and transition creation, comparing the resulting field
values.

### Files to change

- `tests/` — a cross-path consistency test (or three assertions sharing one fixture and one expected value).

### Acceptance criteria

- [ ] Given identical parent values and an unset child field, manual, agent, and transition paths all yield the same inherited `priority` and `release` (NFR-5).
- [ ] Test coverage exercises each of the three paths through the shared helper (FR-8).
- [ ] `go vet ./...`, `go build ./...`, and `go test ./... -short` pass with no new failures; `pnpm exec vue-tsc --noEmit` passes.

---

## Cross-links

- Backend plan under test: [[inherit-priority-and-release]] backend plan.
- Frontend payload audit: [[inherit-priority-and-release]] frontend plan.
- Reused override controls / columns: [[artefact-priority-inline-edit]], [[inline-release-display-edit]], [[artefacts-list-release-priority-columns]].
- Requirement / idea lineage: [[inherit-priority-and-release]].
