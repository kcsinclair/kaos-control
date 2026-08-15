---
title: "Frontend Plan — Directive Migration Offer & Refresh Affordance"
type: plan-frontend
status: draft
lineage: agent-directives-generation
parent: lifecycle/requirements/agent-directives-generation-2.md
labels:
    - frontend
    - architecture
    - onboarding
    - directives
release: KC-Release5
---

# Frontend Plan — Directive Migration Offer & Refresh Affordance

This plan gives the SPA the three user-facing touchpoints for [[agent-directives-generation-2]]:
the **first-run migration offer** (notify the user that directive handling is now multi-CLI and
offer to migrate automatically — FR-16), a **refresh-directives** action (re-run generation after
a stack change — FR-14), and the **opt-in scaffolding panel** the Architecture Wizard hands off to
(FR-17). It stays thin — all generation, migration, and diffing happen in
[[agent-directives-generation-3-be]]; the frontend surfaces state, shows diffs, and calls the
endpoints.

**Scope boundary.** This plan does **not** build the wizard question flow or the promotion screens
— those are [[onboarding-architecture-selection]]; this plan provides only the "generate directives
now?" panel that flow calls at its end. It does not render directive file *content* specially — the
generated AGENTS.md/CLAUDE.md/GEMINI.md are ordinary root files, not lifecycle artefacts.

Baseline (verified): the app already surfaces init state via `initialised` on the project summary
(`web/src/types/api.ts:20`) and renders
[web/src/components/project/InitRequiredBanner.vue](web/src/components/project/InitRequiredBanner.vue)
from [web/src/views/project/WorkspaceView.vue](web/src/views/project/WorkspaceView.vue) (usage line
103). This plan follows that established banner/modal pattern.

Cross-references:
- [[agent-directives-generation-3-be]] — endpoints: project-summary `directivesMigrationAvailable`, `POST …/migrate-directives`, `POST …/directives/refresh`.
- [[agent-directives-generation-5-test]] — Test plan.
- [[onboarding-architecture-selection]] — wizard flow that reuses this plan's opt-in panel.

---

## Milestone 1 — API module + summary state

### Description

Add the directives API client and surface the new `directivesMigrationAvailable` flag from the
project summary so the UI can decide whether to offer migration.

### Files to change

- **New** `web/src/api/directives.ts`:
  - `migrateDirectives(project, { force })` → `POST /api/projects/{project}/migrate-directives`.
  - `refreshDirectives(project, { force })` → `POST /api/p/{project}/directives/refresh`.
  - Both return the typed `GenerateResult` (files with `created|changed|skipped|diff`,
    `disabledAgents`, `skipped`).
- **Edit** `web/src/types/api.ts`:
  - Add `directivesMigrationAvailable?: boolean` to the project-summary type (beside `initialised`,
    line 20/69) and a `GenerateResult` interface mirroring the backend struct.
- **Edit** `web/src/stores/project.ts`:
  - Expose `directivesMigrationAvailable` from the loaded summary for components to read.

### Acceptance criteria

- `pnpm build` + `pnpm test` clean.
- Unit test (vitest): the API module posts to the correct URLs with the `force` flag; the store
  exposes `directivesMigrationAvailable` from a mocked summary.

---

## Milestone 2 — First-run migration offer banner + modal

### Description

When `directivesMigrationAvailable` is true, show a dismissible banner explaining the multi-CLI
directive model and offering **Migrate automatically**; the modal previews the file changes (and
any diff for a user-edited `AGENTS.md`) before applying, and the decline path tells the user the
`kaos-control migrate-directives` CLI is available on demand (FR-16).

### Files to change

- **New** `web/src/components/project/DirectiveMigrationBanner.vue`:
  - Mirror `InitRequiredBanner.vue`'s structure/styling. Copy: "Directive handling is now multi-CLI
    — AGENTS.md is canonical, with CLAUDE.md and GEMINI.md pointing to it." Actions: **Migrate now**
    (opens modal) and **Not now** (dismiss; show the copyable `kaos-control migrate-directives`
    command).
- **New** `web/src/components/project/DirectiveMigrationModal.vue`:
  - On open, describe the exact changes (rename `CLAUDE.md` → `AGENTS.md`; `CLAUDE.md` becomes
    `@AGENTS.md`; add `GEMINI.md`). Call `migrateDirectives(project, { force: false })`; if the
    response carries a `diff` (user-edited AGENTS.md), render it (reuse the app's existing diff/code
    display) and require an explicit **Overwrite** confirm that re-calls with `force: true`.
    On success show the created/changed/skipped list and refresh the summary so the banner clears.
- **Edit** `web/src/views/project/WorkspaceView.vue`:
  - Render `DirectiveMigrationBanner` when `directivesMigrationAvailable` (alongside the existing
    `InitRequiredBanner` at line 103), gated so it does not show simultaneously with the
    not-initialised banner.

### Acceptance criteria

- `pnpm build` + `pnpm test` clean.
- Component test: banner renders only when `directivesMigrationAvailable`; **Migrate now** →
  modal → success clears the banner; a `diff` response requires the explicit force-overwrite step;
  **Not now** dismisses and reveals the CLI hint.

---

## Milestone 3 — Refresh-directives action + wizard opt-in panel

### Description

Provide a manual **Refresh directives** action (re-run generation after a stack change — FR-14)
and the reusable opt-in panel the Architecture Wizard invokes as its final scaffolding step
(FR-17). Both call `refreshDirectives` and show the same file report + diff-gated overwrite as M2.

### Files to change

- **New** `web/src/components/project/DirectiveRefreshPanel.vue`:
  - A self-contained panel: "Regenerate directive files and agent prompts from the current
    architecture + stack." Button → `refreshDirectives(project, { force: false })`; render the
    created/changed/skipped + `disabledAgents` report; diff-gate any user-edited file behind an
    explicit overwrite (shared logic with M2 — extract a `useDirectiveApply()` composable so the
    modal and panel share the call/diff/confirm flow).
- **New** `web/src/composables/useDirectiveApply.ts`:
  - Wraps call → inspect `diff` → optional force re-call → summary refresh; consumed by the modal
    (M2) and panel (M3).
- **Edit** project settings/menu (where project-level actions live — verify the settings view used
  by `EditProjectModal.vue`'s entry point): add a "Refresh directives" entry that opens the panel.
- **Wizard hand-off:** expose `DirectiveRefreshPanel` for [[onboarding-architecture-selection]] to
  render as its opt-in final step (this plan ships the component; the wizard wires it in). Document
  the prop contract (project id; optional `onDone` callback) so the wizard can chain to its
  completion screen.

### Acceptance criteria

- `pnpm build` + `pnpm test` clean.
- Component test: the refresh panel calls `refreshDirectives`, renders the file report and
  `disabledAgents`, and a `diff` response is gated behind an explicit overwrite; the shared
  `useDirectiveApply` composable is exercised by both the modal and the panel.
- Manual smoke: after promoting a different stack via the wizard, the opt-in panel regenerates
  AGENTS.md + patches the agents; the file report matches the backend result.

---

## Risk notes

- **Two banners at once** — the migration banner must be mutually exclusive with the
  not-initialised banner; gate on `initialised && directivesMigrationAvailable`. Covered by the M2
  test.
- **Diff display reuse** — reuse the existing diff/code viewer rather than adding a dependency; if
  none is suitable, a `<pre>` unified-diff is acceptable for v1 (the backend supplies the diff
  text).
- **Scope creep into wizard UX** — this plan stops at the reusable opt-in panel; the question flow
  and promotion screens remain [[onboarding-architecture-selection]].

## Verification (end-to-end)

1. `pnpm lint` + `pnpm build` clean.
2. `pnpm test` (vitest) clean.
3. Manual smoke against a running backend: a legacy project shows the migration banner → migrate →
   AGENTS.md/CLAUDE.md/GEMINI.md appear and the banner clears; refresh-directives after a stack
   change reports the regenerated files; a hand-edited AGENTS.md forces the explicit-overwrite path.
