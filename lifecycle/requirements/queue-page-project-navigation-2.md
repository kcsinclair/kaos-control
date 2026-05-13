---
title: Queue Page Project Navigation
type: requirement
status: blocked
lineage: queue-page-project-navigation
created: "2026-05-13"
priority: normal
parent: lifecycle/ideas/queue-page-project-navigation.md
labels:
    - queue
    - frontend
    - vue
    - usability
    - enhancement
release: KC-Release1
assignees:
    - role: product-owner
      who: agent
---

# Queue Page Project Navigation

## Problem

The Queue page (`/queue`) is a global, app-level view that shows running, pending, and recent jobs across all registered projects. It currently has no mechanism for filtering by project or navigating to a specific project's detail view. Users monitoring multiple concurrent agent runs must leave the queue, go back to the project picker, select a project, and then navigate to the relevant page — a multi-step detour that breaks flow.

Additionally, the project name shown in the running-job panel (`QueueRunningPanel.vue`, line 44) is rendered as plain text. There is no clickable path back to the project that owns the currently running job.

## Goals / Non-goals

### Goals

1. **Project sidebar** — Add a collapsible left sidebar to the Queue page listing all registered projects, enabling one-click filtering and navigation.
2. **Clickable project name** — Make the project name in the running-job panel (and wherever it appears in pending/recent tables) a link that navigates to the project's dashboard view (`/p/:project/dashboard`).
3. **Queue filtering** — Allow users to filter the queue view to show only jobs belonging to a selected project.

### Non-goals

- Restructuring the queue to be project-scoped (it remains a global app-level route at `/queue`).
- Adding project management capabilities (create/delete/edit) from the queue sidebar.
- Real-time search or fuzzy-find within the project sidebar list.
- Changing the queue data model or API; all data needed (`QueueJob.project`, project list API) already exists.

## Detailed Requirements

### Functional

#### F1 — Project Sidebar

- A left sidebar is added to the Queue page layout.
- The sidebar fetches and displays all registered projects using the existing `useProjectStore.fetchProjects()` / `listProjects()` API.
- Each project entry is a clickable item that:
  - Filters the queue (running, pending, recent sections) to show only jobs where `job.project` matches the selected project.
  - Visually highlights the selected project.
- An "All Projects" option at the top of the sidebar clears the filter and shows the full queue (default state).
- The sidebar is collapsible. When collapsed, it reduces to a narrow icon strip or hides entirely. Collapse state should persist for the duration of the session (not across page reloads).
- The sidebar displays a badge or count next to each project name showing the number of pending + running jobs for that project.

#### F2 — Clickable Project Name

- In `QueueRunningPanel`, the project name field (`job.project`) is rendered as a `<RouterLink>` to `/p/:project/dashboard`.
- In `QueuePendingTable` and `QueueRecentTable`, any project name column is likewise rendered as a `<RouterLink>` to the project dashboard.
- Links use standard anchor styling consistent with existing link styles in the app (e.g., the artifact link already present in the running panel).

#### F3 — Sidebar–Queue Interaction

- Selecting a project in the sidebar sets a reactive filter. The running panel shows "Nothing running" if the running job belongs to a different project. Pending and recent tables show only matching jobs.
- The selected project filter is reflected in the URL as a query parameter (e.g., `/queue?project=my-project`) so that the filter survives page reloads and is shareable.
- If the query parameter references a project that does not exist in the project list, the filter is silently cleared to "All Projects".

### Non-functional

- **NF1 — Performance**: The project list should load in parallel with the queue snapshot (both fetches fire on mount). The sidebar must not block queue rendering.
- **NF2 — Responsive**: On narrow viewports (< 768px), the sidebar should default to collapsed and be togglable via a hamburger-style button.
- **NF3 — Accessibility**: Sidebar items and project links must be keyboard-navigable and have appropriate ARIA roles (`role="navigation"`, `aria-current` on selected item).

## Acceptance Criteria

- [ ] Queue page displays a left sidebar listing all registered projects.
- [ ] Each sidebar entry shows the project name and a job count badge (pending + running).
- [ ] Clicking a project in the sidebar filters running/pending/recent sections to that project's jobs only.
- [ ] An "All Projects" option is present and selected by default, showing unfiltered queue.
- [ ] The sidebar is collapsible and defaults to collapsed on viewports < 768px.
- [ ] The selected project filter is stored as a `?project=` query parameter in the URL.
- [ ] Invalid or missing `?project=` values fall back to "All Projects" without error.
- [ ] Project name in the running panel is a `<RouterLink>` to `/p/:project/dashboard`.
- [ ] Project name in pending and recent tables is a `<RouterLink>` to `/p/:project/dashboard`.
- [ ] Project list and queue snapshot load concurrently on mount — sidebar does not block queue content.
- [ ] Sidebar items are keyboard-navigable and have correct ARIA attributes.
- [ ] No new API endpoints are required; implementation uses existing `listProjects()` and `QueueSnapshot` APIs.

## Open Questions

- Should the sidebar project list auto-refresh (e.g., via WebSocket event) when a new project is registered, or is fetch-on-mount sufficient?
- Should clicking a project name link in the queue tables navigate immediately, or open in a new tab? Current artifact links in the running panel navigate in the same tab — should this be consistent?
