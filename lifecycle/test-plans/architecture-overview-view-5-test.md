---
title: Architecture Overview View — Test Plan
type: plan-test
status: draft
lineage: architecture-overview-view
parent: lifecycle/requirements/architecture-overview-view-2.md
created: "2026-08-20T09:10:00Z"
---

# Architecture Overview View — Test Plan

Verifies the backend plan [[architecture-overview-view]] (be) and frontend plan
[[architecture-overview-view]] (fe) against every acceptance criterion in
[[architecture-overview-view]] (the requirement). Reuses the existing test
infrastructure: Go `go test` (unit + `-tags integration` in `tests/`) and the
Vue/Vitest component/unit suites under `web/src`. No new test dependency (NFR-1).

## Conformance to recorded architecture

- Tests live in the existing suites (`internal/**_test.go`, `tests/` integration,
  `web/src/**/__tests__`) — no new harness, service, or dependency (NFR-6).
- Integration tests use the existing `testEnv` (admin auto-login, real project
  fixtures) per the project memory note; read endpoints are asserted through the
  established auth/session path — no client-IP/header trust
  ([[adr-no-header-based-client-ip-trust]]).

## Milestone T1 — Backend model + classification (verifies M-B1)

**Description.** Unit-test `LoadOverview` / `classifyRole` over fixture projects
covering the full model and each degraded shape.

**Files to change.**
- `internal/architecture/overview_test.go` (owned by backend plan; this plan
  specifies the cases): full model; empty `standards/`; no ADRs; no chosen
  architecture; archive present; a `catalog`-labelled candidate.

**Acceptance criteria.**
- Every catalog-role is assigned correctly; promoted-root architecture/stack are
  `chosen-*` and never `catalog`; `standards/*` and `architecture-summary.md`
  classify by **path** despite `type: doc` (OQ-2); `archive/*` → `archive`.
- ADRs come back strictly newest-first by ADR number (FR-7).
- No-chosen-architecture fixture returns `has_chosen_architecture=false`, null
  summary, empty (non-nil) slices, and **no error** (FR-10).

## Milestone T2 — Overview endpoint integration (verifies M-B2, M-B3)

**Description.** Integration tests in `tests/` driving `GET
/api/p/{project}/architecture/overview` against real project fixtures via
`testEnv`.

**Files to change.**
- `tests/architecture_overview_test.go` (new, `//go:build integration`).

**Acceptance criteria.**
- Populated fixture → `200` with the classified model matching the fixture
  (FR-2…FR-7, FR-9 role tags present).
- Empty/unpromoted fixture → `200` with the degraded model, never `500`
  (FR-10/NFR-5).
- Route requires auth but not an editor role: authenticated non-editor read
  succeeds (NFR-2).
- No write side effect: index/artifact bytes unchanged after the call (NFR-2).
- Disk change (add a standard / new ADR) is reflected on the next request with no
  manual rebuild (FR-12); promotion + ADR-create still emit `artifact.indexed`.

## Milestone T3 — Overview view, panels & empty states (verifies F1–F3)

**Description.** Vitest component/unit tests for the composable, view, and panels.

**Files to change.**
- `web/src/composables/__tests__/useArchitectureOverview.spec.ts` (new).
- `web/src/views/project/__tests__/ArchitectureOverviewView.spec.ts` (new).
- `web/src/components/architecture/overview/__tests__/*.spec.ts` (new, per panel).

**Acceptance criteria.**
- Composable fetches on mount; a mocked `artifact.indexed` / `file.changed` event
  triggers exactly one `reload()` (FR-12); API error sets `error`, no throw.
- Panels render FR-2…FR-7 from a model fixture with working click-through routes;
  ADR list is newest-first with title/status/date and no colour-only signalling
  (FR-7, NFR-4).
- Tech-stack mapping renders the hard `related_to`/wiki-link references (OQ-3);
  summary panels render sections as-is with links (OQ-1).
- No chosen architecture → view-level empty state (wizard + map links), no error;
  missing summary / empty standards / no ADRs → only the affected panel shows an
  absent state (FR-10/NFR-5).

## Milestone T4 — Routing, section default & navigation actions (verifies F4)

**Description.** Router/guard and action-bar tests.

**Files to change.**
- `web/src/router/__tests__/architectureSectionDefault.spec.ts` (new).
- Extend `ArchitectureOverviewView.spec.ts` for the action bar.

**Acceptance criteria.**
- Chosen architecture present → Architecture section resolves to the overview
  route; absent → resolves to the relationship map, and the overview URL still
  renders the empty state (FR-1, FR-10). Overview has its own URL in both cases.
- One-click links to relationship map and wizard resolve to their routes; "raise
  a new ADR" opens `NewAdrModal.vue` (FR-8).

## Milestone T5 — Zone ownership + demoted list/board toggle (verifies F5)

**Description.** Tests for the broadened zone predicate and the off-by-default
list/board behaviour, plus the archive strip.

**Files to change.**
- `web/src/types/__tests__/isCatalogMaterial.spec.ts` (update/rename to the
  broadened predicate).
- `web/src/views/project/__tests__/ArtifactListView.*` and a `useKanbanBoard`
  spec — assert default exclusion.
- `web/src/components/architecture/overview/__tests__/ArchiveStrip.spec.ts` (new).

**Acceptance criteria.**
- Default list/board exclude the **whole** architecture zone — candidates, chosen
  architecture, ADRs, standards, summary, archive (FR-9a); enabling "show
  architecture inline" reveals them.
- The predicate is role/zone-based, not `type`-based (FR-9): a `type: adr` or
  `type: architecture` file under `lifecycle/architecture/` is excluded by
  default.
- Archive strip shows ≤10 items and is collapsed on open (OQ-5).

## Milestone T6 — Non-functional gates (verifies NFR-1…NFR-6)

**Description.** Cross-cutting assertions folded into the suites above.

**Acceptance criteria.**
- **NFR-1**: no new frontend/backend runtime dependency — `go.mod` and
  `web/package.json` runtime deps unchanged in the diff (CI/diff check).
- **NFR-2**: no artifact-content write from the view or endpoint (asserted in T2).
- **NFR-3**: overview endpoint responds promptly for the curated model
  (single arch/stack, tens of standards/ADRs) — assert well under a generous
  bound in the integration test.
- **NFR-4**: a11y — newest-first ADR order + non-colour-only status (asserted in
  T3).
- **NFR-5**: graceful degradation across all missing/empty parts (T1–T3).
- **NFR-6**: endpoint honours the existing auth path, adds no client-IP behaviour
  ([[adr-no-header-based-client-ip-trust]]) — no header-derived branch in the new
  handler (code + test review).

## Out of scope (test)

- No tests for the wizard flow ([[onboarding-architecture-selection]]) or the
  catalog relationship graph ([[architecture-relationship-map]]) beyond the
  one-click navigation links.
- No auto-diagram tests — deferred with the feature ([[architecture-auto-diagram]],
  FR-11); only assert the layout reserves space.
