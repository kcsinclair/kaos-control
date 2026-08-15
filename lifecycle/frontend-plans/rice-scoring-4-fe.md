---
title: "Frontend Plan — RICE Scoring Column, Inline & Detail Editing"
type: plan-frontend
status: done
lineage: rice-scoring
parent: lifecycle/requirements/rice-scoring-2.md
---

# Frontend Plan — RICE Scoring Column, Inline & Detail Editing

## Overview

Surfaces the RICE score as a sortable column in the primary artifact list and
provides inline editing of the four components from both the list row and the
artifact detail view, each with a **live computed-score preview** that mirrors the
canonical backend formula in [[rice-scoring]] (backend plan). Saves go through a
new `patchRice` call that reuses the backend `PATCH .../rice` path, then optimistic
update + revert-on-error exactly as `PriorityDropdown.vue` does today.

Interpretation of the requirement's resolved "default values" answer: the defaults
(Reach 100, Impact 0.25, Confidence 25, Effort 1) are **editor pre-fill only** —
when the RICE editor is opened on an item that has *no* RICE fields set, the four
inputs seed with these values as a starting point. They are **not** applied to
unscored rows in the list, which continue to show `N/A` until saved (requirement
§21 is explicit). The `impact` input is constrained to the conventional RICE tiers
`0.25 / 0.5 / 1 / 2 / 3` via a dropdown (resolved question); `effort` is labelled
"person-months" and accepts fractions; `confidence` is a 0–100 percentage.

Scope: **primary artifact list only** (`ArtifactListView.vue`) this iteration —
kanban side-lists, search, and release panels are deliberately excluded (resolved
question; a separate enhancement will extend them).

Builds on [[sortable-table-columns]] (the `useSortableTable` + `SortHeader`
pattern) and reuses the interaction shape of [[artefact-priority-inline-edit]] and
[[artefact-inline-status-change]].

---

## Milestone 1 — Types + API client + shared formula mirror

### Description

Extend the artifact types with the four optional components and the derived
`rice_score`, add `patchRice`, and add the TS derivation mirror used by every live
preview. The mirror must be byte-for-byte equivalent in output to the Go
`artifact.RiceScore` (verified by the [[rice-scoring]] test plan).

- `web/src/lib/rice.ts` — `riceScore(c): number | null` returning `null` for
  `N/A` (missing component, or `effort ≤ 0`, or any out-of-range value), the raw
  score otherwise; `formatRice(v): string` → `v.toFixed(2)` or `'N/A'`;
  `validateRiceComponent(field, value): string | null` returning a field-level
  message or `null`; `RICE_DEFAULTS` and `IMPACT_TIERS = [0.25,0.5,1,2,3]`.
- `web/src/types/api.ts` — add optional `rice_reach/rice_impact/rice_confidence/
  rice_effort` to `ArtifactFrontmatter` and `rice_score?: number` to
  `ArtifactRow`.
- `web/src/api/artifacts.ts` — `patchRice(project, path, components)` →
  `PATCH .../rice`, each component `number | null` (null clears).

### Acceptance criteria

- [ ] `riceScore` returns `null` for every `N/A` case in requirement §7–§8 and the
      exact formula value otherwise (unrounded); `formatRice` rounds to 2 dp.
- [ ] `patchRice` sends only the keys provided; a `null` value clears a component.
- [ ] Type additions are all optional — existing artifacts type-check unchanged.

---

## Milestone 2 — RICE column in the primary artifact list

### Description

Add a sortable **RICE** column to `ArtifactListView.vue`. Register a `rice` entry
in the existing `useSortableTable` map as `type: 'number'` with
`getValue: (row) => row.rice_score ?? riceScore(row.frontmatter) ?? null`. Because
`useSortableTable` pins `null`/empty to the end regardless of direction
(`useSortableTable.ts:103-108`), `N/A` rows automatically group together after
scored rows in both directions — satisfying requirement §12 with no extra logic.
Render the cell as `formatRice(...)`; non-`idea`/`defect` rows render `N/A`.

### Files to change

- `web/src/views/project/ArtifactListView.vue` — `useSortableTable` column map;
  `<SortHeader label="RICE" column="rice" …>` in `<thead>`; the `<td>` cell.

### Acceptance criteria

- [ ] A **RICE** column appears with a computed score or `N/A` per row
      (requirement §10, §11).
- [ ] Toggling the header sorts ascending then descending; in both directions all
      `N/A` rows are grouped after all scored rows (requirement §12).
- [ ] Sorting reads the indexed `rice_score` (falling back to the local mirror)
      and never triggers an extra artifact fetch (requirement §25).

---

## Milestone 3 — Inline RICE editor (shared component, list + detail)

### Description

Build one reusable editor used by both the list cell and the detail view, modelled
on `PriorityDropdown.vue` (trigger badge → popover; outside-click/Escape close;
optimistic update with revert on error; `changed`/`error` emits).

- `web/src/components/artifact/RiceEditor.vue` — props
  `{ project, path, type, frontmatter, readonly }`; a popover with four fields
  (Reach number, Impact tier dropdown, Confidence 0–100, Effort person-months
  fractional), a **live preview** of `formatRice(riceScore(localComponents))` that
  updates on every keystroke *before* save (requirement §14, §17), per-field
  validation messages via `validateRiceComponent` (requirement §19), a "Clear"
  affordance per field that removes it (sends `null`), and a Save that calls
  `patchRice`. On open with no components set, seed inputs from `RICE_DEFAULTS`
  (editor pre-fill, see Overview). Save is optimistic; on failure revert and emit
  `error`.

### Files to change

- `web/src/components/artifact/RiceEditor.vue` — new component.
- `web/src/views/project/ArtifactListView.vue` — mount `RiceEditor` in the RICE
  cell for `idea`/`defect` rows (read-only badge otherwise); on `changed`, update
  the row so the score/sort reflect the new value without navigation
  (requirement §13, §15).

### Acceptance criteria

- [ ] Editing components shows a live preview that recomputes on each change before
      saving (requirement §14).
- [ ] Invalid input shows a field-level message and Save is blocked; nothing is
      persisted (requirement §19).
- [ ] Clearing a field sends `null`; the component is removed and the score reverts
      to `N/A` when fewer than four valid components remain (requirement §20).
- [ ] A successful save persists via `patchRice`, updates the row in place, and the
      list re-sorts without a page navigation (requirement §13, §15).
- [ ] Read-only (no permission / foreign lock → non-2xx) renders a static badge,
      matching `PriorityDropdown` behaviour.

---

## Milestone 4 — Detail-view RICE editor

### Description

Expose the same `RiceEditor` in the artifact detail view for `idea`/`defect`
types, so detail edits use the identical live preview and the identical save path,
guaranteeing identical stored results for identical inputs (requirement §16–§18).

### Files to change

- `web/src/components/artifact/FrontmatterPanel.vue` (and/or
  `FrontmatterEditor.vue`) — render `RiceEditor` as a RICE panel for `idea`/
  `defect` artifacts, wired to the same `patchRice` path.

### Acceptance criteria

- [ ] The detail view shows the RICE editor for `idea`/`defect` only, with the same
      live preview as the list (requirement §16, §17).
- [ ] Saving from detail produces the same stored file as the list editor for
      identical inputs (requirement §18) — asserted by the [[rice-scoring]] test
      plan.
- [ ] A live WebSocket `artifact.indexed` update refreshes the displayed score
      (parity with existing `PriorityDropdown` prop-watch behaviour).
