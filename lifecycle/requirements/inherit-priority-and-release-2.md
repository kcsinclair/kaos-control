---
title: Inherit Priority and Release Through Lineage
type: requirement
status: planning
lineage: inherit-priority-and-release
created: "2026-06-26T00:00:00+10:00"
priority: high
parent: lifecycle/ideas/inherit-priority-and-release.md
labels:
    - feature
    - workflow
    - artefacts
release: KC-Release4
assignees:
    - role: product-owner
      who: agent
---

## Problem

When a child artifact is created from a parent in a lineage chain (idea → requirement → plans → tests → defects), its `priority` and `release` frontmatter fields are not carried forward. Today the value of these fields depends entirely on the creation path:

- **`POST /api/p/:project/artifacts`** (`internal/http/write.go:35`) writes exactly the frontmatter the caller supplies; nothing is copied from the parent.
- **Agent / LLM generation** (`internal/ideachat/generate.go:111`) hard-codes `priority: "normal"` and never sets `release`, even when a source/parent path is known.
- **Rejection artifacts** (`internal/http/transition.go:252`) copy `title`, `type`, `lineage`, and `parent` from the source but leave `priority` and `release` empty.

The result is metadata drift across a single lineage: a `priority: high`, `release: KC-Release4` idea spawns requirements and plans that default to blank or `normal` with no release. This makes release-planning and prioritisation views (roadmap, release pages, priority/release columns) unreliable, because artifacts belonging to one idea scatter across milestones and priorities instead of clustering under the idea's chosen values. Correcting it requires tedious manual editing of every descendant.

## Goals / Non-goals

### Goals

1. **Inherit at creation time.** When a new artifact is created with a `parent`, default its `priority` and `release` from that parent artifact unless the caller supplies an explicit value for the field.
2. **Cover every creation path.** Apply the same inheritance rule to manual creation (`POST /artifacts`), agent/LLM-generated artifacts, and rejection artifacts produced during workflow transitions.
3. **Snapshot, not live link.** Inheritance copies the parent's value into the child's own frontmatter once, at creation. Later changes to the parent do not retroactively alter existing children.
4. **Per-artifact override.** A user (or agent) can set a `priority` or `release` on a child that differs from the parent. Overriding one artifact must not affect its siblings, its parent, or its descendants already created.
5. **Visible inheritance in the editor.** The artifact editor UI indicates, for `priority` and `release`, whether the current value matches the parent's value (i.e. is "inherited") versus has been overridden.

### Non-goals

- Inheriting fields other than `priority` and `release` (e.g. `labels`, `assignees`, `sprint`). Out of scope for this requirement.
- Retroactively back-filling `priority`/`release` onto artifacts that already exist on disk (no migration). Inheritance applies only to newly created artifacts.
- A "live" propagation model where editing a parent cascades to all descendants. Explicitly excluded by Goal 3.
- Cascading an override down to descendants of the overridden artifact at override time (descendants are only affected when they are themselves created).
- Originating artifacts (no `parent`) — they have no source to inherit from and are unchanged.

## Detailed Requirements

### Functional

**FR-1: Inheritance source.** The inheritance source for a new artifact is the artifact referenced by the new artifact's `parent` frontmatter field (a lifecycle-relative path). If `parent` is empty/absent, no inheritance occurs.

**FR-2: Field selection — priority.** When creating an artifact that has a `parent`, if the incoming `priority` is empty/unset, set the child's `priority` to the parent's `priority`. If the parent also has no `priority`, leave it unset (do not fabricate a value).

**FR-3: Field selection — release.** When creating an artifact that has a `parent`, if the incoming `release` is empty/unset, set the child's `release` to the parent's `release`. If the parent has no `release`, leave it unset.

**FR-4: Explicit value wins.** If the caller supplies a non-empty `priority` and/or `release` in the create request, that value is preserved verbatim and inheritance does **not** overwrite it — even if it differs from the parent.

**FR-5: Manual creation path.** `handleCreateArtifact` (`internal/http/write.go`) applies FR-2…FR-4 after assembling the frontmatter and before `buildMarkdown`/write. The parent artifact is resolved from the index/disk; if the parent cannot be resolved, creation proceeds without inheritance (it must not fail the request) and the condition is logged.

**FR-6: Agent / LLM generation path.** The generation path (`internal/ideachat/generate.go`, and any agent-driven artifact creation that sets a `parent`/source) must apply the same inheritance. Specifically, the current hard-coded `priority: "normal"` must be replaced by inheritance from the source/parent when a parent is present, falling back to `normal` only when no parent exists or the parent has no priority. `release` must be inherited when the parent has one.

**FR-7: Workflow transition / rejection path.** `writeRejectionArtifact` (`internal/http/transition.go`) and any other transition that produces a child artifact must inherit `priority` and `release` from the source artifact in addition to the fields already copied (`title`, `type`, `lineage`, `parent`).

**FR-8: Single enforcement point.** Inheritance logic must be implemented once (e.g. a helper such as `applyInheritedFields(child, parent)`) and reused by every creation path, so the three paths above cannot diverge. The helper operates on `artifact.Frontmatter` (`internal/artifact/artifact.go`) values.

**FR-9: Override persistence.** When a child's `priority` or `release` is changed after creation (via full edit, or the existing inline priority/release controls — see [[artefact-priority-inline-edit]], [[inline-release-display-edit]]), the new value is written to that artifact only. No other artifact's frontmatter is read or written as a side effect.

**FR-10: Inherited-vs-overridden indication (UI).** In the artifact editor, for the `priority` and `release` fields, the UI must indicate when the child's current value equals the parent's current value ("inherited") versus differs ("overridden" / set on this artifact). When the artifact has no parent, no indicator is shown. The indicator is computed by comparing the child's value to the parent's current value at view time (the comparison is display-only and does not mutate stored frontmatter).

**FR-11: Validation of inherited release.** An inherited `release` value is copied as-is from the parent and is not re-validated against the project's release list (the parent already holds it). Overrides applied through release controls remain subject to existing validation (e.g. the 422 behaviour in [[inline-release-display-edit]]).

### Non-functional

**NFR-1: No regression to explicit values.** Existing callers that supply `priority`/`release` explicitly must observe identical behaviour to today (FR-4). Existing artifacts on disk are untouched.

**NFR-2: Round-trip safety.** Inheritance must not drop or reorder unrelated frontmatter fields. Setting `priority`/`release` uses the existing frontmatter marshaling so that artifacts written with inheritance are byte-comparable to the same artifact written with those values supplied explicitly.

**NFR-3: Performance.** Resolving the parent for inheritance must add at most one index/disk lookup per artifact creation and must not perform recursive walks of the lineage. Creation latency must remain dominated by the existing write + re-index cost.

**NFR-4: Failure isolation.** Inability to resolve or read a parent must degrade gracefully to "no inheritance" (FR-5) and never block artifact creation.

**NFR-5: Consistency across paths.** Given the same parent and an unset child field, all three creation paths (manual, agent, transition) must produce the same inherited value (guaranteed structurally by FR-8).

## Acceptance Criteria

- [ ] Creating an artifact via `POST /artifacts` with a `parent` and no `priority` yields a child whose `priority` equals the parent's `priority`.
- [ ] Creating an artifact via `POST /artifacts` with a `parent` and no `release` yields a child whose `release` equals the parent's `release`.
- [ ] Supplying an explicit `priority` and/or `release` in the create request preserves the supplied value even when it differs from the parent (inheritance does not overwrite).
- [ ] When the parent has no `priority`/`release`, the created child also has none (no fabricated default) — except the agent/LLM path, which falls back to `priority: normal` only when no parent value exists.
- [ ] Agent/LLM-generated artifacts created with a known parent/source inherit `priority` and `release` instead of hard-coding `priority: normal` and omitting `release`.
- [ ] A rejection artifact produced during a transition inherits the source artifact's `priority` and `release`.
- [ ] Editing a child's `priority` or `release` after creation changes only that artifact's file; the parent and sibling files are unchanged on disk.
- [ ] The editor shows an "inherited" indicator for `priority`/`release` when the child's value matches the parent's, and an "overridden" (or absent-indicator) state when it differs; no indicator appears for artifacts with no parent.
- [ ] Inheritance adds no recursive lineage walk; an unresolvable parent results in no inheritance and does not fail creation (verified by a test with a dangling `parent`).
- [ ] A single shared helper applies inheritance for all three creation paths (verified by test coverage exercising each path).
- [ ] Existing artifacts on disk are not modified by deploying this change (no migration).
- [ ] `go vet ./...` and `go build ./...` pass with no new errors; `pnpm exec vue-tsc --noEmit` passes with no new errors.
- [ ] Related artifacts: [[inherit-priority-and-release]], [[artefact-priority-inline-edit]], [[inline-release-display-edit]], [[artefacts-list-release-priority-columns]]

## Resolved Questions

1. **Inherited-state indicator semantics.** FR-10 computes "inherited vs overridden" by live-comparing the child's value to the parent's *current* value. This means an override that happens to equal the parent reads as "inherited," and a parent edit can flip a previously-inherited child to "overridden" in the display. Is value-comparison acceptable, or should we persist an explicit marker (e.g. `inherited_fields: [priority, release]`) at creation to track provenance precisely? A persisted marker adds a frontmatter field and round-trip considerations (the `Frontmatter` struct is a strict field list).

> Inherit only, typically this will be on creation of the new artefacts, so when the requirements-analyst creates the requirements, they should include the priority and release of the idea.  The system does not need to change existing values.

2. **Defects.** Defects raised by QA (`lifecycle/defects/`) carry a `parent`. Should they inherit `priority`/`release` from the artifact under test, or are defect priority/release independently triaged? Current assumption: treat them like any other child (inherit), but confirm.

> they should inherit `priority`/`release` from the artifact under test

3. **Agent fallback priority.** FR-6 keeps `priority: normal` as a fallback only when there is no parent value. Confirm `normal` remains the desired default for genuinely parentless agent-generated artifacts (e.g. ideas from brain-dump with no source).

> Yes, normal is the default.

4. **Visual treatment.** What exactly should the inherited indicator look like (muted "inherited from parent" hint text, a badge, a tooltip on the field)? Should it offer a one-click "reset to inherited" affordance? Defer to product-owner / UX preference.

> No visual difference is required.
