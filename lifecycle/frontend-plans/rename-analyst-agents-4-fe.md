---
title: "Frontend Plan — Rename Analyst Agents to Phase-First Convention"
type: plan-frontend
status: in-development
lineage: rename-analyst-agents
parent: lifecycle/requirements/rename-analyst-agents-2.md
---

# Frontend Plan — Rename Analyst Agents

This plan covers all Vue/TypeScript source changes required to rename `analyst-requirements` → `requirements-analyst` and `analyst-planner` → `planning-analyst` in the frontend SPA.

Cross-links: [[rename-analyst-agents]] (backend plan), [[rename-analyst-agents]] (test plan).

---

## Milestone 1 — Update AgentLaunchModal agent-to-type mapping

### Description

The `AGENT_INPUT_TYPE_MAP` in `AgentLaunchModal.vue` maps agent names to the artifact type they accept as input. Update the two analyst keys to the new phase-first names.

### Files to change

- `web/src/components/agent/AgentLaunchModal.vue`

### Changes

1. Line 28: `'analyst-requirements': 'idea'` → `'requirements-analyst': 'idea'`
2. Line 29: `'analyst-planner': 'requirement'` → `'planning-analyst': 'requirement'`

### Acceptance criteria

- [ ] `grep -c 'analyst-requirements\|analyst-planner' web/src/components/agent/AgentLaunchModal.vue` returns 0.
- [ ] `grep -c 'requirements-analyst' web/src/components/agent/AgentLaunchModal.vue` returns 1.
- [ ] `grep -c 'planning-analyst' web/src/components/agent/AgentLaunchModal.vue` returns 1.
- [ ] The agent launcher modal correctly maps `requirements-analyst` → `idea` and `planning-analyst` → `requirement`.

---

## Milestone 2 — Type-check and build verification

### Description

Confirm the frontend builds cleanly after the rename.

### Commands (run from `web/`)

```sh
pnpm exec vue-tsc --noEmit
pnpm build
```

### Acceptance criteria

- [ ] `vue-tsc --noEmit` passes with no type errors.
- [ ] `pnpm build` succeeds and produces `web/dist/` assets.
