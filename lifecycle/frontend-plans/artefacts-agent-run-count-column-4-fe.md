---
title: "Frontend Plan: Artefacts Agent Run Count Column"
type: plan-frontend
status: approved
lineage: artefacts-agent-run-count-column
parent: lifecycle/requirements/artefacts-agent-run-count-column-2.md
---

# Frontend Plan: Artefacts Agent Run Count Column

This plan implements FR3 (new table column), FR4 (column sorting), FR5 (column position), and the Q1 active-agent pill indicator. It depends on the [[artefacts-agent-run-count-column]] backend plan delivering `agent_run_count` and `active_agent_status` in the API response.

## Milestone 1 — Extend `ArtifactRow` TypeScript interface

### Description

Add the two new fields returned by the backend to the `ArtifactRow` interface so TypeScript is aware of them throughout the frontend.

### Files to change

- `web/src/types/api.ts` — add to the `ArtifactRow` interface:
  ```typescript
  agent_run_count: number
  active_agent_status?: 'running' | 'queued'
  ```

### Acceptance criteria

- [ ] `ArtifactRow` includes `agent_run_count` (required, number) and `active_agent_status` (optional string union).
- [ ] `pnpm exec vue-tsc --noEmit` passes with no type errors.

---

## Milestone 2 — Add "Runs" column to `ArtifactListView.vue`

### Description

Add a sortable "Runs" column to the artefacts table. Per FR5 it appears after "Type" and before "Created". The column uses the existing `useSortableTable` composable with `type: 'number'`.

### Files to change

- `web/src/views/project/ArtifactListView.vue`:
  1. **Column definition** — add `agent_run_count` to the `useSortableTable` config (after `type`, before `created`):
     ```typescript
     agent_run_count: { type: 'number' },
     ```
  2. **Table header** — add a `<SortHeader>` element after the Type header and before the Created header:
     ```vue
     <SortHeader
       label="Runs"
       column="agent_run_count"
       :sort-column="sortColumn"
       :sort-direction="sortDirection"
       :sortable="true"
       @toggle="onToggleSort"
     />
     ```
     Per Q2, the header text is "Runs". Add `title="Agent Run Count"` for a tooltip.
  3. **Table cell** — add a `<td>` in the corresponding position in the row template:
     ```vue
     <td class="cell-runs">{{ row.agent_run_count }}</td>
     ```
     Display `0` when the count is 0 (never blank or dash, per FR3).

### Acceptance criteria

- [ ] "Runs" column header appears between "Type" and "Created".
- [ ] Column displays the integer count for each row.
- [ ] A count of 0 displays as `0`.
- [ ] Clicking the header sorts ascending; clicking again sorts descending (FR4).
- [ ] Header tooltip reads "Agent Run Count".
- [ ] `pnpm build` and `vue-tsc --noEmit` pass.

---

## Milestone 3 — Active-agent status pill

### Description

When `active_agent_status` is `"running"` or `"queued"`, display a small pill/badge next to the artefact title in the Path/Name column. This addresses the Q1 requirement: "If there is a job running or queued for an artefact, something in the row should indicate that."

### Files to change

- `web/src/views/project/ArtifactListView.vue`:
  1. In the title/path `<td>`, after the link text, conditionally render a pill:
     ```vue
     <span
       v-if="row.active_agent_status"
       class="agent-status-pill"
       :data-status="row.active_agent_status"
     >
       {{ row.active_agent_status === 'running' ? 'Agent Running' : 'Work Queued' }}
     </span>
     ```
  2. Add styles for `.agent-status-pill` following the existing `.badge` and `.priority-pill` patterns:
     - `running`: use an animated/pulsing accent colour (e.g. `var(--badge-in-progress-bg)`).
     - `queued`: use a muted colour (e.g. `var(--badge-planning-bg)`).
     - Small font size, inline display, left margin to separate from the title.

### Acceptance criteria

- [ ] Pill reading "Agent Running" appears when `active_agent_status === 'running'`.
- [ ] Pill reading "Work Queued" appears when `active_agent_status === 'queued'`.
- [ ] No pill appears when `active_agent_status` is absent/empty.
- [ ] Pill styling is consistent with existing `.badge` / `.priority-pill` patterns.

---

## Milestone 4 — Column styling and responsiveness

### Description

Ensure the "Runs" column follows numeric-data conventions (NFR2) and handles narrow viewports gracefully (NFR3).

### Files to change

- `web/src/views/project/ArtifactListView.vue` (scoped styles):
  1. `.cell-runs` — right-align text, use tabular-nums font-feature, match padding of other cells.
  2. Add a `@media` rule (or extend an existing responsive breakpoint) to hide `.cell-runs` and its header on narrow viewports, consistent with how other lower-priority columns are handled.

### Acceptance criteria

- [ ] Run count is right-aligned in its column.
- [ ] On narrow viewports the "Runs" column is hidden (matching existing responsive behaviour).
- [ ] Column width does not cause horizontal overflow on typical artefact lists.

---

## Milestone 5 — WebSocket-driven refresh picks up new counts

### Description

Verify that the existing `artifact.indexed` WebSocket listener in `ArtifactListView.vue` (lines 175-179) already re-fetches the list after an agent run finishes, which will now include updated `agent_run_count` and `active_agent_status`. Additionally, listen to `agent.finished` to ensure immediate refresh when a run completes (the `artifact.indexed` event may fire slightly later due to re-indexing delay).

### Files to change

- `web/src/views/project/ArtifactListView.vue` — add a second WebSocket listener:
  ```typescript
  useWebSocket(project, 'agent.finished', (_e: WsEvent) => {
    store.invalidate()
    store.fetchList(project, { limit: 0, offset: undefined })
  })
  ```

### Acceptance criteria

- [ ] After an agent run finishes, the "Runs" count increments without a full page reload (AC6 from requirement).
- [ ] The active-agent pill disappears when the run finishes.
- [ ] No duplicate fetches if both `artifact.indexed` and `agent.finished` fire in quick succession (the store's loading guard or debounce handles this).
