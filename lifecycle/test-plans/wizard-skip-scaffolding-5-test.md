---
title: "Test Plan — Wizard Skip / Selective Scaffolding"
type: plan-test
status: done
lineage: wizard-skip-scaffolding
parent: lifecycle/requirements/wizard-skip-scaffolding-2.md
created: "2026-08-22T14:30:00+10:00"
---

# Test Plan — Wizard Skip / Selective Scaffolding

Verifies [[wizard-skip-scaffolding]] against its acceptance criteria across three
layers: Go unit tests (contract + directives reference impl), Go integration
tests (HTTP seam, auth, no-writes guarantees), and Vitest component/store tests
(Skip / Finish, presence display, selection). Cross-references the backend plan
([[wizard-skip-scaffolding]] backend plan) and frontend plan
([[wizard-skip-scaffolding]] frontend plan).

Existing coverage to preserve/adapt:
`tests/integration/architecture_wizard_scaffold_test.go`,
`internal/directives/scaffolder_test.go`,
`web/src/components/architecture/__tests__/ScaffoldStep.spec.ts`.

## Milestone 1 — Backend unit tests: contract + presence detection

**Description.** Cover the extended structs and the read-only presence check in
`directives.Scaffolder.Available`.

**Files to change.**
- `internal/architecture/scaffold_test.go` (new or extend)
- `internal/directives/scaffolder_test.go` (extend)

**Acceptance criteria.**
- **Contract marshalling (FR-4, FR-9):** `ScaffoldStep{Present:true}` marshals
  `"present":true`; `ScaffoldChoice` decodes `"selected":true`, and a body
  omitting `selected` decodes to `Selected:false`.
- **Presence false when absent (FR-4):** on a temp project with no directive
  files, `Available(root,…)[0].Present == false`.
- **Presence true when present (FR-4, FR-5):** after writing `AGENTS.md` +
  `CLAUDE.md` (and `GEMINI.md` only when a gemini driver is configured),
  `Present == true`.
- **Gemini conditionality:** with a gemini driver configured but `GEMINI.md`
  missing, `Present == false`; with no gemini driver and `GEMINI.md` absent but
  `AGENTS.md`/`CLAUDE.md` present, `Present == true`.
- **Read-only (FR-5, NFR-1):** a file-tree snapshot before/after `Available`
  is identical (no writes).
- **Sandbox fail-closed (FR-6):** presence resolution uses `internal/sandbox`;
  a traversal/absolute input (defensive helper test) yields the sandbox error
  rather than a raw stat.

## Milestone 2 — Backend unit tests: selective Run

**Description.** Cover `Scaffolder.Run` honouring `Selected` (FR-10, FR-11, FR-12).

**Files to change.**
- `internal/directives/scaffolder_test.go` (extend)

**Acceptance criteria.**
- **Zero selection = no-op (FR-10):** `Run` with `choices == nil`, `choices ==
  []`, or the step present with `Selected:false` writes nothing (before/after
  file-tree count equal) and returns an empty `ScaffoldResult` (no `Applied`, no
  `GitCommands`, `Committed==false`).
- **Selected runs (existing behaviour):** `Run` with the step `Selected:true`
  generates `AGENTS.md`/`CLAUDE.md` and bootstraps `lifecycle/devops/{build,lint,
  test}.yaml`.
- **Unselected present item untouched (FR-11):** with directive files already on
  disk and the step `Selected:false`, no file mtime changes and nothing is
  reported under `Applied`.
- **Idempotent re-run (FR-12):** running the selected step twice yields no net
  change on the second run and reports the files as skipped.

## Milestone 3 — Integration tests: HTTP seam, auth, no-writes

**Description.** Exercise the endpoints end-to-end via the integration harness,
including the presence field, selection, auth split, and the no-writes
guarantees.

**Files to change.**
- `tests/integration/architecture_wizard_scaffold_test.go` (extend + adapt)

**Acceptance criteria.**
- **Adapt existing run test:** the existing
  `TestWizardScaffold_DirectivesScaffolder_GeneratesDirectiveFiles` choice body
  gains `"selected":true` and still generates files + returns `git_commands`
  (proves the explicit-flag contract, backend plan cross-cutting note).
- **GET reports presence (FR-4):** after committing and running scaffolding once,
  a second `GET …/wizard/scaffold?architecture=…&tech_stack=…` returns
  `steps[0].present == true`; before any run it returns `present == false`.
- **GET is authenticated-user only (NFR-3):** a logged-in non-product-owner user
  gets `200` from `GET …/wizard/scaffold`.
- **POST is product-owner only (NFR-3):** a non-PO user gets `403` from
  `POST …/wizard/scaffold`; no files are written.
- **Partial selection (FR-9, FR-11):** POST with the step `selected:false` writes
  nothing and reports nothing applied; POST with `selected:true` writes the
  step's files. (Single-step scaffolder — assert the selected/unselected contrast
  rather than multi-step partitioning.)
- **Zero-selection POST = no writes (FR-10, NFR-1):** POST with all choices
  `selected:false` leaves the project file tree unchanged.
- **No-scaffolder degrade preserved (NFR-4):** existing
  `TestWizardScaffoldGet_NoScaffolderRegistered_ReturnsUnavailable` and
  `TestWizardScaffoldPost_NoScaffolderRegistered_GracefulNoWrites` still pass
  unchanged.
- **No new index/watcher artefacts (NFR-5):** a real run's files are picked up by
  existing paths; no new artefact type appears in the index (assert via existing
  index/query helpers — commit-then-run leaves the architecture tree file count
  governed only by promotion, not by scaffolding adding index entries).

## Milestone 4 — Frontend unit tests (Vitest)

**Description.** Cover the ScaffoldStep UX and store terminal wiring with mocked
API (`getScaffold`/`runScaffold`), asserting Skip issues no run call.

**Files to change.**
- `web/src/components/architecture/__tests__/ScaffoldStep.spec.ts` (extend)
- `web/src/views/project/__tests__/ArchitectureWizardView.spec.ts` (extend)
- `web/src/stores/__tests__/architectureWizard.spec.ts` (extend — `scaffoldSettled`)

**Acceptance criteria.**
- **Skip / Finish present in all states (FR-1):** rendered and non-subordinate
  when `available:false`, when `steps` is empty, and in the normal step list.
- **Skip issues no run (FR-2):** clicking Skip / Finish emits `finish` and never
  calls the mocked `runScaffold`.
- **All-present state (FR-7):** with every step `present:true`, the "everything's
  already in place" copy renders and Skip / Finish is the primary, focused action
  while Run is not presented as the expected action.
- **Default selection empty (OQ-4):** on load, every step's selection control is
  unchecked; Run is disabled until ≥1 step is selected.
- **Presence badge (FR-8):** each step shows present vs missing state from
  `step.present`.
- **Selection drives payload (FR-9):** toggling a step and clicking Run calls
  `runScaffold` with that step `selected:true` and others `selected:false`.
- **Zero selection = skip (FR-10):** with nothing selected, Run is disabled and
  Skip / Finish completes the flow with no `runScaffold` call.
- **Terminal wiring (FR-3):** `finish` from `ScaffoldStep` sets
  `store.scaffoldSettled` and `store.step = 'done'`; `WizardSuccess` then hides
  "Set up scaffolding"; `store.reset()` clears `scaffoldSettled`.

## Exit criteria

- `make lint`, `make test-unit`, the integration suite
  (`go test -tags integration ./tests/...`), and `pnpm test` (Vitest) all pass.
- Every acceptance-criteria checkbox in [[wizard-skip-scaffolding]] maps to at
  least one test above (FR-1…FR-12, NFR-1…NFR-5, OQ-1…OQ-4).
