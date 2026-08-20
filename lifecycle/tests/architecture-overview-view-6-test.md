---
title: Architecture Overview View — Overview Endpoint Integration Tests
type: test
status: approved
lineage: architecture-overview-view
parent: lifecycle/test-plans/architecture-overview-view-5-test.md
created: "2026-08-20T19:00:00Z"
---

# Architecture Overview View — Overview Endpoint Integration Tests

Implements Milestone T2 (and the integration-testable slice of Milestone T6)
of [[architecture-overview-view]] (test plan). Scope: `GET
/api/p/{project}/architecture/overview` exercised end to end through a live
server (`tests/integration`'s `testEnv`), against the already-implemented
backend (`internal/architecture/overview.go`, `internal/http/architecture.go`
`handleArchitectureOverview` — backend plan status `done`).

Milestones T1 (backend unit tests) and T3–T5 (frontend Vitest suites) are
owned by the backend-developer and frontend-developer roles respectively per
`AGENTS.md`; this artifact covers only the `tests/` integration slice
(test-developer scope).

## File

- `tests/integration/architecture_overview_test.go` (new, `//go:build
  integration`)

## Scenarios covered

- **`TestArchitectureOverview_PopulatedModel_ClassifiesAndOrders`** — seeds
  catalog candidates, promotes an architecture + tech-stack pair through the
  real `POST .../architecture/promote` endpoint, creates two ADRs through the
  real `POST .../architecture/adrs` endpoint, and seeds a summary, a
  standard, and an archived file. Asserts: the promoted root items classify
  as `chosen-architecture`/`chosen-stack` and never as `catalog`; the
  untouched catalog source files (promotion doesn't mutate them) still
  classify as `catalog`; summary/standard/archive items carry the right
  `catalog_role`; ADRs come back strictly newest-first by number (FR-7).
  Also pins NFR-3 with a generous 2s bound. Covers FR-2–FR-7, FR-9,
  M-B1/M-B2, and (by exercising promote/ADR-create against the endpoint) the
  M-B3 regression concern that those flows keep working.
- **`TestArchitectureOverview_DegradedEmptyModel_Never500`** — an empty
  project (nothing promoted, no summary/standards/ADRs) returns `200` with
  `has_chosen_architecture=false`, a null summary, and `standards`/`adrs`/
  `archive`/`catalog` present as empty JSON arrays (asserted via a type
  assertion that fails if the field decoded as `null` instead) — pins FR-10/
  NFR-5's "empty, non-nil slices, no error" contract.
- **`TestArchitectureOverview_RequiresAuth`** — unauthenticated request →
  401, mirroring the existing `TestArchitectureMap_RequiresAuth` pattern.
- **`TestArchitectureOverview_NoEditorRoleRequired`** — a user holding only
  the `reviewer` role (via a custom project config,
  `reviewerOnlyCfgYAML`) can read the overview (`200`) but is `403`'d by the
  same project's `POST .../architecture/promote` and `POST
  .../architecture/adrs`, which are gated on `RolesArtifactEditors`. Proves
  the overview endpoint carries no role gate, as the backend plan specifies.
- **`TestArchitectureOverview_NoWriteSideEffect`** — three consecutive GETs
  leave the seeded architecture file's bytes and mtime unchanged, and the
  `/artifacts` item count identical before/after (NFR-2).
- **`TestArchitectureOverview_DiskChangeReflectedWithoutRebuild`** — adding
  and then removing a standard file directly on disk after boot is reflected
  on the very next request with no watcher wait and no manual rebuild step,
  because `LoadOverview` enumerates `lifecycle/architecture/**` directly off
  disk on every call (unlike the index-backed `/architecture-map`). Pins
  FR-12 at the model layer per backend plan Milestone B3.

## Conformance notes

- No new test dependency or harness (NFR-1) — reuses `tests/integration`'s
  existing `testEnv` (real project fixtures, admin auto-login) and the
  `go test -tags integration` suite.
- NFR-6 (no client-IP-derived behaviour,
  [[adr-no-header-based-client-ip-trust]]) is satisfied by code inspection:
  `handleArchitectureOverview` (`internal/http/architecture.go`) reads only
  `projectFromCtx(r.Context())` and calls `architecture.LoadOverview` — no
  header or remote-IP is read anywhere in the handler or `LoadOverview`. No
  synthetic header-spoof test was added since there is no such code path to
  exercise.
- NFR-4 (a11y: newest-first ADR order surfaced without colour-only
  signalling) and the remainder of NFR-5's per-panel degradation are UI
  concerns and belong to Milestone T3's Vitest suites
  (`web/src/**/__tests__`), not this integration layer.
- `internal/architecture/overview_test.go` and
  `internal/http/architecture_overview_test.go` already cover `LoadOverview`
  and the handler in isolation (Milestones B1/B2 of the backend plan); this
  file exercises the same contract end to end through a live server, per the
  test plan's Milestone T2.

## Verification

- `go build -tags integration ./tests/...` — clean.
- `go vet -tags integration ./...` — clean.
- `go test -tags integration ./tests/integration/... -run TestArchitectureOverview -v` —
  all 6 tests pass.
- `go test -tags integration ./tests/integration/...` (full suite) — no
  regressions.
