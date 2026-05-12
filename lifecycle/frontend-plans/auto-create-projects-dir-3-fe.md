---
title: "Frontend Plan: Auto-Create Projects Directory on First Run"
type: plan-frontend
status: done
lineage: auto-create-projects-dir
parent: lifecycle/ideas/auto-create-projects-dir.md
---

# Frontend Plan: Auto-Create Projects Directory on First Run

This feature is entirely a backend/startup concern — the Vue SPA does not interact with the projects directory, does not display its existence, and does not need to handle a "missing directory" state. The API contract is unchanged.

## Milestone 1 — No frontend changes required

### Description

Confirm that no frontend changes are needed. The backend change (see [[auto-create-projects-dir]] backend plan `auto-create-projects-dir-2-be`) ensures the directory exists before the HTTP server starts. The project list API (`/api/projects`) continues to return an empty array on fresh installs, which the SPA already handles gracefully.

### Files to change

None.

### Acceptance criteria

- [ ] `pnpm exec vue-tsc --noEmit` passes with no new errors.
- [ ] `pnpm build` succeeds.
- [ ] The SPA loads correctly on a fresh install where no projects are registered (empty project list).

## Cross-links

- [[auto-create-projects-dir]] — originating idea.
- Backend plan (`auto-create-projects-dir-2-be`) handles the actual directory creation.
- Test plan (`auto-create-projects-dir-4-test`) covers integration verification.
