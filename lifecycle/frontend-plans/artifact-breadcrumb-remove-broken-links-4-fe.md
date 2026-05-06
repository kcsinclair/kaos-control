---
title: "Frontend Plan: Remove Non-Functional Hyperlinks from Artifact Breadcrumb"
type: plan-frontend
status: approved
lineage: artifact-breadcrumb-remove-broken-links
parent: lifecycle/requirements/artifact-breadcrumb-remove-broken-links-2.md
created: "2026-05-06T00:00:00+10:00"
assignees:
    - role: frontend-developer
      who: agent
---

## Summary

Modify `LineageBreadcrumb.vue` so that intermediate path segments (`lifecycle`, stage directory name) render as plain `<span>` text instead of clickable `<button>` elements, while preserving the root `artifacts` link and the final filename indicator.

Currently every segment between the root and the filename is a `<button class="crumb-link">` with a click handler that calls `router.push()`. These intermediate segments point to directory paths that are not valid artifact routes, producing broken navigation.

## Milestone 1 — Classify Segments by Role

### Description

Update the `segments` computed property in `LineageBreadcrumb.vue` to annotate each segment with its role: `root`, `intermediate`, or `current`. The root (`artifacts`) is handled separately in the template already, so focus on distinguishing `intermediate` segments from the `current` (final) segment.

### Files to change

- `web/src/components/artifact/LineageBreadcrumb.vue` — modify the `segments` computed or add a helper to identify the segment role.

### Acceptance criteria

- [ ] Each segment object exposes enough information to determine whether it should be interactive.
- [ ] The final segment is identifiable as the current-page indicator.
- [ ] All other segments (between root and final) are identifiable as intermediate.

## Milestone 2 — Render Intermediate Segments as Plain Text

### Description

Update the `<template>` block so that intermediate segments render as `<span class="crumb-intermediate">` instead of `<button class="crumb-link">`. Remove the `@click` handler and interactive styling from these elements.

### Files to change

- `web/src/components/artifact/LineageBreadcrumb.vue` — template section: change the `v-if="i < segments.length - 1"` branch to distinguish intermediate from root. Only root-level and final segments should be interactive.

### Acceptance criteria

- [ ] Intermediate segments render as `<span>` elements, not `<button>`.
- [ ] Intermediate segments have no `@click` handler, no `cursor: pointer`, and no hover underline.
- [ ] The `artifacts` root button remains a `<button class="crumb-link">` with its existing click handler.
- [ ] The final segment remains a `<span class="crumb-current">`.
- [ ] Separator `/` rendering is unchanged.

## Milestone 3 — Style the Intermediate Segments

### Description

Add a `.crumb-intermediate` CSS class for plain-text breadcrumb segments. These should use the muted text colour (matching `.sep` or similar) and must not have pointer, underline, or focus ring styles.

### Files to change

- `web/src/components/artifact/LineageBreadcrumb.vue` — `<style scoped>` section.

### Acceptance criteria

- [ ] `.crumb-intermediate` uses `color: var(--color-text-muted)` (or equivalent non-interactive colour).
- [ ] No `cursor: pointer` or `:hover` effects on `.crumb-intermediate`.
- [ ] The element is not focusable by keyboard (inherent with `<span>`; no `tabindex`).

## Milestone 4 — Accessibility and Regression Check

### Description

Verify that intermediate segments are not announced as interactive by screen readers, and that the breadcrumb displays correctly for artifacts in every stage directory.

### Files to change

- None (manual / automated testing only).

### Acceptance criteria

- [ ] Intermediate `<span>` elements have no `role="button"`, no `tabindex`, and are not focusable.
- [ ] Breadcrumb renders correctly for artifacts in all stage directories: `ideas`, `requirements`, `backend-plans`, `frontend-plans`, `dev-plans`, `test-plans`, `tests`, `prototypes`, `releases`, `sprints`, `defects`.
- [ ] No console errors or Vue Router warnings when viewing any artifact's breadcrumb.
- [ ] The `toArtifact()` function is removed or updated since it is no longer called for intermediate segments.

## Cross-references

- Backend plan: [[artifact-breadcrumb-remove-broken-links]]-3-be — no backend changes needed.
- Test plan: [[artifact-breadcrumb-remove-broken-links]]-5-test — integration tests for breadcrumb behaviour.
