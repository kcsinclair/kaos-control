---
title: Rename Analyst Agents to Phase-First Convention
type: requirement
status: planning
lineage: rename-analyst-agents
created: "2026-05-06"
priority: high
parent: lifecycle/ideas/rename-analyst-agents.md
labels:
    - agent
    - process
assignees:
    - role: product-owner
      who: agent
---

## Problem

The two analyst agents are currently named `analyst-requirements` and `analyst-planner`, placing the role first and the lifecycle phase second. Every other agent in the system is named with the phase/function as the primary identifier (e.g. `backend-developer`, `frontend-developer`, `test-developer`). The analyst naming is inconsistent: when scanning a list of agents sorted alphabetically, both analyst agents cluster together instead of appearing near the phase they serve. This makes it harder to find the right agent at a glance and breaks the implicit `<phase>-<role>` convention used elsewhere.

## Goals / Non-goals

### Goals

1. Rename `analyst-requirements` to `requirements-analyst` and `analyst-planner` to `planning-analyst` across the entire codebase.
2. Update all configuration, code, tests, documentation, and lifecycle artifacts that reference the old names.
3. Maintain backward compatibility for existing artifacts — no backfill of historical frontmatter or git history is required.

### Non-goals

- Renaming the `analyst` role itself (it remains `analyst`).
- Changing agent behaviour, prompts, or scoped write paths (only the `name` and `git_identity` fields change).
- Backfilling old agent names in previously-committed artifact frontmatter or git commit metadata.
- Changing the email local-part convention (it simply follows the new name: `requirements-analyst@kaos-control.local`).

## Detailed Requirements

### Functional

1. **Config rename** — In `lifecycle/config.yaml`, rename the agent entries:
   - `name: analyst-requirements` → `name: requirements-analyst`
   - `git_identity.name: Analyst (Requirements) Agent` → `git_identity.name: Requirements Analyst Agent`
   - `git_identity.email: analyst-requirements@kaos-control.local` → `git_identity.email: requirements-analyst@kaos-control.local`
   - `name: analyst-planner` → `name: planning-analyst`
   - `git_identity.name: Analyst (Planner) Agent` → `git_identity.name: Planning Analyst Agent`
   - `git_identity.email: analyst-planner@kaos-control.local` → `git_identity.email: planning-analyst@kaos-control.local`

2. **Frontend agent-to-type mapping** — In `web/src/components/agent/AgentLaunchModal.vue`, update the `AGENT_INPUT_TYPE_MAP` (or equivalent mapping) keys from `analyst-requirements`/`analyst-planner` to `requirements-analyst`/`planning-analyst`.

3. **Frontend test fixtures** — In `tests/web/helpers/agent_launch_fixtures.ts`, update the agent name keys in any agent-to-type mappings.

4. **Integration test helpers** — In `tests/integration/agent_helpers_test.go`, update the embedded config YAML and any comments referencing the old names.

5. **Integration test cases** — In the following test files, update all string literals and comments referencing `analyst-requirements` or `analyst-planner`:
   - `tests/integration/agent_status_test.go`
   - `tests/integration/agent_ws_test.go`
   - `tests/integration/agents_api_test.go`

6. **Frontend test cases** — Update agent name string literals in:
   - `tests/web/ArtifactRunHistory.test.ts`
   - `tests/web/AgentsRunsView.sort.test.ts`
   - `tests/web/agent-launch/defect-inclusion.test.ts`
   - `tests/web/agent-launch/input-status-filter.test.ts`
   - `tests/web/agent-launch/empty-state.test.ts`

7. **Go workflow tests** — Update comments in `internal/workflow/workflow_test.go` that reference the old agent names.

8. **Documentation** — Update `CLAUDE.md` where it lists the six agents by name.

9. **Project plans** — Update references in `plans/role-vocabulary-migration.md`, `plans/agent-runtime-hardening.md`, and `plans/PROJECT_PLAN.md` to use the new names.

10. **Lifecycle artifacts** — Update references to old agent names in lifecycle artifact bodies (requirements, plans, tests, defects) where they appear as documentation rather than historical record. Specifically:
    - `lifecycle/requirements/agent-launcher-panels-2.md`
    - `lifecycle/requirements/analyst-agent-sees-draft-ideas-2.md`
    - `lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md`
    - `lifecycle/frontend-plans/agent-launcher-panels-4-fe.md`
    - `lifecycle/frontend-plans/analyst-agent-sees-draft-ideas-4-fe.md`
    - `lifecycle/backend-plans/analyst-missing-in-progress-status-2-be.md`
    - `lifecycle/test-plans/agent-launcher-panels-5-test.md`
    - `lifecycle/test-plans/analyst-missing-in-progress-status-4-test.md`
    - `lifecycle/test-plans/analyst-agent-sees-draft-ideas-5-test.md`
    - `lifecycle/tests/analyst-missing-in-progress-status-5.md`
    - `lifecycle/tests/analyst-agent-sees-draft-ideas-6-test.md`
    - `lifecycle/tests/agent-launcher-panels-6.md`
    - `lifecycle/defects/analyst-missing-in-progress-status.md`

### Non-functional

1. **No runtime behaviour change** — Agent execution, status transitions, allowed write paths, prompt templates, and model selection must remain identical; only the identifier strings change.
2. **Atomic rename** — All references must be updated in a single coordinated change to avoid a state where some code uses old names and some uses new names.
3. **Build verification** — `make build`, `make lint`, `make test-unit`, and all integration/web tests must pass after the rename.

## Acceptance Criteria

- [ ] `lifecycle/config.yaml` contains `name: requirements-analyst` and `name: planning-analyst` with no remaining references to the old names.
- [ ] `git_identity.email` fields use `requirements-analyst@kaos-control.local` and `planning-analyst@kaos-control.local`.
- [ ] `web/src/components/agent/AgentLaunchModal.vue` maps `requirements-analyst` → `idea` and `planning-analyst` → `requirement`.
- [ ] `CLAUDE.md` lists agents as `requirements-analyst`, `planning-analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`.
- [ ] `grep -r 'analyst-requirements\|analyst-planner' internal/ web/src/ tests/ CLAUDE.md lifecycle/config.yaml` returns zero matches.
- [ ] `make build` succeeds (Go compiles, frontend builds).
- [ ] `make lint` passes with no new warnings.
- [ ] `make test-unit` passes.
- [ ] All integration tests in `tests/integration/` pass.
- [ ] All web tests in `tests/web/` pass.
- [ ] Existing artifacts produced before the rename continue to render and index correctly (no backfill required).
- [ ] The agent launcher modal correctly lists `requirements-analyst` and `planning-analyst` with the correct input artifact types.

## Resolved Questions

1. **Lifecycle artifact updates** — Should references to the old names in existing lifecycle artifacts (plans, requirements, test docs, defects) be updated to the new names, or left as historical record? This requirement assumes they should be updated for consistency, but the idea states "no backfill of historical artifacts is required." Clarification needed on whether lifecycle artifact *body text* (not frontmatter) counts as "historical."

> Yes, please update the frontmatter.

2. **`project-notes.md`** — This file references `analyst-requirements` in prose. Should it be updated, or is it considered a personal scratch file outside the rename scope?

> outside scope.
