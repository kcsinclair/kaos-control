---
title: 'Frontend Plan: Ready-Count Badge and Running-State Highlight'
type: plan-frontend
status: in-development
lineage: agent-panel-status-and-ready-count
parent: lifecycle/requirements/agent-panel-status-and-ready-count-2.md
---

## Overview

Add a ready-count badge and running-state visual highlight to each agent card in `AgentPanelRow.vue`. Counts update in real time via WebSocket events. Implements [[agent-panel-status-and-ready-count]] requirements FR-1, FR-3, FR-4, FR-5.

## Milestone 1: Pinia Store — Ready Counts State and Action

**Description:** Extend `stores/agents.ts` with a `readyCounts` reactive map and a `fetchReadyCounts()` action that calls the new backend endpoint from [[agent-panel-status-and-ready-count]]-be.

**Files to change:**
- `web/src/stores/agents.ts` — add `readyCounts: ref<Record<string, number>>({})`, add `fetchReadyCounts(project: string)` action, add `runningCountByAgent` computed (groups `activeRuns` by `agent_name`)

**Implementation details:**
- `fetchReadyCounts()` calls `GET /api/p/:project/agents/ready-counts` and assigns `response.counts` to `readyCounts.value`
- `runningCountByAgent` computed: reduces `activeRuns` into `Record<string, number>` keyed by `agent_name`
- Call `fetchReadyCounts()` inside existing `fetchAgents()` flow (or alongside it in the view)

**Acceptance criteria:**
- [ ] `readyCounts` state populated after `fetchReadyCounts()` resolves
- [ ] `runningCountByAgent` correctly returns per-agent running count from `activeRuns`
- [ ] `vue-tsc --noEmit` passes with no type errors

## Milestone 2: Real-Time Updates via WebSocket

**Description:** Refresh ready counts when artifacts change and keep running state synced via existing agent events.

**Files to change:**
- `web/src/views/project/WorkspaceView.vue` — add `artifact.indexed` event handling to trigger `agentsStore.fetchReadyCounts()`
- `web/src/stores/agents.ts` — ensure `onWsEvent` for `agent.started`/`agent.finished`/`agent.failed` already updates `runs` (it does — `activeRuns` computed will react automatically)

**Implementation details:**
- In `WorkspaceView.vue` WebSocket handler (line 39 area), add a case for `artifact.indexed` that calls `agentsStore.fetchReadyCounts(project)` (debounce 500 ms to avoid thrashing during bulk re-index)
- Running state is already reactive: `activeRuns` updates on `agent.started`/`agent.finished`/`agent.failed` via existing `onWsEvent` handler

**Acceptance criteria:**
- [ ] When an artifact status changes (triggers `artifact.indexed`), `readyCounts` refreshes within 2 seconds
- [ ] When an agent starts/finishes, `runningCountByAgent` updates without page refresh
- [ ] No excessive API calls during bulk re-indexing (debounce protects)

## Milestone 3: Ready-Count Badge UI

**Description:** Render a pill-shaped badge on each agent card showing the ready-artifact count from the store.

**Files to change:**
- `web/src/components/agent/AgentPanelRow.vue` — add badge element to each agent button

**Implementation details:**
- Import `useAgentsStore` and read `readyCounts` in the component
- For each agent, show badge only if agent has an `active_status` (check `agent.active_status` field from `AgentSummary`)
- Badge displays `readyCounts[agent.name] ?? 0`
- Badge is a small pill (`display: inline-flex; border-radius: 9999px; padding: 2px 8px; font-size: 0.7rem; font-weight: 600`)
- Position in the card's trailing area (after agent name or roles)
- Add `aria-label` e.g. `"3 artifacts ready"` for accessibility
- Zero count shown as `0` (not hidden)
- Badge clickable: navigates to `/p/:project/artifacts?status=<active_status>` (per resolved Q1)

**Acceptance criteria:**
- [ ] Badge visible on all agent cards with `active_status` configured
- [ ] Badge hidden for agents without `active_status`
- [ ] Zero displayed as `0`, not hidden
- [ ] `aria-label` present with correct count
- [ ] Clicking badge navigates to filtered artifact list
- [ ] No layout shift when count updates (fixed min-width on badge)

## Milestone 4: Running-State Highlight

**Description:** Apply a green border/glow with pulse animation to agent cards that have active runs, consistent with `AppHeader.vue` styling. Show run count per resolved Q2.

**Files to change:**
- `web/src/components/agent/AgentPanelRow.vue` — add conditional class and run-count indicator

**Implementation details:**
- Add class `.agent-panel--running` when `runningCountByAgent[agent.name] > 0`
- CSS for `.agent-panel--running`:
  - `border-color: #22c55e` (green-500, matches AppHeader)
  - `box-shadow: 0 0 8px rgba(34, 197, 94, 0.3)` (glow)
  - `animation: pulse 1.5s ease-in-out infinite` (reuse existing `@keyframes pulse` from AppHeader or define locally)
- Show small running count badge (e.g. green circle with number) when count > 0
- Respect `prefers-reduced-motion`: disable animation
- Secondary cue beyond colour: the pulse animation itself + a small running icon (lucide `Play` or `Loader2` spinning)

**Acceptance criteria:**
- [ ] Agent cards with active runs show green border + glow + pulse animation
- [ ] Run count displayed when > 0 (per Q2)
- [ ] Cards revert to default styling when all runs finish/fail
- [ ] Animation disabled for `prefers-reduced-motion`
- [ ] Works for all driver types (Claude Code CLI, Ollama, inline)
- [ ] `pnpm build` passes with no errors
- [ ] No visible layout shift when highlight activates/deactivates
