---
title: Artifact Editor Details Panel Missing Inbound/Outbound Relationship Info
type: defect
status: done
lineage: artifact-editor-missing-inbound-outbound-relationships
created: "2026-04-29T15:35:47+10:00"
priority: normal
labels:
    - defect
    - frontend
    - artefacts
    - vue
assignees:
    - role: frontend-developer
      who: agent
release: May2026
---

# Artifact Editor Details Panel Missing Inbound/Outbound Relationship Info

## Reproduction Steps

1. Open the artifact editor at `/p/kaos-control/artifacts/`
2. Select any artifact to open it in the editor
3. Observe the details panel on the right-hand side
4. Compare with the artifact modal (e.g. opened from the graph view)

## Expected Behaviour

The artifact editor details panel should display inbound and outbound relationship information for the selected artifact, consistent with what is shown in the artifact modal.

## Actual Behaviour

The artifact editor details panel does not show inbound or outbound relationship information. This data is only visible in the artifact modal, leaving the editor details panel incomplete by comparison.
