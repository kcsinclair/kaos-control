---
title: 'Agent Panel: Ready Count Badge and Running-State Highlight'
type: requirement
status: approved
lineage: agent-panel-status-and-ready-count
created: "2026-05-10"
priority: normal
parent: lifecycle/ideas/agent-panel-status-and-ready-count.md
labels:
    - agent
    - frontend
    - enhancement
    - usability
    - vue
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

## Problem

The Agents screen shows agent cards with static configuration data (name, role, driver, model) but provides no insight into live system state. Users cannot tell at a glance:

1. **How much work is queued** — to see artifacts waiting for a given agent, a user must navigate to the artifact list, filter by the agent's `active_status`, and mentally map results back to agents. This friction discourages monitoring and delays action.
2. **Which agents are currently running** — the app header shows a global running-agent count, but the Agents screen itself gives no per-agent visual feedback. Users must cross-reference the run history table to determine which specific agent is active.

Both gaps slow down the feedback loop between "work is ready" and "an agent is processing it."

## Goals / Non-goals

### Goals

- Display a badge on each agent card showing the count of artifacts whose status matches that agent's `active_status`, giving an at-a-glance queue depth.
- Visually distinguish agent cards that have at least one run in `running` state, using styling consistent with the existing green running-state indicator in `AppHeader.vue`.
- Keep counts live — update automatically when artifacts are indexed or agent runs start/finish, without requiring a page refresh.
- Ensure the feature works for all agent driver types (Claude Code CLI, Ollama, inline).

### Non-goals

- Showing a detailed artifact list per agent on the Agents screen (the existing artifact views handle that).
- Adding click-to-run or auto-run behaviour based on ready counts (out of scope for this change).
- Changing the app header's existing global running-agent indicator.
- Providing historical queue-depth metrics or charts.

## Detailed Requirements

### Functional

#### FR-1: Ready-count badge on agent cards

Each agent card rendered by `AgentPanelRow.vue` must display a numeric badge indicating the number of artifacts whose `status` equals the agent's `active_status`.

- If the agent has no `active_status` configured (empty string), display no badge.
- A count of zero should display as `0` (not hidden) so users can distinguish "nothing ready" from "badge not applicable."
- The badge must be visually prominent but not dominant — e.g. a small pill-shaped element near the agent name or in the card's trailing area.

#### FR-2: Backend endpoint for ready counts

Provide an API endpoint that returns the ready-artifact count for each agent in a single request:

```
GET /api/p/:project/agents/ready-counts
```

Response shape:

```json
{
  "counts": {
    "requirements-analyst": 3,
    "planning-analyst": 1,
    "backend-developer": 0,
    ...
  }
}
```

The endpoint must use the existing `index.Count(Filter{Status: agent.ActiveStatus})` path to avoid duplicating query logic. Only agents with a non-empty `active_status` should appear in the response.

#### FR-3: Running-state visual highlight

When an agent has one or more runs currently in `running` state, its card must display a visual highlight:

- Apply a green border or background tint consistent with the pulsing green badge used in `AppHeader.vue` (lines 41-48, CSS classes around `bg-green-500`).
- The highlight should include a subtle animation (pulse or glow) to convey liveness.
- When no runs are active for the agent, the card reverts to its default styling.

#### FR-4: Real-time updates via WebSocket

Ready counts and running state must update in real time without polling:

- **Artifact indexed** (`artifact.indexed` WS event): the frontend must refresh ready counts. This can be done by re-fetching the ready-counts endpoint or by the backend pushing updated counts in the event payload.
- **Agent started/finished/failed** (`agent.started`, `agent.finished`, `agent.failed` WS events): the agents store already tracks these — use the existing `activeRuns` computed property to derive per-agent running state.

#### FR-5: Pinia store integration

Extend `stores/agents.ts`:

- Add a `readyCounts` reactive map (`Record<string, number>`) populated from the ready-counts endpoint.
- Add a computed `isRunning(agentName: string): boolean` helper (or equivalent) derived from `activeRuns`.
- Add a `fetchReadyCounts()` action that calls the new endpoint.
- Call `fetchReadyCounts()` on store initialisation and on receipt of `artifact.indexed` events.

### Non-functional

#### NFR-1: Performance

The ready-counts endpoint must complete in under 50 ms for projects with up to 10,000 indexed artifacts. It runs simple `COUNT(*)` queries against the SQLite index with a status filter, so this should be trivially met.

#### NFR-2: No layout shift

Adding the badge and running highlight must not change the card's dimensions or cause visible layout reflow when counts update.

#### NFR-3: Accessibility

- The badge count must be readable by screen readers (use `aria-label` such as "3 artifacts ready").
- The running-state highlight must not rely solely on colour — include a secondary cue (animation, icon, or text label).

## Acceptance Criteria

- [ ] Each agent card displays a numeric badge showing the count of artifacts matching the agent's `active_status`
- [ ] Agents with no `active_status` show no badge
- [ ] A count of zero is displayed as `0`, not hidden
- [ ] `GET /api/p/:project/agents/ready-counts` returns correct counts for all agents with an `active_status`
- [ ] When an artifact's status changes (indexed via filesystem or API write), the badge updates within 2 seconds without page refresh
- [ ] Agent cards for agents with active runs show a green highlight with animation, consistent with [[agent-panel-status-and-ready-count]] header styling
- [ ] Agent cards revert to default styling when all runs for that agent finish or fail
- [ ] The running-state highlight works for all driver types (Claude Code CLI, Ollama, inline)
- [ ] Badge includes `aria-label` for screen reader accessibility
- [ ] No visible layout shift when badge counts change
- [ ] `pnpm build` and `vue-tsc --noEmit` pass with no new errors
- [ ] `go build ./...` and `go vet ./...` pass with no new errors

## Resolved Questions

- **Q1**: Should the ready-count badge be clickable to navigate to a filtered artifact list (e.g. artifacts with that status)? This would add useful drill-down but may be better scoped as a follow-up.

> Yes, great idea.

- **Q2**: Should the running-state highlight differentiate between a single running instance and multiple concurrent runs of the same agent (e.g. show a run count)?

> Yes, show a run count.

- **Q3**: For the Ollama driver, the `active_status` field behaves the same as for Claude Code CLI — confirm there are no driver-specific edge cases for ready-count semantics.

> We need to work on Ollama, focus on Claude for now.
