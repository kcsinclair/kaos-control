---
title: Kanban View — Integration Tests
type: test
status: approved
lineage: kanban-view
parent: test-plans/kanban-view-6-test.md
---

# Kanban View — Integration Tests

Integration test suite covering the Kanban view feature. All tests live in
`tests/integration/` and target the backend API introduced by the kanban-view
backend plan (`lifecycle/backend-plans/kanban-view-4-be.md`).

## Test files

### `tests/integration/kanban_config_test.go`

Tests for `GET /api/p/:project/config/kanban`. Five scenarios:

1. **TestKanbanConfig_Full** — Config with complete kanban section (columns,
   uncategorised, card_fields). Asserts all three sections are returned with
   correct values.
2. **TestKanbanConfig_None** — Config without a `kanban` key. Asserts response
   is `{"kanban": null}`.
3. **TestKanbanConfig_Minimal** — Config with only `kanban.columns`. Asserts
   `card_fields` is empty/absent.
4. **TestKanbanConfig_EmptyColumns** — `kanban.columns: []`. Asserts 200 with
   empty columns array or null (both acceptable for an empty YAML list).
5. **TestKanbanConfig_ReloadAfterEdit** — Writes config, fetches, updates config
   on disk, fetches again. Asserts the new column appears without server restart,
   confirming the handler re-reads from disk on every request.

### `tests/integration/kanban_grouping_test.go`

Tests that the artifact list API (`GET /api/p/:project/artifacts`) supplies all
data the frontend needs to group artifacts into kanban columns. Four scenarios:

1. **TestKanbanGrouping_ArtifactsHaveStatus** — Seeds artifacts with five
   distinct statuses (draft, approved, in-development, done, custom). Asserts
   each returned row has the correct `status` field.
2. **TestKanbanGrouping_ArtifactsHaveCreated** — Seeds artifacts, fetches list.
   Asserts every row has a non-empty `created` field for age computation.
3. **TestKanbanGrouping_ArtifactsHaveCardFields** — Seeds an artifact with
   priority and labels. Asserts top-level fields (`title`, `type`, `lineage`)
   and the `frontmatter` sub-object (including `priority` and `labels`) are all
   present in the list response.
4. **TestKanbanGrouping_FilterByStatus** — Seeds two draft and one approved
   artifact. Applies `?status=draft` filter. Asserts only the two draft
   artifacts are returned.

### `tests/integration/kanban_routing_test.go`

Tests for SPA catch-all routing and authentication. Four scenarios:

1. **TestKanbanRouting_BoardRouteServesSPA** — `GET /p/:project/artifacts/board`
   returns 200 with SPA HTML, confirming no accidental 404 at the server level.
2. **TestKanbanRouting_ListRouteUnchanged** — `GET /p/:project/artifacts` still
   returns 200 with SPA HTML.
3. **TestKanbanRouting_ArtifactEditorRouteUnchanged** — Deep artifact path
   `/p/:project/artifacts/requirements/kanban-view-3.md` still serves the SPA
   and is not captured by the board route.
4. **TestKanbanRouting_KanbanConfigRequiresAuth** — Unauthenticated `GET
   /api/p/:project/config/kanban` should return 401. (This test validates
   the security requirement; if it fails it indicates the endpoint is missing
   `requireAuth` middleware.)

### `tests/integration/kanban_validation_test.go`

Edge-case tests for unusual kanban configurations. Four scenarios:

1. **TestKanbanValidation_DuplicateStatuses** — The same status appears in two
   columns. Asserts 200 (backend does not reject duplicate statuses).
2. **TestKanbanValidation_ColumnEmptyStatuses** — A column with
   `statuses: []`. Asserts 200 and the column is present in the response.
3. **TestKanbanValidation_UnknownCardFields** — `card_fields` contains
   `nonexistent_field`. Asserts 200 (backend does not validate field names).
4. **TestKanbanValidation_ManyColumns** — 20 columns. Asserts 200 and all 20
   columns are present.

## Running the tests

```sh
# All kanban tests
go test ./tests/integration/ -tags integration -run TestKanban -short

# By milestone
go test ./tests/integration/ -tags integration -run TestKanbanConfig -short
go test ./tests/integration/ -tags integration -run TestKanbanGrouping -short
go test ./tests/integration/ -tags integration -run TestKanbanRouting -short
go test ./tests/integration/ -tags integration -run TestKanbanValidation -short
```
