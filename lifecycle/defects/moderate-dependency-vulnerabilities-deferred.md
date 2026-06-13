---
title: Dependency Vulnerabilities — residual dev-only advisories after Vite 6 / Vitest 4 upgrade
type: defect
status: planning
lineage: moderate-dependency-vulnerabilities-deferred
created: "2026-05-09T09:53:43+10:00"
priority: low
labels:
    - defect
    - security
    - testing
release: KC-Release4
parent: lifecycle/tests/release-artefacts-6-test.md
---

# Dependency Vulnerabilities — residual dev-only advisories after Vite 6 / Vitest 4 upgrade

**Dev/test-only exposure — none of these ship in the Go binary. Real-world risk ≈ zero.**

## Update — 2026-06-13: Vite and Vitest already upgraded

The vitest upgrade this defect was deferred behind is **done**. `web/` runs on
Vite 6 + Vitest 4, and `tests/web/` was brought in line on 2026-06-13 (commit
`fec64df9`: explicit `vite ^6.4.2`, `@vitejs/plugin-vue ^5.2.4`, `vitest ^4.1.6`).
See `plans/vitest-upgrade.md`.

The two CVEs that originally blocked this defect are now **resolved**:

| CVE | Package | Status |
|---|---|---|
| GHSA-4w7w-66w2-5vf9 | vite — `.map` path traversal | ✅ fixed (Vite 6) |
| GHSA-67mh-4wv8-2f99 | esbuild — dev server reachable by any website | ✅ fixed (esbuild bumped past 0.24.2) |

## Reproduction Steps

1. Run `pnpm audit` in `web/` and `tests/web/`.
2. Observe the remaining advisories below.

## Expected Behaviour

`pnpm audit` reports no moderate-or-higher advisories in either web dependency
tree, or each remaining advisory has an accepted, documented mitigation.

## Actual Behaviour

After the Vite/Vitest upgrade, two **new** transitive advisories remain (these
postdate the original 2026-05-09 report):

| Severity | CVE | Package | Path | Notes |
|---|---|---|---|---|
| High | GHSA-gv7w-rqvm-qjhr | esbuild | `web/` and `tests/web/` → `vite > esbuild` | "Missing binary integrity verification **in Deno**." Patched in esbuild ≥0.28.1; Vite 6.4.3 still bundles an older esbuild. **Not applicable** — we install via pnpm/Node, not Deno, and esbuild is build-time-only (never in the shipped Go binary). |
| Moderate | GHSA-58qx-3vcg-4xpx | ws | `tests/web/` → `happy-dom > ws` (<8.20.1) | Uninitialized memory disclosure. Test-time only; requires connecting to a malicious WebSocket server during a test run. |

## Mitigation / resolution path

Real exposure is negligible (both are dev/test-only and one is Deno-specific),
so this stays **low priority**. To get a clean `pnpm audit`, add pnpm
`overrides` in each affected `package.json` and reinstall:

```json
"pnpm": {
  "overrides": {
    "esbuild": ">=0.28.1",
    "ws": ">=8.20.1"
  }
}
```

Both are dev-only dependencies, so the override is low-risk; verify the frontend
suites still pass afterwards (`pnpm test` in `tests/web/`). Alternatively, wait
for a Vite 6.x patch that bundles esbuild ≥0.28.1 and a happy-dom release that
pulls ws ≥8.20.1, then drop the overrides.
