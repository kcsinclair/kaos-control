# Vitest upgrade plan — 1.6.1 → 3.x or 4.x

## Status

- Drafted: 2026-05-09
- Origin: residual Dependabot alerts after merging PRs #1 (vite 5→6) and #2 (happy-dom 14→20). The remaining medium-severity alerts are transitive through `vitest 1.6.1` in `tests/web/` and cannot be patched in place.

## Context

After this week's vite + happy-dom upgrades, two Dependabot alerts remain open:

| Severity | CVE | Package |
|---|---|---|
| Medium | GHSA-4w7w-66w2-5vf9 | vite — path traversal in `.map` handling |
| Medium | GHSA-67mh-4wv8-2f99 | esbuild — dev server can be hit by any website |

Both are transitive through vitest 1.6.1. The patches landed in **vite 6.x** and **esbuild 0.25.x**. Vitest 1.x peers with vite 5 only, and vite 5.4.21 is the last 5.4.x release — no patch is being backported.

Practical exposure today: ~zero (both CVEs require an attacker to reach a localhost-only dev server during `pnpm test`). The alerts are tagged "accepted risk" in the GitHub UI; this plan is the path to close them properly.

## Goals / Non-goals

### Goals

- Bring `tests/web/` to a vitest version whose vite range includes a patched vite.
- Keep the existing 851 frontend tests passing.
- Resolve the two transitive Dependabot alerts.

### Non-goals

- Adding new test infrastructure (browser mode, coverage providers, UI).
- Rewriting tests against a different runner (Jest, etc.).
- Changing `web/` (the SPA build) — that already runs vite 6.4.2.

## Compatibility matrix

| vitest | depends on vite | notes |
|---|---|---|
| **1.6.1** (current) | `^5.0.0` | Both transitive CVEs apply. |
| 2.x (last 2.1.9) | `^5.0.0` | Same vite range — does not resolve the alerts. |
| 3.x (last 3.2.4) | `^5.0.0 \|\| ^6.0.0 \|\| ^7.0.0-0` | Resolves alerts when paired with a `vite ^6` override. |
| 4.x (last 4.1.5) | `^6.0.0 \|\| ^7.0.0 \|\| ^8.0.0` | Resolves alerts unconditionally. Required Node ≥ 20. |

`@vue/test-utils@2.4.10` (current) is compatible with vitest 1, 2, 3, and 4 — its peers are Vue 3.x only, vitest is independent.

`happy-dom@20.x` (just upgraded) is compatible across the vitest range.

## Two viable targets

### Option A — vitest 3.x (lower risk)

- One major bump (1 → 3, skipping 2 because 2 doesn't fix the issue).
- Minimum change to silence the alerts.
- Stays on a recent-but-not-bleeding-edge runner.
- Vite 5/6/7 all valid; we'd override to vite ^6 to get the patches.

### Option B — vitest 4.x (recommended end state)

- Three major bumps (1 → 4).
- Latest stable; matches `web/`'s vite 6 directly.
- Better long-term posture — fewer follow-up upgrades in the next 12 months.
- Larger breakage surface; requires more verification work.

**Recommendation: do A first, B as a follow-up.** A gets the security alerts closed in a single small PR. B can be its own piece of work without time pressure.

## Known breaking changes by major

These are the documented changes most likely to bite this codebase. Verify each against the actual release notes when executing.

### vitest 1 → 2

- **Pool defaults changed** — `threads` is no longer the default; `forks` is now used in some scenarios. Our `vitest.config.ts` already pins `poolMatchGlobs` for `*.perf.test.ts` and `*.perf.spec.ts` — verify that still resolves.
- **Reporter API** — custom reporters changed shape. We don't use any, so likely no impact.
- **`vi.hoisted` semantics** — minor changes to hoisting order. We don't appear to use it.
- **`expect.poll` / `expect.assertions`** — unchanged but verify if used.
- **`vi.mock` behaviour around hoisting** — verify all 50 test files. Most use the standard `vi.mock('@/api/...')` pattern; that should be unaffected.

### vitest 2 → 3

- **Workspace config moved to `vitest.config.ts`** — we don't use workspaces.
- **Default pool changed to `forks`** — we pin per-file pool already; verify perf timings haven't shifted.
- **`vi.spyOn` return type changes** — verify any custom spy patterns.
- **Coverage v8 default thresholds** — we don't run coverage in the suite.
- **Snapshot serializer changes** — we don't use snapshot tests.

### vitest 3 → 4

- **Node ≥ 20 required** — verify the CI environment / dev machines. (`node --version` locally and in any CI to be added.)
- **`environmentOptions` reorganised** — our `happyDOM: { url: ... }` config may need to move under a different key. Read the migration guide.
- **Browser mode rewritten** — not in use.
- **Several deprecated APIs removed** — `vi.fn().mockImplementationOnce` chaining behaviour, `vi.useFakeTimers` modes, etc.

## Risks and how to mitigate

| Risk | Likelihood | Mitigation |
|---|---|---|
| Mock hoisting changes break `vi.mock` calls | medium | Run the full suite once after each major bump; failures will surface immediately |
| `vitest.config.ts` schema changes (`environmentOptions`, `poolMatchGlobs`) | medium | Read the migration guide for each major; apply minimal config changes |
| Performance tests' threshold timings shift due to pool changes | low–medium | Adjust thresholds in `*.perf.test.ts` if needed; pool is already pinned |
| `@vue/test-utils` ↔ vitest matrix breakage | low | Both libraries advertise stable interop across recent majors |
| TypeScript types break due to `@types/node` peer change | low | Pin `@types/node` to `^20` if needed |
| New peer dep warnings clutter install output | low | Acceptable noise; filter in plans |

## Proposed approach (Option A — vitest 3.x)

Single PR that covers:

1. Bump `vitest ^1.6.0` → `^3.2.4` in `tests/web/package.json`.
2. Add `pnpm.overrides` for `vite: ^6.0.0` and `esbuild: ^0.25.0` in `tests/web/package.json` so the transitive deps land on patched versions.
3. Run `pnpm install` in `tests/web/`. Resolve any peer-dep complaints by pinning `@types/node` if needed.
4. Read `vitest.config.ts` against the vitest 3 schema — adjust `environmentOptions.happyDOM.url` if the key path moved (likely unchanged in 3.x).
5. Run `pnpm test` and tally failures.
6. For each failure, classify: real test bug exposed by stricter behaviour, vs API change requiring test refactor. Fix or `it.skip()` with a comment.
7. Run `pnpm test` again until 851/851 green.

## Verification plan

Run in order; each step is a hard gate.

1. **`pnpm install` in `tests/web/`** completes without errors. Peer-dep warnings are OK.
2. **`pnpm list vite esbuild --depth Infinity`** shows vite ≥ 6.0.0 and esbuild ≥ 0.25.0 across all transitive paths.
3. **`pnpm test`** — 851/851 green, 0 unhandled rejections.
4. **`go build ./...`** — unchanged (the SPA build doesn't depend on `tests/web/`).
5. **GitHub Dependabot alerts** — both medium alerts auto-resolve within 24 hours of the PR landing on `main`.

## Rollback plan

Cleanest rollback is to revert the single PR. The change is contained to two files in `tests/web/`:

- `tests/web/package.json` — vitest version + pnpm.overrides block
- `tests/web/pnpm-lock.yaml` — regenerated lockfile

Reverting these on `main` puts the test suite back on vitest 1.6.1 immediately. The two alerts re-open, but the test suite is fully functional.

If only some tests break and the team wants to bank progress, push the working subset, `it.skip()` the rest with TODO comments, and file a follow-up issue per skipped test.

## Effort estimate

- **Best case**: 30 minutes — vitest 3 happens to be fully compatible, install + test + commit.
- **Likely case**: 2–4 hours — 5–15 tests need small adjustments (mock hoisting, config keys, occasional fixture timing).
- **Worst case**: 1 day — `vi.mock` semantic change requires updating dozens of test files; or a peer-dep collision requires pinning multiple `@types/*` packages.

## Follow-up: Option B (vitest 3 → 4)

After Option A is stable, take a separate PR to move 3 → 4. Two extra steps:

1. Verify Node 20+ is the floor for development and CI. Document it in the README.
2. Read the vitest 4 migration guide; adjust `environmentOptions` and any deprecated-API call sites.

Don't bundle A and B — they're two independent decisions with two independent risks.

## Open questions

- **Is there a CI environment yet?** Currently no GitHub Actions workflow runs on push. If one's added, vitest's `--reporter=github-actions` becomes available in 3.x and is worth turning on as part of this work.
- **Should we add coverage as part of the upgrade?** vitest 3 ships with v8 coverage built-in. Not required to silence the alerts but cheap to add if there's appetite.
- **Is `tests/web/` going to merge into `web/` eventually?** If so, the override should be applied there from the start. Track separately.
