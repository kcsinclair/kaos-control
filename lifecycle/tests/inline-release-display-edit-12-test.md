---
title: "Test Infrastructure Fix: web/package.json test scripts for inline-release-display-edit"
type: test
status: draft
lineage: inline-release-display-edit
parent: lifecycle/defects/inline-release-display-edit-11-defect.md
---

## Overview

This artifact documents the fix for the missing vitest test infrastructure in
`web/package.json`, resolving defect `inline-release-display-edit-11-defect.md`.

The 17 component tests that existed in `web/src/components/artifact/__tests__/`
could not be executed because `web/package.json` lacked the `test` and
`test:watch` scripts.

## Fix Applied

Added to `web/package.json` `scripts`:

```json
"test": "vitest run",
"test:watch": "vitest"
```

`web/vitest.config.ts` and the `vitest`, `@vue/test-utils`, `jsdom`
devDependencies were already present; no other changes were required.

## How to Run

```sh
cd web && pnpm install && pnpm test
```

## Scenarios Covered

### ReleaseDropdown component — `web/src/components/artifact/__tests__/ReleaseDropdown.spec.ts`

| TC | Scenario |
|---|---|
| TC1 | Renders the current release name when the `release` prop is set |
| TC2 | Renders "None" when the `release` prop is `null` |
| TC3 | Opens dropdown on click and calls `listReleases` to fetch options |
| TC4 | Renders each release option with both a `.release-name` and `.release-status` span |
| TC5 | Shows the "None" option first in the dropdown (`[role="option"]` index 0) |
| TC6 | Clicking a release option calls `patchRelease` with the release name and emits `changed` |
| TC7 | Clicking the "None" option calls `patchRelease` with `null` and emits `changed` |
| TC8 | Optimistic update: trigger button text updates immediately before PATCH resolves |
| TC9 | Error rollback: on `patchRelease` rejection, value reverts and `error` is emitted |
| TC10 | Readonly mode: no `<button>` rendered; clicking does not open the dropdown |
| TC11 | Keyboard navigation: Enter opens; ArrowDown/ArrowUp move focus; Escape closes |
| TC12 | Outside click: clicking a DOM node outside the component closes the dropdown |
| TC13 | ARIA attributes: `aria-haspopup="listbox"`, `aria-expanded` toggles, `role="listbox"` set |

### FrontmatterPanel integration — `web/src/components/artifact/__tests__/FrontmatterPanel.spec.ts`

| TC | Scenario |
|---|---|
| TC1 | First three `<dt>` elements are "Status", "Priority", "Release" |
| TC2 | Release row always rendered; shows "None" when `artifact.frontmatter.release` is absent |
| TC3 | `ReleaseDropdown` stub present when `project` and `targetPath` props are provided |
| TC4 | When `ReleaseDropdown` emits `changed`, `FrontmatterPanel` re-emits `releaseChanged` |

## Test Results

All 17 tests pass (`2 test files, 17 tests`) in vitest 4.x with jsdom environment.
