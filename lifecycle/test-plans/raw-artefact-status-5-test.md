---
title: Test Plan — Add 'raw' Artefact Status Before Draft
type: plan-test
status: approved
lineage: raw-artefact-status
parent: lifecycle/requirements/raw-artefact-status-2.md
---

# Test Plan — Add `raw` Artefact Status Before Draft

Verifies the changes specified in [[raw-artefact-status]] (backend
state machine, frontend defaults and surfaces, indexer round-trip,
auto-block interaction). Unit-level coverage is owned by the backend
and frontend plans; this plan focuses on integration scenarios and the
non-functional checks (accessibility, no-regression sweep).

Cross-links: backend behaviour under test lives in
[[raw-artefact-status]] backend plan; frontend surfaces under test
live in [[raw-artefact-status]] frontend plan.

## Milestone 1 — Parser & vocabulary unit tests

**Description.** Pin the new status into the parser vocabulary with
unit-level coverage. These tests are the floor — every other test in
this plan presumes they pass.

**Files to change.**
- `internal/artifact/artifact_test.go` — table-driven case parsing a
  fixture with `status: raw` and asserting `len(ParseErrs) == 0` and
  `KnownStatuses["raw"] == true`.
- `internal/workflow/workflow_test.go` — coverage for the new
  transition rules per the backend plan's Milestone 2 matrix.

**Acceptance criteria.**
- `go test ./internal/artifact/... ./internal/workflow/...` all green.
- A `raw` artefact with no other unusual frontmatter produces zero
  parse errors and zero log warnings.

## Milestone 2 — Round-trip integration test (API + index + WS)

**Description.** A full integration test exercising the create → read
→ transition path end-to-end. Uses the project's existing test
harness (`testEnv` auto-logins as admin per project memory) and the
NDJSON run-log conventions where applicable.

Scenario:
1. Create a project via the test harness.
2. `POST /artifacts` with body `{ status: "raw", type: "idea", title:
   "...", lineage: "..." }`.
3. Assert 201 and parse the returned path.
4. `GET /artifacts/{path}` and assert `status == "raw"`.
5. Open a WebSocket to the project, subscribe to `artifact.indexed`,
   then transition the artefact `raw → draft` as role `analyst` via
   `PUT /artifacts/{path}` (or the dedicated transition endpoint).
6. Assert the WebSocket emits an `artifact.indexed` event whose
   payload reflects `status == "draft"`.
7. `GET /artifacts/{path}/allowed-targets` as role `analyst` while
   status is `raw` and assert the set contains `draft` and `blocked`
   and excludes `raw`.

**Files to change.**
- Add a new file under `tests/` (project convention is integration
  tests live there) — e.g. `tests/raw_status_test.go`. If a sibling
  workflow integration test already exists, extend it instead.
- Add a corresponding test artefact under `lifecycle/tests/` describing
  what this integration test covers (per project convention).

**Acceptance criteria.**
- `go test ./tests/... -run RawStatus` passes from a clean checkout.
- The test cleans up its project / temp files (no leftover state).
- The WebSocket assertion has a generous timeout (≥ 2 s) but does not
  rely on `time.Sleep` alone for synchronisation.

## Milestone 3 — Frontend unit / component tests

**Description.** Lock the brain-dump default and the badge styling
with frontend-level tests.

Scenarios:
1. **Brain-dump default.** A unit test for `brainDump.ts` (or a
   component test for `BrainDumpModal.vue`) submits the modal and
   asserts the API payload contains `status: 'raw'`.
2. **Status dropdown for `raw`.** A component test renders
   `StatusDropdown.vue` with a `raw` artefact and a mocked
   `allowed-targets` response of `["draft", "rejected", "abandoned",
   "blocked"]`, then asserts those four entries are rendered and
   `raw` is not.
3. **List view filter.** A component test mounts `ArtifactListView.vue`
   with three fixture artefacts (`raw`, `draft`, `done`) and asserts
   `raw` is visible by default and disappears when the "hide non-
   active" / "hide done" filter is toggled in the appropriate way.

**Files to change.**
- `web/src/stores/brainDump.test.ts` (or `.spec.ts` matching project
  convention) — new test file.
- `web/src/components/artifact/StatusDropdown.test.ts` — new or
  extended test file.
- `web/src/views/project/ArtifactListView.test.ts` — new or extended
  test file.

**Acceptance criteria.**
- `pnpm test` (or whichever runner the frontend uses) passes the new
  cases.
- Snapshot tests, if used, are updated and the diff is intentional.

## Milestone 4 — Auto-block-on-open-questions integration

**Description.** Verify resolved question #5: a `raw` artefact whose
body contains `## Open Questions` is auto-transitioned to `blocked`.
This is partly covered by the unit-level test in the backend plan
Milestone 4; this milestone exercises the same path through the live
indexer + watcher.

Scenario:
1. Boot a project via the test harness.
2. Write a markdown file under `lifecycle/ideas/` with frontmatter
   `status: raw, type: idea, title: ..., lineage: ...` and a body
   containing `## Open Questions\n- Q1\n`.
3. Allow the watcher to fire (poll for index update with a timeout).
4. Re-read the on-disk file and assert frontmatter now reads
   `status: blocked` and `assignees` contains `{role: product-owner,
   who: agent}`.
5. Remove the `## Open Questions` section, save, allow the watcher to
   fire.
6. Assert the artefact is now in `status: draft` (per the existing
   auto-unblock contract — `blocked → draft`, not back to `raw`).

**Files to change.**
- `tests/raw_autoblock_test.go` — new integration test (or extend an
  existing autoblock test file).

**Acceptance criteria.**
- The test passes consistently (run it three times in a row locally —
  flakiness on the watcher debounce is a known risk).
- A WebSocket subscriber observes both the auto-block and auto-unblock
  events in order.

## Milestone 5 — Accessibility check

**Description.** The new badge must meet WCAG AA contrast in both
themes (non-functional requirement #4).

Steps:
1. Open the artefact list view with at least one `raw` artefact in
   light theme. Use the Chrome/Firefox devtools accessibility inspector
   (or `axe-core`) to measure the badge contrast.
2. Switch to dark theme. Repeat.
3. Record the measured contrast ratio for both themes.

**Files to change.**
- None (manual check). Record the measured values in the test plan's
  status-checker output or in the PR description.

**Acceptance criteria.**
- Measured contrast ratio ≥ 4.5:1 for both light and dark themes.
- If either theme fails, file a defect rather than weakening the
  acceptance criterion — adjust the token values in the frontend plan
  Milestone 1 and re-measure.

## Milestone 6 — Regression sweep

**Description.** Confirm no existing transition test, workflow
integration, or list / kanban / dashboard test regresses. This is the
final gate before the test plan can be marked `approved`.

Steps:
1. `make lint` — green.
2. `make test-unit` — green (Go).
3. `pnpm --filter web test` — green (frontend).
4. `go test ./tests/...` — green (integration).
5. Manual smoke: brain-dump → list view → kanban → graph — each shows
   the `raw` artefact in the expected colour.
6. Manual smoke: existing `draft → clarifying → planning` lineage
   still transitions correctly (no rule was accidentally shadowed).

**Acceptance criteria.**
- All five automated suites green.
- The smoke checklist completes without surprise.
- No new TODO / `FIXME` comments introduced by this lineage's
  changes (grep the diff).

## Out of scope

- Performance benchmarking of the indexer with a large number of
  `raw` artefacts — defer until measured impact.
- Migration tooling for existing `draft` artefacts — explicit non-goal
  in the requirement.
- UX research on the chosen badge colour — the requirement specifies
  "desaturated grey or slate"; that is the design budget.
