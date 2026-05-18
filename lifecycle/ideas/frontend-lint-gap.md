---
title: Frontend Lint Coverage Gap
type: idea
status: blocked
lineage: frontend-lint-gap
priority: medium
labels:
    - frontend
    - tooling
    - quality
assignees:
    - role: product-owner
      who: agent
---

# Frontend Lint Coverage Gap

## Problem

The Go side has comprehensive lint via `make lint`:
`go vet` + `staticcheck` + `govulncheck` + `gosec` + `gitleaks`. Every
commit gets these run by the user (and via the `lifecycle/devops/all-tests.yaml`
pipeline). Lint failures block merges to `main`.

The frontend has **no equivalent**. A grep across the repo for `.eslintrc*`,
`eslint.config*`, `.prettierrc*` returns zero hits. The only frontend
quality gate that exists is `web/package.json`'s `"type-check": "vue-tsc --noEmit"`
— which is:

1. Not wired into `make lint`.
2. Not run by any CI / pipeline step (`lifecycle/devops/all-tests.yaml`
   covers Go unit, Go integration, and Playwright e2e but not vue-tsc).
3. Only invoked if a developer remembers `pnpm run type-check`
   manually.

Net result: ~50 `.vue` files plus ~100+ `.ts` files have no rules
checking unused vars, no-floating-promises, consistent-imports,
no-unused-components, no-mutating-props, etc. Real bugs have shipped
because nothing catches them — including the stale `mockResolvedValueOnce`
queue-mismatch fixed in `5d28a4e2` (unit-test drift after silent component
churn), and the wrong-shape API parses in flows 04 and 10 (caught only
when the e2e suite was finally run).

## Goals / Non-goals

### Goals

- Type errors fail `make lint` alongside Go's, before agent runs commit
  unbuildable code.
- A small, principled ESLint rule set catches the bugs most likely to
  ship — unused variables, floating promises, mutating-props,
  unused-components.
- Pipeline integration so the workflow runs the new lint step
  automatically.

### Non-goals

- Comprehensive style enforcement (indentation, semicolons, etc.) —
  defer to a Prettier follow-up if wanted.
- Strict-mode TypeScript rules that would require touching every
  file's signatures — that's a separate "tighten types" piece of work.
- A full migration to Biome — option to consider but out of scope for
  the first cut.

## Proposal — two-stage approach

### Stage 1 — wire vue-tsc into `make lint` (cheap win, today)

```make
lint: lint-go lint-frontend

lint-frontend:
	cd web && pnpm install && pnpm run type-check
```

Maybe ~15 minutes. Catches type errors as part of the regular lint
loop. No new dependency, no new rules — just enforcement of what
`web/package.json` already declares.

### Stage 2 — add ESLint with a minimal rule set

1. Add to `web/package.json` (and `tests/web/package.json` if we want
   the same rules for tests):

   ```json
   "devDependencies": {
     "eslint": "^9.x",
     "@eslint/js": "^9.x",
     "typescript-eslint": "^8.x",
     "eslint-plugin-vue": "^9.x",
     "@vue/eslint-config-typescript": "^14.x"
   },
   "scripts": {
     "lint": "eslint . --max-warnings 0"
   }
   ```

2. Flat-config `eslint.config.js` with the minimal correctness rules:

   - `@typescript-eslint/no-unused-vars` (with `_` prefix exemption)
   - `@typescript-eslint/no-floating-promises`
   - `@typescript-eslint/no-misused-promises`
   - `vue/no-unused-components`
   - `vue/no-mutating-props`
   - `vue/no-v-html` (relax per-file where needed)
   - `eqeqeq`
   - `prefer-const`

3. Run, fix, commit. Expected violations on first run: a few dozen
   based on rough scan (most likely no-unused-vars and
   no-floating-promises).

4. Wire `pnpm run lint` into `make lint-frontend`.

5. Add an "ESLint" step to `lifecycle/devops/all-tests.yaml`.

## Why now

Pre-1.0 is the cheapest moment to add lint. The codebase is bounded:
~150 source files in `web/`, ~90 test files in `tests/web/`. Each
additional month makes the baseline cleanup more expensive.

The Go side proved the value: every gate caught real bugs the first
week it was wired in. Same calculation here.

## Caveats

- **ESLint major-version churn.** Flat config (9.x) is now stable and
  the right target — don't pull in legacy `.eslintrc` plugins.
- **Plugin ecosystem drift.** Some `eslint-plugin-vue` rules need
  parser config to understand `<script setup lang="ts">` correctly.
  Use `@vue/eslint-config-typescript` to get the parser+plugin combo
  right out of the box.
- **`tests/web/` rules.** Test files want different ergonomics — e.g.
  `any` is fine in mocks, unused-imports happen in skipped suites.
  Either a separate `eslint.config.js` for `tests/web/`, or a section
  override block in the root config.
- **Performance.** ESLint on a project this size runs in 2-3s.
  Pre-commit hook is feasible but optional.

## Effort estimate

| Piece | Effort |
|---|---|
| Stage 1: `make lint` wires `pnpm run type-check` | ~15 min |
| Stage 2 setup: package.json, `eslint.config.js`, first `pnpm install` | ~30 min |
| Fixing initial rule violations (rough estimate; depends on findings) | ~2 hours |
| Wire into `make lint` and `all-tests.yaml` pipeline | ~15 min |
| **Total** | **~3 hours** |

## Smallest viable proof-of-concept

Just Stage 1. One `Makefile` line; one re-run of `make lint`. If the
codebase passes vue-tsc cleanly, Stage 1 alone is a meaningful
improvement — and Stage 2 can be its own follow-up commit.

## Open Questions

- **Preset choice.** `eslint:recommended` + `@vue/eslint-config-typescript`
  recommended, or strict (`typescript-eslint/strict-type-checked`)?
  Strict catches more but requires more upfront cleanup; conservative
  catches the high-value classes only. Lean conservative for v1.
- **Run on `tests/web/` too?** Pros: same correctness floor. Cons: test
  files need looser rules. Probably yes with an override block.
- **Pre-commit hook?** Catches things earlier but slows the loop.
  Defer — `make lint` is already the contract.
- **Biome alternative?** Faster (Rust), single binary, fewer plugins.
  But the Vue support story is weaker than `eslint-plugin-vue`'s, and
  we have a lot of `.vue` files. Stick with ESLint for now; revisit
  in 12 months.
- **Prettier?** Out of scope for this idea. If we want format
  enforcement, file a separate `frontend-format-prettier` idea.
- **Stylelint for `.vue` `<style>` blocks?** Probably overkill. Skip.
