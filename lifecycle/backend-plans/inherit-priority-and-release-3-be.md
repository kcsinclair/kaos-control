---
title: "Backend Plan — Inherit Priority and Release Through Lineage"
type: plan-backend
status: done
lineage: inherit-priority-and-release
parent: lifecycle/requirements/inherit-priority-and-release-2.md
---

# Backend Plan — Inherit Priority and Release Through Lineage

## Overview

When a child artifact is created with a `parent`, default its `priority` and
`release` from that parent unless the caller supplied an explicit value. The
rule must apply identically to all three server-side creation paths:

1. Manual creation — `handleCreateArtifact` (`internal/http/write.go:36`).
2. Agent / LLM generation — `ideachat.Generate` (`internal/ideachat/generate.go:111`,
   hard-codes `priority: "normal"`, never sets `release`) and its caller
   `handleIdeaGenerate` (`internal/http/idea_generate.go:105`).
3. Workflow rejection — `writeRejectionArtifact` (`internal/http/transition.go:252`).

Inheritance is a **snapshot at creation** (FR-3 / Goal 3): the parent's value is
copied once into the child's own frontmatter. No live link, no migration of
existing files, no recursive lineage walk (NFR-3). An unresolvable parent
degrades silently to "no inheritance" and never fails creation (FR-5 / NFR-4).

The data is already available: `Frontmatter` has `Priority` and `Release` string
fields (`internal/artifact/artifact.go:84`, `:90`), and `index.Get` returns an
`ArtifactRow` whose `FM` field carries the parsed parent frontmatter
(`internal/index/index.go:644`, `:700`).

Per the requirement's Resolved Question 4 ("No visual difference is required"),
the FR-10 / Goal-5 editor indicator is **descoped** — there is no backend work to
expose an "inherited vs overridden" state. See the [[inherit-priority-and-release]]
frontend plan for the (minimal) UI implications.

---

## Milestone 1 — Shared inheritance helper (single enforcement point)

### Description

Implement the one helper required by FR-8 so the three paths cannot diverge
(NFR-5). It operates purely on `artifact.Frontmatter` values and applies FR-2,
FR-3 and FR-4:

```go
// applyInheritedFields fills child.Priority / child.Release from parent when the
// child's value is empty. A non-empty child value always wins (FR-4). Empty
// parent values are left as-is (no fabricated default — FR-2/FR-3).
func applyInheritedFields(child *artifact.Frontmatter, parent artifact.Frontmatter)
```

Place it in `internal/artifact/` (e.g. `inherit.go`) so every package can reuse
it without an import cycle, exported as `artifact.ApplyInheritedFields`. It only
mutates `Priority` and `Release`; it must not touch any other field (NFR-2).

### Files to change

- `internal/artifact/inherit.go` (new) — `ApplyInheritedFields(child *Frontmatter, parent Frontmatter)`.

### Acceptance criteria

- [ ] Empty child `priority` + non-empty parent `priority` → child gets parent's value.
- [ ] Empty child `release` + non-empty parent `release` → child gets parent's value.
- [ ] Non-empty child `priority`/`release` is preserved verbatim even when it differs from parent (FR-4).
- [ ] Empty parent `priority`/`release` leaves the child's field empty (no fabricated value).
- [ ] No field other than `Priority`/`Release` is read or written by the helper.
- [ ] Helper is pure (no I/O, no index access) and has no other dependencies.

---

## Milestone 2 — Parent resolution helper (graceful, single lookup)

### Description

Add a helper that resolves a child's `parent` (a lifecycle-relative path) to the
parent's `Frontmatter`, used by the manual and transition paths. It performs at
most one index/disk lookup (NFR-3): try `p.Idx.Get(parentPath)` first; on miss,
fall back to a single `os.ReadFile` + `artifact.Parse` via `sandbox.Resolve`.
Any failure returns `(Frontmatter{}, false)` so callers skip inheritance and log
at debug/info — creation must never fail because the parent is missing
(FR-5 / NFR-4). No recursion up the lineage.

### Files to change

- `internal/http/write.go` — add unexported `resolveParentFrontmatter(p *project.Project, parentPath string) (artifact.Frontmatter, bool)` (shared within the `http` package; also used by `transition.go`).

### Acceptance criteria

- [ ] Resolvable parent (in index) returns its `Frontmatter` and `true` with a single `Idx.Get`.
- [ ] Parent absent from index but present on disk is resolved via one read+parse fallback.
- [ ] Dangling/unresolvable `parent` returns `false`; the condition is logged; no error is propagated.
- [ ] Empty/absent `parent` returns `false` without any lookup (FR-1).
- [ ] No recursive walk of the lineage occurs.

---

## Milestone 3 — Manual creation path (`POST /artifacts`)

### Description

In `handleCreateArtifact`, after the frontmatter is assembled and `Lineage`/
`Created` are stamped, and before `buildMarkdown`/write (`internal/http/write.go:102-106`):
if `req.Frontmatter.Parent != ""`, resolve the parent (Milestone 2) and call
`artifact.ApplyInheritedFields(&req.Frontmatter, parentFM)`. If the parent cannot
be resolved, proceed unchanged (FR-5). NFR-2 round-trip safety holds because we
reuse the existing `buildMarkdown` marshaling.

### Files to change

- `internal/http/write.go` — `handleCreateArtifact` (insert inheritance step before `buildMarkdown` at ~line 105).

### Acceptance criteria

- [ ] `POST /artifacts` with a `parent` and no `priority` → child `priority` equals parent's.
- [ ] `POST /artifacts` with a `parent` and no `release` → child `release` equals parent's.
- [ ] Explicit `priority`/`release` in the request is preserved even if it differs from parent (FR-4 / NFR-1).
- [ ] Parent with no `priority`/`release` → child has none (no fabricated default).
- [ ] Dangling `parent` → creation still succeeds (201) with no inheritance.
- [ ] Output is byte-identical to the same artifact created with the values supplied explicitly (NFR-2).

---

## Milestone 4 — Agent / LLM generation path

### Description

Replace the hard-coded `priority: "normal"` in `ideachat.Generate`
(`internal/ideachat/generate.go:117-124`) with inheritance from the source/parent
when one is present, falling back to `normal` only when there is no parent or the
parent has no priority (FR-6, confirmed by Resolved Question 3). `release` is
inherited when the parent has one.

`Generate` is a pure function without index access, and its result frontmatter is
a `map[string]any`. Resolve the parent in the handler (`handleIdeaGenerate`,
`internal/http/idea_generate.go`) where index access exists, and pass the parent's
priority/release into `GenerateOptions` as new fields
(`SourcePriority`, `SourceRelease`). `Generate` then applies: child gets
`SourcePriority` if set else `"normal"`; `release` set to `SourceRelease` when
non-empty (omit otherwise). The parent is the artifact at `SourcePath`
(already threaded through for the doc flow).

> Implementation note: this preview is later persisted via `POST /artifacts`,
> which also inherits (Milestone 3). Setting the values here ensures the
> generated **preview** reflects the inherited priority/release (FR-6) rather
> than showing `normal`, keeping the two paths consistent (NFR-5).

### Files to change

- `internal/ideachat/generate.go` — add `SourcePriority`, `SourceRelease` to `GenerateOptions`; replace the hard-coded `"priority": "normal"` with inherited-or-`normal` logic; set `release` in the `fm` map when present.
- `internal/http/idea_generate.go` — `handleIdeaGenerate`: when `SourcePath != ""`, resolve the source artifact's `Priority`/`Release` from the index and pass them into `GenerateOptions`.

### Acceptance criteria

- [ ] Agent-generated artifact with a known parent inherits the parent's `priority` instead of `normal`.
- [ ] Agent-generated artifact with a known parent inheriting a `release` carries that `release`.
- [ ] Parentless agent generation still defaults to `priority: normal` and omits `release` (Resolved Q3).
- [ ] When the parent has no `priority`, the agent path falls back to `normal` (FR-6).
- [ ] `GenerateResult.Frontmatter` for a parented generation shows the inherited values in the preview response.

---

## Milestone 5 — Workflow rejection path

### Description

In `writeRejectionArtifact` (`internal/http/transition.go:252`), after building the
`fm` that already copies `title`, `type`, `lineage`, `parent`
(`internal/http/transition.go:262-268`), apply inheritance from the source
artifact's frontmatter (`row.FM`, already in hand). Since the source is the
`row` being transitioned, no extra lookup is needed — call
`artifact.ApplyInheritedFields(&fm, row.FM)` (FR-7). This also covers defects
created with a `parent` going through the same helper (Resolved Question 2:
defects inherit from the artifact under test).

### Files to change

- `internal/http/transition.go` — `writeRejectionArtifact`: apply `ApplyInheritedFields(&fm, row.FM)` before `buildMarkdown`.

### Acceptance criteria

- [ ] A rejection artifact inherits the source's `priority` and `release` in addition to the existing copied fields.
- [ ] No additional index/disk lookup is introduced (source FM is already loaded as `row.FM`).
- [ ] When the source has no `priority`/`release`, the rejection artifact has none.

---

## Milestone 6 — Override isolation & validation (verification, no new mutation)

### Description

Confirm FR-9 and FR-11 already hold and add no regression. The inline controls
`handlePatchPriority` / `handlePatchRelease` (`internal/http/write.go:455`, `:545`)
write only the target file and read no other artifact — that satisfies FR-9
(override isolation). FR-11: inherited `release` is copied as-is and is **not**
re-validated against the project release list at creation (validation lives only
in `handlePatchRelease`, the override path). Verify the create paths do not call
`release.Store.GetByName` on inherited values, so an inherited release that
predates a release-list change still round-trips.

### Files to change

- None expected (verification milestone). Add a code comment at the create-path inheritance site noting that inherited `release` is intentionally not re-validated (FR-11).

### Acceptance criteria

- [ ] Editing a child's `priority`/`release` after creation changes only that file; parent and siblings are untouched on disk (FR-9).
- [ ] Inherited `release` is written without a release-list validation call at creation time (FR-11).
- [ ] Override via `PATCH .../release` remains subject to the existing 422 validation.
- [ ] `go vet ./...` and `go build ./...` pass with no new errors.

---

## Cross-links

- Requirement / idea lineage: [[inherit-priority-and-release]]
- Inline override controls reused unchanged: [[artefact-priority-inline-edit]], [[inline-release-display-edit]]
- Columns consuming the resulting metadata: [[artefacts-list-release-priority-columns]]
- Frontend implications (indicator descoped): see the [[inherit-priority-and-release]] frontend plan.
- Test coverage: see the [[inherit-priority-and-release]] test plan.
