---
title: "Queue Page Project Navigation — Frontend Plan"
type: plan-frontend
status: in-development
lineage: queue-page-project-navigation
parent: lifecycle/requirements/queue-page-project-navigation-2.md
created: "2026-05-13"
---

# Queue Page Project Navigation — Frontend Plan

This plan implements the three functional requirements (F1 Project Sidebar, F2 Clickable Project Names, F3 Sidebar–Queue Interaction) and the three non-functional requirements (performance, responsive, accessibility).

## Milestone 1 — Create QueueSidebar Component

### Description

Build a new `QueueSidebar.vue` component that renders a collapsible left sidebar listing all registered projects. Each entry shows the project name and a badge with the count of running + pending jobs for that project. An "All Projects" entry at the top clears the filter.

### Files to create

- `web/src/components/queue/QueueSidebar.vue`

### Implementation Details

- Use `useProjectStore().fetchProjects()` to load the project list. Call this on component mount — it must not block the queue content (F1, NF1).
- Compute job counts reactively from `useQueueStore().snapshot`: for each project, count jobs in `pending` and `running` where `job.project === project.name`.
- Render a `<nav>` element with `role="navigation"` and `aria-label="Project filter"`.
- Each project item is a `<button>` (not a link — it filters, not navigates) with `aria-current="true"` when selected.
- "All Projects" is the first item, selected by default.
- Collapse/expand toggle: a button in the sidebar header toggles a reactive `collapsed` ref. When collapsed, the sidebar shrinks to a narrow strip showing only the toggle button. Collapse state is session-only (reactive ref, not persisted).
- On viewports < 768px, default `collapsed` to `true`. Use a `matchMedia` listener or CSS media query to set the initial value.
- Badge styling: small pill next to project name, muted when count is 0, highlighted when count > 0.

### Acceptance Criteria

- [ ] Sidebar renders all registered projects with names and job-count badges.
- [ ] "All Projects" entry is present and selected by default.
- [ ] Sidebar collapses and expands on toggle click.
- [ ] Sidebar defaults to collapsed on viewports < 768px.
- [ ] Sidebar items are keyboard-navigable (Tab, Enter/Space to select).
- [ ] Selected item has `aria-current="true"`.
- [ ] `<nav>` has `role="navigation"` and descriptive `aria-label`.

## Milestone 2 — Integrate Sidebar into QueueView Layout

### Description

Modify `QueueView.vue` to incorporate the sidebar alongside the existing queue content in a flex layout. The sidebar and queue content sit side by side. The project list and queue snapshot must load concurrently on mount (NF1).

### Files to change

- `web/src/views/QueueView.vue`

### Implementation Details

- Wrap the existing content in a flex container: sidebar on the left, queue content on the right (flex-grow: 1).
- Import and render `QueueSidebar` component.
- The sidebar emits a `select` event (or uses a shared reactive/store value) with the selected project name (or `null` for "All Projects").
- Ensure `queueStore.fetch()` and `projectStore.fetchProjects()` are both called on mount (via `Promise.all` or independent calls — they must not be sequential).
- Pass the selected project filter down to child components or provide it via a composable/provide-inject.

### Acceptance Criteria

- [ ] QueueView displays sidebar and queue content side by side.
- [ ] Both project list and queue snapshot begin loading on mount concurrently.
- [ ] Sidebar does not block queue content rendering (each has independent loading state).
- [ ] Layout is visually consistent with existing app styling.

## Milestone 3 — Implement Queue Filtering by Project (F3)

### Description

Wire the sidebar selection to filter the running panel, pending table, and recent table. Reflect the selected project in the URL as a `?project=` query parameter so the filter survives page reloads and is shareable.

### Files to change

- `web/src/views/QueueView.vue`
- `web/src/components/queue/QueueRunningPanel.vue`
- `web/src/components/queue/QueuePendingTable.vue`
- `web/src/components/queue/QueueRecentTable.vue`

### Implementation Details

- Read `?project=` from `route.query.project` on mount. If the value matches a project in the fetched project list, set it as the active filter. If it does not match (or is absent), default to "All Projects".
- When the sidebar selection changes, update `router.replace({ query: { project: selectedProject || undefined } })` to sync the URL without a navigation event.
- Provide the filter value to child components via props or provide/inject.
- **QueueRunningPanel**: If the filter is active and `snapshot.running?.project` does not match, show "Nothing running" (same as the null-running state). Otherwise show the running job as normal.
- **QueuePendingTable**: Compute a filtered `pendingJobs` array: `snapshot.pending.filter(j => !filter || j.project === filter)`. Render only filtered jobs. Update the empty state to "No pending jobs for this project" when filtered.
- **QueueRecentTable**: Same filtering pattern as pending. Empty state: "No recent jobs for this project".
- Sidebar selection and URL parameter are kept in two-way sync: changing one updates the other.

### Acceptance Criteria

- [ ] Selecting a project in the sidebar filters running/pending/recent to that project's jobs.
- [ ] "All Projects" clears the filter and shows all jobs.
- [ ] Selected project is reflected as `?project=<name>` in the URL.
- [ ] Navigating to `/queue?project=my-project` pre-selects that project in the sidebar.
- [ ] Invalid `?project=` value silently falls back to "All Projects" without errors.
- [ ] Filtered empty states display project-specific messages.

## Milestone 4 — Clickable Project Names (F2)

### Description

Convert the plain-text project name in `QueueRunningPanel`, `QueuePendingTable`, and `QueueRecentTable` into `<RouterLink>` elements pointing to the project's agents view.

### Files to change

- `web/src/components/queue/QueueRunningPanel.vue` (line ~44, project name display)
- `web/src/components/queue/QueuePendingTable.vue` (project column)
- `web/src/components/queue/QueueRecentTable.vue` (project column)

### Implementation Details

- Replace the plain `{{ job.project }}` text with `<RouterLink :to="'/p/' + job.project + '/agents'">{{ job.project }}</RouterLink>`.
- Exception per requirement: the running panel links to `/p/:project/dashboard` while pending and recent link to `/p/:project/agents`.
- Apply the same anchor styling used by existing artifact links in the running panel (underline on hover, muted colour, inherits font).
- Links navigate in the same tab (standard RouterLink behaviour, confirmed in Resolved Questions).

### Acceptance Criteria

- [ ] Project name in running panel is a `<RouterLink>` to `/p/:project/dashboard`.
- [ ] Project name in pending table is a `<RouterLink>` to `/p/:project/agents`.
- [ ] Project name in recent table is a `<RouterLink>` to `/p/:project/agents`.
- [ ] Link styling is consistent with existing artifact links in the queue.
- [ ] Clicking a project link navigates in the same tab.

## Milestone 5 — Responsive and Accessibility Polish

### Description

Final pass ensuring the sidebar and new links meet the non-functional requirements for responsive layout and accessibility.

### Files to change

- `web/src/components/queue/QueueSidebar.vue`
- `web/src/views/QueueView.vue`

### Implementation Details

- **Responsive (NF2)**: On viewports < 768px, sidebar defaults to collapsed. Add a hamburger-style toggle button (use lucide `PanelLeftOpen` / `PanelLeftClose` or `Menu` icon) visible at all viewport sizes. When collapsed on mobile, the sidebar should not occupy layout space (use `position: absolute` or `display: none` rather than a narrow strip).
- **Accessibility (NF3)**: Verify all sidebar items are reachable via Tab key. Verify Enter and Space activate selection. Add `aria-expanded` to the collapse toggle button. Add `aria-current="page"` on the selected sidebar item. Ensure project links in tables have descriptive `aria-label` attributes (e.g., `aria-label="Go to project my-project"`).
- Verify colour contrast of badge counts and selected-state highlighting meet WCAG AA.

### Acceptance Criteria

- [ ] Sidebar defaults to collapsed on viewports < 768px.
- [ ] Toggle button is visible and functional at all viewport widths.
- [ ] All interactive elements are keyboard-navigable (Tab, Enter, Space).
- [ ] `aria-expanded`, `aria-current`, and `aria-label` attributes are correctly applied.
- [ ] No accessibility regressions in existing queue components.

## Cross-references

- [[queue-page-project-navigation]] backend plan — project name consistency (Milestones 1–2) is a prerequisite for reliable filtering.
- [[queue-page-project-navigation]] test plan — integration and E2E tests cover sidebar filtering, URL sync, and clickable links.
