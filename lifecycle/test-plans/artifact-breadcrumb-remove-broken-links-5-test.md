---
title: "Test Plan: Remove Non-Functional Hyperlinks from Artifact Breadcrumb"
type: plan-test
status: draft
lineage: artifact-breadcrumb-remove-broken-links
parent: lifecycle/requirements/artifact-breadcrumb-remove-broken-links-2.md
created: "2026-05-06"
assignees:
    - role: test-developer
      who: agent
---

## Summary

Write integration tests verifying that `LineageBreadcrumb.vue` renders intermediate path segments as non-interactive `<span>` elements while keeping the root link and final segment functional. Tests follow the existing pattern in `tests/web/` using the project's test framework.

## Milestone 1 — Test: Intermediate Segments Are Non-Interactive

### Description

Create a test file `tests/web/LineageBreadcrumb.test.ts` that mounts `LineageBreadcrumb` with a representative artifact path (e.g., `lifecycle/requirements/login-2.md`) and asserts that intermediate segments are plain text.

### Files to change

- `tests/web/LineageBreadcrumb.test.ts` — new file.

### Acceptance criteria

- [ ] Test mounts `LineageBreadcrumb` with `path="lifecycle/requirements/login-2.md"`, `project="test"`, `lineage="login"`.
- [ ] Asserts that the `lifecycle` segment renders as a `<span>`, not a `<button>`.
- [ ] Asserts that the `requirements` segment renders as a `<span>`, not a `<button>`.
- [ ] Asserts that neither intermediate segment has a click handler (clicking does not trigger `router.push`).
- [ ] Asserts that intermediate spans do not have `cursor: pointer` style (via class check — `.crumb-intermediate` present, `.crumb-link` absent).

## Milestone 2 — Test: Root Link Remains Clickable

### Description

Verify that the `artifacts` root button is still rendered as a `<button>` and navigates to the artifact list on click.

### Files to change

- `tests/web/LineageBreadcrumb.test.ts` — additional test case.

### Acceptance criteria

- [ ] The `artifacts` element is a `<button>` with class `crumb-link`.
- [ ] Clicking it calls `router.push` with the path `/p/test/artifacts`.

## Milestone 3 — Test: Final Segment Is Current-Page Indicator

### Description

Verify the last path segment renders with class `crumb-current` and is not clickable.

### Files to change

- `tests/web/LineageBreadcrumb.test.ts` — additional test case.

### Acceptance criteria

- [ ] The final segment (`login-2.md`) renders as a `<span>` with class `crumb-current`.
- [ ] It is not a `<button>` and has no click handler.

## Milestone 4 — Test: Breadcrumb Renders for All Stage Directories

### Description

Parameterised test that mounts `LineageBreadcrumb` for each stage directory and verifies consistent rendering.

### Files to change

- `tests/web/LineageBreadcrumb.test.ts` — parameterised test case.

### Acceptance criteria

- [ ] Test covers paths in all stage directories: `ideas`, `requirements`, `backend-plans`, `frontend-plans`, `dev-plans`, `test-plans`, `tests`, `prototypes`, `releases`, `sprints`, `defects`.
- [ ] For each path: intermediate segments are `<span>` elements; root is a `<button>`; final segment has class `crumb-current`.
- [ ] No errors or warnings emitted during render.

## Milestone 5 — Test Artifact

### Description

Create a corresponding test artifact in `lifecycle/tests/` documenting what the test code covers.

### Files to change

- `lifecycle/tests/artifact-breadcrumb-remove-broken-links-6-test.md` — new artifact (type: `test`, parent: this plan).

### Acceptance criteria

- [ ] Artifact frontmatter has correct `type: test`, `lineage: artifact-breadcrumb-remove-broken-links`, `parent` pointing to this plan.
- [ ] Body summarises the test coverage for this feature.

## Cross-references

- Frontend plan: [[artifact-breadcrumb-remove-broken-links]]-4-fe — the implementation being tested.
- Backend plan: [[artifact-breadcrumb-remove-broken-links]]-3-be — confirms no backend test surface.
