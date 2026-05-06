# Role Vocabulary Migration + Agent Expansion

## Context

The project currently has one agent (`backend-planner`) and a role list that conflates planning and coding (`backend-planner`, `frontend-planner`, `developer`). The product is ready to onboard the rest of the agent roster, and this change is the prerequisite.

The new model splits the workflow into three phases:
- **Think**: `analyst` reads ideas → writes requirements, reads requirements → writes plans (backend + frontend + test).
- **Make**: `backend-developer`, `frontend-developer`, `test-developer` each read their plan and produce code / tests.
- **Verify**: `qa` runs integration tests and files defects in the new `lifecycle/defects/` stage, assigning each defect to whichever developer role caused it.

This plan implements that model end-to-end: config, code defaults, workflow transitions, a new `defect` artifact type, and the spec. Decisions approved up front:
- Analyst is implemented as **two agents sharing `role: [analyst]`** — `requirements-analyst` and `planning-analyst` — for focused prompts and safer path scoping.
- Analyst can self-submit (draft→clarifying, clarifying→planning).
- Tickets are gated: all three plans must be approved before entering `in-development`.
- The spec file is updated in the same commit to prevent drift.

## Role changes

| Old | New | Notes |
|---|---|---|
| product-owner | product-owner | unchanged |
| backend-planner | backend-developer | now codes, does not plan |
| frontend-planner | frontend-developer | now codes, does not plan |
| developer | test-developer | now writes integration tests only |
| (none) | analyst | new; writes requirements + plans |
| qa | qa | unchanged; now owns defect creation |
| reviewer | reviewer | unchanged |
| approver | approver | unchanged |

## New agents (six total)

| Agent | role | writes to |
|---|---|---|
| `requirements-analyst` | analyst | `lifecycle/requirements/` |
| `planning-analyst` | analyst | `lifecycle/backend-plans/`, `lifecycle/frontend-plans/`, `lifecycle/test-plans/` |
| `backend-developer` | backend-developer | `internal/`, `cmd/` |
| `frontend-developer` | frontend-developer | `web/src/` |
| `test-developer` | test-developer | `tests/` (repo root), `lifecycle/tests/` |
| `qa` | qa | `lifecycle/defects/` |

All agents use `driver: claude-code-cli`. Each gets its own focused prompt template.

## Files modified

### 1. `internal/config/config.go`

- **`defaultRoles`**: replaced the 7-element slice with the new 8-element slice:
  ```go
  var defaultRoles = []string{
      "product-owner", "analyst",
      "backend-developer", "frontend-developer", "test-developer",
      "qa", "reviewer", "approver",
  }
  ```
- **`defaultStages`**: appended `{Name: "defects", Dir: "defects"}`.

### 2. `internal/artifact/artifact.go`

- **`KnownTypes`**: added `"defect": true`.
- **`stageToType()`**: added `case "defects": return "defect"`.

### 3. `internal/workflow/workflow.go`

Replaced `defaultRules` with:

```go
var defaultRules = []rule{
    {from: "draft", to: "clarifying", roles: []string{"product-owner", "analyst"}},
    {from: "clarifying", to: "planning", roles: []string{"product-owner", "reviewer", "analyst"}},
    {from: "planning", to: "in-development", roles: []string{"approver"}},
    {from: "in-development", to: "in-qa", roles: []string{"backend-developer", "frontend-developer", "test-developer"}},
    {from: "in-qa", to: "approved", roles: []string{"qa"}},
    {from: "approved", to: "done", roles: []string{"approver"}},
    {from: "clarifying", to: "draft", roles: []string{"product-owner", "analyst"}},
    {from: "", to: "rejected", roles: []string{"reviewer"}},
    {from: "", to: "abandoned", roles: []string{"product-owner", "approver"}},
}
```

`GateReady()` needed no code change — it already reads `required` from config. We just populated it.

### 4. `lifecycle/config.yaml`

Full rewrite of `roles`, `agents`, and `required_plans` sections. Preserved top-level `git:` and `users:`. Added `stages:` stanza explicitly so `defects` is visible.

Key structural additions:
```yaml
required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []

stages:
  - {name: ideas,          dir: ideas}
  - {name: requirements,   dir: requirements}
  - {name: backend-plans,  dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: dev-plans,      dir: dev-plans}
  - {name: test-plans,     dir: test-plans}
  - {name: tests,          dir: tests}
  - {name: prototypes,     dir: prototypes}
  - {name: releases,       dir: releases}
  - {name: sprints,        dir: sprints}
  - {name: defects,        dir: defects}
```

Agents got full prompt templates (drafted below). Each uses `{target_path}` substitution (already supported by the existing driver).

### 5. `lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md`

Targeted edits:
- §2 Personas table: rebuilt with new role list.
- §4.2 `type` vocabulary: added `defect`.
- §5.1 Directory layout: added `defects/` and `tests/` (repo-root).
- §5.2 Scope of access: noted developer agents now write to multiple code paths.
- §6.2 Transition matrix: updated to match `workflow.go` defaults.
- §6.3 Plan branches: changed default to require all three plans.
- §7.1 Agent example: replaced `claude-planner` example with `planning-analyst` + `backend-developer`.
- §7.3 Trigger-model wording: example phrase updated.
- §13.3 Project-level config example: roles list, users binding, required_plans aligned.

### 6. `tests/.gitkeep`

Created at repo root so `test-developer`'s `allowed_write_paths` (`tests`, `lifecycle/tests`) targets an existing directory.

## Prompt templates (draft)

All use `{target_path}` substitution and reference CLAUDE.md for lineage conventions.

**requirements-analyst** — idea → requirement, with clarifying Q&A if needed. Frontmatter: `type: ticket, status: draft, parent: <idea path>`. Sections: Problem, Goals/Non-goals, Requirements, Acceptance Criteria, Open Questions.

**planning-analyst** — requirement → three plan artifacts. Each plan has `type: plan-{backend|frontend|test}`, linked via shared lineage. Body structured as ordered milestones with acceptance criteria per milestone.

**backend-developer** — backend plan → Go code in `internal/` + `cmd/`. Runs `go build ./...` and `go vet ./...` per milestone before commit.

**frontend-developer** — frontend plan → Vue/TS in `web/src/`. Runs `pnpm exec vue-tsc --noEmit` + `pnpm build` per milestone.

**test-developer** — test plan → integration tests in `tests/` + a `test` artifact in `lifecycle/tests/` documenting coverage.

**qa** — runs tests relevant to `{target_path}`; for each failure writes a `defect` artifact in `lifecycle/defects/` with frontmatter `type: defect, status: draft, lineage: <feature lineage>, parent: <failing test or feature>, assignees: [{role: backend-developer|frontend-developer|test-developer, who: agent}]` and body containing repro steps, expected vs actual, logs.

## Verification

1. **Build clean**: `go build ./...` and `go vet ./...` pass.
2. **Config parses**: validated by loading `lifecycle/config.yaml` through the real `config.LoadProject` Go loader — all six agents recognised, 11 stages, `required_plans.ticket = [plan-backend, plan-frontend, plan-test]`.
3. **Roles propagate to UI**: on the graph, click a node with no frontmatter → artifact list filter shows new roles/types when filtering.
4. **Transition matrix**: pick a `draft` artifact, log in as a user with role `analyst` (bind via `users:` in config), verify the Change Status dropdown shows `clarifying`. Log in as `reviewer` only → dropdown does not show `clarifying` for draft.
5. **Gate enforcement**: create a ticket in `planning` without any plans → attempt `planning→in-development` → server rejects with `missing_plans`. Add approved backend/frontend/test plans → transition succeeds.
6. **Defect type**: create a file in `lifecycle/defects/` with `type: defect` in frontmatter → watcher indexes it → appears in artifact list under the new `defects` stage filter.
7. **Agent invocation**: open the Run Agent dialog from an idea → `requirements-analyst` listed and invokable.

## Out of scope (explicit non-goals)

- Auto-triggering agents on status changes.
- Pull-request workflow (agent commits to branch → PR → review).
- Migration of existing `agent_runs` rows that reference `backend-planner` — historical rows stay as-is; new runs use new names.
- Per-role git identity beyond what each agent config already declares.
