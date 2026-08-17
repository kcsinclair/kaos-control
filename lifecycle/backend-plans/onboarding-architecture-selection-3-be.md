---
title: "Backend Plan — Architecture Wizard (Guided Selection)"
type: plan-backend
status: done
lineage: onboarding-architecture-selection
parent: lifecycle/requirements/onboarding-architecture-selection-2.md
labels:
    - backend
    - architecture
    - onboarding
    - wizard
release: KC-Release5
---

# Backend Plan — Architecture Wizard (Guided Selection)

Implements the server side of [[onboarding-architecture-selection]]: the question-set
configuration, the deterministic rule-based recommendation engine, the `architecture-summary.md`
writer, prior-run detection, resumable in-progress state, and the wizard orchestration
(`commit`) that drives promotion + summary + ADR authoring.

**Scope boundary.** This lineage owns the wizard *flow and scoring*. It is a **caller** of the
already-built on-disk primitives in [[architectural-artefacts]] (`internal/architecture`:
`Promote`, `CreateADR`, `WriteADR0001`, `NextADRNumber` and the
`POST /architecture/promote` · `POST /architecture/adrs` · `GET /architecture/adrs/next`
endpoints). It does **not** re-implement promotion, ADR numbering, type registration, or the
lineage-validation relaxation — those are done. Catalog seed content + standards seed set are
[[architecture-templates]]; concrete agent-directive prose is [[agent-directives-generation]];
the map/overview rendering is [[architecture-relationship-map]] / [[architecture-overview-view]].

**Architecture conformance.** kaos-control has not yet run its own wizard, so
`lifecycle/architecture/` holds only the catalog — there is no promoted architecture, summary,
ADR, or `standards/` to conform to yet. These plans therefore conform to the tool's de-facto
**Go + Vue** stack described in CLAUDE.md. Scoring is **deterministic and rule-based, no LLM**
(FR-12), so no new model/runtime dependency is introduced — no ADR-worthy deviation. Two
dependencies are **not built**: the standards seed set ([[architecture-templates]] §4) and the
scaffolding/agent-directive generators ([[agent-directives-generation]], currently `blocked`).
The wizard exposes them as **opt-in seams that degrade gracefully** (M7); it never blocks the
core select→promote→summarise→ADR flow on them. This is a recorded cross-lineage dependency,
not a silent deviation.

Cross-references:
- [[onboarding-architecture-selection-4-fe]] — Frontend plan (wizard UI, entry points, resume).
- [[onboarding-architecture-selection-5-test]] — Test plan.
- [[architectural-artefacts]] — the promotion / ADR / type primitives this wizard calls.
- [[architecture-templates]] — catalog labels + de-duplicated standards seed (FR-16, OQ-5).
- [[agent-directives-generation]] — scaffolding + directive generation (FR-17/FR-18 hand-off).

---

## Milestone 1 — Question-set configuration (OQ-4)

### Description

The questionnaire's question set and its question→label mapping live in the per-project
`lifecycle/config.yaml` (OQ-4 resolved). Add an `architecture_wizard` config section modelling
questions, each with: `id`, plain-language `prompt`, `kind` (`hard` constraint | `soft` signal |
`language`), and the catalog `labels` an answer contributes (FR-7, FR-8). Also model the
documented **default bias** (FR-11): the fallback architecture slug (`modular-monolith`) and how
the language answer ranks stacks (FR-10). Ship built-in defaults and self-repair a missing
section, mirroring the existing agent-template repair path (`ValidateAndRepair` /
`RepairNote`) so older project configs keep working.

### Files to change

- **Edit** `internal/config/config.go`:
  - Add `ArchitectureWizard ArchitectureWizardConfig` to the `Project` struct
    (`yaml:"architecture_wizard,omitempty" json:"architecture_wizard,omitempty"`, ~line 640).
  - Define `ArchitectureWizardConfig { Questions []WizardQuestion; DefaultArchitecture string;
    Enabled *bool }` and `WizardQuestion { ID, Prompt, Kind string; Labels []string;
    Options []WizardOption }`, where `WizardOption { Value, Label string; Labels []string;
    Hard bool }` lets a single question carry multiple selectable answers each mapping to
    labels (e.g. "strongest language" → `go`/`ts`/`python`…; "must work offline" → hard
    `offline-capable`). Cap enforcement (≤10) is validated, not hard-coded.
  - Extend `ValidateAndRepair` to fill a missing/empty `architecture_wizard` from a new
    `defaultArchitectureWizard()` built-in and append a `RepairNote`; validate `len(Questions) <= 10`
    and that every `Kind` is one of `hard|soft|language` (repair/skip invalid entries with a note).
- **Edit** `lifecycle/config.yaml`: add the `architecture_wizard:` section with the default
  question set — offline (`offline-capable`, hard), multi-user edit (`collaborative`, soft),
  realtime/streaming (`realtime`, soft), scale (`high-scale`, soft), phone-first (`mobile`, hard),
  AI/ML-central (`ai-ml`, soft), ops tolerance (`low-complexity`…`high-complexity`, soft),
  cost (`low-cost-start`, soft), strongest language (`go`/`ts`/`python`/`java`/`php`, language) —
  ≤10 questions, each with the plain-language prompt the UI shows. Keep it byte-identical to
  `defaultArchitectureWizard()` so the repair path is a no-op on this repo.

### Acceptance criteria

- `go build ./... && go vet ./...` clean; `make run` boots with no config-parse error.
- Unit test `internal/config`: a config with **no** `architecture_wizard` loads, is repaired to
  the built-in set, and records a `RepairNote`; a config with 11 questions is trimmed/flagged; an
  invalid `kind` is rejected with a note. The shipped `lifecycle/config.yaml` round-trips with an
  empty repair set (defaults already present).
- The parsed set is reachable from a project's runtime config (asserted via the M5 endpoint test).

---

## Milestone 2 — Recommendation engine (`internal/architecture/recommend.go`)

### Description

A pure, deterministic scorer (FR-8–FR-12). Given the catalog (architectures with `labels` +
`related_to`, stacks with language `labels`) and a set of answers, it (a) applies **hard
constraints** as a filter, (b) **scores** the survivors by soft-signal label overlap, (c) returns
the **top 2–3** with a machine-readable "why" (which answers drove each), and (d) ranks
**compatible stacks** for a chosen architecture by the language answer. Zero-match handling
(OQ-2): return the **closest** candidates with the list of **dropped constraints** rather than an
empty set. Weak/ambiguous signals (FR-11): bias toward `DefaultArchitecture`
(`modular-monolith`) plus a language-matched stack. No LLM, no clock, no randomness — identical
inputs yield an identical ranking (FR-12).

### Files to change

- **New** `internal/architecture/catalog.go`:
  - `type CatalogItem struct { Path, Slug, Title, Summary, Type string; Labels, RelatedTo []string; Pros, Cons []string }`.
  - `func LoadCatalog(projectRoot string) (arches, stacks []CatalogItem, err error)` — reads
    `lifecycle/architecture/architectures/*.md` and `tech-stacks/*.md` via the existing
    `artifact.Parse`, through `sandbox.Resolve`. Pros/Cons parsed best-effort from the `## Pros`/
    `## Cons` sections (optional; scoring never depends on them). Deterministic ordering (sort by
    slug).
- **New** `internal/architecture/recommend.go`:
  - `type Answer struct { QuestionID, Value string }` and `type Recommendation struct {
    Item CatalogItem; Score int; Why []string }`.
  - `func Recommend(arches []CatalogItem, questions []config.WizardQuestion, answers []Answer)
    (recs []Recommendation, droppedConstraints []string, err error)`:
    1. Resolve answers → contributed labels via the question/option mapping; classify each as
       `hard` or `soft`.
    2. Filter arches whose labels satisfy **all** hard labels. If that yields zero, relax
       constraints one at a time (weakest = fewest catalog matches first, deterministic tiebreak
       by label name) recording each dropped one, until ≥1 candidate remains (OQ-2).
    3. Score survivors by count of soft-label overlaps; build `Why` from the specific answers that
       matched. Stable sort by `(Score desc, DefaultArchitecture-first, slug asc)`.
    4. If the top scores are all equal/zero (ambiguous, FR-11), ensure `DefaultArchitecture` is
       ranked first with a `Why` of "low-regret default — signals were weak".
    5. Return the top 3 (or fewer).
  - `func RankStacks(chosen CatalogItem, stacks []CatalogItem, languageAnswer string)
    []CatalogItem` — restrict to `chosen.RelatedTo`, then stable-sort so stacks whose labels
    include `languageAnswer` come first (FR-6, FR-10). OQ-1 guarantees `related_to` is present, so
    no empty-set fallback is required; still return `[]` safely if a catalog entry is malformed.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit test `internal/architecture/recommend_test.go`, seeded from the real catalog fixture:
  - offline=hard → only offline-capable architectures survive (desktop/edge/mobile), server-only
    ones excluded (FR-8).
  - An over-constrained combo (e.g. offline + high-scale + low-cost) filters to zero, then relaxes
    and returns closest + the exact dropped-constraint list (OQ-2).
  - No/weak answers → `modular-monolith` ranked first with the documented default-bias `Why`
    (FR-11).
  - `Recommend` called twice with the same inputs is byte-identical (determinism, FR-12).
  - `RankStacks` for a chosen architecture returns only `related_to` stacks, language-matched
    first (FR-6/FR-10).

---

## Milestone 3 — `architecture-summary.md` writer (`internal/architecture/summary.go`)

### Description

The Architecture Summary is created by the wizard ([[architectural-artefacts]] FR-8/FR-9); it is
**not** yet implemented. Add a deterministic, idempotent writer that records the
**architecture-breaking requirements** surfaced by the questionnaire and their mapping to the
chosen architecture + stack, the full **Q&A trail**, and links to the promoted architecture,
promoted stack, ADR(s), and standards (FR-14, FR-15). `type: doc`, clean-slug filename
`architecture-summary.md` at the `lifecycle/architecture/` root (per architectural-artefacts
OQ-2 / FR-10 / FR-19).

### Files to change

- **New** `internal/architecture/summary.go`:
  - `type SummaryInput struct { Architecture, TechStack string; BreakingRequirements []BreakingReq;
    QA []QAPair; ADRPaths, StandardPaths []string }`, `BreakingReq { Label, Requirement, Mapping string }`,
    `QAPair { Question, Answer string }`.
  - `func WriteSummary(projectRoot string, in SummaryInput) (relPath string, err error)`:
    - Renders markdown with frontmatter `title: Architecture Summary`, `type: doc`,
      `status: approved` and body sections: *Architecture-breaking requirements* (table:
      requirement → how the chosen architecture+stack satisfies it), *Selection Q&A*, *Links*
      (`[[…]]`/relative to promoted arch, stack, ADRs, standards).
    - Writes `lifecycle/architecture/architecture-summary.md` atomically (reuse the package's
      `writeAtomic`); **idempotent** — overwrites in place, never duplicates (NFR-2/NFR-3).
    - No clock; the caller may pass a date into the body if wanted.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit test `internal/architecture/summary_test.go`: writing produces one
  `architecture-summary.md` with `type: doc`, the breaking-requirement mapping table, the Q&A
  trail, and resolvable links; writing twice yields exactly one file, byte-identical
  (idempotent).

---

## Milestone 4 — Resumable wizard state store (OQ-3)

### Description

The wizard supports partial completion + resume (OQ-3 resolved: **yes**). Because **NFR-1
forbids any write under `lifecycle/architecture/` before confirm**, in-progress state must live
**outside** that tree. Persist it in the project's runtime state area (alongside the SQLite
index / project runtime dir), keyed per project and per user, holding the path taken, answers so
far, and the current step. It is scratch state, not a lifecycle artefact, so it is **not**
indexed or committed to the lifecycle tree.

### Files to change

- **New** `internal/architecture/wizardstate.go` (or a small table in `internal/index`):
  - `type WizardState struct { Path string; Answers []Answer; ChosenArchitecture, ChosenTechStack string;
    Step string; UpdatedUnix int64 }`.
  - `func SaveWizardState(projectRuntimeDir, userID string, st WizardState) error`,
    `func LoadWizardState(projectRuntimeDir, userID string) (WizardState, bool, error)`,
    `func ClearWizardState(projectRuntimeDir, userID string) error` — a JSON file under the
    project's runtime/state dir (e.g. `<runtime>/wizard-state/<userID>.json`) or an
    `index`-managed table. Chosen because it is disposable and must survive a page reload but
    never touch `lifecycle/`.
- Confirm the resolved location does **not** fall under `lifecycle/` (guard with a test).

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Unit test: save → load returns the same state; clear removes it; the state path is asserted to
  be outside `lifecycle/architecture/` (NFR-1 guard). Saving mid-flow writes **nothing** under
  `lifecycle/architecture/`.

---

## Milestone 5 — Wizard read endpoints (start, recommend, stacks, state)

### Description

Expose the read/scoring/resume surface over the existing chi router. These endpoints perform
**no** writes to `lifecycle/architecture/` (NFR-1). Prior-run detection (FR-2/FR-3) reports
whether the project already has a promoted architecture / `architecture-summary.md` /
`adr-0001-*.md`, so the UI can offer continue-or-exit.

### Files to change

- **New** `internal/http/architecture_wizard.go`:
  - `GET /api/p/{project}/architecture/wizard` — returns `{ questions, default_architecture,
    prior_run: { detected, architecture, tech_stack, adr_path, summary_path }, resumable_state }`.
    Prior-run detection scans the `lifecycle/architecture/` root for a promoted `type: architecture`
    file, `architecture-summary.md`, and `decisions/adr-0001-*.md` (FR-2). Loads any saved
    `WizardState` for the requesting user (M4).
  - `POST /api/p/{project}/architecture/wizard/recommend` — body `{ answers: [...] }` →
    `{ recommendations: [{ slug, title, score, why }], dropped_constraints: [...] }` via
    `architecture.Recommend` over `LoadCatalog` (deterministic; FR-9, FR-12, OQ-2).
  - `GET /api/p/{project}/architecture/wizard/stacks?architecture=<slug>&language=<lang>` →
    ranked compatible stacks via `architecture.RankStacks` (FR-6, FR-10).
  - `PUT /api/p/{project}/architecture/wizard/state` — persist in-progress `WizardState` (M4).
  - `DELETE /api/p/{project}/architecture/wizard/state` — discard saved state (abandon; NFR-1).
- **Edit** `internal/http/server.go`: mount the five routes in the `/p/{project}` group beside the
  existing `/architecture/*` routes (~line 266). Read/recommend/stacks require a normal
  authenticated project member; `state` writes are per-user scratch (any member).

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Integration coverage in [[onboarding-architecture-selection-5-test]]: `wizard` returns the
  configured questions + `prior_run.detected=false` on a fresh project and `true` after a commit;
  `recommend` is deterministic and honours hard filters + OQ-2 fallback; `stacks` returns only
  `related_to` stacks language-ranked; `state` PUT/GET/DELETE round-trips and writes nothing under
  `lifecycle/architecture/`.

---

## Milestone 6 — Wizard commit orchestration + re-run superseding ADR

### Description

The single **write** entry point (FR-13, FR-14, FR-16, NFR-1, NFR-2). On confirm it orchestrates
the already-built primitives in one server-side transaction-like sequence, gated to the
**product-owner** role (OQ-5). First run: `Promote` → `WriteSummary` → `WriteADR0001`. Re-run
that **changes** the selection: `Promote` (archives prior copies, [[architectural-artefacts]]
FR-7) → new **superseding** ADR via `CreateADR` (marks the prior ADR `status: superseded` with a
"Superseded by" pointer) → `WriteSummary` refresh (FR-16). Re-run with the **same** selection is
idempotent — overwrite-in-place, no new ADR, no duplicates (NFR-2). Nothing is written until this
call (NFR-1); on success the saved `WizardState` is cleared (M4).

### Files to change

- **New** in `internal/architecture/`: `func Supersede(projectRoot, priorADRRel, newADRRel string) error`
  — sets the prior ADR's `status: superseded` and appends a "Superseded by <newADRRel>" line via
  the frontmatter/body patch helpers; and `func SelectionChanged(projectRoot string, req PromotionRequest) (bool, error)`
  — compares the requested sources against the currently-promoted root copies' `parent:` stamps.
- **Edit** `internal/http/architecture_wizard.go`:
  - `POST /api/p/{project}/architecture/wizard/commit` — body `{ architecture_path, tech_stack_path,
    answers, breaking_requirements, qa }`:
    - Role gate `requireRole(w, r, p, RoleProductOwner)` (OQ-5).
    - Determine first-run vs changed vs same via `SelectionChanged` + prior-run detection.
    - Call `architecture.Promote`; build the ADR/Q&A body from `qa`+`answers`+rejected recs.
    - First run → `WriteADR0001`. Changed → `CreateADR` (new number) titled "Re-adopt … " then
      `Supersede(priorADR, newADR)`. Same → re-run `WriteADR0001`/no-op ADR.
    - `WriteSummary` with the breaking-requirement mapping + full Q&A + links to promoted files,
      ADR(s), and any seeded standards.
    - Synchronously re-index every written/archived path (reuse `reindexPath`, mirroring the
      existing promote handler, NFR-2); clear `WizardState`; return the created/updated paths.
- **Edit** `internal/http/server.go`: mount the commit route (product-owner gated).

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Integration coverage: first commit produces the two promoted root copies (with `parent:`),
  `architecture-summary.md`, and `adr-0001-*.md`; the API-write re-index path fires (the files
  appear in `GET …/artifacts`); a non-product-owner gets `403`; **no** files exist under
  `lifecycle/architecture/` if the wizard is abandoned before commit (NFR-1); a same-selection
  re-commit leaves exactly one of each file (idempotent, NFR-2); a changed-selection re-commit
  archives the prior promoted copies, writes a new superseding ADR, marks the prior ADR
  `superseded`, and refreshes the summary (FR-16).

---

## Milestone 7 — Opt-in scaffolding hand-off seam (FR-17/FR-18)

### Description

After persistence the wizard **offers (opt-in, never automatic)** scaffolding: config, pipelines
(incl. testing + security scanning), agent directives, and a repo skeleton, by handing off to
[[architecture-templates]] §4 and [[agent-directives-generation]]. Those generators are **not
built** ([[agent-directives-generation]] is `blocked`), so this milestone builds the **seam**
only: an endpoint that reports what scaffolding is available for the chosen architecture+stack,
accepts naming choices with a "decide for me" default (FR-18), and dispatches to the generator
**if present**, otherwise returns a clear "not-yet-available" status. The core flow (M6) never
depends on this.

### Files to change

- **New** in `internal/http/architecture_wizard.go`:
  - `GET /api/p/{project}/architecture/wizard/scaffold` — returns the available scaffolding steps
    for the chosen stack and the naming fields each needs (with computed "decide for me"
    defaults). Returns an empty/`available:false` set until the generators land.
  - `POST /api/p/{project}/architecture/wizard/scaffold` — product-owner gated; body carries the
    naming choices (or requests defaults); dispatches to the generator interface if registered,
    else `501`/`{available:false}` with a message pointing at [[agent-directives-generation]].
- **New** `internal/architecture/scaffold.go`: a small `Scaffolder` interface + registry so
  [[architecture-templates]]/[[agent-directives-generation]] can register an implementation later
  without touching the wizard.

### Acceptance criteria

- `go build ./... && go vet ./...` clean.
- Integration: with no scaffolder registered, `GET scaffold` returns `available:false` and `POST`
  returns a graceful not-available response (no error, no partial writes); committing the wizard
  **without** scaffolding still leaves a complete, valid project (M6 unaffected).

---

## Risk notes

- **Determinism** — `Recommend`/`RankStacks` must avoid map iteration order and any clock; unit
  tests assert byte-identical repeat output (FR-12).
- **NFR-1 blast radius** — only M6 `commit` writes under `lifecycle/architecture/`; resume state
  (M4) lives outside it. A test enumerates the tree after abandon to prove zero writes.
- **Unbuilt dependencies** — standards seeding and scaffolding/agent-directives are gated behind
  seams (M7) and never block M1–M6; documented as cross-lineage dependencies, not deviations.

## Verification (end-to-end)

1. `make lint` clean.
2. `make test-unit` clean (config repair, recommend, summary, state, supersede unit tests).
3. `make test-integration` clean ([[onboarding-architecture-selection-5-test]]).
4. Manual smoke via `make run`: `GET …/architecture/wizard` shows questions + `prior_run:false`;
   `POST …/recommend` returns ranked candidates + why; `POST …/commit` (as product-owner) yields
   promoted copies + summary + ADR-0001; re-commit a different architecture → archived prior +
   new superseding ADR; abandon before commit → tree unchanged.

## Addendum (2026-08-17) — full-catalog listing endpoint

Resolves [[onboarding-architecture-selection-4-fe]] **OQ-6**: the frontend Browse step needs the
whole candidate catalog (every architecture + tech-stack, including `pros`/`cons`) before any pick,
which no endpoint returned — `recommend` needs answers, `stacks` needs a chosen architecture, and
`pros`/`cons` live only in `## Pros`/`## Cons` markdown bodies parsed by `LoadCatalog`.

- **Added** `GET /api/p/{project}/architecture/wizard/catalog` → `{ architectures: CatalogItem[],
  tech_stacks: CatalogItem[] }` — `handleListWizardCatalog` in
  [internal/http/architecture_wizard.go](internal/http/architecture_wizard.go), mounted in
  [internal/http/server.go](internal/http/server.go) beside the other `wizard/*` routes. Takes no
  inputs; auth-gated like the sibling reads; a thin marshal of `architecture.LoadCatalog()`
  (`CatalogItem` already JSON-tags `pros`/`cons`).
- **Test** `tests/integration/architecture_wizard_catalog_test.go` — full catalog returned with
  pros/cons and no query params; unauthenticated → 401.
