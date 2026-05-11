---
title: "Frontend: Display Per-Agent Ready Counts from Backend Endpoint"
type: plan-frontend
status: done
lineage: agent-panel-ready-count-not-role-specific
parent: lifecycle/defects/agent-panel-ready-count-not-role-specific.md
---

# Frontend: Display Per-Agent Ready Counts from Backend Endpoint

## Problem Summary

The agents Pinia store (`web/src/stores/agents.ts`) fetches a single `approvedCount` from the artifacts list endpoint and displays the same number for every agent panel. The backend already exposes a per-agent ready-counts endpoint (`GET /api/p/:project/agents/ready-counts`) that is defined in `web/src/api/agents.ts` but never called. The frontend must switch to using per-agent counts.

---

## Milestone 1: Use Per-Agent Ready Counts from the Backend Endpoint

### Description

Replace the `fetchReadyCounts` implementation in the agents store to call the existing `getReadyCounts()` API function (already defined in `web/src/api/agents.ts`) and populate the existing `readyCounts` ref with the per-agent map.

### Files to Change

- `web/src/stores/agents.ts` — rewrite `fetchReadyCounts()` to call `agentsApi.getReadyCounts(project)` and store the returned `counts` map into `readyCounts`. Remove (or deprecate) `approvedCount`.

### Acceptance Criteria

- [ ] `fetchReadyCounts()` calls `GET /api/p/:project/agents/ready-counts`.
- [ ] `readyCounts` ref is a `Record<string, number>` keyed by agent name.
- [ ] `approvedCount` is removed or no longer used by any component.
- [ ] `pnpm exec vue-tsc --noEmit` passes.

---

## Milestone 2: Update AgentPanelRow to Read Per-Agent Count

### Description

Modify the `readyCount()` helper in `AgentPanelRow.vue` to look up the agent's name in `agentsStore.readyCounts` instead of returning the shared `approvedCount`.

### Files to Change

- `web/src/components/agent/AgentPanelRow.vue` — change `readyCount(agent)` to return `agentsStore.readyCounts[agent.name] ?? 0`.

### Acceptance Criteria

- [ ] Each agent panel displays the count specific to that agent (not a shared value).
- [ ] Agents with no entry in the counts map display `0 ready`.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 3: Update Badge Click Navigation to Filter by Type

### Description

Currently the badge click navigates to `?status=approved`. Update it to also include the agent's source type(s) so users see only the artifacts relevant to that agent.

### Files to Change

- `web/src/components/agent/AgentPanelRow.vue` — update `handleBadgeClick` to include `type` query parameter based on the agent's configuration (source types). If the agent summary exposes `source_types`, use the first entry; otherwise fall back to status-only filtering.
- `web/src/api/agents.ts` or `web/src/types/` — ensure the `AgentSummary` type includes `source_types?: string[]` from the backend response.

### Acceptance Criteria

- [ ] Clicking the ready badge on `backend-developer` navigates to `?status=in-development&type=plan-backend` (or equivalent).
- [ ] Clicking the badge on agents without `source_types` falls back to `?status=<active_status>`.
- [ ] `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 4: Expose `source_types` in the Agent List API Response

### Description

If the backend agent list endpoint doesn't already include `source_types` in the JSON payload, this milestone is handled by the [[agent-panel-ready-count-not-role-specific]] backend plan. On the frontend side, ensure the TypeScript type definition includes the field.

### Files to Change

- `web/src/types/` or inline type in `web/src/api/agents.ts` — add `source_types?: string[]` to the `AgentSummary` / agent response type.

### Acceptance Criteria

- [ ] `AgentSummary` type includes `source_types?: string[]`.
- [ ] No TypeScript errors.
- [ ] Badge click uses the field when present.

---

## Cross-References

- [[agent-panel-ready-count-not-role-specific]] backend plan — provides the per-agent counts endpoint and `source_types` in agent config.
- [[agent-panel-ready-count-not-role-specific]] test plan — verifies distinct badge values per agent.
