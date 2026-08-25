---
title: 'Test Plan: Frontend Lint Coverage Integration'
type: plan-test
status: blocked
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

## Open Questions

Both plans this test plan depends on are `status: blocked` with zero code committed:

- [[frontend-lint-gap-3-be]] — Backend & DevOps plan. Blocked because every "Files to
  change" target (`Makefile`, `lifecycle/devops/*.yaml`, `lifecycle/architecture/go-vue.md`,
  `lifecycle/config.yaml`) falls outside the `backend-developer` write scope
  (`internal/**`, `cmd/**`).
- [[frontend-lint-gap-4-fe]] — Frontend plan. Blocked because Milestone 1
  (`web/eslint.config.js`, `web/package.json`) falls outside the `frontend-developer`
  write scope (`web/src/**`), and Milestones 3–5 cannot run without that config existing.

I independently verified the resulting gap in the current tree before starting this plan:

- `Makefile` has no `lint-go` or `lint-frontend` target — only the original Go-only `lint`
  target (`go vet`, `staticcheck`, `govulncheck`, `gosec`, `gitleaks`).
- `web/eslint.config.js` does not exist.
- `web/package.json` has no `"lint"` script.
- `lifecycle/devops/all-tests.yaml` and `lifecycle/devops/test-lint.yaml` still describe the
  `Lint` step as `go vet + staticcheck` only — not updated for frontend checks.

Every milestone in this test plan exercises infrastructure that does not exist yet:

1. **Milestone 1** asserts `make lint-frontend` / `make lint-go` behaviour — neither target
   exists.
2. **Milestone 2** runs ESLint against fixtures using rules declared in
   `web/eslint.config.js` — the file doesn't exist, so there is no rule set to assert
   against.
3. **Milestone 3** verifies the `tests/web/` override block inside the same nonexistent
   config.
4. **Milestone 4** asserts the devops `Lint` step output includes "both Go and frontend
   lint diagnostics" — the pipeline YAML hasn't been updated to run anything but `make
   lint` (Go-only).
5. **Milestone 5** benchmarks `make lint-frontend` execution time and asserts a clean
   `make lint` baseline — again, the target and the zero-warning baseline don't exist to
   measure.

I cannot write meaningful integration tests against commands, files, and rule
configurations that don't exist without guessing at their exact shape (flag names, output
format, exit-code conventions) — and doing so risks tests that pass against my own guessed
fixtures but fail (or worse, silently no-op) once the real implementation lands with a
different shape. Per my instructions, I'm stopping rather than guessing.

Blocking questions for product-owner:

1. Should this test plan wait until [[frontend-lint-gap-3-be]] and [[frontend-lint-gap-4-fe]]
   are unblocked and implemented (i.e. should it be resequenced to run *after* those two,
   not in parallel)?
2. Once unblocked, will the resulting `make lint-frontend` / `make lint-go` output format
   (plain text vs. structured/JSON) and exit-code semantics be specified anywhere I can
   reference, or should I derive assertions directly from whatever the backend/frontend
   agents actually implement at that time?
3. Is there any part of this test plan that's safe to scaffold now against a mocked/fake
   `make lint-frontend` (e.g. a fixture Makefile) purely to validate my test harness
   approach, while the real integration remains blocked — or should all five milestones
   wait together?

No test code has been committed for this plan; stopping here pending direction.
