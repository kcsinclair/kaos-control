---
title: "Test Plan — Architecture Wizard (Guided Selection)"
type: plan-test
status: in-development
lineage: onboarding-architecture-selection
parent: lifecycle/requirements/onboarding-architecture-selection-2.md
labels:
    - test
    - architecture
    - onboarding
    - wizard
release: KC-Release5
---

# Test Plan — Architecture Wizard (Guided Selection)

Verifies every acceptance criterion of [[onboarding-architecture-selection]] against the backend
([[onboarding-architecture-selection-3-be]]) and frontend
([[onboarding-architecture-selection-4-fe]]) plans. Layered: Go **unit** tests beside source,
Go **integration** tests under `tests/integration/` (build tag `integration`, harness
`newTestEnv` / `newTestEnvWithCfgYAML`, `env.doRequest`, `env.login`, `requireStatus`,
`readJSON`), and Vitest **component/store** tests under `web/src/**/__tests__/`. Each milestone
names the FR/AC it closes.

**Fixtures.** A catalog fixture seeding a representative subset of the real
`lifecycle/architecture/architectures/*.md` + `tech-stacks/*.md` (with `labels` + `related_to`),
plus a `lifecycle/config.yaml` carrying the default `architecture_wizard` question set. Reuse the
seed helpers in `tests/integration/architecture_promote_test.go`.

Cross-references:
- [[onboarding-architecture-selection-3-be]] · [[onboarding-architecture-selection-4-fe]]
- [[architectural-artefacts]] — promotion/ADR primitives exercised transitively.
- `lifecycle/tests/` — the artifact describing this test suite is authored alongside the code.

---

## Milestone 1 — Unit: recommendation engine determinism & rules (FR-8, FR-9, FR-11, FR-12, OQ-2)

### Description

Pure-function coverage of `internal/architecture` `Recommend` / `RankStacks` and catalog loading.

### Files to change

- **New** `internal/architecture/recommend_test.go` (and extend `catalog` loading tests).

### Acceptance criteria

- Hard constraint filters: `offline` excludes server-only architectures; `mobile` restricts to
  mobile-native. *(FR-8)*
- Over-constrained combo filters to zero → returns closest candidates + the exact
  **dropped-constraints** list. *(OQ-2)*
- Weak/empty answers → `modular-monolith` ranked first with the documented default-bias "why".
  *(FR-11)*
- Top result set is 2–3, each with a non-empty `why`. *(FR-9)*
- `Recommend` and `RankStacks` are **byte-identical across repeated calls** with identical inputs
  (no map-order/clock dependence). *(FR-12)*
- `RankStacks` returns only the chosen architecture's `related_to` stacks, language-matched first.
  *(FR-6, FR-10)*

---

## Milestone 2 — Unit: summary, ADR-0001, supersede, state store (FR-14, FR-15, FR-16, OQ-3, NFR-1/2/3)

### Description

Coverage of the writers and the resume store.

### Files to change

- **New** `internal/architecture/summary_test.go`, extend `adr_test.go`, **new**
  `internal/architecture/wizardstate_test.go`.

### Acceptance criteria

- `WriteSummary` writes one `architecture-summary.md` (`type: doc`) with the breaking-requirement
  → architecture/stack mapping, the Q&A trail, and resolvable links; writing twice is idempotent
  (one file, byte-identical). *(FR-14, FR-15, NFR-3)*
- `WriteADR0001` twice → exactly one `adr-0001-*.md`; a changed-selection path via `CreateADR` +
  `Supersede` marks the prior ADR `status: superseded` with a "Superseded by" pointer and never
  reuses a number. *(FR-16)*
- Wizard state save→load round-trips; clear removes it; the state path is asserted to be
  **outside** `lifecycle/architecture/`. *(OQ-3, NFR-1)*

---

## Milestone 3 — Unit: config question-set load & repair (FR-7, OQ-4)

### Description

`internal/config` coverage of the `architecture_wizard` section.

### Files to change

- **New/extend** `internal/config/*_test.go`.

### Acceptance criteria

- A config with no `architecture_wizard` is repaired to the built-in set with a `RepairNote`; the
  shipped `lifecycle/config.yaml` round-trips with an empty repair set. *(OQ-4)*
- Validation rejects >10 questions and invalid `kind` values (with notes). *(FR-7)*
- Each question exposes its plain-language prompt + label mapping to the API. *(FR-7)*

---

## Milestone 4 — Integration: read endpoints, prior-run detection, resume (FR-2, FR-3, FR-7, FR-9, FR-10, OQ-2, NFR-1)

### Description

HTTP-level coverage of the non-writing surface.

### Files to change

- **New** `tests/integration/architecture_wizard_read_test.go`.

### Acceptance criteria

- `GET …/architecture/wizard` on a fresh project returns the configured ≤10 questions and
  `prior_run.detected=false`. *(FR-7, FR-2)*
- After a committed selection, `GET …/architecture/wizard` returns `prior_run.detected=true` with
  the existing architecture/stack/ADR/summary references. *(FR-2, FR-3)*
- `POST …/wizard/recommend` is deterministic across repeated identical requests, honours hard
  filters, and returns the OQ-2 dropped-constraints fallback when over-constrained. *(FR-9, FR-12, OQ-2)*
- `GET …/wizard/stacks` returns only `related_to` stacks, language-ranked. *(FR-6, FR-10)*
- `PUT`/`GET`/`DELETE …/wizard/state` round-trips and writing state leaves **zero** files under
  `lifecycle/architecture/` (enumerate the tree). *(OQ-3, NFR-1)*

---

## Milestone 5 — Integration: commit orchestration, re-run, idempotency, role gate (FR-13–FR-16, NFR-1, NFR-2, OQ-5)

### Description

HTTP-level coverage of the single write path and its guarantees.

### Files to change

- **New** `tests/integration/architecture_wizard_commit_test.go`.

### Acceptance criteria

- First commit (as product-owner) promotes the chosen architecture + stack (root copies with
  `parent:`), writes `architecture-summary.md` (breaking-reqs + mapping + Q&A), and writes
  `adr-0001-*.md` titled "Adopt <arch> with <stack>" with the Q&A trail + rejected alternatives;
  all appear in `GET …/artifacts` (re-index fired). *(FR-13, FR-14, FR-15, NFR-2)*
- Abandoning before commit (no commit call) leaves **no** files under `lifecycle/architecture/`
  beyond the pre-existing catalog. *(NFR-1)*
- Same-selection re-commit → exactly one of each file, no duplicates/orphans (idempotent). *(NFR-2)*
- Changed-selection re-commit → prior promoted copies archived, a **new superseding ADR** written,
  the prior ADR marked `superseded`, and the summary refreshed. *(FR-16)*
- A non-product-owner user receives `403` from `POST …/wizard/commit`. *(OQ-5)*

---

## Milestone 6 — Integration: scaffolding hand-off seam degrades gracefully (FR-17, FR-18)

### Description

Confirms the opt-in scaffolding step never breaks the core flow while its generators are unbuilt.

### Files to change

- **New** `tests/integration/architecture_wizard_scaffold_test.go`.

### Acceptance criteria

- With no scaffolder registered, `GET …/wizard/scaffold` returns `available:false` and
  `POST …/wizard/scaffold` returns a graceful not-available response with no partial writes.
  *(FR-17)*
- A wizard committed **without** scaffolding still yields a complete, valid project
  (promoted files + summary + ADR present, no scaffold artefacts). *(FR-17, FR-18)*

---

## Milestone 7 — Frontend: store + component/step coverage (FR-1, FR-3, FR-4, FR-5, FR-7–FR-13, NFR-4, NFR-5)

### Description

Vitest coverage of the wizard store and each step component, including the less-technical UX
guarantees.

### Files to change

- **New** `web/src/stores/__tests__/architectureWizard.test.ts` and per-component tests under
  `web/src/components/architecture/__tests__/`.

### Acceptance criteria

- Store: `start()` loads questions + prior-run + resumable state; `commit()` posts architecture +
  stack + answers + Q&A; server errors surface without throwing. *(FR-1)*
- `PriorRunGate` blocks advancement until Continue/Exit; Exit issues no commit. *(FR-3, NFR-1)*
- `PathChoiceStep` offers Browse/Guided and a persistent "show me everything anyway" that switches
  Guided→Browse preserving answers. *(FR-4)*
- `BrowseCatalogStep` renders one card per architecture with labels/pros/cons and handles an empty
  catalog with guidance (not a crash). *(FR-5)*
- `GuidedQuestionStep` renders exactly the configured (≤10) questions, each Skippable with a
  "decide for me" default; skipping omits the answer. *(FR-7, NFR-5)*
- `RecommendationStep` renders 2–3 candidates each with a visible "why" and a dropped-constraints
  banner when present; override reaches any catalog item. *(FR-9, NFR-4, OQ-2)*
- `StackChoiceStep` shows only compatible stacks, language-ranked, with confirm/override. *(FR-10)*
- `ConfirmStep` shows architecture + stack + standards and issues **no** write until "Confirm &
  write". *(FR-13, NFR-1)*
- Sidebar/route wizard entry is gated to product-owner. *(FR-1, OQ-5)*

---

## Milestone 8 — E2E happy paths (both routes) + artifact write-up

### Description

Full-stack smoke over both paths, plus the `lifecycle/tests/` artifact describing what this suite
covers.

### Files to change

- **New** `tests/integration/architecture_wizard_e2e_test.go` (or extend the frontend E2E suite),
  and **new** `lifecycle/tests/onboarding-architecture-selection-test.md` (`type: test`)
  documenting coverage → AC mapping.

### Acceptance criteria

- **Guided E2E:** start → answer/skip → recommendation (with why) → confirm architecture → pick
  ranked stack → confirm → commit → promoted files + summary + ADR-0001 exist and index. Covers
  the deterministic-ranking, confirm-before-write, and persistence ACs end-to-end.
- **Browse E2E:** start → Browse → pick architecture → pick compatible stack → confirm → commit →
  same persisted outcome. Covers FR-4/FR-5/FR-6 end-to-end.
- **Re-run E2E:** re-launch → prior-run gate → change selection → commit → archived prior +
  superseding ADR + refreshed summary. *(FR-16)*
- The `lifecycle/tests/` artifact enumerates each AC and the test that closes it.

## Verification

1. `make test-unit` green (Milestones 1–3).
2. `make test-integration` green (Milestones 4–6, 8).
3. `pnpm test` green (Milestone 7).
4. `make lint` clean.
