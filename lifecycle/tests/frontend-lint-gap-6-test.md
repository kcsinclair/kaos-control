---
title: Frontend Lint Coverage Integration Tests
type: test
status: draft
lineage: frontend-lint-gap
parent: lifecycle/test-plans/frontend-lint-gap-5-test.md
created: "2026-08-25T19:30:00+10:00"
---

# Frontend Lint Coverage Integration Tests

Implements all 5 milestones of [[frontend-lint-gap-5-test]] against the real
`Makefile`, `web/eslint.config.js`, `tests/web/eslint.config.js`, and
`lifecycle/devops/{test-lint,all-tests}.yaml` — not mocks or synthetic
copies, per the plan's "No Mock Scaffolding" resolution. All new files are
build-tagged `integration`; run with:

```
go test -tags=integration ./tests/... -run 'TestMakefile|TestFrontendLint|TestDevopsLintPipeline|TestMakeLint'
```

## Milestone 1 — Makefile targets & fail-fast

**File:** `tests/frontend_lint_makefile_test.go`

| Test | Scenario |
|---|---|
| `TestMakefileLintFrontend_InvokesExpectedCommands` | `make -n lint-frontend` dry-run shows `pnpm run lint` then `vue-tsc --noEmit` |
| `TestMakefileLintGo_InvokesExpectedCommands` | `make -n lint-go` dry-run shows `go vet`, `staticcheck`, `govulncheck`, `gosec`, `gitleaks` |
| `TestMakefileLint_InvokesBoth` | `make -n lint` dry-run shows lint-go's commands before lint-frontend's |
| `TestMakefileLint_FailFast_BackendFailureStopsFrontend` | lint-go failure halts `make lint` before lint-frontend runs |
| `TestMakefileLint_FailFast_FrontendFailurePropagates` | lint-frontend failure (lint-go passing) still fails `make lint` |

Dry-run (`make -n`) is used for command-composition assertions so these tests
need no linters installed. Fail-fast is proven against a synthetic Makefile
with the same `lint: lint-go lint-frontend` dependency shape (isolating GNU
Make's default sequential-halt-on-first-failure behaviour from the real
recipes, which need a fully provisioned toolchain).

## Milestone 2 — ESLint rule enforcement

**Files:** `tests/frontend_lint_rules_test.go`, `tests/fixtures/frontend-lint/{violations,valid}/*`

Runs the real `web/eslint.config.js` (via `web/node_modules/.bin/eslint -f json`)
against one fixture per FR-3 pattern. `TestFrontendLintRules_ViolationsAreFlagged`
asserts each of the 8 patterns (unused var, floating promise, misused
promise, unused component, prop mutation, `v-html`, `eqeqeq`, `prefer-const`)
fires its exact rule ID with a positive line/column (NFR-4).
`TestFrontendLintRules_ValidPatternsPass` asserts the permitted counterpart
of each (`_`-prefixed args, `void` promises, sync-wrapped async calls, strict
equality / `== null`, `const`, a used component with read-only props and no
`v-html`) lints clean.

`tests/fixtures/frontend-lint/tsconfig.json` anchors typescript-eslint's
automatic project-service discovery so the type-aware rules
(`no-floating-promises`, `no-misused-promises`) work on fixture files outside
`web/`'s own tsconfig — no change to any production config was needed.

## Milestone 3 — Test suite override ergonomics

**Files:** `tests/frontend_lint_test_overrides_test.go`, `tests/fixtures/frontend-lint/test-overrides/*`

Runs the real `tests/web/eslint.config.js` against fixtures verifying
`@typescript-eslint/no-explicit-any` is off for mocks/spies
(`TestFrontendLintTestOverrides_AnyIsPermittedForMocks`), while a floating
promise and a non-`_`-prefixed unused import are still hard failures
(`TestFrontendLintTestOverrides_RealBugsAreCaught`) — `no-unused-vars` keeps
the same `^_` override as production; only `no-explicit-any` is relaxed.

## Milestone 4 — DevOps pipeline execution

**File:** `tests/devops_lint_pipeline_test.go`

`TestDevopsLintPipeline_LoadsRealDefinitions` loads the actual
`lifecycle/devops/test-lint.yaml` and `all-tests.yaml` via
`internal/devops.Discover` and asserts both declare `Lint` / `make lint` as
step 1. `TestDevopsLintPipeline_ExecutesAgainstRealRepo` drives
`test-lint.yaml`'s single Lint step through `internal/devops.Runner` with the
real repo root as the working directory (a synthetic fixture project has no
Makefile, so `make lint` needs the genuine tree) and asserts the runner
executes the step to completion well under the 2-minute budget, with Go
diagnostics in the captured output. `all-tests.yaml`'s other 4 steps
(unit/frontend/integration/e2e) are checked at the metadata level only —
actually running them would itself spawn long, resource-heavy nested test
suites, out of scope for verifying the Lint step's wiring.

## Milestone 5 — Clean baseline & performance

**File:** `tests/frontend_lint_benchmark_test.go`

`TestMakeLint_CleanBaseline` runs the real `make lint` and asserts exit 0.
`TestMakeLintFrontend_PerformanceBudget` warms the ESLint cache with one
untimed run, then asserts a second `make lint-frontend` run completes in
under 5.0s (NFR-1). `TestMakeLintFrontend_OfflineExecution` runs
`make lint-frontend` with `http(s)_proxy` pointed at an unreachable local
port and asserts it still succeeds, proving no network dependency (NFR-2).

## Known gap: current lint baseline is not clean

`TestMakeLint_CleanBaseline` and the second half of
`TestDevopsLintPipeline_ExecutesAgainstRealRepo` (frontend diagnostics
present in the Lint step's output) **fail as of this writing**, independent
of this test-writing work. `make lint` currently fails during `lint-go`:
gosec flags `internal/agent/errors.go:23`
(`FailureReasonTurnTokenCeiling = "turn_token_ceiling"`) as `G101` (potential
hardcoded credential) — a false positive on the constant name, introduced by
an unrelated commit (`118aa48d`, "Milestone 4 — Structured Error Taxonomy &
Event Enrichment") landed on this branch shortly before this test suite was
written. Because `make lint`'s targets are fail-fast (Milestone 1 above), `lint-go`
failing means `lint-frontend` never runs, so the devops execution test
correctly reports no frontend diagnostics in that run's output.

Both are genuine, working tests correctly reporting the repo's current
state — not defects in this test suite. Fixing the false positive (either a
`#nosec G101` annotation on that line, or a `G101` gosec exclusion in the
Makefile alongside the existing G104/G124/etc. list) is backend-developer
scope (`internal/agent/`, `Makefile`) and outside this agent's
`allowed_write_paths` (`tests/**`, `lifecycle/tests/`,
`lifecycle/architecture/decisions/`). All other 20 tests across the 5
milestones pass.
