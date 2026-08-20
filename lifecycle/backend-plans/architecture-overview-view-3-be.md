---
title: Architecture Overview View — Backend Plan
type: plan-backend
status: in-development
lineage: architecture-overview-view
parent: lifecycle/requirements/architecture-overview-view-2.md
created: "2026-08-20T09:00:00Z"
---

# Architecture Overview View — Backend Plan

Backend companion to the frontend plan [[architecture-overview-view]] (fe) and
test plan [[architecture-overview-view]] (test). The view is read-mostly and
derives entirely from the artifacts on disk; the backend's job is to **assemble
and classify** the architecture zone into one read-only model so the frontend
does not have to re-implement path/role discovery or newest-first ordering, and
to keep the freshness contract (FR-12) on the existing index/watcher path.

## Conformance to recorded architecture

- Stays inside the **modular monolith** / **Go + Vue** stack. New code is a
  read-only aggregation in the existing `internal/architecture` package plus one
  handler in `internal/http` — no new service, dependency, or persistence store
  (NFR-1, NFR-6).
- Adds **no** endpoint that derives behaviour from client IP or request headers;
  the handler follows the existing auth/session read path only, honouring
  [[adr-no-header-based-client-ip-trust]] (NFR-6).
- No new ADR is required — the requirement's own Architecture-Breaking analysis
  concluded none is needed, and this plan introduces no deviation. If review
  finds the aggregate endpoint should instead be assembled client-side from the
  existing artifacts index (a genuine architecture choice), that is a design
  note, not a break — record it in this plan, do not deviate silently.

## Milestone B1 — Overview model + catalog-role classification

**Description.** Add a pure, read-only assembler in `internal/architecture` that
loads the whole architecture zone and classifies each artifact by **catalog-role**
(FR-9), not by artifact `type`. Roles: `catalog` (candidate `architectures/` +
`tech-stacks/` carrying the `catalog` label), `chosen-architecture` and
`chosen-stack` (promoted `type: architecture` / `type: tech-stack` at the
`lifecycle/architecture/` root), `summary` (`architecture-summary.md`, located by
**path** per OQ-2), `standard` (`standards/*`, by path per OQ-2), `adr`
(`decisions/*`, `type: adr`), and `archive` (`archive/*`, superseded promotions).
Discovery reuses the existing SQLite index rows for everything under
`lifecycle/architecture/` (falling back to disk enumeration of `decisions/`,
`standards/`, and `archive/` so an unindexed-but-present file still appears).

The model carries, per item: repo-relative `path`, `title`, `status`, `type`,
`created`/date, and `catalog_role`. ADRs are sorted **newest-first** (FR-7) by
ADR number descending (parsed from the `adr-NNNN-` filename via the existing
`adrNumberRe`), tie-broken by `created`. The model exposes booleans/absence for
graceful degradation (FR-10/NFR-5): `has_chosen_architecture`, `summary` may be
null, `standards`/`adrs`/`archive` may be empty. **No panel content is inlined**
— bodies are fetched by the frontend through the existing `GET /artifacts/*path`
(NFR-1); the model returns only classification + light metadata.

**Files to change.**
- `internal/architecture/overview.go` (new) — `type Overview struct{…}`,
  `type OverviewItem struct{…}`, `func LoadOverview(projectRoot string, idx *index.Index) (Overview, error)`, and an unexported `classifyRole(relPath string, fm Frontmatter) Role` helper. Reuse `LoadCatalog`, the promoted-root detection used by `promote.go`/`stackprofile.go`, and `adrNumberRe`.
- `internal/architecture/overview_test.go` (new) — table tests for
  `classifyRole` and `LoadOverview` fixtures (full model; empty `standards/`; no
  ADRs; no chosen architecture; archive present).

**Acceptance criteria.**
- Given a fixture project with a promoted architecture + stack, a summary, N
  standards, M ADRs, and K archived files, `LoadOverview` returns exactly those,
  each with the correct `catalog_role`, and `adrs` ordered strictly by descending
  ADR number.
- Catalog candidates (label `catalog`) classify as `catalog`; the promoted root
  architecture/stack classify as `chosen-*` and are **never** `catalog`; files
  under `archive/` classify as `archive`; `standards/*` and
  `architecture-summary.md` classify by path even though their `type` is `doc`
  (OQ-2).
- With no promoted architecture, `has_chosen_architecture` is false, `summary` is
  null, and `standards`/`adrs` are empty slices (not nil-panics); `LoadOverview`
  returns no error (FR-10).
- The assembler performs **no writes** and opens no artifact for mutation
  (NFR-2); `go test ./internal/architecture/...` passes.

## Milestone B2 — Read-only overview endpoint

**Description.** Expose the model at `GET /api/p/{project}/architecture/overview`
returning the assembled JSON. Read-only: requires an authenticated session but no
editor role (mirrors `GET /architecture-map`, not the promote/ADR-create
writers). Malformed/absent parts degrade to empty/null in the payload rather than
5xx (NFR-5). Response includes a `catalog_role` on every item so the frontend and
the list/board exclusion (FR-9a) share one discriminator vocabulary.

**Files to change.**
- `internal/http/architecture.go` — add `handleArchitectureOverview` calling
  `architecture.LoadOverview(p.Entry.Path, p.Idx)` and `writeJSON`. Follow the
  `handleArchitectureMap` shape (project-from-ctx guard, no editor-role gate).
- `internal/http/server.go` — register `r.Get("/architecture/overview",
  s.handleArchitectureOverview)` in the project subrouter, next to the other
  `architecture/*` GETs (around line 271–284 / 328).
- `internal/http/architecture_overview_test.go` (new) — handler test asserting
  200 + shape for a populated fixture and 200 + degraded shape for an empty one;
  assert no write side effects.

**Acceptance criteria.**
- `GET …/architecture/overview` returns `200` with the classified model for a
  populated project and `200` with an empty/degraded model when nothing is
  promoted (never `500` for absent parts).
- The route requires auth but not an editor role; an authenticated non-editor
  reader receives the model (read-mostly, NFR-2).
- The endpoint issues no artifact-content write and leaves the index unchanged
  (verified by the handler test).
- `go vet ./...` and `go test ./internal/http/...` pass.

## Milestone B3 — Freshness on the existing index/watcher path (FR-12)

**Description.** Confirm and lock in that the overview reflects on-disk changes
**without** new plumbing: because the endpoint re-assembles on every request and
the frontend re-fetches on `artifact.indexed` / `file.changed` WS events (see the
fe plan, mirroring `useArchitectureMap`), the existing fsnotify → index →
WebSocket path already satisfies FR-12. The one gap to close: ensure a change to
a **promoted root artifact, the summary, a standard, or an ADR** produces one of
those broadcast events. Promotion and ADR-creation already broadcast
`artifact.indexed` (`handlePromoteArchitecture`, `handleCreateADR`); external/disk
edits are covered by the watcher. Add a regression test rather than new code if
the watcher already covers `lifecycle/architecture/**`.

**Files to change.**
- `internal/watcher/*` — **only if** a test shows `lifecycle/architecture/**`
  (esp. `standards/`, `decisions/`, `archive/`) is not watched; otherwise no
  change. Document the finding in this plan either way.
- `internal/http/architecture_overview_test.go` — extend with a
  disk-change → re-`LoadOverview` assertion (add a standard, re-load, see it
  appear) to pin FR-12 at the model layer.

**Acceptance criteria.**
- Adding/removing/changing a promoted artifact, the summary, a standard, or an
  ADR on disk is observable on the next `LoadOverview`/endpoint call with no
  manual rebuild step (FR-12).
- Promotion and ADR creation continue to emit `artifact.indexed` (existing tests
  still green); the watcher covers the architecture subtree (asserted or fixed).
- No polling, no new persistence, no new dependency introduced (NFR-1).

## Out of scope (backend)

- No editing/authoring endpoints for architecture content (NFR-2) — raise-ADR
  and re-run-wizard already exist (`handleCreateADR`, wizard routes) and are the
  only mutating flows the view links to.
- No auto-diagram/codebase analysis endpoint — deferred to
  [[architecture-auto-diagram]] (FR-11); the model just needn't preclude it.
- No change to `type` vocabulary or the on-disk artefact model — that is
  [[architectural-artefacts]]; this plan is a pure consumer.
