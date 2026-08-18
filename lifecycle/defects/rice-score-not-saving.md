---
title: RICE Score Not Saved — Displays N/A After Entry
type: defect
status: approved
lineage: rice-score-not-saving
created: "2026-08-19T09:28:11+10:00"
priority: normal
labels:
    - defect
    - frontend
    - ui
    - persistence
release: KC-Release5
rice_reach: 100
rice_impact: 0.25
rice_confidence: 25
---

# RICE Score Not Saved — Displays N/A After Entry

## Reproduction Steps

1. Open an artifact that supports RICE scoring.
2. Enter 0.1 for months
3. Save the artifact.
4. Observe the displayed RICE score.

## Expected Behaviour

After entering RICE values and saving, the calculated RICE score should persist and be displayed correctly (e.g. a numeric score derived from the entered values).

## Actual Behaviour

The RICE score continues to display "N/A" after saving. No error messages or console errors are visible. The entered values appear to not be persisted or not correctly calculated/rendered after the save operation.

Months should support decimal places
