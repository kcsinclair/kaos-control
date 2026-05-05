---
title: Remove Non-Functional Hyperlinks from Artifact Breadcrumb Path
type: idea
status: draft
lineage: artifact-breadcrumb-remove-broken-links
created: "2026-05-06T07:46:40+10:00"
priority: normal
labels:
    - frontend
    - artefacts
    - defect-fix
    - usability
    - vue
---

# Remove Non-Functional Hyperlinks from Artifact Breadcrumb Path

When viewing an artifact, the breadcrumb path displays segments such as `artifacts / lifecycle / requirements / agent-questions-trigger-blocked-status-2.md`. The intermediate path segments (`lifecycle`, `requirements`, `ideas`, `plans`, etc.) are rendered as hyperlinks but do not navigate anywhere useful, creating a confusing and broken user experience.

The fix should remove hyperlink behaviour from all breadcrumb segments except the final filename. Intermediate segments — specifically `lifecycle` and the stage directory (which may be `requirements`, `ideas`, `backend-plans`, `frontend-plans`, `test-plans`, `defects`, etc.) — should be rendered as plain, non-clickable text.

This applies wherever the artifact path breadcrumb is rendered in the UI. No backend changes are required; this is a purely frontend adjustment to the breadcrumb component.
