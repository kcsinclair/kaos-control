---
title: 'Tests: Remove Non-Functional Hyperlinks from Artifact Breadcrumb'
type: test
status: draft
lineage: artifact-breadcrumb-remove-broken-links
parent: lifecycle/test-plans/artifact-breadcrumb-remove-broken-links-5-test.md
created: "2026-05-06T00:00:00+10:00"
---

## Summary

Integration tests for `LineageBreadcrumb.vue` verifying that intermediate path segments render as non-interactive `<span>` elements while the root "artifacts" link and final filename segment retain their correct roles.

## Test file

`tests/web/LineageBreadcrumb.test.ts` — 59 tests across 5 describe blocks.

## Scenarios covered

### Milestone 1 — Intermediate segments are non-interactive

- `lifecycle` and `requirements` (and equivalent segments in deeper paths) render as `<span class="crumb-intermediate">`, never as `<button>`.
- No intermediate segment carries the `crumb-link` class.
- Clicking any intermediate segment does **not** invoke `router.push`.
- Exactly one `<button>` is present in the rendered output (the root link).

### Milestone 2 — Root link remains clickable

- The root "artifacts" element is a `<button class="crumb-link">`.
- Clicking it calls `router.push('/p/{project}/artifacts')` with the correct project slug from props.

### Milestone 3 — Final segment is current-page indicator

- The last path segment renders as `<span class="crumb-current">`.
- It is not a `<button>` and has no `crumb-link` class.
- Clicking it does not invoke `router.push`.
- Exactly one `crumb-current` span exists per render.

### Milestone 4 — All stage directories (parameterised)

Covers all eleven stage directories: `ideas`, `requirements`, `backend-plans`, `frontend-plans`, `dev-plans`, `test-plans`, `tests`, `prototypes`, `releases`, `sprints`, `defects`.

For each stage directory:

- Intermediate segments (`lifecycle`, stage dir) are `<span class="crumb-intermediate">`.
- Root is `<button class="crumb-link">` labelled "artifacts".
- Final segment has class `crumb-current`.
- No `console.warn` or `console.error` calls are emitted during render.
