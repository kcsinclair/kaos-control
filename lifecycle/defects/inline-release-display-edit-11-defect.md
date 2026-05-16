---
title: Test infrastructure for inline-release-display-edit not committed (vitest.config.ts and package.json scripts missing)
type: defect
status: in-development
lineage: inline-release-display-edit
parent: lifecycle/tests/inline-release-display-edit-10-test.md
labels:
    - defect
release: KC-Release2
assignees:
    - role: test-developer
      who: agent
---

## Reproduction Steps

1. Clone the repo and check out `kc-dev`.
2. Follow the documented run instructions from `lifecycle/tests/inline-release-display-edit-10-test.md`:
   ```sh
   cd web && pnpm install && pnpm test
   ```
3. Observe the error.

## Expected Behaviour

`pnpm test` runs vitest and executes all 17 component tests (TC1–TC13 in `ReleaseDropdown.spec.ts` and TC1–TC4 in `FrontmatterPanel.spec.ts`) without any manual setup.

## Actual Behaviour

```
ERR_PNPM_NO_SCRIPT  Missing script: test
Command "test" not found.
```

The `test` and `test:watch` scripts are absent from `web/package.json`.  
`web/vitest.config.ts` does not exist in the repository.  
`vitest`, `@vue/test-utils`, and `jsdom` are not listed as devDependencies in `web/package.json` and are therefore not installed after a fresh `pnpm install`.

The test artifact (`inline-release-display-edit-10-test.md`) documents these items as already in place ("New files (infrastructure)"), but none were actually committed.

## Logs / Output

```
$ cd web && pnpm install && pnpm test

ERR_PNPM_NO_SCRIPT  Missing script: test

Command "test" not found.
```

Running `ls web/vitest.config.ts` confirms the file does not exist:
```
zsh: no matches found: web/vitest.config.ts
```

Running `grep -c vitest web/package.json` returns `0`.

## Notes

- The 17 tests themselves pass once vitest is manually installed and a `vitest.config.ts` is created — the defect is solely in the missing committed infrastructure.
- Files that must be added / updated:
  - `web/vitest.config.ts` — jsdom environment, `@` alias pointing to `web/src`
  - `web/package.json` — add `vitest`, `@vue/test-utils`, `jsdom` to `devDependencies`; add `"test": "vitest run"` and `"test:watch": "vitest"` to `scripts`
