---
title: "Tests: 'raw' artefact status"
type: test
status: done
lineage: raw-artefact-status
parent: lifecycle/test-plans/raw-artefact-status-5-test.md
---

# Tests: 'raw' artefact status

Integration and unit tests verifying the `raw` artefact status feature: parser
vocabulary, workflow transition rules, API round-trip, WebSocket events, auto-block
behaviour, and frontend component coverage.

## Test files

- `tests/integration/raw_status_test.go` — API round-trip and WebSocket integration tests
- `tests/integration/raw_autoblock_test.go` — Auto-block/unblock on `raw` artefacts via watcher and startup scan
- `web/src/stores/__tests__/brainDump.spec.ts` — Store unit test: `createDoc` sends `status:'raw'`
- `web/src/components/artifact/__tests__/StatusDropdown.spec.ts` — Component test: `StatusDropdown` for a `raw` artefact
- `web/src/views/project/__tests__/ArtifactListView.spec.ts` — Component test: `raw` visible by default, `done` hidden

> **Note — Milestone 1 (parser & workflow unit tests):** These were already implemented
> in earlier lineage runs and are not duplicated here.
> - `internal/artifact/artifact_test.go` — `TestParse_RawStatus`, `TestKnownStatuses_Raw`
> - `internal/workflow/workflow_test.go` — `TestRawToDraftTransitions`, `TestDraftToRawTransition`,
>   `TestRawEscapeHatches`, `TestAllowedTargetsFromRawForAnalyst`, `TestSystemRoleCanBlockFromAnyStatus`

## Milestone 2 — Round-trip integration test (API + index + WS)

File: `tests/integration/raw_status_test.go`

Run with:

```
go test ./tests/integration/... -tags=integration -run TestRawStatus
```

Scenarios covered:

1. **TestRawStatus_CreateAndGet** — `POST /artifacts` with `status: "raw"`, assert 201,
   then `GET` the created path and assert `status == "raw"`.

2. **TestRawStatus_AllowedTargetsForAnalyst** — `GET /allowed-targets` as a user with
   only the analyst role while artefact status is `raw`. Asserts `draft` and `blocked`
   are present; asserts `raw` is absent (no self-transition).

3. **TestRawStatus_TransitionRawToDraft** — `POST /transition` with `{to: "draft"}` as
   analyst. Asserts 200 response, on-disk file updated, git commit recorded.

4. **TestRawStatus_WSEventOnTransition** — Registers a hub listener before the
   transition. Asserts an `artifact.indexed{action:transitioned, to:draft}` event
   arrives within 2 s (channel-based, no sleep).

5. **TestRawStatus_NonAnalystCantTransitionToDraft** — Dev user (backend-developer only)
   attempts `raw → draft`; expects 403 with error code `"forbidden"`.

## Milestone 3 — Frontend unit / component tests

### brainDump store

File: `web/src/stores/__tests__/brainDump.spec.ts`

Run with (from `web/`):

```
pnpm test --run brainDump
```

Scenarios covered:

1. **sends status "raw" in the POST payload when creating a doc** — Mocks `api.post`,
   calls `createDoc`, asserts `frontmatter.status === 'raw'` and `type === 'doc'`.

2. **still sends status "raw" when sourceLineage is provided** — Same assertion with a
   `sourceLineage` and `sourcePath` option; also verifies `parent` is set.

3. **returns null and makes no API call when input is empty** — Empty/whitespace input
   results in `null` return and no `api.post` call.

### StatusDropdown component

File: `web/src/components/artifact/__tests__/StatusDropdown.spec.ts`

Run with:

```
pnpm test --run StatusDropdown
```

Scenarios covered:

1. **renders allowed targets when the dropdown is opened** — Mounts with `status='raw'`,
   mocks `getAllowedTargets` to return `['draft','rejected','abandoned','blocked']`, opens
   the dropdown, asserts all four options are rendered.

2. **does not list "raw" as a transition target** — Same setup; asserts `raw` is not in
   the rendered option list.

3. **shows the current status badge as "raw" before opening** — Checks `[data-status="raw"]`
   badge is present in the initial (closed) state.

4. **calls transitionArtifact with "draft" when the draft option is selected** — Clicks the
   `draft` option; asserts `transitionArtifact('testproject', path, 'draft')` was called.

### ArtifactListView — filter behaviour

File: `web/src/views/project/__tests__/ArtifactListView.spec.ts`

Run with:

```
pnpm test --run ArtifactListView
```

Scenarios covered:

1. **TERMINAL_STATUSES does not include "raw"** — Asserts the exported constant is correct.

2. **TERMINAL_STATUSES includes "done"** — Sanity check for the terminal set.

3. **shows raw and draft rows but hides done rows by default** — Shallow-mounts
   `ArtifactListView` with three fixture artefacts (`raw`, `draft`, `done`); asserts
   "Raw Idea" and "Draft Idea" are in the HTML and "Done Idea" is absent.

4. **shows done rows after toggling showCompleted on** — Sets the "Show completed"
   checkbox; asserts all three titles are now present.

## Milestone 4 — Auto-block-on-open-questions for raw artefacts

File: `tests/integration/raw_autoblock_test.go`

Run with:

```
go test ./tests/integration/... -tags=integration -run TestRawAutoBlock
```

Scenarios covered:

1. **TestRawAutoBlock_WatcherTriggersBlock** — Writes a `raw` artefact with
   `## Open Questions` to disk; polls until `status == "blocked"` (3 s timeout);
   asserts `{role:product-owner, who:agent}` assignee. Then removes the section
   (writes as `blocked`), polls until `status == "draft"` (auto-unblock goes to
   `draft`, not back to `raw`).

2. **TestRawAutoBlock_StartupScanBlocksRawWithOQ** — Seeds a `raw` artefact with
   open questions; asserts the startup scan has already blocked it before the first
   HTTP request returns.

3. **TestRawAutoBlock_WSEventsOnWatcherBlock** — Registers a hub listener; triggers
   watcher auto-block and auto-unblock; asserts `artifact.indexed{blocked_reason:
   open_questions_detected}` and `feed.new` events arrive for the block, then
   `artifact.indexed` and `feed.new{open_questions_resolved}` events arrive for the
   unblock.

## Milestone 5 — Accessibility (manual)

No automated test file. See test plan §Milestone 5 for manual WCAG AA contrast
measurement steps.

## Milestone 6 — Regression sweep

See test plan §Milestone 6 for the full regression checklist. Run:

```
make lint
make test-unit
pnpm --filter web test
go test ./tests/integration/... -tags=integration
```
