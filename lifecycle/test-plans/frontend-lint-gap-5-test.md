---
title: 'Test Plan: Frontend Lint Coverage Integration'
type: plan-test
status: in-development
lineage: frontend-lint-gap
created: "2026-08-24T19:30:00+10:00"
parent: lifecycle/requirements/frontend-lint-gap-2.md
labels:
    - test
    - tooling
    - quality
release: KC-Release6
assignees:
    - role: product-owner
      who: agent
---

# Test Plan: Frontend Lint Coverage Integration

## Overview

Integration, verification, and regression tests for [[frontend-lint-gap-2]]. This test plan covers the automated verification of the new Makefile targets (`make lint-frontend`, `make lint-go`, `make lint`), ESLint 9.x flat config rule enforcement via positive and negative test cases, test override ergonomics in `tests/web/`, DevOps pipeline execution (`lifecycle/devops/all-tests.yaml` and `lifecycle/devops/test-lint.yaml`), and performance benchmarking (< 5s execution budget).

Cross-references:
- [[frontend-lint-gap]] — Originating idea.
- [[frontend-lint-gap-2]] — Requirement specification.
- [[frontend-lint-gap-3-be]] — Backend & DevOps plan (Makefile targets, pipeline updates, stack profile).
- [[frontend-lint-gap-4-fe]] — Frontend plan (ESLint Flat Config, rules, and baseline remediation).
- [[architecture-summary]] — Architectural constraints.
- [[go-vue]] — Promoted tech-stack profile.

---

## Milestone 1 — Makefile Target & Fail-Fast Integration Tests

### Description

Author integration tests verifying that `make lint-frontend`, `make lint-go`, and `make lint` execute the expected sub-commands, enforce fail-fast semantics, and propagate non-zero exit codes when errors or warnings occur.

### Files to change

- `tests/frontend_lint_makefile_test.go` (new) — Integration tests covering:
  - `make lint-frontend` executes `pnpm run lint` and `vue-tsc --noEmit`.
  - `make lint-go` executes Go linters and security checks (`go vet`, `staticcheck`, `govulncheck`, `gosec`, `gitleaks`).
  - `make lint` invokes both `lint-go` and `lint-frontend`.
  - A failure in `lint-frontend` causes `make lint` to fail with non-zero exit code.
  - A failure in `lint-go` causes `make lint` to fail with non-zero exit code.

### Acceptance criteria

- [ ] All Makefile targets are exercised in test assertions.
- [ ] Fail-fast behaviour is verified for both backend and frontend failures.
- [ ] Non-zero exit code is correctly returned on failure.
- [ ] `go test -tags=integration ./tests/frontend_lint_makefile_test.go` passes.

---

## Milestone 2 — ESLint Rule Enforcement & Negative Test Validation

### Description

Create synthetic test fixtures and test cases verifying that the ESLint configuration accurately flags and fails on each prohibited defect pattern defined in FR-3:
1. Unused variable or import without `_` prefix (`@typescript-eslint/no-unused-vars`).
2. Unhandled floating promise (`@typescript-eslint/no-floating-promises`).
3. Misused promise in a synchronous callback context (`@typescript-eslint/no-misused-promises`).
4. Unused component declared in a Vue SFC (`vue/no-unused-components`).
5. Direct prop mutation inside a child component (`vue/no-mutating-props`).
6. Raw `v-html` without scoped disable comment (`vue/no-v-html`).
7. Loose equality comparison (`eqeqeq`).
8. Reassigned `let` that should be `const` (`prefer-const`).

Also verify positive test cases (e.g. `_`-prefixed arguments, `void` promise calls, strict equality `===`) pass without errors.

### Files to change

- `tests/fixtures/frontend-lint/` (new directory) — Fixtures demonstrating violations and valid exceptions.
- `tests/frontend_lint_rules_test.go` (new) — Integration test executing ESLint against the fixture files and asserting expected diagnostic messages and exit codes.

### Acceptance criteria

- [ ] Each prohibited pattern triggers a lint failure with rule name and location.
- [ ] Permitted patterns (`_` prefixes, `void` promises, strict comparisons) pass cleanly.
- [ ] Diagnostics include file path, line, column, and rule name (NFR-4).

---

## Milestone 3 — Test Suite Override & Mock Ergonomics Validation

### Description

Verify that test files in `tests/web/` can leverage testing conveniences without triggering lint errors, while remaining protected against syntax and floating promise bugs.

### Files to change

- `tests/frontend_lint_test_overrides_test.go` (new) — Verify:
  - Usage of `any` in mock objects and spy return values is permitted in test files.
  - Unused imports in skipped test suites or fixture files do not cause hard failures if configured.
  - Unhandled promises in test files are still flagged.

### Acceptance criteria

- [ ] Test files using `any` for mocks pass without lint errors.
- [ ] Actual bugs in test files (e.g. floating promises, syntax errors) are caught.

---

## Milestone 4 — DevOps Pipeline Execution & Timeout Verification

### Description

Validate that the updated DevOps pipeline specifications (`lifecycle/devops/all-tests.yaml` and `lifecycle/devops/test-lint.yaml`) run the unified `make lint` check within the execution engine.

### Files to change

- `tests/devops_lint_pipeline_test.go` (new or extended devops runner tests) — Test cases:
  - Loading `all-tests.yaml` and executing step 1 (`Lint`).
  - Executing `test-lint.yaml` pipeline.
  - Asserting the step completes in < 2 minutes.

### Acceptance criteria

- [ ] Pipeline runner successfully executes the `Lint` step.
- [ ] Step output includes both Go and frontend lint diagnostics.
- [ ] Execution completes well under the 2m timeout budget.

---

## Milestone 5 — Clean Baseline, Performance Benchmarking & Determinism

### Description

Verify that the entire repository passes `make lint` with zero errors and zero warnings, and benchmark `make lint-frontend` execution speed to ensure compliance with the < 5s threshold (NFR-1) and offline operation (NFR-2).

### Files to change

- `tests/frontend_lint_benchmark_test.go` (new) — Benchmark and baseline test:
  - Runs `make lint` on the repository: asserts exit code 0, 0 errors, 0 warnings.
  - Measures duration of `make lint-frontend`: asserts duration < 5.0 seconds.
  - Verifies offline execution without network dependencies.

### Acceptance criteria

- [ ] `make lint` passes cleanly across the entire codebase with `--max-warnings 0`.
- [ ] `make lint-frontend` completes in < 5.0 seconds on standard developer hardware.
- [ ] No network calls occur during lint execution (deterministic offline operation).

---

## Companion Test Artifact

Following implementation of the test suite, the `test-developer` agent will author the companion test artifact:
- `lifecycle/tests/frontend-lint-gap-6-test.md` (type: `test`, lineage: `frontend-lint-gap`, parent: `lifecycle/test-plans/frontend-lint-gap-5-test.md`).

---

## Resolved Questions

1. **Execution Resequencing**: The prerequisite plans [[frontend-lint-gap-3-be]] (Backend & DevOps) and [[frontend-lint-gap-4-fe]] (Frontend) have been unblocked (`status: approved`) and granted necessary path permissions. In the execution lifecycle, BE and FE plans will land first. Once `make lint-frontend` and `web/eslint.config.js` exist in the repo, this test plan will execute all 5 milestones.
2. **Output Format & Exit Code Semantics**: Standard Unix exit codes MUST be asserted (0 on success, non-zero on lint/type errors or failure). Output diagnostics will follow default ESLint/`vue-tsc` human-readable formatting (`<file>:<line>:<col> <rule-name>`).
3. **No Mock Scaffolding**: All 5 milestones will wait for the BE and FE implementation tasks to land so that integration tests are written directly against the live, authoritative targets and configuration files.
