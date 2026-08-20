---
title: Architecture Overview View — Visualise the Chosen Architecture
type: requirement
status: blocked
lineage: architecture-overview-view
created: "2026-08-20T00:00:00Z"
parent: lifecycle/ideas/architecture-overview-view.md
labels:
    - architecture
    - visualization
    - frontend
    - ux
    - feature
release: KC-Release5
assignees:
    - role: product-owner
      who: agent
---

# Architecture Overview View — Visualise the Chosen Architecture

## Problem

Once a project has run the Architecture Wizard
([[onboarding-architecture-selection]]) and promoted its choices into
`lifecycle/architecture/` ([[architectural-artefacts]]), the resulting picture
is spread across many files: the promoted architecture and tech-stack artifacts,
`architecture-summary.md` (wizard Q&A rationale plus architecture-breaking
requirements), `decisions/` (ADRs), and `standards/` (non-functional baselines).
There is no single surface that assembles these into a coherent, reviewable
picture of *the architecture this project has chosen and why*.

The relationship map ([[architecture-relationship-map]]) is a **browse** surface
for the catalog of options; it does not show the project's own selected
architecture, its rationale, its decisions, or its standards. Meanwhile the
generic lifecycle surfaces (list, board) are about **flow** (idea → requirement
→ plan → dev → QA → release), and everything under `lifecycle/architecture/` is
**standing reference**, not flow — so it does not belong on those surfaces and
currently clutters them behind an interim "Show catalog" toggle.

This requirement defines the **Architecture Overview** view: the read-mostly home
for a project's chosen architecture, and the proper owner of the architecture
zone in the navigation.

## Goals / Non-goals

### Goals

- Provide a single, read-mostly **overview of the chosen architecture** that
  assembles, on one screen: the promoted architecture, the promoted tech stack
  and its mapping onto the architecture's components, the wizard Q&A rationale,
  the architecture-breaking requirements and their mapping, the standards, and
  the ADRs (newest first, each click-through to its artifact).
- Make the overview the **default destination** of the Architecture section
  *once a project has a chosen architecture*, converging with
  [[architecture-relationship-map]] FR-1.
- Keep the relationship map and the Architecture Wizard **one click away** from
  the overview, and offer entry points to **re-run the wizard** and to **raise a
  new ADR**.
- Have the view **own the architecture zone**: surface the catalog (candidates),
  the chosen architecture + stack, the summary, ADRs, standards, and the archive
  of superseded choices — so this material no longer needs to appear on the
  list/board.
- Derive everything **from the artifacts on disk** — they remain the single
  source of truth; the view never becomes an authoring surface for their content.
- **Degrade gracefully** when parts of the model are absent (no summary yet,
  empty `standards/`, no ADRs, or no chosen architecture at all).

### Non-goals

- **No in-app editing** of architecture content. Editing happens in the
  artifacts (via the existing editor); this view is read-mostly and only offers
  navigational entry points (open artifact, re-run wizard, raise ADR).
- **Not** the wizard question flow or scoring — that is
  [[onboarding-architecture-selection]].
- **Not** the catalog relationship graph — that is
  [[architecture-relationship-map]]; this view links to it but does not replace
  it.
- **Not** the on-disk artefact model (promotion, summary format, ADR numbering,
  standards seeding) — that is [[architectural-artefacts]]; this view is a pure
  consumer of that model.
- **No auto-generated diagram of the *actual* current system** derived from the
  codebase — deferred to [[architecture-auto-diagram]] as a later enhancement
  (see FR-11).
- **No** deletion of any `lifecycle/architecture/` material by this view.

## Detailed Requirements

### Functional

**FR-1 — Overview as section default when a selection exists.** The Architecture
left-menu section routes to the overview view as its **default destination when
the project has a chosen (promoted) architecture**. When no selection exists, the
section defaults to the relationship map ([[architecture-relationship-map]] FR-1)
and the overview route, if visited, shows an empty state (FR-10). The overview
has its own route/URL in both cases.

**FR-2 — Chosen architecture panel.** The view renders the **promoted
architecture artifact** (the `type: architecture` file at the
`lifecycle/architecture/` root): its title, summary, and body (components /
interactions), with a click-through to open the underlying artifact.

**FR-3 — Tech-stack panel and mapping.** The view renders the **promoted
tech-stack artifact** (the `type: tech-stack` file at the root) and presents how
the stack maps onto the architecture's components (sourced from the stack
artifact's stack-profile / body content). It offers a click-through to the stack
artifact.

**FR-4 — Wizard Q&A rationale.** The view surfaces the **questions and answers /
selection rationale**, sourced from `architecture-summary.md`, as the trail that
led to this selection.

**FR-5 — Architecture-breaking requirements.** The view lists the **critical /
architecture-breaking requirements** and, for each, how it maps to (is satisfied
by) the chosen architecture and stack — sourced from `architecture-summary.md`.
Where a requirement links to an ADR or a requirement artifact, the link is
click-through.

**FR-6 — Standards / non-functional requirements.** The view lists the contents
of `lifecycle/architecture/standards/` (the design elements and non-functional
baselines the agents follow), each item click-through to its artifact.

**FR-7 — Architecture Decision Records.** The view lists the ADRs from
`lifecycle/architecture/decisions/` (`type: adr`), **newest first**, each showing
at least title, status, and date, and each click-through to the ADR artifact.

**FR-8 — Navigation & actions.** From the overview the user can, in one click:
open the relationship map, open the Architecture Wizard (re-run selection), and
initiate raising a new ADR. "Raise a new ADR" and "re-run wizard" are entry
points into existing flows — this view does not itself author ADR content or run
the wizard.

**FR-9 — Architecture zone ownership.** The overview presents the whole
architecture zone, discriminated by **catalog-role** (not artifact `type`):
- the **catalog** (candidate `architectures/` + `tech-stacks/`, carrying the
  `catalog` label) as the browsable menu of options for change;
- the **chosen** architecture + tech stack (promoted to the root);
- the **summary**, **ADRs**, and **standards** as the live architecture;
- the **archive** (`lifecycle/architecture/archive/` — superseded promoted
  choices) as a history/provenance strip, not as live work items.
Consequent to this ownership, the list/board surfaces exclude the architecture
zone by default (see FR-9a).

**FR-9a — Demote the interim list/board toggle.** With the overview owning the
zone, the interim per-view "Show catalog" toggle in `ArtifactListView.vue` /
`KanbanBoardView.vue` (keyed on `isCatalogMaterial()` in `web/src/types/api.ts`)
is **demoted**: the list/board exclude the architecture zone by default. The
discriminator stays **catalog-role**, not artifact `type`, so the chosen
architecture, ADRs, and standards are excluded along with candidates and archive
(the whole zone has its home here). An optional "show architecture inline" escape
hatch may remain, but is off by default.

**FR-10 — Empty / partial states.** The view renders sensibly when the model is
incomplete: with **no chosen architecture**, it shows an empty state that invites
running the wizard and links to the relationship map (does not error). With a
chosen architecture but a **missing summary, empty `standards/`, or no ADRs**,
each affected panel shows its own empty/absent state rather than failing the
whole view.

**FR-11 — Auto-diagram placeholder (forward-compatibility).** The layout
accommodates a later side-by-side **auto-generated diagram of the actual current
system** ([[architecture-auto-diagram]]) beside the intended architecture for
drift comparison. v1 does **not** implement the diagram; it only avoids a layout
that would preclude adding it.

**FR-12 — Derived & fresh.** All panels are derived from the current artifacts on
disk. When a promoted artifact, the summary, a standard, or an ADR is
added/removed/changed on disk, the view reflects the change consistent with the
existing index/watcher behaviour, without a manual rebuild step.

### Non-functional

**NFR-1 — Reuse existing stack.** The view reuses the existing Vue 3 / Pinia /
Vue Router frontend, the existing markdown rendering (`markdown-it`) and
editor/viewer, and the existing index/REST API. It introduces **no new
frontend or backend runtime dependency** and no new persistence store.

**NFR-2 — Read-mostly integrity.** The view issues no write to artifact content.
Its only mutating actions are the existing flows it links to (raise ADR, re-run
wizard); artifacts remain the sole source of truth.

**NFR-3 — Performance.** Initial render for the curated project model (one
architecture, one stack, a summary, ≤ tens of standards/ADRs) completes promptly;
navigating between panels/sections is interactive (sub-second) at this scale.

**NFR-4 — Accessibility.** Panels, lists, and the newest-first ADR ordering are
navigable and readable; any status/decision signalling does not rely on colour
alone (aligns with the existing a11y direction).

**NFR-5 — Graceful degradation.** Missing, empty, or malformed
summary / standards / ADR / promoted-artifact data degrades to a per-panel
empty/absent state and never breaks the overall view (see FR-10).

**NFR-6 — Conformance to recorded architecture.** The feature stays within the
modular-monolith architecture and the Go + Vue stack; it adds no service, no
network trust boundary, and no client-IP-derived behaviour (consistent with
[[adr-no-header-based-client-ip-trust]]).

### Architecture-Breaking Requirements

None. This feature is a read-mostly Vue view inside the existing single-binary
**modular monolith** on the **Go + Vue** stack, consuming the already-defined
architecture artefact model. Assessed against the usual break axes:

- **Offline operation** — no change; the SPA is embedded and served by the same
  single binary, as today. Satisfied by the recorded architecture.
- **Collaboration / realtime** — none introduced. The view is read-mostly and
  derives from disk; live freshness (FR-12) reuses the **existing** fsnotify →
  index → WebSocket path, not a new realtime mechanism. Satisfied.
- **Scale** — bounded by one project's curated architecture model (single
  architecture + stack, tens of standards/ADRs). No new scale pressure. Satisfied.
- **Security / compliance** — no new endpoint trust or client-IP behaviour;
  honours [[adr-no-header-based-client-ip-trust]]. Read access follows the
  existing auth/session model. Satisfied.
- **Cost / new dependency** — none; reuses existing rendering and API (NFR-1).
  Satisfied.

No conflict exists against `lifecycle/architecture/architecture-summary.md`. (In
the kaos-control repo itself that summary and the `standards/` set are not yet
populated — FR-10/NFR-5 require the view to degrade gracefully in exactly that
state, so the absence is a robustness requirement here, not an architecture
break.) No new ADR is required for this requirement.

## Acceptance Criteria

- [ ] When the project has a chosen architecture, the Architecture section
      defaults to the overview view on its own route; with no selection it
      defaults to the relationship map and the overview route shows an empty
      state. *(FR-1, FR-10)* — see [[architecture-relationship-map]]
- [ ] The overview renders the promoted architecture artifact (title, summary,
      components/interactions) with click-through to the artifact. *(FR-2)*
- [ ] The overview renders the promoted tech-stack artifact and its mapping onto
      the architecture's components, with click-through. *(FR-3)*
- [ ] The overview surfaces the wizard Q&A rationale sourced from
      `architecture-summary.md`. *(FR-4)*
- [ ] The overview lists the architecture-breaking requirements with per-item
      mapping to the chosen architecture and stack, sourced from the summary,
      with click-through where links exist. *(FR-5)* — see [[architectural-artefacts]]
- [ ] The overview lists `standards/` items, each click-through to its artifact.
      *(FR-6)*
- [ ] The overview lists `decisions/` ADRs newest-first (title, status, date),
      each click-through to the ADR. *(FR-7)*
- [ ] The overview offers one-click entry points to the relationship map, the
      Architecture Wizard (re-run), and raising a new ADR. *(FR-8)* — see
      [[onboarding-architecture-selection]]
- [ ] The overview presents the architecture zone discriminated by catalog-role
      (catalog, chosen, summary/ADRs/standards, archive), not by artifact `type`.
      *(FR-9)*
- [ ] The list/board exclude the architecture zone by default; the interim
      "Show catalog" toggle is demoted to an off-by-default escape hatch keyed on
      catalog-role. *(FR-9a)*
- [ ] With a chosen architecture but missing summary, empty `standards/`, or no
      ADRs, each affected panel shows an empty/absent state and the view does not
      error. *(FR-10, NFR-5)*
- [ ] The layout accommodates a future side-by-side auto-diagram without
      implementing it in v1. *(FR-11)* — see [[architecture-auto-diagram]]
- [ ] Adding/removing/changing a promoted artifact, the summary, a standard, or
      an ADR on disk is reflected in the view without a manual rebuild. *(FR-12)*
- [ ] The view adds no new frontend/backend runtime dependency and issues no
      artifact-content write. *(NFR-1, NFR-2)*
- [ ] No architecture-breaking requirement is introduced; the feature conforms to
      the modular-monolith architecture and Go + Vue stack and honours
      [[adr-no-header-based-client-ip-trust]]. *(NFR-6)*

## Open Questions

- **OQ-1 — Summary parsing granularity.** FR-4/FR-5 source the Q&A and the
  architecture-breaking requirements from `architecture-summary.md`. Does the view
  parse structured sections/headings out of that markdown (requiring a stable
  summary structure), or render the relevant summary sections as-is with links?
  A stable, agreed heading contract for the summary would make FR-5's per-item
  mapping robust.

> render the relevant summary sections as-is with links

- **OQ-2 — Standards/summary type discovery.** [[architectural-artefacts]] uses
  `type: doc` for the summary and standards in v1. Should the overview locate
  them by **path** (`architecture-summary.md`, `standards/*`) or by `type`/label?
  Path-based discovery is proposed for v1 given `type: doc` is shared.

> Path-based works.

- **OQ-3 — Tech-to-component mapping source (FR-3).** Is the architecture ↔ stack
  component mapping derived from existing artifact content (stack profile / body
  wiki-links), or does it need an explicit mapping field the wizard would record
  in the summary? v1 assumes it is derived from existing content; confirm whether
  an explicit mapping is wanted.

- **OQ-4 — Removal vs. demotion of the list/board toggle (FR-9a).** Should the
  interim "Show catalog" toggle be **removed** entirely once the zone is fully
  represented here, or kept as an off-by-default "show architecture inline"
  escape hatch? Default assumption: keep it demoted/off.

- **OQ-5 — Archive strip scope (FR-9).** How much of the archive should the
  history/provenance strip show by default (latest superseded choice only vs. full
  history), and is it collapsed by default?
