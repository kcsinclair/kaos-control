---
title: Queue Page Shows 'Invalid Date' and 'NaNh' for All Entries
type: defect
status: approved
lineage: queue-invalid-date-nan-elapsed
created: "2026-05-13T14:12:56+10:00"
priority: normal
labels:
    - defect
    - frontend
    - queue
    - vue
---

# Queue Page Shows 'Invalid Date' and 'NaNh' for All Entries

## Reproduction Steps

1. Navigate to the Queue page in the application.
2. Observe the date columns and time elapsed fields for any queue entries.

## Expected Behaviour

All date fields display correctly formatted dates (e.g. '13 May 2026') and time elapsed fields display a human-readable duration (e.g. '2h', '45m').

## Actual Behaviour

All date fields render as 'Invalid Date' and all time elapsed fields render as 'NaNh', indicating that the date values received from the API are either null, undefined, or in an unexpected format that the frontend date parsing logic cannot handle.
