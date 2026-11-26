---
title: QA Agent Defect Role Assignment
type: requirement
status: blocked
lineage: qa-agent-defect-role-assignment
created: "2026-08-24T18:32:41+10:00"
priority: normal
parent: lifecycle/ideas/qa-agent-defect-role-assignment.md
labels:
    - qa
    - defect
    - workflow
    - agent
release: KC-Release7
assignees:
    - role: product-owner
      who: agent
---

# QA Agent Defect Role Assignment

## Problem

When the QA agent or automated test runners execute test suites and detect failures, defect artifacts are generated in `lifecycle/defects/`. In earlier versions of the workflow and prompt configurations, defects were created with labels (e.g. `defect`, component tags) but lacked explicit role assignment. Consequently, defect artifacts remained unassigned after creation, stalling in backlog/triage columns and requiring manual intervention from a product owner or lead developer to inspect the failure logs and route the defect to the responsible role.

Additionally, early informal requirements referenced setting an `assigned_to` scalar field, whereas the canonical schema implemented across the codebase, parser (`internal/artifact/artifact.go`), UI (`FrontmatterPanel.vue`), and [[frontmatter-role-assignment]] requires an `assignees` array containing structured `{role: <role>, who: <who>}` mappings.

To achieve an uninterrupted, automated lifecycle from QA verification through to developer remediation, all QA defect generation mechanisms (LLM QA agents, test-runner agents, and programmatic test-runner filers) must consistently and automatically route defects to the appropriate functional role (`backend-developer`, `frontend-developer`, or `test-developer`) with `who: agent` formatted in canonical frontmatter.

## Goals / Non-goals

### Goals

1. **Automated Defect Routing:** Automatically assign every newly generated defect artifact to the responsible developer role (`backend-developer`, `frontend-developer`, or `test-developer`) based on failure layer analysis.
2. **Canonical Frontmatter Conformance:** Enforce the standard `assignees` array format (`- role: <role>\n  who: agent`) across all defect generation paths, eliminating legacy or informal `assigned_to` references.
3. **Template and Scaffolding Parity:** Ensure that all shipped project configurations (`lifecycle/config.yaml`, `lifecycle/config-default.yaml`, `lifecycle/config-ollama-dev.yaml`) and the new-project init template (`internal/initcmd/templates/config.yaml.tmpl`) carry explicit role-routing directives in their QA agent prompt templates.
4. **Programmatic Defect Filer Parity:** Ensure the programmatic test failure filer (`internal/testrunner/defect.go`) applies consistent label- and path-based routing rules and emits valid `assignees` frontmatter.
5. **Role Vocabulary Validation:** Ensure all assigned roles are valid entries in the project's configured `roles` list (`lifecycle/config.yaml`).

### Non-goals

- **Specific Human User Assignment:** Automatically assigning defects to specific individual user accounts (e.g. user email addresses). Automated assignment targets functional roles (`role: <role>`, `who: agent`); individual users or agents claim work by role.
- **Complex External Machine Learning Triage:** Introducing external machine learning triage services or third-party bug tracking APIs. Layer detection is performed via deterministic path/label heuristics in code and contextual reasoning within LLM agent prompts.
- **Modifying the Assignee Frontmatter Schema:** Changing the existing `Assignee` struct definition or introducing alternative schema fields.
- **Altering Non-Defect Creation Workflows:** Changing the assignment rules for ideas, requirements, or plan artifacts.

## Detailed Requirements

### Functional Requirements

**FR-1: Canonical Assignee Frontmatter Structure**
All defect artifacts produced by QA agents or automated test runners must include the `assignees` YAML frontmatter field formatted as a list of objects with `role` and `who` fields:
```yaml
assignees:
    - role: <developer-role>
      who: agent
```
The field name must be `assignees` (not `assigned_to`). The `who` field must default to `agent`.

**FR-2: Layer-to-Role Routing Rules**
Defects must be assigned based on the architectural layer where the failure occurred:
1. **Backend Layer (`role: backend-developer`):**
   - Failures in Go source code (`internal/**`, `cmd/**`).
   - REST API handler errors, HTTP status code mismatches, payload validation failures.
   - Database and SQLite index errors (`internal/index/**`).
   - Workflow state machine transition failures (`internal/workflow/**`).
   - Agent orchestration, CLI commands, and configuration loading failures.
2. **Frontend Layer (`role: frontend-developer`):**
   - Failures in Vue components, views, or stores (`web/src/**`).
   - TypeScript compilation (`vue-tsc`) and Vite build failures.
   - Frontend unit/component test failures (`tests/web/**`, Vitest).
   - UI layout, styling, and user interaction defects.
3. **Test Harness / Automation Layer (`role: test-developer`):**
   - Flaky, broken, or obsolete test assertions where application logic meets specification but the test implementation is defective.
   - Integration test fixture, harness, or test environment setup failures in `tests/**`.
   - Missing or misconfigured test metadata in `lifecycle/tests/**`.

**FR-3: LLM QA Agent Prompt Directives**
The prompt templates for QA agents (including `qa`, `test-runner`, and any role-bound verification agents) must include explicit instructions defining:
1. The requirement to generate the `assignees` frontmatter block for each created defect.
2. The layer-to-role routing criteria specified in FR-2.
3. A fallback rule: if the failure cause cannot be isolated to a single layer with certainty, the defect must default to `role: backend-developer`.

**FR-4: Configuration Template Synchronization**
The QA agent prompt templates across all configuration files must be synchronized and include the frontmatter specification and role-routing rules:
- `lifecycle/config.yaml`
- `lifecycle/config-default.yaml`
- `lifecycle/config-ollama-dev.yaml`
- `internal/initcmd/templates/config.yaml.tmpl` (used by `kaos-control init` for new projects)

**FR-5: Programmatic Test Runner Defect Filer Parity**
The automated defect generator (`internal/testrunner/defect.go`) must enforce role routing:
1. **Label-based routing:** If the associated `lifecycle/tests/*.md` artifact has labels containing `backend` / `backend-developer`, `frontend` / `frontend-developer`, or `test` / `test-developer`, route to that role.
2. **Path-based fallback:** If no matching label exists on the test artifact, route failures in `tests/web` and `tests/e2e` to `frontend-developer`, and all other packages to `backend-developer`.
3. **Frontmatter output:** Emit valid YAML containing `assignees: [{role: <role>, who: agent}]`.

**FR-6: Project Role Vocabulary Validation**
Assigned roles must belong to the active project's `roles` array declared in `lifecycle/config.yaml`. If a custom project configuration renames or removes standard developer roles, the prompt and filer fallbacks must conform to the declared role vocabulary.

**FR-7: Lineage, Parent, and Metadata Inheritance**
Defects created by QA agents must maintain lineage consistency:
- `lineage:` must match the lineage of the feature or test under test.
- `parent:` must point to the test artifact or requirement being verified.
- `priority:` and `release:` must be inherited from the parent artifact as specified in [[inherit-priority-and-release]].

### Non-functional Requirements

**NFR-1: Schema Validity and Round-Trip Safety**
All defect artifacts produced with `assignees` must parse cleanly into `artifact.Frontmatter` (`internal/artifact/artifact.go`) with zero parse errors and round-trip faithfully through `PUT /artifacts/*` and `GET /artifacts/*` without dropping or corrupting fields (conforming to [[frontmatter-role-assignment]]).

**NFR-2: Zero External Dependencies**
Role determination and frontmatter generation must run entirely locally within the single binary and agent runtime without relying on external cloud triage services or network lookups.

**NFR-3: Deterministic Fail-Safe Routing**
Every defect created by an automated tool or agent must have a valid role assigned; no defect artifact may be written with an empty `role` or empty `assignees` list.

### Architecture-Breaking Requirements

None. This requirement refines agent prompt directives and defect frontmatter generation within the existing architecture and tech stack recorded in `lifecycle/architecture/architecture-summary.md`. Explicit assessment against the standing architectural constraints:

- **Single self-contained binary:** *Satisfied.* No external database, triage service, or cgo dependency is introduced. Defect generation is handled by embedded templates and standard Go packages.
- **Local filesystem is the source of truth:** *Satisfied.* Defect artifacts are written as markdown files directly to `lifecycle/defects/` and indexed incrementally into the SQLite cache ([[index-is-a-cache]]).
- **Agents execute arbitrary tools / scope-enforced writes:** *Satisfied.* QA agents continue to write strictly within `allowed_write_paths` (`lifecycle/defects` and `lifecycle/architecture/decisions`), mediated by the driver permission model ([[adr-0006-mediated-agent-driver-permission-model]], [[filesystem-sandboxing]]).
- **Direct-served, no trusted proxy hop:** *Satisfied.* No change to HTTP transport, authentication, or client IP handling ([[adr-0001-no-header-based-client-ip-trust]]).
- **Offline operation & low operational complexity:** *Satisfied.* All prompt templates, validation logic, and defect filing logic function completely offline.

There is **no conflict** with `lifecycle/architecture/architecture-summary.md`; no new ADR is required.

## Acceptance Criteria

- [ ] All QA agent prompt templates in `lifecycle/config.yaml`, `lifecycle/config-default.yaml`, and `lifecycle/config-ollama-dev.yaml` explicitly instruct the agent to include `assignees` frontmatter with `who: agent` and the appropriate developer `role`. *(FR-1, FR-3, FR-4)*
- [ ] `internal/initcmd/templates/config.yaml.tmpl` includes the complete defect role routing instructions and `assignees` frontmatter specification for the `qa` and `test-runner` agents. *(FR-3, FR-4)*
- [ ] QA agent prompt templates define clear layer-to-role routing: Backend API/logic/data to `backend-developer`, UI/Vue/component to `frontend-developer`, and flaky/broken tests to `test-developer`. *(FR-2, FR-3)*
- [ ] Programmatic defect generation in `internal/testrunner/defect.go` creates defects containing `assignees:` with `who: agent` and routes role by test label and file path. *(FR-1, FR-5)*
- [ ] Defect artifacts generated by QA agents parse into `artifact.Frontmatter` with non-empty `Assignees` and index into the SQLite database without warnings. *(FR-1, NFR-1)*
- [ ] Assigned roles match valid entries in the project `roles` list. *(FR-6)*
- [ ] Defect artifacts inherit `priority` and `release` from their parent test/feature artifact. *(FR-7)* — see [[inherit-priority-and-release]]
- [ ] Frontmatter editor and Kanban boards correctly display and filter QA-generated defects by assigned role. *(NFR-1)* — see [[frontmatter-role-assignment]]
- [ ] `go vet ./...` and `go test ./...` pass with no errors.
- [ ] Related: [[qa-agent-defect-role-assignment]], [[frontmatter-role-assignment]], [[inherit-priority-and-release]]

## Open Questions

None. The schema, routing rules, prompt structures, and programmatic implementation mechanisms are fully aligned with the project's architecture and existing codebase.
