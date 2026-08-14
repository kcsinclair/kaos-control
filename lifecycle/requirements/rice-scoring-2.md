---
title: RICE Scoring for Ideas and Defects
type: requirement
status: blocked
lineage: rice-scoring
priority: high
parent: lifecycle/ideas/rice-scoring.md
labels:
    - feature
    - frontend
    - backend
    - ux
    - artifacts
    - usability
release: KC-Release5
assignees:
    - role: product-owner
      who: agent
---

## Problem

Product owners and analysts have no structured way to compare and rank ideas
and defects. Prioritisation today is implicit and happens by opening artifacts
one at a time, which does not scale during triage or planning sessions and
makes ranking decisions hard to justify or reproduce.

RICE (Reach, Impact, Confidence, Effort) is a well-known prioritisation
framework that produces a single comparable score. The tool should capture the
four RICE components on `idea` and `defect` artifacts, derive the score
automatically, and surface it where items are compared (list views) and edited
(detail view), so ranking is fast, visible, and consistent.

## Goals / Non-goals

### Goals

- Store optional RICE component values (Reach, Impact, Confidence, Effort) in
  the frontmatter of `idea` and `defect` artifacts.
- Derive the RICE score automatically (`Reach × Impact × Confidence / Effort`)
  whenever all four components are present and valid.
- Display the RICE score as a sortable list-view column, showing `N/A` when the
  score cannot be computed.
- Provide inline editing of the four components — from both the list view and
  the artifact detail view — with a live preview of the computed score.
- Require **no migration**: artifacts without RICE fields remain valid and load
  unchanged.

### Non-goals

- No RICE scoring for artifact types other than `idea` and `defect`.
- No prescribed scale or normalisation policy beyond the defaults defined below
  (e.g. no organisation-configurable Impact tables in this iteration).
- No automatic re-ranking, roadmap placement, or workflow-state changes driven
  by the score.
- No historical tracking of score changes beyond ordinary git history of the
  artifact file.

## Detailed Requirements

### Functional — data model

1. Four optional frontmatter fields are defined on `idea` and `defect`
   artifacts: `rice_reach`, `rice_impact`, `rice_confidence`, `rice_effort`.
2. Value constraints:
   - `rice_reach` — number ≥ 0.
   - `rice_impact` — number ≥ 0 (e.g. the conventional 0.25 / 0.5 / 1 / 2 / 3
     scale is permitted but not enforced).
   - `rice_confidence` — number in the range 0–100 (interpreted as a
     percentage).
   - `rice_effort` — number > 0 (person-months or equivalent; zero is invalid
     because it is the divisor).
3. Each field is independently optional; any subset may be present.
4. Fields absent from frontmatter are treated as unset, not zero.

### Functional — score derivation

5. The RICE score is computed as
   `(rice_reach × rice_impact × (rice_confidence / 100)) / rice_effort`.
6. The score is derived, never stored: it is computed on read/render and not
   written back to frontmatter.
7. The score is defined (a number) only when **all four** components are present
   and each satisfies its constraint above; otherwise the score is `N/A`.
8. When `rice_effort` is present but `≤ 0`, the score is `N/A` (not an error and
   not infinity).
9. The rendered score is rounded to a consistent precision (2 decimal places)
   for display; sorting uses the unrounded value.

### Functional — list view

10. List views that show ideas and/or defects include a **RICE** column.
11. The column renders the computed score, or `N/A` when it cannot be computed.
12. The column is sortable ascending and descending. Under either direction all
    `N/A` rows sort together as a group after all scored rows (they never
    interleave with numeric scores).
13. RICE component values can be entered/updated inline from the list row
    without navigating to the detail view.
14. Inline editing shows a live preview of the computed score as component
    values change, before the change is saved.
15. Saving an inline edit persists the changed components to the artifact
    frontmatter via the existing artifact write path and re-indexes so the list
    reflects the new score.

### Functional — detail view

16. The artifact detail view for an `idea` or `defect` exposes a RICE editor
    (inline or dedicated panel) for the four components.
17. The detail editor shows the same live computed-score preview as the list
    view.
18. Saving from the detail view uses the same write + re-index path and yields
    the same stored result as the list view for identical inputs.

### Functional — validation & feedback

19. Non-numeric or out-of-range input is rejected at edit time with a clear,
    field-level message; invalid input is never persisted.
20. Clearing a component field removes it from frontmatter (returns it to the
    unset state) rather than writing `0` or an empty string.

### Non-functional

21. **Backward compatibility** — artifacts with no RICE fields load, index, and
    render without error and show `N/A` in the RICE column.
22. **Consistency** — the same derivation logic is used by the list column, the
    detail view, and any API response, producing identical results for
    identical inputs (single source of truth for the formula).
23. **Persistence fidelity** — writing RICE fields preserves all other
    frontmatter fields, ordering conventions, and the artifact body unchanged.
24. **Indexing** — the derived score (or its `N/A` state) is available to
    list/sort queries without opening each artifact file per request; component
    values participate in the normal incremental re-index on write and on
    external file change.
25. **Performance** — sorting a list by RICE score is O(n log n) over indexed
    values and does not require reparsing artifact bodies.

## Acceptance Criteria

- [ ] An `idea` or `defect` artifact can store any subset of `rice_reach`,
      `rice_impact`, `rice_confidence`, `rice_effort` in frontmatter, and each
      field is optional.
- [ ] An artifact with all four valid components displays a computed score equal
      to `(reach × impact × (confidence/100)) / effort`, rounded to 2 dp.
- [ ] An artifact missing any component, or with `rice_effort ≤ 0`, displays
      `N/A`.
- [ ] Existing artifacts with no RICE fields load and render unchanged and show
      `N/A` (verifies no migration required).
- [ ] A **RICE** column appears in list views containing ideas/defects and is
      sortable ascending and descending, with all `N/A` rows grouped together
      after scored rows in both directions.
- [ ] RICE components can be edited inline from the list view, with a live score
      preview, and the change persists and re-indexes without a page navigation.
- [ ] RICE components can be edited from the artifact detail view with the same
      live preview, and identical inputs produce the same stored result as the
      list-view editor.
- [ ] Invalid input (non-numeric, negative reach/impact, confidence outside
      0–100, effort ≤ 0) is rejected with a field-level message and is not
      persisted.
- [ ] Clearing a component removes the field from frontmatter (unset), and the
      score reverts to `N/A` if fewer than four valid components remain.
- [ ] Saving RICE fields leaves all other frontmatter fields and the artifact
      body byte-for-byte unchanged except for the RICE fields themselves.
- [ ] Score derivation is computed from a single shared implementation used by
      list, detail, and API paths.
- [ ] Relates to [[artefact-priority-inline-edit]] and
      [[artefact-inline-status-change]] for the inline-edit interaction pattern,
      and [[sortable-table-columns]] for the sortable column behaviour.

## Open Questions

- **Impact scale enforcement** — should Impact be a free number, or constrained
  to the conventional RICE tiers (0.25 / 0.5 / 1 / 2 / 3) via a dropdown? This
  requirement currently allows any non-negative number.

> Use the conventional RICE tiers.

- **Confidence units** — is 0–100 (percentage) the right representation, or
  should Confidence be a 0–1 fraction? Chosen 0–100 for editing ergonomics;
  needs product-owner confirmation.

> 0-100 as a percentage

- **Effort units** — person-months is assumed as the conventional unit; should
  the unit be labelled/configurable in the UI?

> Yes, person-months is the convention, allow/encourage fractions and show person-months in the UI.

- **Default values** — the idea mentions "default values or N/A … if all
  blank." This requirement treats missing components strictly as `N/A`. Are
  non-`N/A` defaults (e.g. Confidence = 100) actually desired for partially
  scored items?

> If there is nothing assigned, use the following defaults.
> Reach = 100
> Impact = 0.25
> Confidence = 25%
> Effort = 1

- **Scope of list views** — should the RICE column appear in every list that
  can contain ideas/defects (search, kanban side lists, release panels), or
  only the primary artifact list in this iteration?
