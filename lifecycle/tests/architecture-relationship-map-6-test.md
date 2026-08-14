---
title: "Integration Tests — Architecture Relationship Map"
type: test
status: in-qa
lineage: architecture-relationship-map
parent: lifecycle/test-plans/architecture-relationship-map-5-test.md
---

# Integration Tests — Architecture Relationship Map

Implements the test-developer's slice of [[architecture-relationship-map-5-test]] against the
backend ([[architecture-relationship-map-3-be]]) and frontend ([[architecture-relationship-map-4-fe]])
implementations: Milestones 3, 4, and 8 — the milestones that require true integration tests in
`tests/`. Milestones 1–2 (Go unit tests in `internal/artifact`, `internal/index`) and Milestone 3's
handler-level contract were already covered directly by the backend developer alongside their
commits (`internal/artifact/artifact_test.go`, `internal/index/architecture_map_test.go`,
`internal/http/graph_test.go`) — outside this write scope (`internal/`) and not duplicated here.
Milestones 5–7 (frontend Vue/TS behaviour tests under `web/src`) are the frontend developer's write
scope, not the test-developer's; see "Gaps found" below — no spec files exist there yet.

## Test files

| File | Milestone | Covers |
|---|---|---|
| `tests/integration/architecture_map_test.go` | 3 (FR-3, FR-8, FR-10) | `GET /architecture-map` through a live server: base-map shape (`labels` on nodes, `kind`+`label` on typed edges), `stack_for` add/omit behaviour, auth parity with `/graph` (401 unauthenticated), no mutating verb (405 on POST/PUT/DELETE/PATCH), and the same error-status contract as `/graph` for an unknown project. |
| `tests/integration/architecture_map_test.go` | 4 (FR-12) | Freshness via the real fsnotify watcher with no process restart: a new architecture file appears in the next `architecture-map` response; removing one drops its node and dangling edges; `/graph` for the same project is unaffected by the feature (non-goal). |
| `tests/e2e/flows/12-architecture-map-smoke.spec.ts` + `tests/e2e/fixtures/lifecycle/architecture/**` | 8 (NFR-2, NFR-3) | Playwright smoke test over a copy of the shipped catalog (10 architectures / 14 tech-stacks, copied from this repo's own `lifecycle/architecture/`): base map renders exactly one node per architecture, 2D↔3D toggling and the stack-reveal toggle complete within a bounded timeout with no page errors. |

## Scenarios covered

- **Endpoint shape (FR-3):** `{nodes, edges}` on 200; every node has a `labels` field; a typed
  `evolves_into` field produces an edge with `kind == "evolves_into"` and a non-empty `label`.
- **Stack ring (FR-8):** omitting `stack_for` returns the architecture-only base map (default-off);
  `?stack_for=<archId>` adds exactly that architecture's `related_to` tech-stack node(s) and
  connecting edge.
- **Auth & read-only (FR-10):** an unauthenticated request gets 401, same as `/graph`; `POST` /
  `PUT` / `DELETE` / `PATCH` on `/architecture-map` all get rejected (405 — chi's default
  method-not-allowed response, since no handler is registered for those verbs); an unknown project
  gets the same error status as `/graph` (404), never a 200 or an unhandled panic.
- **Freshness (FR-12):** a file written to `lifecycle/architecture/architectures/` after boot
  appears in `architecture-map` within 2s via the live watcher, no restart; removing a file drops
  its node and any edge that referenced it; `/graph`'s node count for the project is unchanged by
  querying `architecture-map` in between.
- **Full-catalog smoke (NFR-2, NFR-3):** against a real 10-architecture / 14-tech-stack catalog
  snapshot, the base map renders exactly 10 nodes; toggling 2D→3D→2D and revealing the stack ring
  for "Local Web-based Application" each complete within a 20s bound with no `pageerror` events.

## Known pre-existing gap hit while writing these tests

`lifecycle/architecture/` is not a configured `stage` (see
[[architectural-artefacts-6-test]], gap #1) — the startup scan silently skips anything placed there
before boot. This meant `newTestEnv`'s `seeds` mechanism (Go) and `spawnKaosControl`'s pre-boot
fixture copy (e2e) could not be used directly for architecture/tech-stack fixtures; both test files
instead write fixtures **after** the server starts and poll until the live fsnotify watcher indexes
them (a pattern the repo already uses in `tests/integration/architecture_lineage_test.go`). This
works reliably and, for Milestone 4, is exactly the behaviour under test anyway — but it means
Milestones 3/8 incidentally depend on the live-watch path rather than the startup-scan path. Not a
test bug; same root cause already tracked in [[architectural-artefacts-6-test]], not fixed here
(outside this write scope).

## Gaps found — flagging, not fixing (outside this write scope)

- **No frontend test coverage yet for Milestones 5–7.** `web/src/views/project/__tests__/ArchitectureMapView.spec.ts`,
  `web/src/components/map/__tests__/archMapStyle.spec.ts`, and
  `web/src/components/map/__tests__/ArchMapLegend.spec.ts` (nav/route/2D-3D equivalence,
  decision-signal encoding + legend, click-through + stack toggle + entry points) do not exist,
  despite the corresponding view/composable/style-helper/legend code having landed
  (`feat(architecture-map)` Milestones 1–6). This is the frontend developer's write scope
  (`web/src`), not the test-developer's (`tests/`, `lifecycle/tests/`) — flagging so it isn't
  mistaken for complete.
- **FR-11 (entry points) appears unimplemented.** The plan requires "the catalog `README` and the
  onboarding/project-create flow both contain a link into the map route." A search of
  `lifecycle/architecture/README.md` and the onboarding/project-create views found no reference to
  `architecture/map` or `architecture-map` — this looks like outstanding frontend work, not a test
  gap; a Milestone 7 test asserting it would currently fail. Not exercised here.
