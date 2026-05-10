---
title: RecentIdeasDefectsWidget fetches limit=7 instead of limit=6
type: defect
status: approved
lineage: dashboard-recent-ideas-defects-widget
parent: lifecycle/tests/dashboard-recent-ideas-defects-widget-6-test.md
labels:
    - defect
assignees:
    - role: frontend-developer
      who: agent
---

# RecentIdeasDefectsWidget fetches limit=7 instead of limit=6

`RecentIdeasDefectsWidget.vue` calls `listArtifacts` with `limit: 7`. The
feature spec and test suite both require `limit: 6` (show the 6 most recent
ideas and defects).

## Reproduction Steps

1. Open `web/src/components/dashboard/widgets/RecentIdeasDefectsWidget.vue`.
2. Locate the `listArtifacts` call inside `fetchItems()` (line 17–21).
3. Observe the `limit` value is `7`, not `6`.
4. Run `npx --prefix tests/web vitest run --root tests/web RecentIdeasDefectsWidget` and observe the failure.

## Expected Behaviour

`listArtifacts` is called with `{ type: 'idea,defect', sort: 'created:desc', limit: 6 }` — exactly 6 items are requested, matching the widget name ("Recent … ×6") and the test-plan spec.

## Actual Behaviour

`listArtifacts` is called with `limit: 7`, returning up to 7 items and causing the following test failure:

## Logs / Output

```
FAIL  RecentIdeasDefectsWidget.test.ts > RecentIdeasDefectsWidget — general > calls listArtifacts with type=idea,defect, sort=created:desc, limit=6
AssertionError: expected "spy" to be called with arguments: [ 'myproject', ObjectContaining{…} ]

Received:

  1st spy call:

  Array [
    "myproject",
-   ObjectContaining {
-     "limit": 6,
+   Object {
+     "limit": 7,
      "sort": "created:desc",
      "type": "idea,defect",
    },
  ]

Number of calls: 1

 ❯ RecentIdeasDefectsWidget.test.ts:521:38
    519|     await flushPromises()
    520|
    521|     expect(vi.mocked(listArtifacts)).toHaveBeenCalledWith(
       |                                      ^
    522|       'myproject',
    523|       expect.objectContaining({
```

Fix: change `limit: 7` to `limit: 6` at
`web/src/components/dashboard/widgets/RecentIdeasDefectsWidget.vue:20`.
