# Graph Label & Priority Filters

## Context

Both the 2D (Cytoscape) and 3D (3d-force-graph) graph views share `stores/graph.ts` and `GraphFilters.vue`. The existing filter panel had chip-selects for type/status/lineage only. Labels existed in the backend (`labels_index` table, `Labels()` API) but were not surfaced in the `Graph()` API response or in either the graph or artifact list filters.

A second change followed immediately: `priority` was added as a new frontmatter field across lifecycle artifacts. Priority needed wiring from the artifact parser through the SQLite schema, server-side filtering, the graph chips, and the artifact list dropdown.

## What was changed

### Labels (graph filter only)

- Added `Labels []string` to `GraphNode` in `internal/index/index.go`; populated via a secondary `SELECT artifact, label FROM labels_index WHERE artifact IN (...)` query after the main node scan.
- Added `labels?: string[]` to `GraphNode` and `GraphFilter` in `web/src/types/api.ts`.
- Added `uniqueLabels` computed (flatMap over rawNodes) + label OR-filter to `stores/graph.ts`.
- Added `uniqueLabels` prop + Labels chip group to `GraphFilters.vue`; `hasFilters()` updated.
- `GraphView.vue` passes `uniqueLabels` and resets labels on Reset.

### Priority (artifact list + graph filter)

- Added `Priority string` to `Frontmatter` in `internal/artifact/artifact.go`.
- Bumped `schemaVersion` to 2 (triggers auto-rebuild on startup) and added `priority TEXT NOT NULL DEFAULT ''` column + index to the `artifacts` table in `internal/index/index.go`.
- Added `Priority string` to `Filter` struct and `buildWhere`; added `priority` to the upsert INSERT; added `Priorities()` method (distinct non-empty values).
- Added `Priority string` to `GraphNode`; included in Graph SELECT + scan.
- Added `handlePriorities` to `internal/http/graph.go`; route `/priorities` registered in `internal/http/server.go`.
- `internal/http/artifacts.go` wires `priority` query param into `Filter`.
- Frontend: `priority` added to `ArtifactFrontmatter`, `GraphNode`, `ArtifactFilter`, `GraphFilter` in `web/src/types/api.ts`.
- `listPriorities` added to `web/src/api/artifacts.ts`; `priority` param added to `filterParams`.
- `priorities` ref + `fetchPriorities` action added to `web/src/stores/artifacts.ts`.
- `uniquePriorities` computed + priority filter (exact match, multi-select) added to `web/src/stores/graph.ts`.
- Priority chip group added to `GraphFilters.vue`; `GraphView.vue` passes `uniquePriorities` and resets on Reset.
- Priority `<select>` dropdown added to `ArtifactListView.vue` filter bar (hidden until priorities exist); `fetchPriorities` called on mount.

## Files modified

| File | Change |
|---|---|
| `internal/artifact/artifact.go` | `Priority` field on `Frontmatter` |
| `internal/index/index.go` | schemaVersion→2; `priority` column + index; `Priority` in Filter/GraphNode/upsert/Graph SELECT/buildWhere; `Priorities()` method |
| `internal/http/graph.go` | `handlePriorities` |
| `internal/http/server.go` | `/priorities` route |
| `internal/http/artifacts.go` | `priority` query param into Filter |
| `web/src/types/api.ts` | `labels`/`priority` on GraphNode; `labels`/`priorities` on GraphFilter; `priority` on ArtifactFilter/ArtifactFrontmatter |
| `web/src/api/artifacts.ts` | `listPriorities`; `priority` in filterParams |
| `web/src/stores/artifacts.ts` | `priorities` ref + `fetchPriorities` |
| `web/src/stores/graph.ts` | `uniqueLabels`, `uniquePriorities` computed; label+priority filter in `filteredNodes` |
| `web/src/components/graph/GraphFilters.vue` | Label + Priority chip groups; props; hasFilters update |
| `web/src/views/project/GraphView.vue` | Pass uniqueLabels/uniquePriorities; reset includes labels/priorities |
| `web/src/views/project/ArtifactListView.vue` | Priority select dropdown; fetchPriorities on mount |
