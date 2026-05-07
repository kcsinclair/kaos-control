---
title: 'Release Creation: Spread Syntax Error Causes Partial Save and 409 Conflict on Retry'
type: defect
status: approved
lineage: release-create-spread-syntax-409-conflict
created: "2026-05-07T08:45:11+10:00"
priority: normal
labels:
    - defect
    - releases
    - frontend
    - backend
---

# Release Creation: Spread Syntax Error Causes Partial Save and 409 Conflict on Retry

## Reproduction Steps

1. Navigate to the releases section of the UI.
2. Initiate creation of a new release (e.g. named "May2026").
3. Fill in the required fields and save.
4. Observe the error: `Spread syntax requires ...iterable[Symbol.iterator] to be a function`.
5. Attempt to save again without navigating away.
6. Observe the second error: `release "May2026" already exists in this project`.

Also observed in the browser console:
`[Error] Failed to load resource: the server responded with a status of 409 (Conflict) (releases, line 0)`

## Expected Behaviour

The release should be created successfully on the first save attempt with no JavaScript errors. If a conflict genuinely exists, a clear, user-friendly message should be shown before any write is attempted. A failed save should not persist partial state server-side.

## Actual Behaviour

The first save triggers a JavaScript spread-syntax runtime error in the frontend, indicating a non-iterable value is being spread (likely a field expected to be an array or iterable is null/undefined). Despite this error, the request appears to reach the server and the release record is created. On the second save attempt the server correctly returns a 409 Conflict, but the user is left in a broken state: the UI shows a JS error on the first attempt and a conflict error on every subsequent attempt, with no way to recover without a page reload or manual intervention.
