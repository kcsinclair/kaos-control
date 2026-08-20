---
created: "2026-08-15T18:03:35+10:00"
title: "Integration Tests — Architecture Wizard (Guided Selection)"
type: test
status: approved
lineage: onboarding-architecture-selection
parent: lifecycle/test-plans/onboarding-architecture-selection-5-test.md
---

# Integration Tests — Architecture Wizard (Guided Selection)

Covers the Architecture Wizard backend surface: read/detect/recommend/stacks/state
(Milestone 4), the single write path — commit, idempotency, re-run/supersede,
role gate (Milestone 5) — and the scaffolding hand-off seam (Milestone 6) of
[lifecycle/test-plans/onboarding-architecture-selection-5-test.md](../test-plans/onboarding-architecture-selection-5-test.md).

Go **unit** tests for the recommendation engine, ADR/summary writers, and
wizard state store (Milestones 1–3) already exist beside their source —
written by backend-developer alongside
[onboarding-architecture-selection-3-be](../backend-plans/onboarding-architecture-selection-3-be.md):
`internal/architecture/recommend_test.go`, `summary_test.go`, `adr_test.go`,
`wizardstate_test.go`, `promote_test.go`, `supersede_test.go`. Not duplicated
here.

All tests below carry the `//go:build integration` tag. Run with:

```sh
go test -tags integration ./tests/integration/... -run TestWizard -v
```

---

## Test files

| File | Milestone |
|---|---|
| `tests/integration/architecture_wizard_read_test.go` | 4 |
| `tests/integration/architecture_wizard_commit_test.go` | 5 |
| `tests/integration/architecture_wizard_scaffold_test.go` | 6 |

---

## Scenarios covered

### Milestone 4 — read endpoints, prior-run detection, resume (`architecture_wizard_read_test.go`)

- **TestWizardGet_FreshProject_ReturnsQuestionsAndNoPriorRun** — `GET
  /architecture/wizard` on a project with no `architecture_wizard` config
  section returns the config-repaired built-in question set (≤10 questions),
  `default_architecture: "modular-monolith"`, and `prior_run.detected: false`.
  *(FR-7, FR-2)*
- **TestWizardGet_AfterCommittedSelection_DetectsPriorRun** — after a commit,
  `GET /architecture/wizard` reports `prior_run.detected: true` with the
  architecture/tech-stack/ADR/summary paths populated. *(FR-2, FR-3)*
- **TestWizardRecommend_DeterministicAndHonoursHardFilter** — two identical
  `POST /wizard/recommend` calls return byte-identical bodies; a "mobile: yes"
  hard-constraint answer restricts the result to the one catalog architecture
  carrying the `mobile` label. *(FR-9, FR-12, FR-8)*
- **TestWizardRecommend_OverConstrained_ReturnsDroppedConstraintsFallback** —
  against a fixture with no `mobile`-labelled architecture, answering
  "mobile: yes" filters to zero, so the endpoint relaxes that constraint and
  reports it in `dropped_constraints`, falling back to the full candidate set.
  *(OQ-2)*
- **TestWizardStacks_RelatedOnlyAndLanguageRanked** — `GET /wizard/stacks`
  returns only the chosen architecture's `related_to` stacks, with the
  language-matching stack ranked first; a stack unrelated to the chosen
  architecture never appears. *(FR-6, FR-10)*
- **TestWizardState_RoundTripsAndWritesNothingUnderLifecycleArchitecture** —
  `PUT`/`DELETE /wizard/state` round-trip through `GET /architecture/wizard`'s
  `resumable_state` field, and neither call ever writes a file under
  `lifecycle/architecture/` (file count enumerated before/after). *(OQ-3, NFR-1)*

### Milestone 5 — commit orchestration, re-run, idempotency, role gate (`architecture_wizard_commit_test.go`)

- **TestWizardCommit_FirstRun_PromotesWritesSummaryAndADR0001** — first commit
  (as product-owner) promotes the chosen architecture + stack to the
  `lifecycle/architecture/` root, writes `architecture-summary.md` and
  `adr-0001-*.md` (containing the Q&A trail), and all four artifacts appear in
  `GET /artifacts` (re-index fired). *(FR-13, FR-14, FR-15, NFR-2)*
- **TestWizardCommit_AbandonedBeforeCommit_LeavesNoFiles** — reading the
  wizard, fetching recommendations, and saving in-progress state leave no
  files under `lifecycle/architecture/` beyond the pre-seeded catalog. *(NFR-1)*
- **TestWizardCommit_SameSelectionRecommit_Idempotent** — recommitting the
  identical selection produces exactly one promoted architecture copy, one
  promoted stack copy, one summary, and one ADR file — no duplicates. *(NFR-2)*
- **TestWizardCommit_ChangedSelectionRecommit_ArchivesAndSupersedes** —
  committing a different architecture/stack archives both prior promoted
  copies, writes a new, distinct ADR, marks the prior `adr-0001-*` file
  `status: superseded` with a "Superseded by" pointer, and refreshes the
  summary to the new selection. *(FR-16)*
- **TestWizardCommit_NonProductOwner_Returns403** — a user holding only the
  `qa` role is denied `POST /wizard/commit`. *(OQ-5)*

### Milestone 6 — scaffolding hand-off seam (`architecture_wizard_scaffold_test.go`)

- **TestWizardScaffoldGet_NoScaffolderRegistered_ReturnsUnavailable** — with no
  `Scaffolder` registered (the current state — no generator package under
  `internal/architecture.RegisterScaffolder` exists yet), `GET
  /wizard/scaffold` returns `available: false` with an explanatory message.
  *(FR-17)*
- **TestWizardScaffoldPost_NoScaffolderRegistered_GracefulNoWrites** — `POST
  /wizard/scaffold` under the same condition returns the same graceful
  not-available response and writes nothing under `lifecycle/architecture/`.
  *(FR-17)*
- **TestWizardCommit_WithoutScaffolding_YieldsCompleteProject** — a commit
  made without ever calling the scaffold endpoints still produces the
  complete, valid outcome (promoted files + summary + ADR-0001). *(FR-17, FR-18)*

### Milestone 7 — frontend store + component/step coverage

**Partially covered, elsewhere; the remainder is not implemented.**

- The wizard store (`start()`, answer/skip mutation, `commit()`, error
  surfacing) is already covered by
  `web/src/stores/__tests__/architectureWizard.spec.ts`.
- `PriorRunGate` (blocks advancement until Continue/Exit; Exit issues no
  commit) is already covered by
  `web/src/views/project/__tests__/ArchitectureWizardView.spec.ts`.
- `PathChoiceStep`, `BrowseCatalogStep`, `GuidedQuestionStep`,
  `RecommendationStep`, `StackChoiceStep`, and `ConfirmStep` **do not exist**
  in `web/src/components/architecture/` — `ArchitectureWizardView.vue`'s step
  body is still the Milestone 2 placeholder. The frontend plan
  ([onboarding-architecture-selection-4-fe](../frontend-plans/onboarding-architecture-selection-4-fe.md))
  is itself `status: blocked` on OQ-6 (no backend catalog-listing endpoint for
  Browse), so none of Milestone 3 onward was built. No tests were written
  against these components — there is nothing to test yet. See this test
  plan's own Open Questions for the corresponding blocker.
- "Sidebar/route wizard entry is gated to product-owner": the sidebar nav item
  is already conditionally rendered (`AppSidebar.vue`'s `hasProductOwnerAccess`
  check), but the route itself carries `meta: { roles: ['product-owner'] }`
  with **no enforcement** — `router.beforeEach` only checks `requiresAuth`,
  and unlike `DevOpsView.vue` (which self-guards with an access-denied state),
  `ArchitectureWizardView.vue` has no client-side role check at all. This is
  the same class of gap as the missing step components — deferred, not tested
  against a spec that doesn't match the code.

### Milestone 8 — E2E happy paths (both routes)

**Not implemented.** Both the Guided and Browse E2E paths require driving the
wizard through its step UI, which does not exist (see Milestone 7 above). The
backend-only flow (recommend → stacks → commit) is already exercised at the
HTTP level by Milestone 4/5 tests above; a true UI-driven E2E cannot be
written until the frontend plan's OQ-6 is resolved and Milestones 3–7 of that
plan land.

---

## Notes

- All tests use the real `newTestEnv` HTTP-level harness (SQLite index, git
  repo, full router) — no mocking. `wizardCatalogSeeds()` seeds a
  representative subset of the real shipped catalog (`modular-monolith`,
  `mobile-native`, `static-site` architectures; `go-vue`, `python-fastapi`,
  `flutter`, `static-html-js` stacks) with the same labels/`related_to` edges
  as production, so the default (self-repaired) question set exercises
  realistic filtering/ranking.
- `make test-integration` (or `go test -tags integration ./tests/... -run
  TestWizard`) is green for all Milestone 4–6 tests above.
