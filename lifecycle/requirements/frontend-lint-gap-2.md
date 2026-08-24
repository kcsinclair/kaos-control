---
title: Frontend Lint Coverage Gap — ESLint & vue-tsc Integration
type: requirement
status: blocked
lineage: frontend-lint-gap
created: "2026-08-24T18:35:00+10:00"
priority: medium
parent: lifecycle/ideas/frontend-lint-gap.md
labels:
    - frontend
    - tooling
    - quality
    - devops
release: KC-Release6
assignees:
    - role: product-owner
      who: agent
---

# Frontend Lint Coverage Gap — ESLint & vue-tsc Integration

## Problem

The Go backend enforces comprehensive static analysis and linting via `make lint` (`go vet`, `staticcheck`, `govulncheck`, `gosec`, and `gitleaks`). Lint failures fail builds and block merges.

In contrast, the Vue 3 + TypeScript frontend currently has **no linting configuration or static analysis enforcement** in standard developer and CI loops:
1. Grepping the repository reveals no ESLint configuration (`.eslintrc*`, `eslint.config*`) or formatting configurations.
2. The only frontend type gate declared is `"type-check": "vue-tsc --noEmit"` in `web/package.json`, which is not invoked by `make lint` or the DevOps pipelines (`lifecycle/devops/all-tests.yaml`, `lifecycle/devops/test-lint.yaml`).
3. Static type checks only run if an engineer or agent remembers to manually execute `pnpm run type-check`.

As a result, ~50 `.vue` Single File Components and ~100+ `.ts` files across `web/` and `tests/web/` lack automated linting to catch high-frequency defect patterns (such as unused variables, unhandled floating promises, misused async functions, mutating props, unused components, and loose equality). This gap has already permitted real bugs to ship undetected until late end-to-end or runtime execution (e.g. stale mock queue drift and API response shape parsing defects).

## Goals / Non-goals

### Goals

- **Comprehensive Linting in `make lint`:** Wire frontend type checking (`vue-tsc`) and ESLint into `make lint` alongside existing Go tooling so that any frontend lint or type error fails fast before code is committed or merged.
- **Minimal Correctness Rule Set:** Implement ESLint (Flat Config, ESLint 9.x) using `@typescript-eslint` and `eslint-plugin-vue` / `@vue/eslint-config-typescript` with a conservative, high-value rule set focused on catching actual bugs (unused variables, floating promises, mutating props, unused components).
- **Test Suite Coverage:** Include `tests/web/` in lint coverage with an override block tailored to testing ergonomics (e.g. allowing `any` in test mocks and looser return assertions).
- **DevOps Pipeline Integration:** Update pipeline specifications (`lifecycle/devops/all-tests.yaml` and `lifecycle/devops/test-lint.yaml`) to run the frontend lint checks automatically.
- **Clean Initial Baseline:** Resolve all baseline violations across `web/src/` and `tests/web/` so `make lint` passes cleanly with zero errors and zero warnings (`--max-warnings 0`).

### Non-goals

- **Format/Style Enforcement (Prettier):** Pure aesthetic style rules (quotes, indentation, semicolons) are out of scope and deferred to a dedicated Prettier evaluation.
- **Strict-Mode TypeScript Migration:** Overhauling existing types to satisfy strict type-checking flags (e.g. `noImplicitAny` everywhere or `strict-type-checked`) is out of scope for this initial gate.
- **Biome Migration:** Retaining standard ESLint + `eslint-plugin-vue` ecosystem for Vue 3 SFC support rather than switching to Biome.
- **Stylelint for CSS:** Linting `<style>` blocks within `.vue` SFCs is excluded as unnecessary overhead for current requirements.
- **Git Pre-commit Hooks:** Enforcing linting via client-side git hooks is out of scope; `make lint` and the automated pipelines remain the authoritative contract.

## Detailed Requirements

### Functional Requirements

- **FR-1: Makefile Frontend Lint Targets**
  - The root `Makefile` MUST provide dedicated frontend linting targets:
    - `lint-frontend`: Executes frontend type checks (`vue-tsc --noEmit`) and ESLint checks (`pnpm run lint`) inside `web/`.
    - `lint-go`: Encapsulates existing Go checks (`go vet`, `staticcheck`, `govulncheck`, `gosec`, `gitleaks`).
  - The top-level `lint` target MUST depend on both `lint-go` and `lint-frontend` (or run them sequentially), failing immediately with a non-zero exit code if either step encounters errors or warnings.

- **FR-2: ESLint Flat Config Setup in `web/`**
  - `web/package.json` MUST include ESLint 9.x flat config dependencies under `devDependencies`:
    - `eslint` (`^9.x`)
    - `@eslint/js` (`^9.x`)
    - `typescript-eslint` (`^8.x`)
    - `eslint-plugin-vue` (`^9.x`)
    - `@vue/eslint-config-typescript` (`^14.x`)
  - A root flat configuration file (`web/eslint.config.js` or `web/eslint.config.mjs`) MUST be introduced, properly configuring parsers for TypeScript and Vue 3 SFCs (`<script setup lang="ts">`).
  - `web/package.json` MUST define a `"lint"` script: `eslint . --max-warnings 0`.

- **FR-3: Targeted Correctness Rule Set**
  - The ESLint configuration MUST enforce the following core correctness rules across production code:
    - `@typescript-eslint/no-unused-vars`: Disallow unused variables while permitting variables and arguments prefixed with `_` (e.g. `argsIgnorePattern: "^_"`, `varsIgnorePattern: "^_"`).
    - `@typescript-eslint/no-floating-promises`: Ensure promises are handled, awaited, or explicitly marked with `void`.
    - `@typescript-eslint/no-misused-promises`: Prevent passing async functions to locations expecting synchronous callbacks.
    - `vue/no-unused-components`: Disallow registration of components that are not used in templates.
    - `vue/no-mutating-props`: Prevent direct mutation of component props.
    - `vue/no-v-html`: Warn/error on `v-html` usage, with explicit inline disable comments permitted only where raw markdown rendering is intentional.
    - `eqeqeq`: Enforce strict equality operators (`===` and `!==`).
    - `prefer-const`: Enforce `const` for variables never reassigned.

- **FR-4: Test Code Ergonomics & Overrides (`tests/web/`)**
  - ESLint configuration MUST cover test files in `tests/web/`.
  - An override configuration block MUST relax rules that hinder test authoring:
    - Allow `@typescript-eslint/no-explicit-any` for test mocks and stubs.
    - Relax unused import/variable constraints for skipped test suites and fixture declarations where appropriate.

- **FR-5: DevOps Pipelines Updates**
  - `lifecycle/devops/all-tests.yaml` MUST ensure its `Lint` step invokes `make lint` (covering both Go and frontend checks) with a timeout sufficient for both suites.
  - `lifecycle/devops/test-lint.yaml` MUST execute the unified `make lint` command.

- **FR-6: Baseline Remediation**
  - All existing lint and type errors in `web/src/` and `tests/web/` MUST be remediated so that `make lint` succeeds out-of-the-box with zero warnings and zero errors.

- **FR-7: Stack Profile & Agent Prompts Alignment**
  - The developer agent prompt templates and stack profile in `lifecycle/architecture/go-vue.md` and `lifecycle/config.yaml` MUST remain consistent with the frontend lint command (`cd web && pnpm run lint` / `cd web && pnpm exec vue-tsc --noEmit`).

### Non-Functional Requirements

- **NFR-1: Execution Speed**
  - Frontend lint execution (`make lint-frontend` / `pnpm run lint` + `vue-tsc`) MUST complete in under 5 seconds on a standard developer machine for the existing codebase (~150 files).
- **NFR-2: Determinism & Offline Operation**
  - Linting MUST run completely offline once dependencies are installed via `pnpm install`, producing deterministic results regardless of environment or execution mode.
- **NFR-3: Zero Warnings Policy**
  - Linting MUST run with `--max-warnings 0`. Warnings MUST fail the build to prevent warning fatigue and gradual code health degradation.
- **NFR-4: Developer Feedback Quality**
  - Output from lint failures MUST clearly state the file path, line number, column number, and rule name to facilitate immediate remediation by human engineers and autonomous agents.

### Architecture-Breaking Requirements

- **Analysis against [[architecture-summary]]:**
  - **Single self-contained binary:** Frontend linting is purely a development-time and CI-time static analysis check. It adds dev dependencies to `web/package.json` and does not affect the production build output (`web/dist/`) or the pure-Go single-binary distribution model ([[adr-0004-embedded-spa-single-binary]]).
  - **Local filesystem source of truth:** Linting inspects local files on disk without external service dependencies, conforming to [[index-is-a-cache]] and [[filesystem-sandboxing]].
  - **Agent permission model:** Developer agents operating in `web/` already have write permissions for frontend source files ([[adr-0006-mediated-agent-driver-permission-model]]).
  - **Verdict:** There are **no architecture-breaking requirements**. The requirement operates entirely within the constraints of the Go + Vue modular monolith ([[modular-monolith]]) and does not require a new ADR or changes to `architecture-summary.md`.

## Acceptance Criteria

- [ ] `Makefile` contains `lint-frontend` and `lint-go` targets, and `make lint` executes both.
- [ ] `web/package.json` includes ESLint 9.x flat config dependencies (`eslint`, `@eslint/js`, `typescript-eslint`, `eslint-plugin-vue`, `@vue/eslint-config-typescript`).
- [ ] `web/eslint.config.js` (or `.mjs`) is present and configures Vue 3 SFCs, TypeScript, and the targeted correctness rules. *(FR-2, FR-3)*
- [ ] `tests/web/` is covered by ESLint with appropriate override rules for test mock flexibility. *(FR-4)*
- [ ] `lifecycle/devops/all-tests.yaml` and `lifecycle/devops/test-lint.yaml` execute the unified lint check. *(FR-5)*
- [ ] Existing codebase passes `make lint` cleanly with 0 errors and 0 warnings under `--max-warnings 0`. *(FR-6, NFR-3)*
- [ ] `make lint-frontend` executes in under 5 seconds locally. *(NFR-1)*
- [ ] Related artifacts linked: [[frontend-lint-gap]], [[go-vue]], [[agent-directives-generation]].

## Open Questions

*None. All design questions raised in the originating idea [[frontend-lint-gap]] have been resolved (conservative ESLint Flat Config 9.x selected, test override block agreed, Prettier and Stylelint deferred/skipped, Biome deferred).*
