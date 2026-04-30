---
title: "Frontend Plan — Move Running Agents Indicator to Menu Bar"
type: plan-frontend
status: in-development
lineage: agents-indicator-in-menu-bar
parent: lifecycle/requirements/agents-indicator-in-menu-bar-2.md
---

# Frontend Plan — Move Running Agents Indicator to Menu Bar

## Summary

Relocate the running-agents indicator from the fixed-position floating pill (`RunStatusChip.vue` with `<Teleport to="body">`) into the application header (`AppHeader.vue`). The indicator consumes the existing `useAgentsStore().activeRuns` computed property — no new data fetching or WebSocket subscriptions are needed. The backend plan ([[agents-indicator-in-menu-bar]]-3-be) confirms no API changes are required.

---

## Milestone 1: Add the indicator to AppHeader

### Description
Create an inline indicator element inside `AppHeader.vue` that shows a pulsing green dot and the count of running agents. The indicator is rendered conditionally when `activeRuns.length > 0` and is scoped to the current project context.

### Files to change
- `web/src/components/layout/AppHeader.vue` — add the indicator markup, script imports, and scoped styles.

### Implementation details
1. Import `useAgentsStore` and `useRoute` from their respective modules.
2. Derive `project` from `route.params.project` — the indicator only renders when this is truthy (FR-8: project context).
3. Compute `activeCount` from `agentsStore.activeRuns.length`.
4. Add a `<RouterLink>` (or clickable element using `router.push`) to the `header-actions` div, positioned **before** the existing user name span (FR-1).
5. Use `v-if="project && activeCount > 0"` to satisfy FR-5 (hidden when idle) and FR-8 (project context).
6. Inner content: a `<span class="run-dot">` followed by text `{{ activeCount }} running agent{{ activeCount === 1 ? '' : 's' }}` (FR-2, FR-3).
7. Set `aria-label` dynamically: `` `${activeCount} running agent${activeCount === 1 ? '' : 's'} — click to view` `` (NFR-3).
8. The `RouterLink` target: `/p/${project}/agents` (FR-6).

### Styles
- `.header-run-indicator` — `display: flex; align-items: center; gap: var(--space-2); padding: var(--space-1) var(--space-3); border: 1px solid var(--color-border-dark); border-radius: var(--radius-md); font-size: var(--text-sm); color: var(--color-sidebar-text-muted); text-decoration: none; cursor: pointer; transition: color 0.15s, border-color 0.15s;`
- `.header-run-indicator:hover` — `color: #fff; border-color: var(--color-sidebar-text); background: rgba(255,255,255,0.08);` (matches existing `btn-icon` / `btn-ghost` hover patterns, NFR-1).
- `.run-dot` — reuse the existing 8px green pulsing dot from `RunStatusChip`. `width: 8px; height: 8px; border-radius: 50%; background: #22c55e; animation: pulse 1.5s ease-in-out infinite;`
- `@keyframes pulse` — same as current: `0%, 100% { opacity: 1; } 50% { opacity: 0.4; }`.
- `@media (prefers-reduced-motion: reduce)` — disable the pulse animation, show a static green dot (NFR-3).
- `@media (max-width: 768px)` — hide the text label, show only the dot and count number (NFR-2).

### Acceptance criteria
- [ ] Indicator appears in `AppHeader` when `activeRuns.length > 0` in the current project.
- [ ] Indicator is hidden when no agents are running.
- [ ] Indicator is not rendered on non-project pages (login, project picker).
- [ ] Count text uses correct singular/plural grammar.
- [ ] Pulsing green dot is visible and animates.
- [ ] Clicking the indicator navigates to `/p/:project/agents`.
- [ ] `aria-label` is present and descriptive.
- [ ] Styles match existing header action items (typography, spacing, colours, hover).

---

## Milestone 2: Remove RunStatusChip and its usage

### Description
Delete the floating pill component and remove its import/usage from `WorkspaceView.vue`. After this milestone there is exactly one running-agents indicator in the UI (FR-7).

### Files to change
- `web/src/components/agent/RunStatusChip.vue` — delete entirely.
- `web/src/views/project/WorkspaceView.vue` — remove the `import RunStatusChip` line and the `<RunStatusChip :project="getProject()" />` element from the template.

### Acceptance criteria
- [ ] `RunStatusChip.vue` no longer exists on disk.
- [ ] `WorkspaceView.vue` has no import or template reference to `RunStatusChip`.
- [ ] No `<Teleport to="body">` remains for the running-agents indicator purpose.
- [ ] The application builds without errors (`pnpm build` succeeds).

---

## Milestone 3: Theme and responsiveness verification

### Description
Manually verify the indicator renders correctly across themes and viewport sizes. No code changes expected — this milestone is a visual QA gate.

### Files to review (no changes expected)
- `web/src/components/layout/AppHeader.vue` — inspect styles added in Milestone 1.
- `web/src/assets/` — confirm CSS custom properties used are defined for both themes.

### Acceptance criteria
- [ ] Indicator renders correctly in light theme (NFR-5).
- [ ] Indicator renders correctly in dark theme (NFR-5).
- [ ] At 768px viewport width, indicator collapses to dot + count only (NFR-2).
- [ ] With `prefers-reduced-motion: reduce`, the dot is static (NFR-3).
