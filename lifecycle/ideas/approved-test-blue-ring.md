---
title: Blue Ring Indicator for Approved Tests in 2D and 3D Maps
type: idea
status: clarifying
lineage: approved-test-blue-ring
created: "2026-04-28T13:18:09+10:00"
priority: medium
labels:
    - frontend
    - feature
    - vue
---

# Blue Ring Indicator for Approved Tests in 2D and 3D Maps

Test artifacts with an `approved` status should be visually distinguished in both the 2D (Cytoscape.js) and 3D (3d-force-graph) map views by rendering a blue ring or outline around their node representation.

The blue ring should only be applied if it does not clash with other node colours or status indicators already in use — if a conflict is identified, an alternative shade or style should be selected to maintain visual clarity.

This enhancement improves at-a-glance comprehension of test coverage health, letting reviewers and the QA agent quickly identify which tests have passed the approval gate without needing to open each artifact.
