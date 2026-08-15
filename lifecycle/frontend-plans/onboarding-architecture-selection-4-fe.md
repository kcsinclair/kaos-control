---
title: Frontend Plan — Architecture Wizard (Guided Selection)
type: plan-frontend
status: blocked
lineage: onboarding-architecture-selection
parent: lifecycle/requirements/onboarding-architecture-selection-2.md
labels:
    - frontend
    - architecture
    - onboarding
    - wizard
    - ux
release: KC-Release5
assignees:
    - role: product-owner
      who: agent
---

# Frontend Plan — Architecture Wizard (Guided Selection)

Implements the onboarding UX of [[onboarding-architecture-selection]]: a launchable multi-step
wizard offering two paths (Browse / Guided) to a chosen architecture + compatible stack, showing
transparent recommendations, and driving persistence via the backend
([[onboarding-architecture-selection-3-be]]). Built in Vue 3 + Pinia + Vue Router per the tool's
own Go+Vue stack.

**Scope boundary.** UI only. All scoring, promotion, summary/ADR authoring, prior-run detection,
and resume storage are the backend's ([[onboarding-architecture-selection-3-be]]). The visual
relationship surface is [[architecture-relationship-map]] (`ArchitectureMapView.vue`) which this
wizard **embeds/links**; the post-selection outcome display is [[architecture-overview-view]].

**Reuse.** `web/src/api/architecture.ts` already exports `promoteArchitecture`, `nextAdrNumber`,
`createAdr` ("exposed here for the onboarding wizard to reuse") — extend it, do not fork. There is
**no generic stepper component** yet; the multi-step, session-backed pattern in
`web/src/stores/ideaChat.ts` + `web/src/components/idea/` and the form/validation pattern in
`web/src/components/project/CreateProjectModal.vue` are the closest templates. Nav lives in
`web/src/components/layout/AppSidebar.vue` (an "Architecture" entry already exists).

**Dependency note.** New projects do **not** yet copy the catalog into `lifecycle/architecture/`
(open defect `project-init-missing-architecture-artefacts.md`). The wizard must handle an **empty
catalog** gracefully (clear "no catalog present — run init / see defect" state) rather than
render a broken Browse. Scaffolding (FR-17/FR-18) hand-off targets are unbuilt; that step renders
only when the backend reports `available:true` (M7 of the backend plan).

Cross-references:
- [[onboarding-architecture-selection-3-be]] — the endpoints this UI calls.
- [[onboarding-architecture-selection-5-test]] — Test plan.
- [[architecture-relationship-map]] — the embedded/linked map.
- [[architecture-overview-view]] — where the persisted outcome is displayed afterwards.
- [[new-project-init-directory-options]] — the New Project flow launch point (FR-1).

---

## Milestone 1 — API client + wizard Pinia store

### Description

Typed client wrappers and a state-machine store that own the whole wizard flow: which path, the
answers, the current step, the recommendations, the chosen architecture + stack, and
save/resume. The store is the single source of truth the views bind to.

### Files to change

- **Edit** `web/src/api/architecture.ts`: add
  `getWizard(project)` → `GET …/architecture/wizard`,
  `recommend(project, answers)` → `POST …/architecture/wizard/recommend`,
  `listStacks(project, architecture, language)` → `GET …/architecture/wizard/stacks`,
  `saveWizardState(project, state)` / `discardWizardState(project)` → `PUT`/`DELETE …/wizard/state`,
  `commitWizard(project, payload)` → `POST …/architecture/wizard/commit`,
  `getScaffold(project)` / `runScaffold(project, choices)` → `GET`/`POST …/wizard/scaffold`.
- **Edit** `web/src/types/api.ts`: add `WizardQuestion`, `WizardRecommendation`, `WizardPriorRun`,
  `WizardState`, `CatalogItem`, `ScaffoldStep` types mirroring the backend JSON.
- **New** `web/src/stores/architectureWizard.ts` (`defineStore('architectureWizard', () => …)`):
  state `path` (`'browse'|'guided'|null`), `step`, `questions`, `answers`, `recommendations`,
  `droppedConstraints`, `chosenArchitecture`, `chosenStack`, `priorRun`, `loading`, `error`;
  actions `start()` (loads questions + prior-run + any resumable state), `setAnswer()`,
  `skip()`, `fetchRecommendations()`, `chooseArchitecture()`, `fetchStacks()`, `chooseStack()`,
  `persistState()` (debounced save for resume), `commit()`, `reset()`. Getters for step
  validity and the "why" strings.

### Acceptance criteria

- `pnpm build` + `pnpm test` (vitest) pass.
- Store unit test: `start()` populates questions + prior-run; `setAnswer`/`skip` mutate state;
  `commit()` calls the commit endpoint with architecture + stack + answers + Q&A; a
  server error surfaces via `error` without throwing.

---

## Milestone 2 — Wizard shell, routing, and entry points (FR-1, FR-2, FR-3)

### Description

A stepper shell view that hosts the step components, plus the two launch points (New Project flow
and the Architecture menu) and the prior-run gate. Launchable **any time**, not only at creation
(FR-1).

### Files to change

- **New** `web/src/views/project/ArchitectureWizardView.vue` — the shell: a step header/progress,
  the active step slot, and Back/Next/Cancel. Reads/writes the M1 store.
- **New** `web/src/components/architecture/WizardStepper.vue` — reusable step-progress indicator
  (no generic one exists).
- **Edit** `web/src/router/index.ts`: add child route `architecture/wizard` →
  `ArchitectureWizardView.vue` under `/p/:project`, with `meta: { roles: ['product-owner'] }`
  (OQ-5) matching the existing `devops` route gating.
- **Edit** `web/src/components/layout/AppSidebar.vue`: add a launch affordance for the wizard
  (either a sub-item under the existing "Architecture" entry or a "Start Architecture Wizard"
  action), gated to product-owner via `NavItem.roles`.
- **Edit** `web/src/components/project/CreateProjectModal.vue` (and/or `InitProjectModal.vue`):
  after successful project creation/init, offer "Set up architecture now" that routes to the
  wizard (FR-1 second entry point). Non-blocking — the user can skip.
- **New** `web/src/components/architecture/PriorRunGate.vue` — when the store's `priorRun.detected`
  is true, show the existing selection (architecture, stack, ADR link) and force an explicit
  **Continue (re-run) / Exit** choice before any step proceeds (FR-3, NFR-1).

### Acceptance criteria

- The wizard route is reachable from the sidebar and from the post-create modal; both require
  product-owner.
- Component test: with `priorRun.detected=true`, the stepper renders `PriorRunGate` and blocks
  advancement until Continue or Exit is chosen; Exit routes away with no commit call.

---

## Milestone 3 — Path selection + Browse path (FR-4, FR-5, FR-6)

### Description

The fork step and the Browse experience: catalog architecture cards / comparison with pros/cons +
decision-signal labels, an embed/link to the relationship map, direct architecture pick, then the
compatible-stack picker. The user may switch from Guided to Browse ("show me everything anyway")
at any time (FR-4).

### Files to change

- **New** `web/src/components/architecture/PathChoiceStep.vue` — Browse vs Guided, plus a
  persistent "Show me everything anyway" control surfaced throughout Guided.
- **New** `web/src/components/architecture/BrowseCatalogStep.vue` — architecture cards (title,
  summary, pros/cons, labels) + a comparison table; embeds or links `ArchitectureMapView.vue`
  ([[architecture-relationship-map]]); selecting a card sets `chosenArchitecture`. Handles the
  **empty-catalog** case with a clear message (dependency note above).
- **New** `web/src/components/architecture/StackChoiceStep.vue` — after an architecture is chosen
  (either path), lists compatible stacks from `listStacks`, language-ranked (FR-6, FR-10), with
  confirm/override. Shared by both paths.

### Acceptance criteria

- Component test: Browse renders one card per catalog architecture with its labels/pros/cons;
  picking one advances to `StackChoiceStep` showing only that architecture's `related_to` stacks;
  the "show me everything anyway" control jumps from Guided into Browse preserving answers.
- Empty catalog renders the guidance state, not a crash/blank.

---

## Milestone 4 — Guided questionnaire (FR-7, FR-8, NFR-5)

### Description

The ≤10-question flow driven by the backend-supplied question set (from `lifecycle/config.yaml`,
OQ-4). Every question is **skippable**; plain language; a "decide for me" / skip default at each
step for less-technical users (NFR-5).

### Files to change

- **New** `web/src/components/architecture/GuidedQuestionStep.vue` — renders the current
  `WizardQuestion` (single/multi-select options), a prominent **Skip** control, and a
  "decide for me" affordance; writes answers to the store; progresses through the set, then calls
  `fetchRecommendations()`.
- **Edit** `web/src/stores/architectureWizard.ts`: sequence questions, track skipped ones, and
  compose the answer payload; auto-`persistState()` after each answer (resume, OQ-3).

### Acceptance criteria

- Component test: exactly the configured questions render (≤10), each Skippable; skipping a
  question omits it from the answer payload; completing/skipping all triggers a `recommend` call;
  answering triggers a debounced `saveWizardState`.

---

## Milestone 5 — Recommendation + stack ranking display (FR-9, FR-10, FR-11, NFR-4)

### Description

Transparent results: the **top 2–3** architectures each with a one-line "why", the OQ-2
**dropped-constraints** notice when applicable, confirm-or-override, then the language-ranked
stack step (reusing M3's `StackChoiceStep`). Never a single black-box answer (NFR-4).

### Files to change

- **New** `web/src/components/architecture/RecommendationStep.vue` — renders `recommendations`
  (2–3) with per-candidate `why`, an "override with any other candidate" expander (falls back to
  Browse to reach any catalog item, FR-4/NFR-4), and, when `droppedConstraints` is non-empty, a
  "no exact match — here is the closest; dropped: …" banner (OQ-2). Weak-signal default-bias
  candidates (FR-11) render with their default-bias "why" string.
- Wire `RecommendationStep` → `StackChoiceStep` (M3) for stack confirm/override (FR-10).

### Acceptance criteria

- Component test: given 3 recommendations, all render with a visible "why"; confirming sets
  `chosenArchitecture` and advances to the stack step; a `droppedConstraints` payload renders the
  closest-match banner listing exactly the dropped constraints.

---

## Milestone 6 — Confirm + commit + success (FR-13, FR-14, NFR-1)

### Description

The pre-write review and the single commit action. Shows the final architecture + stack + any
**seeded standards to be created** before anything is written (FR-13, NFR-1). On confirm, calls
`commitWizard`; on success shows the promoted files, `architecture-summary.md`, and ADR link, and
offers to view them in [[architecture-overview-view]] / [[architecture-relationship-map]].

### Files to change

- **New** `web/src/components/architecture/ConfirmStep.vue` — read-only summary of the selection
  and standards-to-be-created (from the backend), an explicit "Confirm & write" button, and copy
  making clear nothing is written until confirmed. Building the Q&A/breaking-requirements payload
  for the commit body from the store's answers.
- **New** `web/src/components/architecture/WizardSuccess.vue` — post-commit confirmation with links
  to the promoted architecture, stack, summary, and ADR-0001; entry to the optional scaffolding
  step (M7). On success, clears resume state (`reset()`).

### Acceptance criteria

- Component test: `ConfirmStep` shows architecture + stack + standards and makes **no** API write
  until "Confirm & write"; Cancel/abandon before confirm issues no commit call (NFR-1);
  on commit success `WizardSuccess` renders the created-file links and offers scaffolding.

---

## Milestone 7 — Opt-in scaffolding step (FR-17, FR-18)

### Description

A final, opt-in step offering config/pipelines/agent-directives/repo-skeleton scaffolding, with
naming prompts that each include a **"decide for me"** default (FR-18). Rendered only when the
backend reports scaffolding is available (backend M7); otherwise a "coming soon — see
[[agent-directives-generation]]" note. Never automatic (FR-17).

### Files to change

- **New** `web/src/components/architecture/ScaffoldStep.vue` — lists available `ScaffoldStep`s from
  `getScaffold`, renders naming fields with per-field "decide for me" defaults, and calls
  `runScaffold`. Hidden/disabled when `available:false`.

### Acceptance criteria

- Component test: with `available:false`, the step renders the not-yet-available note and issues no
  `runScaffold` call; with `available:true`, naming fields render with working "decide for me"
  defaults and submit dispatches `runScaffold`.

---

## Accessibility & less-technical UX (NFR-5) — cross-cutting

- Every step: keyboard navigable, focus managed on step change, plain-language copy, a visible
  skip/default at each decision point (path choice, ambiguous scoring, naming).
- Covered by targeted a11y assertions folded into the component tests above and in
  [[onboarding-architecture-selection-5-test]].

## Verification (end-to-end)

1. `pnpm build` clean; `pnpm test` (vitest) green.
2. Manual: launch from sidebar (as product-owner) and from post-create modal; run Guided to a
   recommendation, override to Browse, pick a stack, confirm, and observe promoted files + summary
   + ADR links; re-launch to see the prior-run gate; abandon mid-Guided and re-open to resume.

## Progress

- **Milestone 1 — API client + wizard Pinia store**: done (`web/src/api/architecture.ts`,
  `web/src/types/api.ts`, `web/src/stores/architectureWizard.ts` + unit tests).
- **Milestone 2 — Wizard shell, routing, and entry points**: done
  (`ArchitectureWizardView.vue`, `WizardStepper.vue`, `PriorRunGate.vue`, the
  `architecture/wizard` route, the sidebar entry, and the post-create/post-init
  "Set up architecture now" hand-offs).
- **Milestone 3 onward**: blocked — see Open Questions below.

## Open Questions

- **OQ-6 (blocking).** `BrowseCatalogStep.vue` (Milestone 3, FR-5) needs, for **every** catalog
  architecture, its `title`/`summary`/`labels`/`related_to`/**`pros`/`cons`** up front, before any
  architecture is chosen — that's what the cards and comparison table render. No backend endpoint
  returns this today:
  - `GET /api/p/{project}/architecture/wizard/recommend` requires `answers` and returns at most 3
    scored `Recommendation`s, never the full catalog.
  - `GET /api/p/{project}/architecture/wizard/stacks` requires an already-chosen `architecture`
    slug — unusable before a pick exists, and it's stacks, not architectures.
  - `GET /api/p/{project}/artifacts?type=architecture` returns index rows with frontmatter
    (`labels`, `related_to`, `summary`) but **no `pros`/`cons`** — those fields don't exist on
    `artifact.Frontmatter` at all.
  - `pros`/`cons` are stored only as `## Pros` / `## Cons` markdown sections in each catalog file's
    **body**, parsed today exclusively by the backend's internal `architecture.LoadCatalog()` /
    `parseProsCons()` (used by `Recommend`/`RankStacks`, never exposed over HTTP). The only way to
    reach that text from the frontend today is `GET /architecture/artifacts/{path}` — the raw
    markdown body — one artifact at a time, which would mean an N+1 fetch plus re-implementing
    `## Pros`/`## Cons` section parsing in TypeScript, duplicating backend logic the plan's own
    "Scope boundary" section reserves for the backend ("All scoring, promotion, summary/ADR
    authoring... is the backend's").
  - This plan's cross-reference [[onboarding-architecture-selection-3-be]] (the backend plan) never
    mentions a catalog-listing/Browse endpoint — it wasn't scoped there either.
  - **What's needed to unblock**: a new backend read endpoint (e.g.
    `GET /api/p/{project}/architecture/wizard/catalog` → `{ architectures: CatalogItem[], tech_stacks:
    CatalogItem[] }`, reusing the existing `architecture.LoadCatalog()` + `CatalogItem` JSON shape)
    so Browse can render cards/comparison without picking an architecture first. This is backend
    work outside this plan's write scope (`web/src/**` only) — routing it to
    [[onboarding-architecture-selection-3-be]] (or a follow-up backend milestone) is the
    product-owner's call.
  - Milestones 3 (Browse path), and by extension 5's "override with any other candidate" expander
    (which falls back to Browse) and 6/7, cannot be implemented and tested against their stated
    acceptance criteria until this is resolved. Milestone 3's `PathChoiceStep.vue` and
    `StackChoiceStep.vue` have no such dependency and could proceed once this question is answered
    and the endpoint (or an approved alternative) exists — implementation stopped rather than
    guessing at a workaround (e.g. N+1 client-side markdown scraping) that would silently deviate
    from the plan's own scope boundary.
