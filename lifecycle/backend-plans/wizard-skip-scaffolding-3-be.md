---
title: "Backend Plan — Wizard Skip Scaffolding / Selective Scaffolding (contract + directives reference impl)"
type: plan-backend
status: done
lineage: wizard-skip-scaffolding
parent: lifecycle/requirements/wizard-skip-scaffolding-2.md
created: "2026-08-22T14:30:00+10:00"
---

# Backend Plan — Wizard Skip Scaffolding / Selective Scaffolding

Implements the backend half of [[wizard-skip-scaffolding]]: extend the scaffold
availability/run **contract** so it reports per-step presence and honours
per-step selection, and update the one shipped `Scaffolder`
(`internal/directives`) as the reference implementation. Skip / Finish itself is
purely a frontend outcome (FR-2 issues *no* POST) — the backend's job is to make
the contract express presence and selection, keep every write PO-gated and
sandbox-resolved, and stay backward-compatible when no scaffolder is registered.

Conforms to the recorded architecture: modular monolith, single Go binary, local
filesystem as source of truth ([[architecture-summary.md]]). No new external
dependency, datastore, artefact type, index entry, or watcher path is introduced
(NFR-5). No ADR required — the requirement's Architecture-Breaking section
confirms this is a contract/UX refinement within the standing constraints.

See the frontend counterpart [[wizard-skip-scaffolding]] (frontend plan) and the
test plan [[wizard-skip-scaffolding]] (test plan).

## Milestone 1 — Extend the scaffold contract (presence + selection)

**Description.** Add a per-step presence indicator and a per-choice selection
flag to the wire contract, and thread the project root into presence detection.
Per resolved **OQ-1**, presence is a single per-**step** flag (not per-artefact).
Per resolved **OQ-2**, selection is an **explicit boolean** on `ScaffoldChoice`
(not expressed by omission).

The `Scaffolder.Available` signature must gain `projectRoot` — presence can only
be computed against the project's files, and FR-6 requires that resolution go
through the sandbox resolver rooted at the project.

**Files to change.**
- `internal/architecture/scaffold.go`
  - `ScaffoldStep`: add `Present bool \`json:"present"\`` (FR-4; step-level per
    OQ-1). Document it as "the artefact(s) this step would create already exist
    on disk; a run would report this step as skipped (FR-5)."
  - `ScaffoldChoice`: add `Selected bool \`json:"selected"\`` (FR-9, OQ-2).
    Document that a choice with `Selected == false` (including the zero value /
    absent field) means "do not scaffold this step" (FR-9/FR-11), and that a run
    with no selected step is a net no-op (FR-10).
  - `Scaffolder` interface: change
    `Available(archSlug, stackSlug string) ([]ScaffoldStep, bool)` to
    `Available(projectRoot, archSlug, stackSlug string) ([]ScaffoldStep, bool)`.
    Document that `Available` MUST be read-only (FR-5) and MUST resolve any path
    it inspects through `internal/sandbox` against `projectRoot` (FR-6).
- `internal/http/architecture_wizard.go` — update the one call site
  (`handleGetWizardScaffold`) to pass `p.Entry.Path` (Milestone 4).
- `internal/directives/scaffolder.go` — update `Available` to the new signature
  (Milestone 2); the `var _ architecture.Scaffolder = Scaffolder{}` assertion
  keeps the two in lock-step at compile time.

**Acceptance criteria.**
- `internal/architecture` and dependents compile; `var _ architecture.Scaffolder
  = Scaffolder{}` still satisfied.
- `ScaffoldStep` marshals a `present` boolean; `ScaffoldChoice` unmarshals a
  `selected` boolean. A JSON body omitting `selected` decodes to `false`.
- `go vet ./...` and `staticcheck` clean (`make lint`).

## Milestone 2 — Read-only presence detection in `directives.Scaffolder.Available`

**Description.** Implement per-step presence for the single `agent-directives`
step. "Present" means every file this step would create already exists on disk,
so a real run would report it under `ScaffoldResult.Skipped` (FR-5). The step
creates `AGENTS.md` and `CLAUDE.md` always, and `GEMINI.md` when a gemini driver
is configured (see `Generate`/`hasGeminiDriver`). Presence is therefore:
`AGENTS.md` present **and** `CLAUDE.md` present **and** (`GEMINI.md` present *or*
no gemini driver configured).

Detection MUST be read-only (no writes, no `Generate` call — `Generate` mutates)
and MUST resolve each filename through `sandbox.Resolve(projectRoot, name)`
before `os.Stat`, failing closed on an unresolvable/out-of-root path (FR-6).
Because these are fixed root-level filenames the resolver never rejects them, but
routing through it is the standard the requirement mandates
([[filesystem-sandboxing]]).

Driver detection reuses the existing `configuredDrivers(projectRoot)` /
`hasGeminiDriver` helpers so presence agrees with what a run would actually
write. A config-load error is treated fail-safe as "not present" (so the step is
still offered) rather than surfaced as an error from a read-only availability
call.

**Files to change.**
- `internal/directives/scaffolder.go`
  - New unexported helper, e.g.
    `directivesPresent(projectRoot string) bool`, using `sandbox.Resolve` +
    `os.Stat` for `AGENTS.md`, `CLAUDE.md`, and conditionally `GEMINI.md`.
  - `Available(projectRoot, archSlug, stackSlug string)` sets
    `Present: directivesPresent(projectRoot)` on the returned step.
- (import `internal/sandbox` and `os` in `scaffolder.go`.)

**Acceptance criteria.**
- On a project with no directive files, `Available(...)[0].Present == false`.
- On a project where `AGENTS.md` + `CLAUDE.md` exist (and `GEMINI.md` exists iff
  a gemini driver is configured), `Present == true`.
- `Available` performs **no** disk writes (verified by a before/after file-tree
  count in the test plan) — FR-5, NFR-1.
- A path that would escape the root (defensive test) is rejected via the sandbox
  error rather than stat'd directly (FR-6).

## Milestone 3 — Honour per-step selection in `Scaffolder.Run`

**Description.** `directives.Scaffolder.Run` currently ignores `choices` and
always generates. Make it scaffold **only** steps whose choice has
`Selected == true`. For the single `agent-directives` step:
- If no choice for `scaffoldStepKey` has `Selected == true`, `Run` writes nothing
  and returns an empty `ScaffoldResult{}` (no `Applied`, no git commands) — this
  is the FR-10 zero-selection / FR-11 unselected-present path, and keeps the
  endpoint safe even though the frontend won't POST in that case.
- If selected, behaviour is exactly as today (generate directives, bootstrap
  devops pipelines, track under git), and remains idempotent: re-running on an
  already-scaffolded project produces no net change and reports present files as
  skipped (FR-12), which the existing `Generate` merge/skip logic already gives.

This preserves FR-11 ("a present item that is not selected MUST NOT be
re-generated, overwritten, or reported as applied"): an unselected step is never
passed to `Generate`, so nothing under it is touched or reported.

**Files to change.**
- `internal/directives/scaffolder.go`
  - Add a `selected(choices, scaffoldStepKey) bool` check at the top of `Run`;
    early-return `architecture.ScaffoldResult{}, nil` when not selected.

**Acceptance criteria.**
- `Run` with `choices == nil` or all `Selected == false` writes nothing and
  returns a zero `ScaffoldResult` (FR-10/FR-11).
- `Run` with the step `Selected == true` behaves as today (directive files +
  devops pipelines generated; `Committed`/`GitCommands` populated per repo
  ownership).
- Re-running the selected step on an already-scaffolded project reports the files
  as skipped and makes no net change (FR-12).

## Milestone 4 — Wire the handlers; preserve auth and backward compatibility

**Description.** Update the HTTP seam to the new contract while keeping the
authorization split and the no-scaffolder degrade path unchanged.

- `handleGetWizardScaffold` (GET): pass `p.Entry.Path` to `Available`; response
  shape (`{available, steps}` / `{available:false, message}`) is unchanged apart
  from each step now carrying `present`. Remains **authenticated-user only**
  (NFR-3). When `ActiveScaffolder() == nil`, still returns
  `{available:false, message}` with no `steps` (NFR-4) — Skip / Finish on the
  frontend does not depend on this call succeeding.
- `handleRunWizardScaffold` (POST): unchanged auth — remains **product-owner
  only** (`requireRole(..., RoleProductOwner)`, NFR-3). It decodes `choices`
  (now carrying `selected`) and forwards them verbatim to `Run`; selection is
  enforced in `Run` (Milestone 3), so no handler-side filtering is needed. With
  no scaffolder registered it still returns the graceful `{available:false,
  message}` and writes nothing.

No new routes; `server.go` route table is untouched. No index/watcher changes —
a real run's files continue to be picked up by the existing config-reload / index
paths exactly as today (NFR-5).

**Files to change.**
- `internal/http/architecture_wizard.go`
  - `handleGetWizardScaffold`: `steps, ok := scaffolder.Available(p.Entry.Path,
    archSlug, stackSlug)`.
  - (No signature/route change to `handleRunWizardScaffold`; confirm it still
    passes `req.Choices` through.)

**Acceptance criteria.**
- `GET …/wizard/scaffold` returns `steps[].present` and requires only an
  authenticated user (a non-PO authenticated user gets `200`) — NFR-3.
- `POST …/wizard/scaffold` still returns `403` for a non-PO user (NFR-3) and
  `{available:false}` when no scaffolder is registered (NFR-4), writing nothing.
- With no scaffolder registered, `GET` returns `{available:false, message}` and
  the response carries no `steps` (NFR-4).
- Full build (`make build`) and `make lint` pass.

## Cross-cutting notes

- **Callers/tests updated together.** Adding an explicit `Selected` flag (OQ-2)
  means existing callers that submit a choice without `selected` now decode to
  `Selected:false` → skipped. The existing integration test
  (`tests/integration/architecture_wizard_scaffold_test.go`) that posts
  `{"step_key":"agent-directives","use_defaults":true}` must add
  `"selected":true`; this is captured in the test plan
  ([[wizard-skip-scaffolding]] test plan).
- **No persisted trace of Skip (OQ-3).** Skip / Finish is transient — the
  backend writes nothing recording it, and no summary/ADR is amended.
- **Sandbox everywhere (FR-6).** Presence stat calls and any run writes resolve
  through `internal/sandbox` / the existing generators, never `filepath.Join` of
  slug/catalog input onto the root ([[filesystem-sandboxing]]).
