---
title: Architecture Overview View — Frontend Plan
type: plan-frontend
status: in-development
lineage: architecture-overview-view
parent: lifecycle/requirements/architecture-overview-view-2.md
created: "2026-08-20T09:05:00Z"
---

# Architecture Overview View — Frontend Plan

The primary surface for this requirement. Companion to the backend plan
[[architecture-overview-view]] (be) and test plan [[architecture-overview-view]]
(test). Builds a read-mostly Vue view that assembles the chosen architecture on
one screen, becomes the Architecture section's default when a selection exists,
and takes ownership of the architecture zone from the list/board. It reuses the
existing Vue 3 / Pinia / Vue Router stack, `markdown-it` rendering, and the
existing artifact/REST API — **no new runtime dependency** (NFR-1).

## Conformance to recorded architecture

- Pure frontend consumer inside the **Go + Vue** modular monolith (NFR-6). No new
  frontend dependency, no new store engine — a single Pinia store + composable
  that call the existing REST API and re-fetch on the existing WS events.
- Read-mostly (NFR-2): the only mutating actions are navigational entry points
  into **existing** flows (open artifact, re-run wizard, raise ADR via the
  existing `NewAdrModal.vue`). The view writes no artifact content.
- Freshness reuses the existing `useWebSocket` `artifact.indexed` / `file.changed`
  path exactly as `useArchitectureMap` does (FR-12) — no new realtime mechanism
  (NFR-6).

## Milestone F1 — Overview data layer (store + composable)

**Description.** Add a composable that loads the assembled overview model from
`GET /api/p/{project}/architecture/overview` (backend M-B2) and keeps it fresh on
`artifact.indexed` / `file.changed`, mirroring `useArchitectureMap`. Expose
`loading`, `error`, and the classified model (chosen architecture/stack refs,
summary ref-or-null, `standards[]`, `adrs[]` newest-first, `archive[]`,
`catalog[]`, and `hasChosenArchitecture`). Panel **bodies** are fetched lazily
via the existing artifacts store / `GET /artifacts/*path` so we reuse
`MarkdownPreview.vue` for rendering (NFR-1).

**Files to change.**
- `web/src/composables/useArchitectureOverview.ts` (new) — modelled on
  `useArchitectureMap.ts`; `reload()` on mount + on the two WS events.
- `web/src/api/architecture.ts` (new or extend existing api module) —
  `getArchitectureOverview(project)`.
- `web/src/types/api.ts` — add `ArchitectureOverview` / `OverviewItem` / a
  `CatalogRole` union matching the backend `catalog_role` values.

**Acceptance criteria.**
- On mount the composable fetches the overview and populates the model; an API
  error sets `error` and does not throw.
- A received `artifact.indexed` or `file.changed` WS event triggers exactly one
  `reload()` (FR-12), verified by unit test with a mocked socket.
- `adrs` preserve the backend's newest-first order without client re-sorting
  (FR-7).

## Milestone F2 — Overview view + panels (FR-2…FR-7)

**Description.** Build `ArchitectureOverviewView.vue` composed of independent
panels, each degrading on its own (FR-10/NFR-5):
- **Chosen architecture panel** (FR-2): title, summary, rendered body
  (components/interactions) via `MarkdownPreview`, click-through to the artifact.
- **Tech-stack panel + mapping** (FR-3): renders the promoted stack and its
  architecture↔component mapping. Per OQ-3 the mapping is the **hard references
  already present** in the `architectures/*` and `tech-stacks/*` documents
  (`related_to` / body wiki-links) — resolve and render those links; no new
  mapping field.
- **Wizard Q&A rationale panel** (FR-4): render the relevant
  `architecture-summary.md` sections **as-is with links** (OQ-1 — no structured
  heading parsing).
- **Architecture-breaking requirements panel** (FR-5): render the summary's
  breaking-requirements section as-is, preserving click-through where the summary
  links to an ADR/requirement.
- **Standards panel** (FR-6): list `standards/*`, each click-through to its
  artifact.
- **ADRs panel** (FR-7): list ADRs newest-first showing title, status, date, each
  click-through; status/decision signalling is not colour-only (NFR-4).

Layout leaves room for a future side-by-side auto-diagram (FR-11) — panels laid
out so a second column can be added later without restructuring (do not implement
the diagram).

**Files to change.**
- `web/src/views/project/ArchitectureOverviewView.vue` (new).
- `web/src/components/architecture/overview/*` (new) — one component per panel
  (`ChosenArchitecturePanel.vue`, `TechStackPanel.vue`, `RationalePanel.vue`,
  `BreakingRequirementsPanel.vue`, `StandardsPanel.vue`, `AdrListPanel.vue`).
- Reuse `web/src/components/artifact/MarkdownPreview.vue` for body rendering.

**Acceptance criteria.**
- Each of FR-2…FR-7 renders from the model with a working click-through to the
  underlying artifact route (`artifact-editor`).
- Tech-stack mapping renders the hard `related_to`/wiki-link references from the
  promoted artifacts (OQ-3); summary panels render sections as-is with links
  intact (OQ-1).
- ADR list is newest-first with title + status + date; no meaning is conveyed by
  colour alone (NFR-4).
- Layout reserves space for a future auto-diagram column without shipping one
  (FR-11).

## Milestone F3 — Empty / partial states (FR-10, NFR-5)

**Description.** Every panel and the view as a whole degrade gracefully. With **no
chosen architecture**: the view shows an empty state inviting the wizard and
linking to the relationship map — it does not error. With a chosen architecture
but **missing summary / empty `standards/` / no ADRs**: each affected panel shows
its own empty/absent state; the rest of the view still renders.

**Files to change.**
- `ArchitectureOverviewView.vue` + each panel component — per-panel empty slots;
  a top-level empty state component (reuse `web/src/components/common/*` empty-state
  pattern if one exists).

**Acceptance criteria.**
- No promoted architecture → empty state with "run the wizard" + relationship-map
  links; no console error, no failed render.
- Missing summary, empty `standards/`, or zero ADRs → only the affected panel
  shows an absent state; sibling panels are unaffected (FR-10/NFR-5).

## Milestone F4 — Section default + routing + navigation actions (FR-1, FR-8)

**Description.** Register the overview route and make it the Architecture section
default **when a chosen architecture exists**; otherwise the section defaults to
the relationship map (converging with [[architecture-relationship-map]] FR-1) and
the overview route, if visited, shows the FR-10 empty state. The overview has its
own URL in both cases. From the overview, one-click actions (FR-8): open the
relationship map, open/re-run the Architecture Wizard, and **raise a new ADR** via
the existing `NewAdrModal.vue`. Update the sidebar so the "Architecture" item
lands on the section default.

**Files to change.**
- `web/src/router/index.ts` — add `architecture/overview` route
  (`ArchitectureOverviewView.vue`); change the `architecture` landing redirect
  (currently hard-coded to `architecture/map`, lines ~65–73) to resolve to the
  overview when a chosen architecture exists, else the map. The section-default
  decision reads the overview model / a cheap "has promoted architecture" signal
  (reuse the store; avoid a blocking redirect — a guard or a lightweight landing
  component that redirects after the model resolves).
- `web/src/components/layout/AppSidebar.vue` — Architecture nav item points to
  the section landing (`/p/{project}/architecture`); overview is reachable and
  the map/wizard stay reachable.
- `ArchitectureOverviewView.vue` — action bar wiring relationship-map link,
  wizard link, and `NewAdrModal.vue`.

**Acceptance criteria.**
- With a chosen architecture, navigating to the Architecture section lands on the
  overview on its own route; with none, it lands on the relationship map and the
  overview route shows the empty state (FR-1, FR-10).
- The relationship map and the wizard are each reachable in one click from the
  overview; "raise a new ADR" opens `NewAdrModal.vue` (FR-8).
- Direct navigation to the overview URL always works (own route in both states).

## Milestone F5 — Architecture-zone ownership + demote list/board toggle (FR-9, FR-9a)

**Description.** The overview owns the zone; the list/board stop showing it by
default. Broaden the discriminator from today's `isCatalogMaterial()` (which
excludes only `catalog`-labelled + `archive/` and **keeps** the chosen
architecture, ADRs, and standards visible) to a **catalog-role / zone**
predicate that excludes the **whole** `lifecycle/architecture/` zone by default —
chosen architecture, ADRs, standards, summary, candidates, and archive alike
(FR-9). Keep the discriminator role-based, not `type`-based. The interim "Show
catalog" toggle is **demoted** to an off-by-default "show architecture inline"
escape hatch keyed on the new predicate (OQ-4 — kept, not removed). Add an
**archive strip** in the overview showing 10 items by default, **collapsed on
open** (OQ-5).

**Files to change.**
- `web/src/types/api.ts` — rename/extend `isCatalogMaterial` → e.g.
  `isArchitectureZone(row)` (path under `lifecycle/architecture/` OR `catalog`
  label OR `archive/`), keeping a compat export if referenced widely; update the
  doc comment.
- `web/src/views/project/ArtifactListView.vue` (~line 49–50, 300) and
  `web/src/composables/useKanbanBoard.ts` (~line 90–91) — filter on the new
  predicate; relabel the toggle "show architecture inline", default off.
- `web/src/components/architecture/overview/ArchiveStrip.vue` (new) — collapsed,
  10-items-default provenance strip fed by `overview.archive`.
- `web/src/types/__tests__/isCatalogMaterial.spec.ts` — update/rename to cover the
  broadened predicate (chosen architecture, ADRs, standards now excluded).

**Acceptance criteria.**
- By default the list and board show **no** `lifecycle/architecture/` artifact —
  candidates, chosen architecture, ADRs, standards, summary, and archive are all
  excluded (FR-9a); toggling "show architecture inline" on reveals them.
- The discriminator is catalog-role/zone-based, not artifact `type`-based (FR-9).
- The overview's archive strip shows ≤10 items and is collapsed on first open
  (OQ-5); expanding reveals the rest.
- Existing `isCatalogMaterial` spec is updated and green under the new behaviour.

## Out of scope (frontend)

- No in-app editing of architecture content (NFR-2); editing stays in the
  existing artifact editor.
- No wizard question flow/scoring ([[onboarding-architecture-selection]]) and no
  catalog relationship graph ([[architecture-relationship-map]]) — both are
  linked, not reimplemented.
- No auto-diagram rendering — layout accommodates it only (FR-11,
  [[architecture-auto-diagram]]).
