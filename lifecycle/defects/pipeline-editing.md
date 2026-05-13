---
title: Make Pipelines Editable
type: defect
status: blocked
lineage: pipeline-editing
created: "2026-05-13T17:01:56+10:00"
priority: normal
labels:
    - defect
    - feature
    - enhancement
    - frontend
release: KC-Release1
assignees:
    - role: product-owner
      who: agent
---

# Make Pipelines Editable

## Reproduction Steps

1. Open the application and navigate to the Pipelines section.
2. Add a new pipeline using the existing add-pipeline functionality.
3. Attempt to modify or edit the newly created pipeline's configuration.

## Expected Behaviour

An existing pipeline should be editable — the user should be able to click into a pipeline and modify its name, configuration, stages, or other properties, then save the changes.

## Actual Behaviour

No edit capability exists for pipelines. Once a pipeline has been created it can only be viewed; there is no UI affordance (e.g. edit button, inline editing) to update its configuration.

## Open Questions

1. **No backend update endpoint exists.** The backend only exposes `POST /p/{project}/devops/pipelines` (create, rejects with 409 if pipeline already exists), with no `PUT` or `PATCH` endpoint to update an existing pipeline's definition. The frontend agent is scoped to `web/src/**` only and cannot modify backend code. Before a frontend edit dialog can be built, the backend needs a `PUT /p/{project}/devops/pipelines/{slug}` (or equivalent) endpoint that accepts a new YAML definition and overwrites the existing file. **Who will implement the backend endpoint, and what should its request/response shape be?**

2. **No milestone breakdown.** This artifact is a defect report without an implementation plan or milestone structure. The frontend agent requires a milestone-by-milestone plan to proceed. Should a separate frontend plan artifact be created (e.g. `lifecycle/frontend-plans/pipeline-editing-N-fe.md`) that describes the UI/UX in detail and breaks the work into milestones?
