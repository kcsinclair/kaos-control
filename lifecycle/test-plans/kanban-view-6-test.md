---
title: "Kanban View — Test Plan"
type: plan-test
status: draft
lineage: kanban-view
parent: requirements/kanban-view-3.md
---

# Kanban View — Test Plan

This plan covers integration tests for the Kanban view feature ([[kanban-view]]). Tests validate the backend config API from [[kanban-view-4-be]] and the end-to-end behaviour implied by [[kanban-view-5-fe]]. All tests go in `tests/integration/` following existing conventions.

## Milestone 1 — Kanban config API tests

### Description

Test the `GET /api/p/:project/config/kanban` endpoint for various config states.

### Files to change

- `tests/integration/kanban_config_test.go` — New file.

### Test cases

1. **Full kanban config** — Write a `lifecycle/config.yaml` with a complete `kanban` section (columns, uncategorised: true, card_fields). Hit the endpoint. Assert the response contains the correct columns array, uncategorised flag, and card_fields list.

2. **No kanban config** — Write a `lifecycle/config.yaml` without a `kanban` key. Hit the endpoint. Assert response is `{"kanban": null}`.

3. **Minimal kanban config** — Write a `lifecycle/config.yaml` with only `kanban.columns` (no `uncategorised`, no `card_fields`). Assert `uncategorised` defaults to true (or null, with frontend applying default), and `card_fields` is empty/null.

4. **Empty columns** — `kanban.columns` is `[]`. Assert endpoint returns 200 with an empty columns array.

5. **Config reload after edit** — Write initial config, fetch kanban config, then update `config.yaml` to add a new column, fetch again. Assert the new column appears without server restart.

### Acceptance criteria

- [ ] All five test cases pass.
- [ ] Tests use the existing integration test helpers (`helpers_test.go`) for server setup and HTTP requests.
- [ ] `go test ./tests/integration/ -run TestKanbanConfig -short` passes.

---

## Milestone 2 — Artifact grouping correctness tests

### Description

Test that the artifact list API returns all the data the frontend needs to group artifacts into kanban columns. These tests verify that the artifact list endpoint returns `status` and `created` fields correctly for kanban consumption — the grouping itself is client-side, but the data contract must be verified.

### Files to change

- `tests/integration/kanban_grouping_test.go` — New file.

### Test cases

1. **Artifacts have status field** — Create artifacts with various statuses (draft, approved, in-development, done, a custom/unknown status). Fetch via `GET /api/p/:project/artifacts`. Assert each artifact row includes the correct `status` field.

2. **Artifacts have created field** — Create artifacts with `created` frontmatter dates. Fetch list. Assert `created` is returned in the response for age computation.

3. **Artifacts with frontmatter fields for card rendering** — Create artifacts with `title`, `type`, `priority`, `labels`, `lineage` frontmatter. Fetch list. Assert all fields are present in the response row and `frontmatter` sub-object.

4. **Filter interaction** — Apply `status=draft` filter. Assert only draft artifacts are returned. This validates that frontend filtering (which may also be done client-side) can fall back to server-side filtering for large datasets.

### Acceptance criteria

- [ ] All four test cases pass.
- [ ] `go test ./tests/integration/ -run TestKanbanGrouping -short` passes.

---

## Milestone 3 — Navigation and routing tests

### Description

Test that the new board route is accessible and the sidebar navigation restructure is correct. These are HTTP-level tests verifying that the SPA catch-all serves the frontend for the board path, and that existing artifact routes are not broken.

### Files to change

- `tests/integration/kanban_routing_test.go` — New file.

### Test cases

1. **Board route serves SPA** — `GET /p/:project/artifacts/board` returns 200 with the SPA HTML (the frontend catch-all route). This confirms the new route does not 404 at the server level.

2. **List route unchanged** — `GET /p/:project/artifacts` still returns 200 with the SPA HTML.

3. **Artifact editor route unchanged** — `GET /p/:project/artifacts/requirements/kanban-view-3.md` still serves the SPA (not intercepted by the board route).

4. **Config kanban endpoint requires auth** — `GET /api/p/:project/config/kanban` without authentication returns 401.

### Acceptance criteria

- [ ] All four test cases pass.
- [ ] Existing artifact-related routing tests (`artifacts_api_test.go`) still pass.
- [ ] `go test ./tests/integration/ -run TestKanbanRouting -short` passes.

---

## Milestone 4 — Config validation edge cases

### Description

Test that invalid or unusual kanban configurations are handled gracefully.

### Files to change

- `tests/integration/kanban_validation_test.go` — New file.

### Test cases

1. **Duplicate statuses across columns** — A status appears in two different columns' `statuses` lists. Assert the endpoint still returns 200 (no server error). The artifact will appear in whichever column is listed first (frontend behaviour, but backend should not reject the config).

2. **Column with empty statuses array** — A column has `statuses: []`. Assert endpoint returns 200 and the column is included in the response.

3. **Unknown card_fields entries** — `card_fields` includes `"nonexistent_field"`. Assert endpoint returns 200 (backend does not validate field names — that is a frontend concern).

4. **Very large number of columns** — Config with 20 columns. Assert endpoint returns 200 and all columns are present.

### Acceptance criteria

- [ ] All four test cases pass.
- [ ] `go test ./tests/integration/ -run TestKanbanValidation -short` passes.

---

## Milestone 5 — Write test artifact for lifecycle tracking

### Description

Create a companion artifact in `lifecycle/tests/` documenting the test coverage for the kanban view feature.

### Files to change

- `lifecycle/tests/kanban-view-7-test.md` — New file. Documents the test suites created above, the scenarios covered, and references the test files in `tests/integration/`.

### Acceptance criteria

- [ ] The test artifact exists with correct frontmatter (type: test, lineage: kanban-view, parent pointing to this test plan).
- [ ] The body summarises the four test files and their scenarios.
