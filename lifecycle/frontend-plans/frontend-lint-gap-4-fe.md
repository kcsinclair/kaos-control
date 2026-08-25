---
title: 'Frontend Plan: ESLint Flat Config & Baseline Remediation'
type: plan-frontend
status: in-development
lineage: frontend-lint-gap
created: "2026-08-24T19:30:00+10:00"
parent: lifecycle/requirements/frontend-lint-gap-2.md
labels:
    - frontend
    - tooling
    - quality
release: KC-Release6
assignees:
    - role: product-owner
      who: agent
---

# Frontend Plan: ESLint Flat Config & Baseline Remediation

## Overview

Implement the frontend static analysis infrastructure and remediate all baseline violations for [[frontend-lint-gap-2]]. This plan introduces ESLint 9.x Flat Config (`web/eslint.config.js`), installs `@typescript-eslint` and `eslint-plugin-vue` / `@vue/eslint-config-typescript` dependencies in `web/package.json`, establishes a targeted correctness rule set, configures ergonomics overrides for `tests/web/`, and remediates existing code violations across all `.vue` Single File Components and `.ts` files to achieve a zero-warning, zero-error baseline under `--max-warnings 0`.

Cross-references:
- [[frontend-lint-gap]] — Originating idea.
- [[frontend-lint-gap-2]] — Requirement specification.
- [[frontend-lint-gap-3-be]] — Backend & DevOps plan (Makefile targets, pipeline updates, stack profile).
- [[frontend-lint-gap-5-test]] — Test plan (verification, negative fixtures, and performance benchmarking).
- [[architecture-summary]] — Architectural constraints (embedded SPA, local filesystem cache).
- [[go-vue]] — Promoted tech-stack profile.

---

## Milestone 1 — ESLint Flat Config & Dependencies Setup

### Description

Install ESLint 9.x flat config dependencies under `devDependencies` in `web/package.json`:
- `eslint` (`^9.x`)
- `@eslint/js` (`^9.x`)
- `typescript-eslint` (`^8.x`)
- `eslint-plugin-vue` (`^9.x`)
- `@vue/eslint-config-typescript` (`^14.x`)

Create `web/eslint.config.js` (ESLint flat configuration format) configured with TypeScript and Vue 3 SFC parsers (`vue-eslint-parser` with `@typescript-eslint/parser` for `<script setup lang="ts">`).

Configure the core correctness rule set (FR-3):
- `@typescript-eslint/no-unused-vars`: Disallow unused variables while permitting `_`-prefixed arguments and variables (`argsIgnorePattern: "^_"`, `varsIgnorePattern: "^_"`).
- `@typescript-eslint/no-floating-promises`: Ensure promises are awaited, returned, or explicitly marked with `void`.
- `@typescript-eslint/no-misused-promises`: Prevent passing async functions where synchronous handlers are expected.
- `vue/no-unused-components`: Prevent registering components that are not referenced in SFC templates.
- `vue/no-mutating-props`: Prevent direct prop mutations within child components.
- `vue/no-v-html`: Flag `v-html` usages (permitting scoped inline disable comments only where markdown rendering is intended).
- `eqeqeq`: Enforce strict equality operators (`===` and `!==`).
- `prefer-const`: Require `const` declarations for variables never reassigned.

Add the lint script to `web/package.json`: `"lint": "eslint . --max-warnings 0"`.

### Files to change

- `web/package.json` — Add ESLint devDependencies and `"lint"` npm script.
- `web/eslint.config.js` (new) — Flat config definition for Vue 3 SFCs, TypeScript, and JavaScript files.

### Acceptance criteria

- [ ] `cd web && pnpm install` succeeds without dependency resolution conflicts.
- [ ] `web/eslint.config.js` is recognised by ESLint 9.x flat config engine.
- [ ] `pnpm run lint` executes ESLint targeting `web/` with `--max-warnings 0`.
- [ ] Correctness rules specified in FR-3 are active and enforced.

---

## Milestone 2 — Test Suite Configuration & Ergonomics Overrides

### Description

Extend the ESLint configuration in `web/eslint.config.js` to cover test files in `tests/web/` (`../tests/web/**/*.{ts,vue}` or configured glob pattern), applying an override block tailored to testing ergonomics (FR-4):
- Allow `@typescript-eslint/no-explicit-any` for test mocks, spies, and stubs.
- Relax unused variable/argument rules where test fixtures or skipped suites define placeholders.
- Maintain core syntax and promise safety across test code.

### Files to change

- `web/eslint.config.js` — Add test files pattern match and rules override section.

### Acceptance criteria

- [ ] Test files under `tests/web/` are included in lint analysis.
- [ ] Mocks using `any` or stub callbacks do not trigger lint errors in test files.
- [ ] Test code adheres to core promise safety and syntax rules.

---

## Milestone 3 — Production Code Baseline Remediation (`web/src/`)

### Description

Execute `pnpm run lint` and `pnpm run type-check` across `web/src/` (~50 `.vue` components and ~50 `.ts` files). Systematically fix all identified violations:
- Remove unused variables/imports or prefix intentional unused parameters with `_`.
- Handle floating promises by adding `await`, returning the promise, or prefixing with `void` for fire-and-forget calls.
- Fix async event handlers and callbacks where misused promises are flagged.
- Replace loose equality (`==`, `!=`) with strict equality (`===`, `!==`).
- Remove unused component imports in Vue SFC `<script>` blocks.
- Fix direct prop mutations by emitting events or using local reactive state.
- Add scoped inline disable comments (`<!-- eslint-disable-next-line vue/no-v-html -->`) strictly on intended markdown rendering sites.

### Files to change

- `web/src/**/*.{ts,vue}` (as identified during the initial lint pass)

### Acceptance criteria

- [ ] `cd web && pnpm run lint` passes on `web/src/` with 0 errors and 0 warnings.
- [ ] `cd web && pnpm run type-check` (`vue-tsc --noEmit`) passes with 0 errors.
- [ ] `cd web && pnpm run build` succeeds cleanly.
- [ ] All existing Vitest tests in `tests/web/` continue to pass without regression.

---

## Milestone 4 — Web Test Suite Baseline Remediation (`tests/web/`)

### Description

Execute `pnpm run lint` against `tests/web/` (~100 test files). Remediate all violations in the test files:
- Resolve unhandled floating promises in async test cases.
- Remove obsolete or unused fixture imports.
- Standardise equality comparisons and variable declarations.

### Files to change

- `tests/web/**/*.ts` (as identified during the initial lint pass)

### Acceptance criteria

- [ ] All test files in `tests/web/` pass `pnpm run lint` with 0 errors and 0 warnings.
- [ ] `cd tests/web && pnpm test` runs and all tests pass cleanly.

---

## Milestone 5 — Performance & Zero-Warnings Policy Verification

### Description

Verify that the frontend lint suite satisfies all non-functional requirements:
1. NFR-1 (Execution Speed): `pnpm run lint` + `vue-tsc --noEmit` runs in < 5 seconds locally.
2. NFR-2 (Determinism & Offline): Linting operates entirely offline without external network calls once `pnpm install` has run.
3. NFR-3 (Zero Warnings Policy): `--max-warnings 0` is strictly enforced so any newly introduced warning fails the command.
4. NFR-4 (Developer Feedback Quality): Diagnostic messages provide clear file path, line number, column number, and rule name.

### Files to change

- None (verification milestone).

### Acceptance criteria

- [ ] `cd web && pnpm run lint` completes in under 5 seconds.
- [ ] Synthetic warning triggers a non-zero exit code.
- [ ] Output diagnostics include file, line, column, and rule identifier.

---

## Resolved Questions

1. **Write Scope Expansion**: `lifecycle/config.yaml` has been updated to widen `frontend-developer`'s `allowed_write_paths` from `web/src` to `web` (the entire package root) as well as `tests/web`. The agent is authorized to edit `web/package.json`, `web/eslint.config.js`, and all configuration files within `web/`.
2. **Milestone Sequencing**: The plan will be executed sequentially by `frontend-developer`: Milestones 1 & 2 establish `web/eslint.config.js` and the `"lint"` script in `web/package.json`. Once those land, Milestones 3 & 4 remediate baseline violations across `web/src/` and `tests/web/`, followed by Milestone 5 verification.
3. **`tests/web/**` Ownership**: `frontend-developer` is authorized to remediate `tests/web/**` for frontend linting compliance, and `tests/web` has been added to `allowed_write_paths` for `frontend-developer` in `lifecycle/config.yaml`.
