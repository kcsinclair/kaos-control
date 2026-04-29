---
title: "Hide Done Items by Default — Test Suite"
type: test
status: approved
lineage: hide-done-items-by-default
parent: lifecycle/test-plans/hide-done-items-by-default-5-test.md
---

# Hide Done Items by Default — Test Suite

Integration and unit tests verifying that the "Show completed" toggle works correctly across the three artifact views (list, kanban, graph), that each view is independent, and that the backend API never suppresses terminal-status artifacts server-side.

## Test files

### Backend API integration tests (Go)

**`tests/integration/hide_done_items_api_test.go`**

Go integration tests that spin up a real kaos-control server and verify the data layer provides everything the frontend needs for client-side filtering.

| Test | Scenario covered |
|------|-----------------|
| `TestHideDoneItems_APIReturnsAllStatuses` | Unfiltered GET /artifacts includes all 8 statuses |
| `TestHideDoneItems_APIReturnsTerminalArtifactsUnfiltered` | Backend does not suppress done/rejected/abandoned |
| `TestHideDoneItems_EachItemHasStatusField` | Every artifact row has a non-empty `status` field |
| `TestHideDoneItems_FilterByTerminalStatus` | API supports `?status=done`, `?status=rejected`, `?status=abandoned` |
| `TestHideDoneItems_FilterByActiveStatus` | API supports all five active status filters |
| `TestHideDoneItems_GraphNodesHaveStatusField` | Graph nodes include `status` field for client-side filtering |
| `TestHideDoneItems_GraphIncludesTerminalStatusNodes` | Graph API does not suppress terminal-status nodes |
| `TestHideDoneItems_GraphNodeCount` | Graph node count matches seeded artifact count |
| `TestHideDoneItems_TerminalArtifactsRetrievableIndividually` | Individual GET /artifacts/:path works for terminal artifacts |
| `TestHideDoneItems_ActiveArtifactCountIsConsistent` | Per-status counts sum correctly |

### Frontend unit tests (TypeScript / Vitest)

All TypeScript tests live under `tests/web/` and are picked up by `tests/web/vitest.config.ts`. They use `happy-dom` + Pinia + `@vue/test-utils` with mocked API layers.

**`tests/web/helpers/seed_artifacts.ts`**

Shared factory helpers — `makeArtifactRow()`, `makeGraphNode()`, `makeGraphEdge()`, `makeArtifactsForAllStatuses()`, `makeGraphNodesForAllStatuses()` — used by all three view test files.

---

**`tests/web/hide-done-items/artifact-list-toggle.test.ts`** (Milestone 2)

Tests the `visibleItems` computed logic from `ArtifactListView` using Vue's `ref` + `computed` directly.

| Test | Scenario covered |
|------|-----------------|
| Default state hides all terminal artifacts | `done`, `rejected`, `abandoned` absent from visibleItems when `showCompleted=false` |
| Default state shows all active artifacts | All 5 active statuses present |
| Count reflects visible set only | `visibleItems.length` is 5, not 8 |
| Toggle reveals all items | After `showCompleted=true`, all 8 artifacts visible |
| Toggle re-hides terminal items | After toggling back to `false`, terminal items hidden again |
| Reset per instance | New `showCompleted` ref starts `false` (simulates navigation reset) |

---

**`tests/web/hide-done-items/kanban-toggle.test.ts`** (Milestone 3)

Tests the `useKanbanBoard` composable with mocked API. Config includes a Done column with statuses `[done, abandoned, rejected]`.

| Test | Scenario covered |
|------|-----------------|
| `hideTerminal` defaults to `true` | Done column absent by default |
| Terminal cards absent from all columns | No done/rejected/abandoned cards in any column |
| Active columns (Backlog, In Progress) present | Non-terminal columns unaffected |
| Toggle reveals Done column | After `hideTerminal=false`, Done column appears with 3 cards |
| Column card counts are accurate | Backlog count unchanged by toggle; Done shows exactly 3 |
| Toggle re-hides Done column | Setting `hideTerminal=true` again removes Done column |
| Other columns unaffected | Backlog and In Progress cards identical regardless of toggle |
| No extra API calls on toggle | `listArtifacts` and `api.get` call counts don't increase after toggle |

---

**`tests/web/hide-done-items/graph-toggle.test.ts`** (Milestone 4)

Tests the Pinia `useGraphStore` store's `filteredNodes`, `filteredEdges`, and `toggleHideTerminal`.

| Test | Scenario covered |
|------|-----------------|
| `hideTerminal` defaults to `true` | Terminal nodes excluded from `filteredNodes` |
| All active nodes included | All 5 active statuses present in `filteredNodes` |
| Edges to hidden terminal nodes pruned | `filteredEdges` removes edges where source or target is terminal |
| Active↔active edges retained | Edges between non-terminal nodes survive |
| Toggle reveals terminal nodes | After `hideTerminal=false`, all 8 nodes in `filteredNodes` |
| Edges to terminal nodes reappear | After `hideTerminal=false`, previously-pruned edges present |
| `toggleHideTerminal()` flips flag | Action correctly inverts `hideTerminal` |
| Explicit status filter overrides `hideTerminal` | When `filter.statuses=['done']`, done node appears despite `hideTerminal=true` |
| Clearing status filter re-activates `hideTerminal` | Removing filter hides terminal nodes again |
| Node count reflects filtered set | `filteredNodes.length < rawNodes.length` when hidden |
| No extra API calls on toggle | `getGraph` call count unchanged after toggle |
| Reset on navigation | Setting `hideTerminal=true` restores hidden state (simulates `onMounted`) |

---

**`tests/web/hide-done-items/cross-view-consistency.test.ts`** (Milestone 5)

Tests that the toggle state is independent across views and resets on remount.

| Test | Scenario covered |
|------|-----------------|
| List toggle does not affect kanban | `showCompleted=true` in list; kanban `hideTerminal` stays `true` |
| Kanban toggle does not affect graph | `hideTerminal=false` in kanban; graph store `hideTerminal` stays `true` |
| Two kanban instances are independent | Separate `useKanbanBoard()` calls return independent refs |
| List resets to false on new instance | Fresh `showCompleted` ref starts `false` |
| Kanban resets to true on new instance | Fresh `useKanbanBoard()` call has `hideTerminal=true` |
| Graph resets via onMounted simulation | Setting `store.hideTerminal = true` restores default |
| All three views start hidden by default | Concurrent check: list, kanban, graph all start with terminal items hidden |
| No API calls when toggling list | `listArtifacts` not re-called on `showCompleted` change |
| No API calls when toggling graph | `getGraph` not re-called on `toggleHideTerminal` |
| No API calls when toggling kanban | `api.get` and `listArtifacts` not re-called on `hideTerminal` change |

## Notes

- The filtering logic is entirely client-side. The backend API always returns all artifacts regardless of status. These tests verify both sides of that contract.
- The graph store's `hideTerminal` ref is bypassed when the user sets an explicit `filter.statuses` selection, so an active status filter always wins.
- `useKanbanBoard` is a plain function (not a Pinia store), so each call returns a fresh reactive scope with independent `hideTerminal`.
- The graph store IS a Pinia singleton; `GraphView.onMounted` resets `store.hideTerminal = true` to simulate navigation reset behaviour.
