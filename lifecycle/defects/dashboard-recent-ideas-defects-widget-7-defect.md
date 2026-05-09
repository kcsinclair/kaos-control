---
title: DashboardGrid missing panel slot — activity-feed widget not rendered
type: defect
status: in-development
lineage: dashboard-recent-ideas-defects-widget
parent: lifecycle/tests/dashboard-recent-ideas-defects-widget-6-test.md
labels:
    - defect
release: KC-Release0
assignees:
    - role: frontend-developer
      who: agent
---

# DashboardGrid missing panel slot — activity-feed widget not rendered

`DashboardGrid.vue` does not implement a `panel` slot. The `panelWidgets`
computed property and the `section[aria-label="Panels"]` DOM section are both
absent from the component. Any widget registered with `{ slot: 'panel' }` (e.g.
`activity-feed`) is silently ignored and never rendered.

## Reproduction Steps

1. Register a widget with slot `panel`:
   ```ts
   registerWidget('activity-feed', FeedStub, { slot: 'panel', order: 0 })
   ```
2. Mount `DashboardGrid` with a valid `project` prop.
3. Assert `wrapper.find('section[aria-label="Panels"]').exists()`.

## Expected Behaviour

A `<section aria-label="Panels">` element is rendered containing all widgets
whose `slot` is `'panel'`, sorted by `order`. If no panel widgets are
registered the section is omitted (v-if).

## Actual Behaviour

`section[aria-label="Panels"]` is not present in the DOM regardless of how many
panel-slot widgets are registered. The component only computes `summaryWidgets`
and `chartWidgets`; there is no corresponding `panelWidgets` computed property
or panel rendering block in the template.

## Logs / Output

```
FAIL  DashboardView.test.ts > DashboardGrid — slot rendering > renders a widget registered to the panel slot
AssertionError: expected false to be true // Object.is equality
 ❯ DashboardView.test.ts:138:30
    136|
    137|     const section = wrapper.find('section[aria-label="Panels"]')
    138|     expect(section.exists()).toBe(true)
       |                              ^

FAIL  DashboardView.test.ts > DashboardGrid — slot rendering > renders widgets across all three slots simultaneously
AssertionError: expected false to be true // Object.is equality
 ❯ DashboardView.test.ts:222:50
    220|     expect(wrapper.find('.stub-summary').exists()).toBe(true)
    221|     expect(wrapper.find('.stub-chart').exists()).toBe(true)
    222|     expect(wrapper.find('.stub-panel').exists()).toBe(true)
       |                                                  ^

FAIL  DashboardView.test.ts > DashboardGrid — Milestone 5: layout with recent-ideas-defects widget > TC1: all 6 named widgets can be registered across the three slots
AssertionError: expected false to be true // Object.is equality
 ❯ DashboardView.test.ts:485:58
    483|     expect(wrapper.find('.stub-recent-ideas').exists()).toBe(true)
    484|     expect(wrapper.find('.stub-velocity').exists()).toBe(true)
    485|     expect(wrapper.find('.stub-activity-feed').exists()).toBe(true)
       |                                                          ^

FAIL  DashboardView.test.ts > DashboardGrid — Milestone 5: layout with recent-ideas-defects widget > TC2: summary-counts is in summary slot; activity-feed is in panel slot
Error: Cannot call find on an empty DOMWrapper.
 ❯ DashboardView.test.ts:505:25
    503|     const panelSection   = wrapper.find('section[aria-label="Panels"]')
    504|     expect(summarySection.find('.tc2-summary').exists()).toBe(true)
    505|     expect(panelSection.find('.tc2-feed').exists()).toBe(true)
```

4 tests failing. 0 panel-slot widgets rendered in any test scenario.
