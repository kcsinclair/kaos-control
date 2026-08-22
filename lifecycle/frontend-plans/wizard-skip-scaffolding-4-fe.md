---
title: "Frontend Plan — Wizard Skip / Finish + Selective Scaffolding (ScaffoldStep.vue)"
type: plan-frontend
status: draft
lineage: wizard-skip-scaffolding
parent: lifecycle/requirements/wizard-skip-scaffolding-2.md
created: "2026-08-22T14:30:00+10:00"
---

# Frontend Plan — Wizard Skip / Finish + Selective Scaffolding

Implements the UX half of [[wizard-skip-scaffolding]]: a first-class **Skip
scaffolding / Finish** action, per-item present/missing display, per-item
selection, and an "everything's already in place" state, all in the wizard's
final `ScaffoldStep.vue`. Depends on the extended contract from the backend plan
([[wizard-skip-scaffolding]] backend plan): `ScaffoldStep.present` and
`ScaffoldChoice.selected`.

Conforms to the recorded stack ([[go-vue]]): Vue 3.5 SFC + Pinia + TypeScript,
embedded SPA ([[adr-0004-embedded-spa-single-binary]]). No new dependency.

### Resolved contradiction — default selection

FR-8's *Detailed Requirements* text says the default selection on a
partially-scaffolded project should be "the missing items only." **OQ-4 in the
same requirement's Resolved Questions explicitly overrides this**, resolving the
default to **"select nothing, user opts in"** (the parenthetical marks FR-8's
default as merely "assumed"). This plan implements OQ-4: the **default selection
is empty** on every load. FR-8's other clause — *show per-item present vs.
missing state* — still stands and is implemented. This is a documented reading
of an explicit resolution, not a guess.

## Milestone 1 — Type mirrors

**Description.** Mirror the extended Go contract in the SPA types so the compiler
enforces the new fields (FR-4, FR-9).

**Files to change.**
- `web/src/types/api.ts`
  - `ScaffoldStep`: add `present?: boolean` (optional so a no-scaffolder /
    older response still type-checks; treated as `false` when absent — NFR-4).
  - `ScaffoldChoice`: add `selected: boolean` (required — every emitted choice
    states its selection explicitly, OQ-2).

**Acceptance criteria.**
- `pnpm build` / `vue-tsc` type-check passes.
- `ScaffoldChoice` objects constructed in `ScaffoldStep.vue` must set `selected`.

## Milestone 2 — Per-item selection + presence display

**Description.** Replace the current "submit a choice for every step" behaviour
with explicit per-step selection and visible presence state (FR-8, FR-9).

- Extend `stepState` to track `selected: boolean` per step, defaulting to
  **false** for every step (OQ-4 — "user opts in").
- Render each step card with:
  - a **checkbox / toggle** bound to `stepState[key].selected` (FR-9);
  - a **present / missing badge** from `step.present` — e.g. "Already present"
    vs "Will be created" (FR-8);
  - the existing naming fields + "decide for me" control, shown only when the
    step is selected (naming an item you won't scaffold is noise).
- `choices` computed now emits a choice **for every offered step**, each with its
  real `selected` value (backend enforces selection; sending all steps with
  explicit flags is the OQ-2 "do these, not those" instruction). Present-but-
  unselected steps carry `selected:false` and are left untouched by the backend
  (FR-11).

**Files to change.**
- `web/src/components/architecture/ScaffoldStep.vue`
  - `stepState` reactive shape gains `selected`.
  - `load()` initialises `selected: false` per step.
  - Template: per-step checkbox + presence badge; gate name-fields on `selected`.
  - `choices` computed includes `selected: stepState[s.key].selected`.

**Acceptance criteria.**
- Each offered step shows a present/missing indicator driven by `step.present`.
- Each step has an independent selection control, all unchecked on load (OQ-4).
- `runScaffold` payload includes every step with an explicit `selected` flag;
  toggling one step changes only that step's flag.

## Milestone 3 — Skip / Finish action + "everything's already in place" state

**Description.** Add the first-class Skip / Finish control and the all-present
terminal messaging (FR-1, FR-2, FR-3, FR-7).

- **Skip / Finish button (FR-1):** rendered with **equal prominence** to *Run
  scaffolding* (same `.btn-primary`/`.btn-secondary` weight, a real button — not
  a text link). It is present in **every** render branch that can complete the
  wizard, including when `availability.available === false` and when `steps` is
  empty (FR-1). Activating it calls **no** API (`getScaffold` may have run;
  `runScaffold` must not — FR-2) and emits a `finish` event (Milestone 4).
- **All-present state (FR-7):** when `available` and every offered step has
  `present === true`, render a clear "Everything's already in place — nothing to
  scaffold" panel with **Skip / Finish as the primary, default-focused action**
  (autofocus / `ref` focus on mount) and do **not** present *Run scaffolding* as
  the expected action (hide it or demote it to a subordinate "Scaffold anyway").
- **Run button gating (FR-10):** *Run scaffolding* is disabled when zero steps
  are selected; a disabled Run plus an active Skip / Finish means "select nothing"
  behaves exactly like Skip (no POST, no writes). Optionally, clicking a disabled-
  intent Run is simply not possible — Skip / Finish is the path out.
- **Post-run completion (FR-3):** after a successful `runScaffold`, the existing
  result panel stays, plus a **Finish** button that emits the same `finish`
  event, so a completed run and a Skip reach the *same* terminal state.

**Files to change.**
- `web/src/components/architecture/ScaffoldStep.vue`
  - Add `defineEmits<{ finish: [] }>()`.
  - Add `skipFinish()` → `emit('finish')` (no API call).
  - Template branches: not-available / empty-steps / all-present / partial / post-
    result — each surfaces Skip / Finish; all-present makes it primary+focused.
  - Disable *Run scaffolding* when no step selected; add *Finish* to the result
    panel.
  - Styles: ensure Skip / Finish matches Run's visual weight (FR-1).

**Acceptance criteria.**
- Skip / Finish is visible and equally prominent in all states, including
  `available:false` and empty `steps` (FR-1).
- Activating Skip / Finish fires no network request to `POST …/scaffold` (FR-2,
  asserted by a mocked `runScaffold` never being called).
- With every step `present`, the step shows the "already in place" copy and Skip
  / Finish is the focused primary action; Run is not the expected action (FR-7).
- With zero steps selected, Run is disabled and Skip / Finish completes the flow
  (FR-10).
- After a run, a Finish control emits `finish` (FR-3).

## Milestone 4 — Terminal wiring in the wizard shell

**Description.** Make both Skip / Finish and post-run Finish land on the same
terminal success state the wizard reaches after a completed run (FR-3), without
letting the user loop back into scaffolding.

`WizardSuccess.vue` (step `done`) is the named success state and currently
*precedes* scaffolding, offering a "Set up scaffolding" entry. To satisfy FR-3
("reach the same terminal/success state (`WizardSuccess`)") cleanly:

- Add a store flag `scaffoldSettled` (ref, default `false`), set `true` when the
  scaffold step emits `finish`.
- `ArchitectureWizardView` handles `ScaffoldStep`'s `finish` by returning to the
  `done` step (`store.step = 'done'`), i.e. `WizardSuccess` — the terminal
  success screen — now with scaffolding settled.
- `WizardSuccess.vue` hides / disables its "Set up scaffolding" button when
  `store.scaffoldSettled` is `true`, preventing a re-entry loop, and may show a
  brief "Scaffolding step complete" confirmation.
- `reset()` clears `scaffoldSettled`.

This keeps the single terminal state for both outcomes (Skip and completed run)
and preserves the existing exit affordances (View relationship map / Cancel).

**Files to change.**
- `web/src/stores/architectureWizard.ts` — add `scaffoldSettled` ref, expose it,
  clear it in `reset()`.
- `web/src/views/project/ArchitectureWizardView.vue` — bind
  `@finish="onScaffoldFinish"` on `<ScaffoldStep>`; `onScaffoldFinish` sets
  `store.scaffoldSettled = true` and `store.step = 'done'`.
- `web/src/components/architecture/WizardSuccess.vue` — guard the "Set up
  scaffolding" button on `!store.scaffoldSettled`.

**Acceptance criteria.**
- Skip / Finish and post-run Finish both land on `WizardSuccess` (step `done`),
  the same state reached after a run (FR-3).
- After finishing, "Set up scaffolding" is not offered again (no loop).
- `store.reset()` (view unmount / Start Again) clears `scaffoldSettled`.

## Cross-cutting notes

- **No writes on load / skip / zero-selection (NFR-1):** only an explicit *Run
  scaffolding* with ≥1 selected step issues a POST. Skip / Finish and empty
  selection issue none — enforced here and (defensively) in the backend
  ([[wizard-skip-scaffolding]] backend plan).
- **Backward compatible (NFR-4):** `present` is optional in the type mirror; with
  no scaffolder registered the not-available branch renders and Skip / Finish
  still completes the wizard.
- **Transient (OQ-3):** the frontend persists nothing about the skip decision.
