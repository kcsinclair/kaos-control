---
title: Releases and Roadmaps
type: requirement
status: blocked
lineage: releases-and-roadmaps
created: "2026-05-06T00:00:00+10:00"
parent: ideas/releases-and-roadmaps.md
labels:
    - workflow
    - roadmaps
    - releases
assignees:
    - role: product-owner
      who: agent
---

# Releases and Roadmaps

## Problem

Innovation Maker currently treats releases as free-form string fields on artifact frontmatter with no dedicated data model, management UI, or visualisation. Product owners have no way to:

- Define releases with date ranges and see what work falls into each.
- View a timeline/Gantt chart of planned releases.
- Automatically generate a roadmap from release definitions.
- Filter existing views (kanban, graph) by release.

As projects grow beyond a handful of ideas the product owner needs structured release planning to prioritise work, communicate timelines to stakeholders, and track progress toward shipping.

## Goals / Non-goals

### Goals

1. **Release entity model** — promote releases from a plain string to a first-class entity stored in SQLite with name, start date, end date, and ordering.
2. **Release CRUD** — provide API endpoints and UI for creating, editing, renaming, and deleting releases.
3. **Artifact–release assignment** — allow ideas and defects to be assigned to a release via a select dropdown in the artifact editor, populated from defined releases.
4. **Gantt chart view** — a new Roadmap page with a Gantt chart showing releases as horizontal bars across configurable time columns (week, month, quarter, half-year, year).
5. **Roadmap graph view** — extend the existing 2D/3D graph to display a release-centric roadmap where releases form the backbone and their assigned ideas/defects are linked nodes.
6. **Kanban release filter** — add a release filter dropdown to the kanban board so users can scope the board to a single release.
7. **Artifact frontmatter sync** — keep the `release` field on artifact markdown files in sync with release entity changes (renames propagate to all assigned artifacts).

### Non-goals

- Sprint management changes — sprints are out of scope for this feature.
- Capacity planning, velocity tracking, or burndown charts.
- Multi-project cross-release views.
- Drag-and-drop reordering of ideas within the Gantt chart.
- Release artifacts in `lifecycle/releases/` — releases are managed as SQLite entities, not markdown artifacts, in this iteration.
- Automated release creation from git tags or CI pipelines.

## Detailed Requirements

### DR-1: Release Data Model

- A **release** is a named entity with the following fields:
  - `id` (integer, auto-increment primary key)
  - `project_id` (foreign key to the project)
  - `name` (string, unique per project, 1–120 characters)
  - `start_date` (date, required)
  - `end_date` (date, required, must be ≥ start_date)
  - `created_at`, `updated_at` (timestamps)
- Alternatively, the user may specify `start_date` + `duration` (in days/weeks); the system calculates `end_date`.
- Releases are ordered by `start_date` for display purposes.
- The SQLite `releases` table is created at startup via schema migration.

### DR-2: Release API

Expose the following REST endpoints under `/api/v1/projects/:project/releases`:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/` | List all releases for the project, ordered by start_date |
| `POST` | `/` | Create a new release |
| `GET` | `/:id` | Get a single release with summary counts of assigned ideas/defects |
| `PUT` | `/:id` | Update a release (name, dates) |
| `DELETE` | `/:id` | Delete a release (does **not** update artifact frontmatter) |
| `GET` | `/:id/artifacts` | List artifacts assigned to this release |

- **Rename propagation**: when a release name changes via `PUT`, the server must update the `release` frontmatter field on all artifacts currently assigned to the old name, commit the changes, and re-index.
- **Delete behaviour**: deleting a release removes it from the database only. Artifacts retain their existing `release` field value (now orphaned). The UI should display a warning before deletion indicating how many artifacts reference this release.

### DR-3: Artifact–Release Assignment

- The `release` field in artifact frontmatter stores the release **name** (string), matching a defined release entity.
- In the artifact editor (`FrontmatterEditor.vue`), replace the free-text `release` input with a `<select>` dropdown populated from `GET /releases`. Include an "Unassigned" option and a "+ Create Release" option that opens the release creation modal inline.
- Only artifact types `idea` and `defect` appear in roadmap views. Other types may still carry a `release` field but are excluded from roadmap visualisations.

### DR-4: Gantt Chart View

- Add a new top-level route `/p/:project/roadmap` accessible from a "Roadmap" item in the left navigation.
- The Gantt chart is the default sub-view of the Roadmap page.
- **Time axis**: configurable granularity via a toolbar toggle — `week`, `month` (default), `quarter`, `half-year`, `year`.
- **Rows**: one row per release, ordered by `start_date`.
- **Bars**: each release is rendered as a horizontal bar spanning from `start_date` to `end_date` across the appropriate columns. The bar displays the release name and a summary badge (e.g., "3 ideas · 1 defect").
- **Click interaction**: clicking a release bar opens a modal listing all ideas and defects assigned to that release as cards. Each card shows title, type, status, and lineage. Clicking a card navigates to the artifact detail view.
- **Release management**: a "Create Release" button in the toolbar opens a modal with fields for name, start date, and duration (auto-calculates end date) or explicit end date. Within the release detail modal, "Edit" and "Delete" buttons are available.
- **Today marker**: a vertical line indicating the current date.
- **Empty state**: when no releases are defined, show a prompt to create the first release.
- Implement using HTML/CSS (no heavy Gantt library dependency); leverage existing component patterns.

### DR-5: Roadmap Graph View

- Add a "Graph" sub-view toggle on the Roadmap page (alongside the Gantt chart).
- The graph view reuses the existing 2D (Cytoscape.js) and 3D (3d-force-graph) components with a filtered dataset:
  - **Nodes**: releases (larger, distinct colour/shape) + ideas and defects assigned to those releases.
  - **Edges**: release → idea/defect (membership), plus any existing `depends_on`/`blocks` edges between included artifacts.
- Releases are connected to each other in chronological order (by `start_date`) to form a timeline backbone.
- Clicking a release or artifact node opens the standard node modal.
- A "Roadmap" entry in the left navigation menu provides access to this view.

### DR-6: Kanban Release Filter

- Add a release filter dropdown to the kanban board toolbar.
- Options: "All Releases" (default), each defined release by name, "Unassigned".
- When a release is selected, only artifacts with a matching `release` field are shown.
- The filter state is preserved in the URL query string (`?release=<name>`) for shareability.

### DR-7: Index / Query Support

- Add a `releases` table to the SQLite schema (see DR-1).
- Add an index on the `release` field in the `artifacts` table (or in `frontmatter_json`) to support efficient filtering by release.
- The `GET /artifacts` endpoint must accept an optional `release` query parameter for server-side filtering.

## Acceptance Criteria

- [ ] A product owner can create a release with a name, start date, and duration via the UI.
- [ ] A product owner can edit a release's name and dates; renaming propagates to all assigned artifact files and commits the changes.
- [ ] A product owner can delete a release; artifacts are not modified; a warning shows the count of affected artifacts.
- [ ] The artifact editor shows a select dropdown for the `release` field populated from defined releases.
- [ ] Selecting "+ Create Release" in the dropdown opens the creation modal and assigns the new release on save.
- [ ] The Roadmap page is accessible from the left navigation under "Roadmap".
- [ ] The Gantt chart displays releases as bars with correct date positioning across week/month/quarter/half-year/year granularities.
- [ ] Clicking a release bar opens a modal showing idea and defect cards assigned to that release.
- [ ] The Gantt chart "Create Release" button opens a creation modal.
- [ ] The roadmap graph view shows releases as backbone nodes with ideas/defects linked, reusing existing graph components.
- [ ] The kanban board has a release filter dropdown that correctly scopes displayed artifacts.
- [ ] The release filter is reflected in the URL query string.
- [ ] Releases persist across server restarts (SQLite storage).
- [ ] The `releases` REST API returns correct responses for all CRUD operations.
- [ ] Artifacts with type `idea` or `defect` appear in roadmap views; other types do not.

## Open Questions

1. Should releases support a `status` field (e.g., `planned`, `active`, `shipped`) to visually distinguish past/current/future releases on the Gantt chart?
2. When deleting a release, should there be an option to reassign its artifacts to another release rather than orphaning them?
3. Should the Gantt chart support drag-to-resize release bars for quick date adjustment, or is the edit modal sufficient for v1?
4. How should "Unscheduled" releases (no end date, used as a parking lot) be represented on the Gantt chart — as a separate section or excluded?
5. Should the roadmap graph view be a separate entry in the left nav, or a tab/toggle within the existing graph page?
