---
title: Kanban View (revised)
type: requirement
status: draft
lineage: kanban-view
created: "2026-04-27T00:00:00+10:00"
priority: normal
parent: requirements/kanban-view-2.md
labels:
    - artefacts
    - workflow
    - frontend
assignees:
    - role: product-owner
      who: agent
---

# Kanban View (revised)

This requirement supersedes [[kanban-view-2]] by incorporating answers to its open questions. All requirements from the prior version carry forward unless explicitly changed below.

## Problem

The artifact list view presents all artifacts in a flat table. Users managing a lifecycle with many artifacts across different statuses have no way to visualise work distribution at a glance. A Kanban board — where cards are organised into columns by status — is a widely understood pattern for tracking work through stages, but the app does not offer one.

Different projects define different workflow statuses and may want to group those statuses into columns that reflect their own process (e.g. grouping `draft` and `clarifying` under "Backlog"). The column-to-status mapping must be configurable rather than hard-coded.

A secondary aspect is renaming the UI label "Artifacts" to "Artefacts" and restructuring the navigation so the current list becomes one sub-view alongside the new Kanban view.

## Goals / Non-goals

### Goals

- Provide a Kanban board view that displays artifacts as cards arranged in status-grouped columns.
- Make the column definitions and status-to-column mapping configurable via the project's `lifecycle/config.yaml`.
- Reuse the existing filter controls (stage, status, type, label, priority) so users can dynamically refine what appears on the board.
- Support a catch-all "Uncategorised" column for artifacts whose status does not match any configured column.
- Allow Kanban card content to be configured in YAML, supporting both raw frontmatter fields and virtual/computed fields.
- Restructure the left-menu "Artifacts" item into "Artefacts" with sub-menu entries for "List" (existing table view) and "Board" (new Kanban view).
- Allow users to reorder columns in the UI during a session (non-persistent).

### Non-goals

- Drag-and-drop to move cards between columns (requires status-transition logic; out of scope).
- Inline editing of artifacts from the Kanban card.
- Swimlanes (e.g. rows by type or assignee) — single-dimension columns only.
- Backend API changes for sorting or grouping — the board is assembled client-side from the existing artifact list endpoint.
- Work-in-progress (WIP) limits on columns.
- Changing URL paths from "artifacts" to "artefacts" — AU English URL changes are deferred.
- Persisting user column reordering across page refreshes.

## Detailed Requirements

### Functional

1. **Kanban configuration in `config.yaml`** — A new top-level `kanban` key in the project config defines the board layout:

   ```yaml
   kanban:
     columns:
       - name: Backlog
         statuses: [draft]
       - name: Approved
         statuses: [approved]
       - name: In-Progress
         statuses: [in-progress, clarifying, planning, in-development, in-qa]
       - name: Blocked
         statuses: [blocked, rejected]
       - name: Done
         statuses: [done]
     uncategorised: true
     card_fields:
       - title
       - type
       - priority
       - labels
       - age            # virtual field
   ```

   - `columns` is an ordered list. Each entry has a `name` (display label) and a `statuses` array of status strings that map into that column.
   - `uncategorised` (boolean, default `true`): when enabled, artifacts whose status does not appear in any column's `statuses` list are collected into a trailing "Uncategorised" column. When disabled, such artifacts are hidden from the board.
   - `card_fields` is an ordered list of field names to render on each Kanban card. `title` is always shown regardless of this list. Fields may be raw frontmatter keys or virtual field identifiers.

2. **Virtual / computed card fields** — In addition to raw frontmatter fields, `card_fields` supports the following virtual fields:

   | Virtual field | Definition |
   |---|---|
   | `age` | Number of days since the artifact's `created` date (or file creation if no `created` field). Displayed as e.g. "12d". |

   The set of virtual fields is fixed at build time. If a `card_fields` entry is neither a known virtual field nor a valid frontmatter key, it is silently ignored.

3. **Kanban column rendering** — Each configured column is rendered as a vertical lane with a header showing the column name and an artifact count badge. Cards stack vertically within each column and the columns scroll horizontally if they overflow the viewport. Each column independently scrolls vertically when its cards exceed the visible height.

4. **Column reordering** — The default column order matches `config.yaml`. Users may drag column headers to reorder columns within the current session. Reordering is transient and resets to the config-defined order on page refresh or navigation away.

5. **Kanban card rendering** — Each card displays:
   - The artifact title (always shown, used as the card heading).
   - Additional fields as specified by `card_fields` in config: type badge, priority indicator, labels as small tags, virtual field values.
   - The artifact's `lineage` slug displayed as a subtle identifier.
   - Clicking a card navigates to the artifact editor view (same as clicking a row in the list).

6. **Filter bar** — The existing filter controls (stage, status, type, label, priority, reset) are rendered above the board, identical to the list view. Filters narrow the set of artifacts distributed across columns. The status filter, when applied, restricts which artifacts appear but does not hide empty columns (columns remain visible but show an empty state).

7. **Navigation restructure** — The left sidebar entry currently labelled "Artifacts" becomes "Artefacts" and expands to show two sub-entries:
   - **List** — routes to the existing `ArtifactListView`.
   - **Board** — routes to the new Kanban view.

8. **UI label rename** — All user-facing occurrences of "Artifacts" in the sidebar, page titles, and headings are updated to "Artefacts". Internal code identifiers (component names, store names, API paths) remain unchanged.

9. **Empty-state handling** — If the config does not contain a `kanban` key, the Board sub-menu item is still shown but the view displays a message: "No Kanban configuration found. Add a `kanban` section to your project's config.yaml."

10. **Real-time updates** — The board listens for `artifact.indexed` WebSocket events and re-renders affected cards (same invalidation pattern as the list view).

11. **Config endpoint** — The backend exposes the kanban configuration to the frontend. If the existing project-config API already returns the full `config.yaml` contents, no new endpoint is needed; otherwise a `GET /api/projects/:project/config/kanban` endpoint returns the parsed kanban block.

### Non-functional

1. **Performance** — The board must render smoothly with up to 500 visible cards distributed across columns. Avoid re-rendering all columns when a single card changes.
2. **Responsiveness** — On narrow viewports (< 768 px) columns should stack vertically or become horizontally scrollable; cards must remain legible.
3. **Accessibility** — Column landmarks use `role="region"` with an `aria-label` matching the column name. Cards are focusable and navigable via keyboard (Tab between cards, Enter to open).
4. **No new runtime dependencies** — Implement the board with plain Vue 3 + CSS. Do not add a Kanban or drag-and-drop library (column reorder uses native HTML drag-and-drop API).
5. **Visual consistency** — Cards reuse existing design tokens (colours, spacing, border-radius, badge styles) from the artifact list and status badge system.

## Acceptance Criteria

- [ ] A `kanban` section in `lifecycle/config.yaml` defines columns, status mappings, uncategorised flag, and card fields.
- [ ] The Board view renders one column per configured entry, with artifacts grouped by their status.
- [ ] Artifacts whose status is not mapped to any column appear in an "Uncategorised" trailing column when `uncategorised: true`.
- [ ] Artifacts with unmapped statuses are hidden from the board when `uncategorised: false`.
- [ ] Each Kanban card displays the title plus the fields specified in `card_fields`.
- [ ] The `age` virtual field is supported in `card_fields` and displays days since artifact creation.
- [ ] Unknown `card_fields` entries are silently ignored (no error, no blank space).
- [ ] Clicking a card navigates to the artifact editor for that artifact.
- [ ] The filter bar (stage, status, type, label, priority, reset) appears above the board and filters the displayed cards.
- [ ] The sidebar shows "Artefacts" with "List" and "Board" sub-entries; both navigate to the correct views.
- [ ] All user-facing text that previously said "Artifacts" now reads "Artefacts".
- [ ] URL paths remain unchanged (no `/artefacts` routes).
- [ ] When no `kanban` config is present, the Board view displays an informative empty-state message.
- [ ] The board updates in real time when `artifact.indexed` WebSocket events arrive.
- [ ] Column headers show the column name and a count of currently visible cards.
- [ ] Users can drag column headers to reorder columns within the current session.
- [ ] Column order resets to config-defined order on page refresh.
- [ ] Columns are horizontally scrollable when they overflow the viewport width.
- [ ] Each column scrolls vertically independently when its cards exceed available height.
- [ ] Cards are keyboard-accessible: focusable via Tab, openable via Enter.
- [ ] The board renders 500 cards without perceptible jank.
- [ ] No new runtime dependencies are introduced.

## No Questions

None — all questions from [[kanban-view-2]] have been resolved.
