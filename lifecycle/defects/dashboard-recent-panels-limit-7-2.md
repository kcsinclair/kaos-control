---
title: RecentIdeasDefectsWidget test asserts stale limit=6 after limit was raised to 7
type: defect
status: approved
lineage: dashboard-recent-panels-limit-7
parent: lifecycle/defects/dashboard-recent-panels-limit-7.md
labels:
    - defect
release: KC-Release0
assignees:
    - role: test-developer
      who: agent
---

# RecentIdeasDefectsWidget test asserts stale limit=6 after limit was raised to 7

## Reproduction Steps

1. From the repository root, run the full Vitest suite:
   ```
   cd tests/web && npx vitest run --reporter=verbose
   ```
2. Observe that `RecentIdeasDefectsWidget.test.ts` fails on the assertion below.

## Expected Behaviour

The test `calls listArtifacts with type=idea,defect, sort=created:desc, limit=6` should pass — or, more correctly, it should be updated to assert `limit: 7` to match the implementation that was changed as part of the `dashboard-recent-panels-limit-7` fix (status: done).

## Actual Behaviour

The test fails because `RecentIdeasDefectsWidget.vue` now calls `listArtifacts` with `limit: 7`, but the test still uses `expect.objectContaining({ limit: 6 })`.

```
RecentIdeasDefectsWidget.test.ts > RecentIdeasDefectsWidget — general >
  calls listArtifacts with type=idea,defect, sort=created:desc, limit=6
    → expected "spy" to be called with arguments: [ 'myproject', ObjectContaining{ limit: 6, … } ]

  1st spy call:
    Array [
      "myproject",
  -   ObjectContaining { "limit": 6, "sort": "created:desc", "type": "idea,defect" },
  +   Object           { "limit": 7, "sort": "created:desc", "type": "idea,defect" },
    ]
```

## Logs / Output

```
❯ RecentIdeasDefectsWidget.test.ts  (20 tests | 1 failed) 99ms
  ❯ RecentIdeasDefectsWidget — general > calls listArtifacts with type=idea,defect,
    sort=created:desc, limit=6
    → expected "spy" to be called with arguments: [ 'myproject', ObjectContaining{…} ]

    Received:
      1st spy call:
      Array [
        "myproject",
    -   ObjectContaining { "limit": 6, "sort": "created:desc", "type": "idea,defect" },
    +   Object { "limit": 7, "sort": "created:desc", "type": "idea,defect" },
      ]

    Number of calls: 1
```

Test file: `tests/web/RecentIdeasDefectsWidget.test.ts`, line 511 and 526.

The fix is to update the test's `expect.objectContaining` assertion to use `limit: 7` and rename the test description to match.
