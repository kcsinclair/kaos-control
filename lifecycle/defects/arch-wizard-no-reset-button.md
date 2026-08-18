---
title: 'Architecture Wizard: No ''Start Again'' Button to Reset Wizard'
type: defect
status: in-development
lineage: arch-wizard-no-reset-button
created: "2026-08-18T10:58:20+10:00"
priority: normal
labels:
    - defect
    - architecture
    - wizard
    - onboarding
    - frontend
    - ux
    - vue
assignees:
    - role: frontend-developer
      who: agent
---

# Architecture Wizard: No 'Start Again' Button to Reset Wizard

## Reproduction Steps

1. Open the Architecture Wizard from the onboarding flow.
2. Progress through one or more steps, making selections.
3. Decide you want to start the wizard from the beginning (e.g. to change an early choice).
4. Look for a 'Start Again' or 'Reset' button.

## Expected Behaviour

A clearly visible 'Start Again' (or equivalent reset) button is available within the wizard UI, allowing the user to discard all current selections and return to the first step of the wizard from any point in the flow.

## Actual Behaviour

No 'Start Again' or reset button exists in the wizard. Users have no way to restart the wizard from scratch without reloading the page or navigating away, making it difficult to recover from an unwanted selection made early in the flow.
