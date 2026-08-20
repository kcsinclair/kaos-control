---
created: "2026-08-14T15:05:41+10:00"
title: "Frontend Plan — Architectural Artefacts On-Disk Model"
type: plan-frontend
status: done
lineage: architectural-artefacts
parent: lifecycle/requirements/architectural-artefacts-2.md
labels:
    - frontend
    - architecture
    - artefacts
release: KC-Release5
---

# Frontend Plan — Architectural Artefacts On-Disk Model

This plan makes the SPA recognise and correctly render the new architectural artefact types
introduced by [[architectural-artefacts-2]], surface the promoted architecture zone without
lineage-validation noise, and provide a minimal **create/propose ADR** affordance wired to the
[[architectural-artefacts-3-be]] endpoints. It deliberately stays thin: the rich
architecture views (summary rendering, relationship map) belong to
[[architecture-overview-view]] and [[architecture-relationship-map]]; the wizard flow that
*triggers* promotion belongs to [[onboarding-architecture-selection]]. This plan only ensures the
underlying types render as first-class citizens everywhere the existing type vocab is enumerated.

Cross-references:
- [[architectural-artefacts-3-be]] — Backend plan (types, promotion, ADR endpoints).
- [[architectural-artefacts-5-test]] — Test plan.
- [[architecture-overview-view]] / [[architecture-relationship-map]] — downstream consumers of the ADR/summary artefacts surfaced here.

---

## Milestone 1 — Extend the type vocabulary across the SPA

### Description

`architecture`, `tech-stack`, and `adr` must validate and display wherever the frontend
enumerates artefact types: list filters, graph/map node colours, and the type→agent mapping.
Today `typeOptions` in
[web/src/views/project/ArtifactListView.vue](web/src/views/project/ArtifactListView.vue) (line
106) and the palette in
[web/src/components/map/graphConstants.ts](web/src/components/map/graphConstants.ts) omit them
(FR-18).

### Files to change

- **Edit** `web/src/views/project/ArtifactListView.vue`:
  - Add `'architecture'`, `'tech-stack'`, `'adr'` to `typeOptions` (line 106) so they appear in
    the type filter dropdown (rendered at line 295).
- **Edit** `web/src/components/map/graphConstants.ts`:
  - Add colour entries for `architecture`, `tech-stack`, `adr` in **both** the dark palette
    (~line 12) and the light palette (~line 72). Choose hues distinct from the existing
    plan/idea/doc families (e.g. a slate/indigo family for architecture & tech-stack, a
    contrasting accent for `adr`) and keep the light-mode variants darker for contrast, matching
    the existing convention documented inline.
- **Edit** `web/src/composables/useAgentForArtifact.ts`:
  - No agent authors `architecture`/`tech-stack` artefacts directly (they are promoted, not
    generated), so leave those unmapped. Add an `adr` entry only if a single owning agent is
    intended; otherwise leave ADRs human/agent-proposed with no default launch mapping and
    document the omission with a comment.
- **Edit** `web/src/components/artifact/FrontmatterEditor.vue` / `FrontmatterPanel.vue`:
  - The type is rendered read-only (span at FrontmatterEditor.vue:138) — confirm it displays the
    raw new type strings correctly. If any `fmt()`/label map special-cases known types, add the
    three new ones so they render with friendly labels ("ADR", "Architecture", "Tech Stack").

### Acceptance criteria

- `pnpm build` + `pnpm test` (vitest) clean.
- Component test: the ArtifactList type filter includes `architecture`, `tech-stack`, `adr`, and
  filtering by `adr` shows only ADR artefacts.
- A graph node of each new type renders with a defined (non-fallback) colour in both light and
  dark palettes — asserted against `graphConstants`.

---

## Milestone 2 — Render clean-slug architecture artefacts without lineage errors

### Description

Files under `lifecycle/architecture/` carry no `lineage:`/`-N` index and a promoted copy's
`parent:` points at a catalog entry rather than a prior lineage step (FR-19/FR-20). The frontend
must not render "missing lineage" validation warnings or broken lineage breadcrumbs for these
artefacts, and a promoted copy's `parent:` link must resolve to the catalog source.

### Files to change

- **Edit** the artefact detail / breadcrumb components (verify: `FrontmatterPanel.vue`,
  breadcrumb component from the `artifact-breadcrumb-remove-broken-links` lineage):
  - Suppress any "missing lineage" / lineage-index badge when the artefact path begins with
    `lifecycle/architecture/`. Reuse a small `isArchitecturePath(path)` helper (mirror the
    backend `IsArchitecturePath`) in a shared util so the check is defined once.
  - Ensure the `parent` edge renders as a normal link; when it targets a catalog entry
    (`lifecycle/architecture/architectures/…`) it should resolve/navigate correctly rather than
    showing a broken-link state.
- **Edit** the graph/map filtering (if it groups nodes by lineage): architecture-zone nodes with
  empty `lineage` must still render (not be dropped by a `lineage`-required filter).

### Acceptance criteria

- `pnpm test` clean.
- Component test: rendering a promoted `architecture` artefact with empty `lineage` shows **no**
  lineage-warning badge and no broken-breadcrumb; a `parent:` pointing at a catalog path renders
  as a resolvable link.
- Manual smoke: with the backend from [[architectural-artefacts-3-be]] running and a promotion
  performed, the promoted copies and `decisions/adr-0001-*.md` appear in the artefact list and
  graph with correct type colours and no validation warnings.

---

## Milestone 3 — Create / propose ADR affordance

### Description

Provide a minimal UI to create an ADR that calls the backend ADR endpoints
([[architectural-artefacts-3-be]] M4): preview the next number, submit title/slug/body, default
status `draft` (OQ-4), and navigate to the created artefact. This is the human "raise an ADR"
path (FR-13); agent-proposed ADRs use the same backend endpoint via their write scope.

### Files to change

- **Edit** `web/src/api/` (add an `architecture.ts` API module):
  - `promoteArchitecture(project, { architecturePath, techStackPath })` → POST
    `…/architecture/promote` (exposed for [[onboarding-architecture-selection]] to reuse, even
    though this plan doesn't build the wizard screens).
  - `nextAdrNumber(project)` → GET `…/architecture/adrs/next`.
  - `createAdr(project, { slug, title, status, body })` → POST `…/architecture/adrs`.
- **New** `web/src/components/artifact/NewAdrModal.vue` (or extend an existing "new artefact"
  entry point):
  - Fields: title, slug (auto-derived from title, editable), body (CodeMirror), status defaulting
    to `draft`. On open, fetch and display the previewed next number ("This will create
    ADR-0004").
  - On submit, call `createAdr`, then route to the returned artefact path and refresh the index
    view. Surface backend 4xx errors inline.
- **Edit** the artefact list/toolbar (where "New Idea"/"New Defect" buttons live —
  `dashboard-new-idea-defect-buttons` lineage): add a "New ADR" action that opens the modal.

### Acceptance criteria

- `pnpm build` + `pnpm test` clean.
- Component test: opening the modal fetches and shows the previewed next number; submitting calls
  `createAdr` with `status: 'draft'` by default and navigates to the returned path.
- Manual smoke: "New ADR" → fill title/body → submit → new `adr-000N-*.md` appears in the list
  with `type: adr`, `status: draft`, and the number matches the previewed value.

---

## Risk notes

- **Palette exhaustion** — adding three types to a already-dense colour map risks low contrast;
  pick from a different hue family than plans/ideas and verify against the dark and light
  backgrounds. Covered by the M1 colour-defined assertion.
- **Scope creep into overview/relationship views** — resist rendering the summary or ADR graph
  specially here; those are [[architecture-overview-view]] / [[architecture-relationship-map]].
  This plan stops at "types render as normal artefacts + ADR can be created".

## Verification (end-to-end)

1. `pnpm lint` + `pnpm build` clean.
2. `pnpm test` (vitest component tests) clean.
3. Manual smoke against a running backend: promote (via API), confirm promoted copies + ADR-0001
   render with correct colours and no lineage warnings; create a new ADR through the modal and
   confirm it lands with the previewed number and `status: draft`.
