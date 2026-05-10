---
title: "RecentIdeasDefectsWidget: limit=7 regression tests"
type: test
status: in-qa
lineage: dashboard-recent-panels-limit-7
parent: lifecycle/defects/dashboard-recent-panels-limit-7-2.md
---

# RecentIdeasDefectsWidget: limit=7 regression tests

Verifies that `RecentIdeasDefectsWidget.vue` calls `listArtifacts` with `limit: 7`
after the limit was raised from 6 to 7 as part of the `dashboard-recent-panels-limit-7`
fix.

## Test file

`tests/web/RecentIdeasDefectsWidget.test.ts`

## Scenarios covered

### TC — General: API call parameters

- **`calls listArtifacts with type=idea,defect, sort=created:desc, limit=7`**  
  Mounts the widget with `project="myproject"`, waits for promises to flush, then
  asserts that `listArtifacts` was called with `{ type: 'idea,defect', sort: 'created:desc', limit: 7 }`.
  This is the direct regression guard for the defect: the previous assertion used
  `limit: 6` and failed after the implementation was updated.

### TC1 — Renders items

- Correct number of `[role="listitem"]` elements is rendered for a 4-item API response.
- Each item title appears in the DOM.
- Item type badges display correct text (`idea` / `defect`) and CSS classes.
- Timestamps are rendered.

### TC2 — Empty state

- When the API returns zero items, the "No recent ideas or defects" message is shown.

### TC3 — Navigation

- Each item is wrapped in a `<router-link>` / `<a>` pointing to
  `/p/{project}/artifacts/{path}`.

### TC4 — Type badges

- `idea` items receive the correct badge class.
- `defect` items receive the correct badge class.
- Badges carry `aria-label` attributes for accessibility.

### TC5 — Live update via WebSocket

- When `useWebSocket` fires an `artifact.indexed` event, the component triggers a
  second `listArtifacts` call (re-fetch).

### TC6 — Accessibility

- Items are rendered as native `<a>` elements (natively focusable; no extra
  `tabindex` needed).
- Type badges include `aria-label`.
