---
title: "Test Coverage: Frontend Component Tests for Inline Release Display and Editing"
type: test
status: in-qa
lineage: inline-release-display-edit
parent: lifecycle/test-plans/inline-release-display-edit-5-test.md
---

## Overview

This artifact documents the frontend component test coverage for the inline
release display and editing feature, implementing Milestones 2 and 3 from the
test plan. The backend integration tests (Milestone 1) were documented earlier
in `lifecycle/tests/inline-release-display-edit-6-test.md`.

## Test Infrastructure Added

**Framework:** vitest 2.x + @vue/test-utils 2.x + jsdom

**New files (infrastructure):**
- `web/vitest.config.ts` — vitest configuration (jsdom environment, `@` alias)
- `web/package.json` — added `vitest`, `@vue/test-utils`, `jsdom` to devDependencies
  and `test`/`test:watch` scripts

Run with:

```sh
cd web && pnpm install && pnpm test
```

## Milestone 2 — ReleaseDropdown Component Tests

**File:** `web/src/components/artifact/__tests__/ReleaseDropdown.spec.ts`

API modules are mocked via `vi.mock('@/api/artifacts')` and
`vi.mock('@/api/releases')` so no real HTTP calls occur.

| Test | Scenario |
|---|---|
| TC1 | Renders the current release name when the `release` prop is set |
| TC2 | Renders "None" when the `release` prop is `null` |
| TC3 | Opens dropdown on click and calls `listReleases` to fetch options |
| TC4 | Renders each release option with both a `.release-name` and `.release-status` span |
| TC5 | Shows the "None" option first in the dropdown (`[role="option"]` index 0) |
| TC6 | Clicking a release option calls `patchRelease` with the release name and emits `changed` |
| TC7 | Clicking the "None" option calls `patchRelease` with `null` and emits `changed` |
| TC8 | Optimistic update: trigger button text updates immediately, before PATCH resolves |
| TC9 | Error rollback: on `patchRelease` rejection, value reverts and `error` is emitted |
| TC10 | Readonly mode: no `<button>` is rendered; clicking does not open the dropdown |
| TC11 | Keyboard navigation: Enter opens; ArrowDown/ArrowUp move focus; Escape closes |
| TC12 | Outside click: clicking a DOM node outside the component closes the dropdown |
| TC13 | ARIA attributes: `aria-haspopup="listbox"` on trigger, `aria-expanded` toggles, `role="listbox"` and `aria-activedescendant` are set |

## Milestone 3 — FrontmatterPanel Integration Tests

**File:** `web/src/components/artifact/__tests__/FrontmatterPanel.spec.ts`

Uses `shallowMount` so child components (StatusDropdown, PriorityDropdown,
ReleaseDropdown, ArtifactRunHistory, RunDetailModal) are replaced by stubs.
Store and API dependencies of child modules are mocked via `vi.mock`.

| Test | Scenario |
|---|---|
| TC1 | First three `<dt>` elements in the rendered `<dl>` are "Status", "Priority", "Release" |
| TC2 | Release row is always rendered; displays "None" when `artifact.frontmatter.release` is absent |
| TC3 | When `project` and `targetPath` props are provided, a `ReleaseDropdown` stub is present |
| TC4 | When the `ReleaseDropdown` stub emits `changed`, FrontmatterPanel re-emits `releaseChanged` |

## Notes

- `tsconfig.app.json` intentionally excludes `src/**/__tests__/*`; the test
  files are compiled by vitest's own bundler (esbuild) and are not checked by
  `vue-tsc --noEmit`. The main app type-check is unaffected.
- `pnpm exec vue-tsc --noEmit` continues to pass with no changes to app source.
