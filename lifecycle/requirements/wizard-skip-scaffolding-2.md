---
title: Architecture Wizard — First-Class "Skip Scaffolding / Finish" for Retrofits
type: requirement
status: blocked
lineage: wizard-skip-scaffolding
created: "2026-08-22T13:30:00+10:00"
parent: lifecycle/ideas/wizard-skip-scaffolding.md
labels:
    - architecture
    - wizard
    - onboarding
    - ux
release: KC-Release5
assignees:
    - role: product-owner
      who: agent
---

# Architecture Wizard — First-Class "Skip Scaffolding / Finish" for Retrofits

## Problem

The Architecture Wizard's final step (`ScaffoldStep.vue`, satisfying FR-17/FR-18
of [[onboarding-architecture-selection]]) offers optional
config / pipelines / agent-directives / repo-skeleton scaffolding. It is already
opt-in — nothing mutates disk until **Run scaffolding** is clicked — but three
gaps make it read as a required action rather than an optional one, which is
wrong for a **retrofit** (a project that already has its config, directives,
pipelines, and repo skeleton — kaos-control adopting its own lifecycle is the
canonical case):

1. **No Skip / Finish path.** The step renders only a *Run scaffolding* button.
   There is no equally-prominent control that completes the wizard cleanly
   *without* invoking the scaffolder, so it is unclear how to finish.
2. **No up-front "already scaffolded" detection.** The availability contract
   (`GET …/architecture/wizard/scaffold` → `WizardScaffoldAvailability`, backed
   by `Scaffolder.Available()` returning `[]ScaffoldStep`) reports only *which*
   steps are offered, not *what already exists*. Presence is discovered only
   *after* a run, in `ScaffoldResult.Skipped`. So on a fully-retrofitted project
   the step still presents Run scaffolding as the expected action.
3. **All-or-nothing run.** `ScaffoldChoice` carries per-step naming values and a
   `use_defaults` flag but no include/exclude selector, and the step submits a
   choice for *every* offered step. A partially-scaffolded project cannot
   scaffold only the missing pieces and skip the rest.

Retrofitting the lifecycle onto an existing codebase is a first-class use case;
the scaffold step is built for greenfield onboarding and should get out of the
way when there is nothing (or little) to do.

## Goals / Non-goals

### Goals

- Add an explicit **"Skip scaffolding / Finish"** action to the scaffold step
  that completes the wizard cleanly without invoking the scaffolder — visually
  as prominent as **Run scaffolding**, and writing nothing to disk.
- **Detect "already present" per offered step** *before* any run, by extending
  the availability contract so the UI can show per-item state and default the
  step toward Skip when everything is already in place.
- When **every** offered artefact is already present, present the step as
  "Everything's already in place — nothing to scaffold" with **Skip / Finish**
  as the primary (default-focused) action.
- Support **selective, per-item scaffolding**: a partially-scaffolded project can
  select only the missing items to scaffold and leave present ones untouched.
- Preserve every existing guarantee: opt-in only, deterministic, idempotent, no
  writes to disk until the user acts (NFR-1 / NFR-2 of
  [[onboarding-architecture-selection]]).

### Non-goals

- The scaffolding **generators themselves** (config, pipelines, repo skeleton,
  agent directives) and what "present" means for each artefact — owned by
  [[architecture-templates]] §4 and [[agent-directives-generation]]. This
  requirement defines the **contract** they must satisfy (report per-item
  presence; honour per-item selection), not their internals.
- The wizard entry, path selection, questionnaire, promotion, summary, and ADR
  authoring — owned by [[onboarding-architecture-selection]] and
  [[architectural-artefacts]]. This requirement refines only the scaffolding
  hand-off step (FR-17/FR-18).
- A CLI equivalent of Skip/selective scaffolding (`kaos-control init` prompts);
  v1 is the web-UI wizard step only, consistent with the wizard's own v1 scope.
- Migrating or diffing existing directive files — owned by
  [[agent-directives-generation]] (FR-11/FR-16).

## Detailed Requirements

### Functional — Skip / Finish

- **FR-1** The scaffold step MUST present a **"Skip scaffolding / Finish"**
  action alongside **Run scaffolding**, rendered with equal prominence (not a
  subordinate text link). Both must be reachable whenever the availability call
  succeeds — including when `available` is `false` (no scaffolder registered) or
  the offered step list is empty.
- **FR-2** Activating Skip / Finish MUST complete the wizard **without** calling
  `POST …/architecture/wizard/scaffold` and MUST write nothing to disk.
- **FR-3** After Skip / Finish the wizard MUST reach the same terminal/success
  state (`WizardSuccess`) it reaches after a completed run, so "finish without
  scaffolding" is a normal, non-error outcome.

### Functional — presence detection (contract)

- **FR-4** The availability contract MUST report, per offered step, whether that
  step's artefact(s) are **already present** on disk. Concretely: `ScaffoldStep`
  (Go `internal/architecture/scaffold.go` and its TypeScript mirror in
  `web/src/types/api.ts`) gains a machine-readable presence indicator (e.g.
  `present bool` and/or `partial bool`), populated by `Scaffolder.Available()`.
- **FR-5** Presence detection MUST be **read-only**: computing it makes no
  changes to the project. It reports the same facts the post-run
  `ScaffoldResult.Skipped` would, without running anything.
- **FR-6** All presence-detection path resolution MUST go through the sandbox
  resolver (`internal/sandbox/`) against the project root — no `filepath.Join`
  of catalog/slug input onto the root, fail-closed on unresolvable/out-of-root
  paths — per [[filesystem-sandboxing]].
- **FR-7** When **every** offered step reports present, the step MUST render a
  clear "Everything's already in place — nothing to scaffold" state with **Skip
  / Finish** as the primary, default-focused action, and MUST NOT present Run
  scaffolding as the expected action.
- **FR-8** When **some** offered steps are present and some are missing, the step
  MUST show per-item state (present vs. missing) and default the selection
  (FR-9) to the missing items only.

### Functional — selective scaffolding

- **FR-9** The user MUST be able to select, per offered step, whether to scaffold
  it, and the step MUST submit choices for the **selected** steps only. The
  selection is expressed in the run payload — either a per-`ScaffoldChoice`
  include flag or by omitting unselected steps from `choices` — such that the
  scaffolder receives an unambiguous "do these, not those" instruction.
- **FR-10** Selecting zero steps MUST be equivalent to Skip / Finish (FR-2): no
  run is issued and nothing is written.
- **FR-11** A present item that is **not** selected MUST NOT be re-generated,
  overwritten, or reported as applied; running with a partial selection scaffolds
  exactly the selected (missing) items and leaves the rest untouched.
- **FR-12** Running remains fully idempotent (per
  [[onboarding-architecture-selection]] NFR-2 and [[architectural-artefacts]]
  FR-7): re-running with the same selection on an already-scaffolded project
  produces no net change and reports the present items as skipped.

### Non-functional

- **NFR-1** No writes to disk occur on load, presence detection, Skip / Finish,
  or a zero-selection submit; only an explicit Run scaffolding with ≥1 selected
  step mutates the project. Abandoning the step leaves the project unchanged.
- **NFR-2** Presence detection is deterministic and adds no perceptible latency
  to loading the step (target: the `GET …/wizard/scaffold` response returns in
  well under 1 s on a local filesystem, consistent with
  [[agent-directives-generation]] NFR-4).
- **NFR-3** The run authorization is unchanged: `POST …/wizard/scaffold` remains
  restricted to the **product-owner** role; the read-only availability/presence
  `GET` requires only an authenticated user, matching the current handlers.
- **NFR-4** The change is backward-compatible at the seam: with **no** scaffolder
  registered (`ActiveScaffolder() == nil`, the current default) the step still
  renders and Skip / Finish still completes the wizard; presence fields are
  simply absent/false.
- **NFR-5** No new artefact types, index entries, or watcher paths are
  introduced. Anything the scaffolder writes on a real run continues to be picked
  up by the existing config-reload / index paths, exactly as today
  ([[agent-directives-generation]] FR-15, [[index-is-a-cache]]).

### Architecture-Breaking Requirements

None. This requirement is a UX + API-contract refinement of an existing wizard
step and introduces no constraint that could invalidate the chosen architecture
or stack recorded in [[architecture-summary.md]]. Explicitly checked against the
standing architecture-breaking constraints:

- **Single self-contained binary.** No new external datastore, service, or cgo
  dependency; presence detection reads local files and the extended contract is
  plain Go structs + their embedded-SPA TypeScript mirror. Satisfied.
- **Local filesystem is the source of truth.** Presence is computed by reading
  the authoritative project files, not by querying the index; the index remains a
  rebuildable cache ([[index-is-a-cache]]). Satisfied.
- **Agents execute arbitrary tools / scope-enforced writes.** No agent execution
  is added. Presence reads and any scaffolder writes resolve through the sandbox
  resolver (FR-6) and remain within the mediated model
  ([[adr-0006-mediated-agent-driver-permission-model]], [[filesystem-sandboxing]]).
  Satisfied.
- **Direct-served, no trusted proxy hop.** No change to request-identity or
  transport handling ([[adr-0001-no-header-based-client-ip-trust]]). Satisfied.

No conflict with `lifecycle/architecture/architecture-summary.md`; no new ADR is
required.

## Acceptance Criteria

- [ ] The scaffold step shows a **Skip scaffolding / Finish** action as prominent
      as **Run scaffolding**, present even when availability is `false` or the
      step list is empty. *(FR-1)* — see [[onboarding-architecture-selection]]
- [ ] Activating Skip / Finish completes the wizard, issues **no**
      `POST …/wizard/scaffold`, and writes nothing to disk. *(FR-2, NFR-1)*
- [ ] After Skip / Finish the wizard reaches the same success state as after a
      completed run. *(FR-3)* — see [[architecture-overview-view]]
- [ ] `Scaffolder.Available()` / the `GET …/wizard/scaffold` payload reports
      per-step presence, and the Go struct and its `web/src/types/api.ts` mirror
      both carry the presence indicator. *(FR-4)*
- [ ] Presence detection performs no disk writes and reports the same items a
      post-run `ScaffoldResult.Skipped` would. *(FR-5, FR-12)*
- [ ] All presence path resolution goes through `internal/sandbox/` and fails
      closed on traversal / out-of-root input. *(FR-6)* — see [[filesystem-sandboxing]]
- [ ] On a project where every offered step is present, the step renders
      "Everything's already in place — nothing to scaffold" with Skip / Finish as
      the primary, default-focused action. *(FR-7)*
- [ ] On a partially-scaffolded project, per-item present/missing state is shown
      and the default selection is the missing items only. *(FR-8)*
- [ ] The user can toggle each step's selection, and Run scaffolding submits
      choices for the selected steps only. *(FR-9)*
- [ ] Selecting zero steps behaves exactly like Skip / Finish — no run, no
      writes. *(FR-10)*
- [ ] Running a partial selection scaffolds only the selected (missing) items;
      unselected present items are neither overwritten nor reported as applied.
      *(FR-11)*
- [ ] Re-running with the same selection on an already-scaffolded project yields
      no net change and reports present items as skipped. *(FR-12)* — see [[architectural-artefacts]]
- [ ] `POST …/wizard/scaffold` still requires the product-owner role; the
      availability `GET` requires only an authenticated user. *(NFR-3)*
- [ ] With no scaffolder registered, the step still renders and Skip / Finish
      still completes the wizard. *(NFR-4)*
- [ ] No new artefact types, index entries, or watcher paths are added; a real
      run's files are picked up by the existing paths. *(NFR-5)* — see [[agent-directives-generation]]

## Open Questions

- **OQ-1** Presence granularity: is a per-**step** `present`/`partial` flag
  sufficient, or should the contract report presence per underlying **artefact**
  (e.g. `CLAUDE.md` present but `GEMINI.md` missing within the directives step)?
  A step-level flag is simpler and matches the current `ScaffoldStep` shape; a
  finer grain gives clearer messaging on partial steps (FR-8).

> per-step is sufficient

- **OQ-2** Selection encoding: add an explicit `selected`/`include` boolean to
  `ScaffoldChoice`, or express "skip this" by **omitting** the step from
  `choices`? Omission is smaller but overloads "absent" to mean "skip"; an
  explicit flag is unambiguous but changes the struct. (Either satisfies FR-9;
  this fixes the wire contract for [[architecture-templates]] and
  [[agent-directives-generation]].)

> add an explicit selected/include boolean to ScaffoldChoice

- **OQ-3** Should Skip / Finish be recorded anywhere (e.g. a note in
  `architecture-summary.md` that scaffolding was intentionally skipped on a
  retrofit), or is it a purely transient UI outcome with no persisted trace?

- **OQ-4** For a partially-scaffolded project, should the default be "select only
  missing" (FR-8, assumed) or "select nothing, user opts in"? The former speeds
  the common finish-the-retrofit case; the latter is maximally conservative.
