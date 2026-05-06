---
title: Move Running Agents Indicator to Menu Bar
type: requirement
status: done
lineage: agents-indicator-in-menu-bar
parent: lifecycle/ideas/agents-indicator-in-menu-bar.md
labels:
    - frontend
    - enhancement
    - usability
    - agent
    - vue
assignees:
    - role: product-owner
      who: agent
---

# Move Running Agents Indicator to Menu Bar

## Problem

The running-agents indicator (`RunStatusChip.vue`) is currently rendered as a fixed-position pill in the bottom-right corner of the viewport via `<Teleport to="body">`. This placement has two problems:

1. **Occlusion** — the pill overlaps content in views with bottom-anchored elements (e.g. editor status bars, scrollable lists) and can be hidden by modals or drawers with higher z-index.
2. **Discoverability** — users who trigger an agent run and navigate away from the Agents view have no persistent, predictable location to check run status. The bottom-right corner is not where users habitually look for global state.

## Goals / Non-goals

### Goals

- Provide a persistent, always-visible indicator of active agent runs in the application header (`AppHeader.vue`).
- Show the count of currently running agents and update it in real time as runs start and stop.
- Allow the user to click the indicator to navigate to the Agents view for the current project.
- Remove the existing bottom-right floating pill so there is a single, canonical location for this information.

### Non-goals

- Displaying a dropdown/popover with per-agent detail directly from the header indicator (future enhancement).
- Showing completed or failed run counts in the header — only active (running) agents are in scope.
- Modifying the Agents list view itself.
- Adding notification badges, sounds, or browser notifications for agent state changes.

## Detailed Requirements

### Functional

1. **FR-1: Header placement.** The running-agents indicator MUST be rendered inside `AppHeader.vue`, positioned in the `header-actions` area to the left of existing controls (theme toggle, user name, sign-out).
2. **FR-2: Count display.** When one or more agents are running, the indicator MUST display the text `N running agent(s)` where N is the count, using correct singular/plural grammar (e.g. "1 running agent", "3 running agents").
3. **FR-3: Pulsing dot.** The indicator MUST include a pulsing green dot (matching the existing `run-dot` animation) to the left of the text, providing a visual heartbeat signal.
4. **FR-4: Real-time updates.** The count MUST update reactively as agent runs start and stop. The existing `useAgentsStore().activeRuns` computed property already provides this via WebSocket events; the indicator MUST consume it.
5. **FR-5: Hidden when idle.** When no agents are running (`activeRuns.length === 0`), the indicator MUST NOT be rendered — no empty-state placeholder, no "0 running agents" text.
6. **FR-6: Click navigation.** Clicking the indicator MUST navigate to the Agents view for the current project (`/p/:project/agents`).
7. **FR-7: Remove floating pill.** The existing `RunStatusChip.vue` component and its `<Teleport to="body">` usage in `WorkspaceView.vue` MUST be removed. There must be exactly one running-agents indicator in the UI.
8. **FR-8: Project context.** The indicator MUST only appear when the user is inside a project workspace (i.e. when a project slug is available from the route). It MUST NOT appear on the project picker or login views.

### Non-functional

1. **NFR-1: Consistency.** The indicator's typography, spacing, and colour palette MUST match the existing `AppHeader` action items (use `--text-sm`, `--color-sidebar-text-muted` for text, `--color-border-dark` for any borders, same `btn-icon`/`btn-ghost` hover patterns).
2. **NFR-2: Responsiveness.** The indicator MUST remain usable at viewport widths down to 768px. If the header becomes crowded, the indicator may collapse to show only the dot and count (no text label) at narrow widths.
3. **NFR-3: Accessibility.** The indicator MUST have an `aria-label` describing the current state (e.g. "2 running agents — click to view"). The pulsing dot animation MUST respect `prefers-reduced-motion` by falling back to a static dot.
4. **NFR-4: Performance.** The indicator MUST NOT introduce additional API polling. It relies solely on the existing Pinia store, which is fed by the WebSocket connection already established per project.
5. **NFR-5: Theme support.** The indicator MUST render correctly in both light and dark themes, using CSS custom properties already defined in the design system.

## Acceptance Criteria

- [ ] The running-agents indicator appears in `AppHeader` when at least one agent is active in the current project.
- [ ] The indicator shows the correct count and updates within 1 second of an agent starting or stopping (bounded by existing WS event latency).
- [ ] Clicking the indicator navigates to `/p/:project/agents`.
- [ ] The indicator is hidden when no agents are running.
- [ ] The indicator is not visible on the project picker or login screens.
- [ ] The old fixed-position `RunStatusChip` pill is fully removed — no `<Teleport to="body">` remains for this purpose.
- [ ] The pulsing dot respects `prefers-reduced-motion`.
- [ ] The indicator has a descriptive `aria-label`.
- [ ] The indicator looks correct in both light and dark themes.
- [ ] At viewport width 768px the indicator remains visible and usable.

## Resolved Questions

1. Should the indicator show a brief transition animation (e.g. fade in/out) when the first agent starts or the last agent stops, or should it appear/disappear instantly?
2. Is there a desired maximum width or truncation behaviour if the agent count reaches double digits (e.g. "12 running agents")?
