---
title: Remove Non-Functional Hyperlinks from Artifact Breadcrumb
type: requirement
status: planning
lineage: artifact-breadcrumb-remove-broken-links
created: "2026-05-06"
priority: medium
parent: lifecycle/ideas/artifact-breadcrumb-remove-broken-links.md
labels:
    - frontend
    - usability
    - vue
assignees:
    - role: product-owner
      who: agent
---

## Problem

The `LineageBreadcrumb` component (`web/src/components/artifact/LineageBreadcrumb.vue`) renders every path segment of an artifact's filepath as a clickable `<button>` that calls `router.push()` to an artifact viewer route. However, intermediate segments such as `lifecycle` and the stage directory (`requirements`, `ideas`, `backend-plans`, etc.) are not valid artifact paths — they are directories, not markdown files. Clicking them navigates to a non-existent or meaningless route, producing a broken user experience.

Only the first segment (`artifacts` — the list view) and the final segment (the artifact filename) are useful navigation targets. The intermediate directory segments should be rendered as plain, non-interactive text.

## Goals / Non-goals

### Goals

- Eliminate broken navigation caused by clicking intermediate breadcrumb segments.
- Render intermediate path segments (`lifecycle`, stage directory name) as plain non-clickable text.
- Keep the `artifacts` root link and the final filename segment in their current form (root as a clickable link to the artifact list; final segment as the current-page indicator).

### Non-goals

- Making intermediate breadcrumb segments navigate to a filtered artifact list (e.g., showing all requirements). This may be valuable but is out of scope.
- Changing the breadcrumb's visual layout, typography, or spacing beyond what is needed to distinguish clickable from non-clickable segments.
- Any backend changes — this is a frontend-only fix.

## Detailed Requirements

### Functional

1. **Intermediate segments must not be clickable.** All breadcrumb segments between the root `artifacts` link and the final filename segment must render as plain text (`<span>`) instead of `<button>` elements. They must not have click handlers, pointer cursors, or link styling.

2. **Root link unchanged.** The leading `artifacts` button must remain a clickable link that navigates to the project's artifact list (`/p/{project}/artifacts`).

3. **Final segment unchanged.** The last segment (the artifact filename) must continue to render as the current-page indicator with class `crumb-current`, matching the existing behaviour.

4. **Separator rendering unchanged.** The `/` separators between segments must continue to render identically.

### Non-functional

5. **Accessibility.** Intermediate text segments must not be focusable or announced as interactive by screen readers. Using `<span>` (not `<button>` with `disabled`) satisfies this.

6. **No regressions.** The breadcrumb must continue to display correctly for artifacts at all stage directories defined in `lifecycle/config.yaml` (`ideas`, `requirements`, `backend-plans`, `frontend-plans`, `dev-plans`, `test-plans`, `tests`, `prototypes`, `releases`, `sprints`, `defects`).

## Acceptance Criteria

- [ ] Intermediate breadcrumb segments (e.g., `lifecycle`, `requirements`) render as plain `<span>` text, not `<button>` elements.
- [ ] Intermediate segments have no `cursor: pointer`, no hover underline, and no click handler.
- [ ] The `artifacts` root link remains clickable and navigates to the artifact list view.
- [ ] The final filename segment renders as the current-page indicator (`crumb-current`).
- [ ] Breadcrumb renders correctly for artifacts in every stage directory listed in [[artifact-breadcrumb-remove-broken-links]] config.
- [ ] No console errors or routing warnings when viewing an artifact's breadcrumb.

## Questions

None — the idea is well-defined and the scope is narrow.
