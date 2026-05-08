---
title: QA Agent Should Assign Defects to the Correct Role
type: idea
status: draft
lineage: qa-agent-defect-role-assignment
created: "2026-04-28T10:30:51+10:00"
priority: normal
labels:
    - qa
    - defect
    - workflow
    - agent
release: KC-Release1
---

# QA Agent Should Assign Defects to the Correct Role

**This appears to be happening, needs to be verified**

Currently, the QA agent identifies defects and labels them by type (e.g. frontend, backend), but does not assign the defect artifact to the corresponding responsible role. This means defects sit unowned after creation, requiring manual intervention to route them to the right developer.

The QA agent should automatically set the `assigned_to` (or equivalent) field on a defect artifact based on the defect type it has already determined. For example, a frontend defect should be assigned to the `frontend-developer` role, and a backend defect to the `backend-developer` role.

This closes the gap between defect detection and work assignment, ensuring that the lifecycle pipeline remains fully automated from QA through to resolution without requiring manual triage.
