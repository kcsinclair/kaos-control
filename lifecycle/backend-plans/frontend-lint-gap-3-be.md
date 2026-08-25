---
title: 'Backend & DevOps Plan: Frontend Lint Coverage Integration'
type: plan-backend
status: in-development
lineage: frontend-lint-gap
created: "2026-08-24T19:30:00+10:00"
parent: lifecycle/requirements/frontend-lint-gap-2.md
labels:
    - backend
    - devops
    - tooling
    - quality
release: KC-Release6
assignees:
    - role: product-owner
      who: agent
---

# Backend & DevOps Plan: Frontend Lint Coverage Integration

## Overview

Implement the backend, build system, and DevOps pipeline updates for [[frontend-lint-gap-2]]. This plan restructures the root `Makefile` to introduce dedicated, composable lint targets (`lint-go`, `lint-frontend`), updates the top-level `lint` target to enforce both Go and frontend static analysis fail-fast, updates the DevOps pipeline specifications (`lifecycle/devops/all-tests.yaml` and `lifecycle/devops/test-lint.yaml`), and aligns the Go + Vue tech-stack profile ([[go-vue]]) and agent configuration directives in `lifecycle/config.yaml`.

Cross-references:
- [[frontend-lint-gap]] — Originating idea.
- [[frontend-lint-gap-2]] — Requirement specification.
- [[frontend-lint-gap-4-fe]] — Frontend plan (ESLint 9.x Flat Config, rules, and baseline remediation).
- [[frontend-lint-gap-5-test]] — Test plan (verification, negative test cases, and performance benchmarking).
- [[architecture-summary]] — Architecture constraints and single-binary distribution.
- [[go-vue]] — Promoted tech-stack profile.

---

## Milestone 1 — Makefile Target Splitting & Unified Lint Target

### Description

Refactor the root `Makefile` to decouple Go static analysis from frontend checks and provide dedicated, composable targets. Currently, `make lint` runs Go tools (`go vet`, `staticcheck`, `govulncheck`, `gosec`, `gitleaks`).

Introduce:
1. `lint-go`: Encapsulates all existing Go static analysis and security tools.
2. `lint-frontend`: Executes frontend type checks (`vue-tsc --noEmit`) and ESLint checks (`pnpm run lint`) inside `web/`.
3. Update `lint`: Sequences `lint-go` and `lint-frontend` (or runs both in dependency order), failing fast with a non-zero exit code if any tool reports errors or warnings.

### Files to change

- `Makefile` — Add `lint-go` and `lint-frontend` targets; update `lint` target and `help` descriptions.

### Acceptance criteria

- [ ] `make lint-go` executes Go checks (`go vet`, `staticcheck`, `govulncheck`, `gosec`, `gitleaks`) and exits with 0 on clean code.
- [ ] `make lint-frontend` navigates to `web/` and runs `pnpm run lint` followed by `pnpm run type-check` (or `vue-tsc --noEmit`).
- [ ] `make lint` executes both `lint-go` and `lint-frontend`.
- [ ] `make lint` halts immediately with a non-zero exit code if either Go or frontend checks fail.
- [ ] `make help` displays clear descriptions for `lint`, `lint-go`, and `lint-frontend`.

---

## Milestone 2 — DevOps Pipeline Specifications Update

### Description

Update the automated pipeline definitions in `lifecycle/devops/` to align with the unified lint targets. Ensure that pipeline runners invoking `all-tests.yaml` and `test-lint.yaml` execute the comprehensive `make lint` check and have sufficient timeout allowances for both Go and frontend linting.

### Files to change

- `lifecycle/devops/all-tests.yaml` — Update the `Lint` step description to reflect both Go and frontend checks; verify the 2m timeout is sufficient for both.
- `lifecycle/devops/test-lint.yaml` — Update description and ensure `make lint` runs as the single gate.

### Acceptance criteria

- [ ] `lifecycle/devops/all-tests.yaml` contains an updated `Lint` step description referencing Go (`go vet`, `staticcheck`, etc.) and frontend (`vue-tsc`, `eslint`).
- [ ] `lifecycle/devops/test-lint.yaml` executes `make lint`.
- [ ] Both pipeline YAML files parse cleanly and pass schema validation.

---

## Milestone 3 — Stack Profile & Agent Directives Alignment

### Description

Update the Go + Vue tech-stack profile in `lifecycle/architecture/go-vue.md` and catalog entry `lifecycle/architecture/tech-stacks/go-vue.md` so the machine-readable `stack_profile.roles.frontend-developer.lint` command reflects the unified frontend lint command (`cd web && pnpm run lint && pnpm exec vue-tsc --noEmit`). Update `lifecycle/config.yaml` prompt templates for the `frontend-developer` role and `test-runner` to ensure developer agents run both ESLint and `vue-tsc` after each milestone.

### Files to change

- `lifecycle/architecture/go-vue.md` — Update `frontend-developer.lint` in `stack_profile`.
- `lifecycle/architecture/tech-stacks/go-vue.md` — Update `frontend-developer.lint` in catalog `stack_profile`.
- `lifecycle/config.yaml` — Update `frontend-developer` prompt template in `agents` list to include `pnpm run lint`.

### Acceptance criteria

- [ ] `lifecycle/architecture/go-vue.md` and `lifecycle/architecture/tech-stacks/go-vue.md` declare `lint: cd web && pnpm run lint && pnpm exec vue-tsc --noEmit` under `frontend-developer`.
- [ ] `lifecycle/config.yaml` prompt template for `frontend-developer` includes running `pnpm run lint` alongside `vue-tsc` and `pnpm build`.
- [ ] `internal/directives` tests (if applicable) parse the updated stack profile without error.

---

## Resolved Questions

1. **Role Ownership**: The plan remains assigned to `backend-developer`. Build system, Makefile targets, DevOps pipeline specifications, and stack profiles fall under backend/devops developer responsibilities.
2. **Write Scope Permissions**: `lifecycle/config.yaml` has been updated to add `Makefile`, `lifecycle/devops`, `lifecycle/architecture`, and `lifecycle/config.yaml` to `allowed_write_paths` for `backend-developer`. The agent is fully authorized to edit these files during execution of this plan.
3. **`internal/directives` behavior**: No new Go code or behavior is required in `internal/directives/config_patch.go`. The Milestone 3 acceptance criterion is purely a verification step to run `go test ./internal/directives/...` and confirm existing tests pass after stack profile YAML edits.
