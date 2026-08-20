---
title: RICE Score Not Saved — Displays N/A After Entry
type: defect
status: done
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

## Resolution (done)

Vue's `v-model` on `<input type="number">` coerces the bound ref to a **number** on edit, but
the editor seeded the refs with strings — so `parseField` called `.trim()` on a number and
threw, killing the whole save (defaults saved only because they were never edited). `parseField`
and `fieldError` now accept `string | number`; decimal effort (months) saves correctly. Verified
in `web/src/components/artifact/RiceEditor.vue` + the new `RiceEditor.spec.ts`.
