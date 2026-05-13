---
title: 'Test Plan: Inline Release Display and Editing'
type: plan-test
status: in-development
lineage: inline-release-display-edit
parent: lifecycle/requirements/inline-release-display-edit-2.md
assignees:
    - role: test-developer
      who: agent
---

## Overview

Integration and unit tests covering the PATCH release endpoint, the ReleaseDropdown component, and the FrontmatterPanel field reorder. Tests follow existing patterns in `tests/` (integration) and any co-located component tests.

## Milestone 1: Backend Integration Tests — PATCH Release Endpoint

**Description:** Test the `PATCH /api/p/:project/artifacts/*/release` endpoint for all success and error paths.

**Files to change:**
- `tests/` directory: New test file (e.g., `tests/patch_release_test.go` or add to existing artifact PATCH test file if one exists).

**Test cases:**
1. **Happy path — set release:** Create an artifact with no release, PATCH with a valid release name → 200, artifact on disk and in index has the release set.
2. **Happy path — change release:** Artifact has release A, PATCH to release B → 200, release updated.
3. **Happy path — clear release:** Artifact has a release, PATCH with `null` → 200, release field removed from frontmatter.
4. **Invalid release name → 422:** PATCH with a release name that doesn't exist in the project → 422 with `invalid_release` error code.
5. **Artifact not found → 404:** PATCH a non-existent artifact path → 404.
6. **Invalid JSON body → 400:** Send malformed JSON → 400.
7. **Lineage lock conflict → 423:** Lock the lineage as user A, PATCH as user B → 423 with lock info.
8. **Re-index verification:** After a successful PATCH, query the index and verify the release value matches.
9. **WebSocket event:** After a successful PATCH, verify that an `artifact.indexed` event with `action: updated` is broadcast (if the test harness supports WS assertions).

**Acceptance criteria:**
- [ ] All 9 test cases pass.
- [ ] Tests use the same test harness/setup pattern as existing integration tests in `tests/`.
- [ ] `make test-unit` passes (or `go test ./tests/... -short` if integration tests are skipped in short mode).

## Milestone 2: Frontend Component Tests — ReleaseDropdown

**Description:** Unit/component tests for `ReleaseDropdown.vue` covering rendering, interaction, and error handling.

**Files to change:**
- Co-located test file or `web/src/components/artifact/__tests__/ReleaseDropdown.spec.ts` (follow existing test file conventions).

**Test cases:**
1. **Renders current release name** when `release` prop is set.
2. **Renders "None"** when `release` prop is null.
3. **Opens dropdown on click** and shows release options fetched from API (mock `listReleases`).
4. **Shows release status** in `Name (status)` format for each option.
5. **"None" option** is present at the top of the dropdown list.
6. **Selecting a release** calls `patchRelease` and emits `changed`.
7. **Selecting "None"** calls `patchRelease` with `null` and emits `changed`.
8. **Optimistic update** — selected value shows immediately before PATCH resolves.
9. **Error rollback** — if PATCH fails, value reverts and `error` is emitted.
10. **Readonly mode** — clicking does not open the dropdown.
11. **Keyboard navigation** — arrow keys move focus, Enter selects, Escape closes.
12. **Outside click** closes the dropdown.
13. **ARIA attributes** — `role="listbox"`, `aria-expanded`, `aria-activedescendant` are present and correct.

**Acceptance criteria:**
- [ ] All 13 test cases pass.
- [ ] Tests follow the existing component test patterns in the project.
- [ ] `pnpm exec vue-tsc --noEmit` passes.

## Milestone 3: FrontmatterPanel Integration Tests

**Description:** Verify the field reorder and ReleaseDropdown integration within FrontmatterPanel.

**Files to change:**
- Existing FrontmatterPanel test file or a new co-located test.

**Test cases:**
1. **Field order** — the first three `<dt>` elements in the rendered `<dl>` are Status, Priority, Release (in that order).
2. **Release always visible** — when artifact has no release, the Release row still renders with "None" text.
3. **ReleaseDropdown rendered** — when `project` and `targetPath` props are provided, the Release row contains a `ReleaseDropdown` component (not static text).
4. **Event propagation** — selecting a release in the dropdown causes FrontmatterPanel to emit `releaseChanged`.

**Acceptance criteria:**
- [ ] All 4 test cases pass.
- [ ] No regressions in existing FrontmatterPanel tests.

## Milestone 4: Test Lifecycle Artifact

**Description:** Create a lifecycle test artifact describing what the test code covers, per project convention.

**Files to change:**
- `lifecycle/tests/` directory: New test artifact documenting the test coverage for inline release display and editing.

**Acceptance criteria:**
- [ ] Test artifact exists with correct frontmatter (type: test, lineage: inline-release-display-edit).
- [ ] Artifact body describes the scope of tests written in Milestones 1-3.

## Cross-links

- Depends on the [[inline-release-display-edit]] backend plan (PATCH endpoint) and frontend plan (ReleaseDropdown component) being implemented first.
- Backend integration tests (Milestone 1) can be written in parallel with the backend implementation.

## Resolved Questions

Milestones 2 and 3 (frontend component tests) cannot be implemented until the
following questions are resolved.

**Q1 — Which frontend test framework should be used?**
`web/package.json` has no test framework installed (no `vitest`, no
`@vue/test-utils`, no `jsdom`). The test plan says "follow existing test file
conventions" but no `*.spec.ts` files exist anywhere in `web/src/`. Before
tests for `ReleaseDropdown.vue` and `FrontmatterPanel.vue` can be written, the
product owner must decide which framework to adopt (vitest + @vue/test-utils is
the natural choice for a Vite/Vue 3 project) and add the required dependencies
to `web/package.json`.

> vitest + @vue/test-util

**Q2 — Where should component test files live?**
The plan mentions both "co-located" files and a
`web/src/components/artifact/__tests__/` directory. The project has no
established pattern. Is the preference co-located (`*.spec.ts` next to the
`.vue` file) or a dedicated `__tests__/` subdirectory?

> dedicated `__tests__/` subdirectory

**Q3 — How should `patchRelease` and `listReleases` API calls be mocked in component tests?**
The plan requires mocking `patchRelease` and `listReleases`. Should tests use
`vi.mock('@/api/artifacts')` / `vi.mock('@/api/releases')` (vitest module
mocks), or is there a different injection/stubbing approach preferred?

> `vi.mock('@/api/artifacts')` / `vi.mock('@/api/releases')`

**Q4 — Is `FrontmatterPanel.vue` ready for the integration tests described in Milestone 3?**
Milestone 3 test case 3 asserts that `ReleaseDropdown` is rendered when
`project` and `targetPath` props are provided. The existing
`FrontmatterPanel.vue` must expose these props and conditionally render
`ReleaseDropdown`. Confirm that the frontend implementation (Milestones 2 and 3
of the frontend plan) is complete before test implementation begins.
- Frontend tests (Milestones 2-3) require the components to exist.

> Yes
