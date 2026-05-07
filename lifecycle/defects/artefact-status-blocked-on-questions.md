---
title: Artefact Status Should Be 'blocked' (Not 'draft') When Questions Are Added
type: defect
status: done
lineage: artefact-status-blocked-on-questions
created: "2026-04-29T15:53:54+10:00"
priority: normal
labels:
    - defect
    - workflow
    - artefacts
release: May2026
---

# Artefact Status Should Be 'blocked' (Not 'draft') When Questions Are Added

## Reproduction Steps

1. Open any artefact (e.g. a requirement) that is in any workflow state.
2. Add one or more questions to the artefact.
3. Save / submit the changes.
4. Observe the resulting status and assignee of the artefact.

## Expected Behaviour

When questions are added to an artefact, the artefact's status should transition to `blocked` and it should be assigned to the `product-owner` role, indicating that a human decision is required before work can continue.

## Actual Behaviour

The artefact's status is set to (or remains as) `draft` and no assignment to the `product-owner` is made, leaving the blocked state invisible and unowned.
