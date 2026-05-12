---
title: Queue Work Button Not Visible on Approved Artifacts; Queue Limited to One Item
type: defect
status: in-development
lineage: queue-work-button-visibility-and-multi-item
created: "2026-05-12T19:01:06+10:00"
priority: normal
labels:
    - defect
    - frontend
    - queue
    - workflow
    - usability
---

# Queue Work Button Not Visible on Approved Artifacts; Queue Limited to One Item

## Reproduction Steps

1. Navigate to a defect artifact that is in `approved` status (e.g. `lifecycle/test-plans/end-to-end-smoke-tests-3-test.md`).
2. Observe the artifact detail view — note the Queue Work button is absent.
3. As a workaround, transition the artifact to `draft`, then back to `approved`.
4. Observe that the Queue Work button now appears.
5. Queue the artifact for work.
6. Attempt to queue a second artifact for work while the first is queued or in-progress.
7. Observe that the second item cannot be queued.

## Expected Behaviour

- The Queue Work button should be visible on any artifact that is in an `approved` status without requiring a state round-trip.
- The work queue should support multiple items queued simultaneously, processing them one at a time in order.

## Actual Behaviour

- The Queue Work button does not render when an artifact arrives at `approved` status directly (e.g. after an agent transition or page load); it only appears after manually toggling the artifact to `draft` and back to `approved`.
- Only one artifact can be queued at a time; attempting to queue additional items is blocked or silently ignored, preventing a multi-item backlog from being worked through sequentially.
