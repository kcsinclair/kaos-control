---
title: "Backend Plan — Rename Analyst Agents to Phase-First Convention"
type: plan-backend
status: done
lineage: rename-analyst-agents
parent: lifecycle/requirements/rename-analyst-agents-2.md
---

# Backend Plan — Rename Analyst Agents

This plan covers all Go source, configuration, documentation, and lifecycle artifact changes required to rename `analyst-requirements` → `requirements-analyst` and `analyst-planner` → `planning-analyst`. No runtime behaviour changes; only identifier strings change.

Cross-links: [[rename-analyst-agents]] (frontend plan), [[rename-analyst-agents]] (test plan).

---

## Milestone 1 — Rename agents in lifecycle/config.yaml

### Description

Update the two analyst agent entries in `lifecycle/config.yaml` so that `name`, `git_identity.name`, and `git_identity.email` use the new phase-first convention.

### Files to change

- `lifecycle/config.yaml`

### Changes

1. `name: analyst-requirements` → `name: requirements-analyst`
2. `git_identity.name: Analyst (Requirements) Agent` → `git_identity.name: Requirements Analyst Agent`
3. `git_identity.email: analyst-requirements@kaos-control.local` → `git_identity.email: requirements-analyst@kaos-control.local`
4. `name: analyst-planner` → `name: planning-analyst`
5. `git_identity.name: Analyst (Planner) Agent` → `git_identity.name: Planning Analyst Agent`
6. `git_identity.email: analyst-planner@kaos-control.local` → `git_identity.email: planning-analyst@kaos-control.local`

### Acceptance criteria

- [ ] `grep -c 'analyst-requirements\|analyst-planner' lifecycle/config.yaml` returns 0.
- [ ] `grep -c 'requirements-analyst' lifecycle/config.yaml` returns at least 2 (name + email).
- [ ] `grep -c 'planning-analyst' lifecycle/config.yaml` returns at least 2 (name + email).

---

## Milestone 2 — Update Go workflow test comments

### Description

Update comments in the Go workflow test file that reference the old agent names.

### Files to change

- `internal/workflow/workflow_test.go`

### Changes

1. Line 59 comment: `analyst-requirements` → `requirements-analyst`.
2. Line 61 comment: `analyst-planner` → `planning-analyst`.

### Acceptance criteria

- [ ] `grep -c 'analyst-requirements\|analyst-planner' internal/workflow/workflow_test.go` returns 0.
- [ ] `go test ./internal/workflow/... -short` passes.

---

## Milestone 3 — Update CLAUDE.md agent listing

### Description

Update the line in `CLAUDE.md` that lists all six agents to use the new names.

### Files to change

- `CLAUDE.md`

### Changes

1. Replace `analyst-requirements`, `analyst-planner` with `requirements-analyst`, `planning-analyst` in the agent listing sentence.

### Acceptance criteria

- [ ] `CLAUDE.md` lists agents as `requirements-analyst`, `planning-analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`.
- [ ] No remaining references to `analyst-requirements` or `analyst-planner` in `CLAUDE.md`.

---

## Milestone 4 — Update project plans

### Description

Update references to old agent names in project plan documents.

### Files to change

- `plans/PROJECT_PLAN.md`
- `plans/role-vocabulary-migration.md`
- `plans/agent-runtime-hardening.md`

### Changes

1. Find-and-replace `analyst-requirements` → `requirements-analyst` in all three files.
2. Find-and-replace `analyst-planner` → `planning-analyst` in all three files.

### Acceptance criteria

- [ ] `grep -r 'analyst-requirements\|analyst-planner' plans/` returns 0 matches.

---

## Milestone 5 — Update lifecycle artifact body text

### Description

Update references to old agent names in lifecycle artifact bodies (requirements, plans, tests, defects). These are documentation references, not historical record.

### Files to change

- `lifecycle/requirements/agent-launcher-panels-2.md`
- `lifecycle/requirements/analyst-agent-sees-draft-ideas-2.md`
- `lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md`
- `lifecycle/requirements/prompt-to-idea-2.md`
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

### Changes

1. In each file, find-and-replace `analyst-requirements` → `requirements-analyst`.
2. In each file, find-and-replace `analyst-planner` → `planning-analyst`.

### Acceptance criteria

- [ ] `grep -r 'analyst-requirements\|analyst-planner' lifecycle/requirements/ lifecycle/backend-plans/ lifecycle/frontend-plans/ lifecycle/test-plans/ lifecycle/tests/ lifecycle/defects/` returns 0 matches (excluding `lifecycle/config.yaml.backup`).

---

## Milestone 6 — Build and lint verification

### Description

Verify the full build pipeline passes after all renames.

### Commands

```sh
make build
make lint
make test-unit
```

### Acceptance criteria

- [ ] `make build` succeeds (Go compiles, frontend embeds).
- [ ] `make lint` passes with no new warnings.
- [ ] `make test-unit` passes.
