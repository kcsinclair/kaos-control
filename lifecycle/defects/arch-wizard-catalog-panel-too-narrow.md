---
title: Architecture Wizard Catalog Panel Too Narrow — Table Overflows
type: defect
status: draft
lineage: arch-wizard-catalog-panel-too-narrow
created: "2026-08-18T10:56:09+10:00"
priority: normal
labels:
    - defect
    - architecture
    - catalog
    - frontend
    - ui
    - usability
    - wizard
---

# Architecture Wizard Catalog Panel Too Narrow — Table Overflows

## Reproduction Steps

1. Open the application and navigate to the Architecture Wizard.
2. Browse the architecture catalog (the panel displaying available architectures and tech stacks).
3. Observe the catalog table within its panel.

## Expected Behaviour

The catalog panel should be wide enough to display the table comfortably without overflow. The panel should be at least twice its current width, and all table columns and text should be fully visible and readable without wrapping or truncation.

## Actual Behaviour

The catalog panel is too narrow, causing the table to overflow the panel boundaries. All text in the table appears squashed and cramped, making it difficult to read catalog entries. The panel does not resize to accommodate the table content.
