---
title: DevOps Pipeline Run Does Nothing; Directory Not Auto-Created
type: defect
status: blocked
lineage: devops-pipeline-run-no-op
created: "2026-05-06T15:55:15+10:00"
priority: normal
labels:
    - defect
    - backend
    - frontend
assignees:
    - role: product-owner
      who: agent
---

# DevOps Pipeline Run Does Nothing; Directory Not Auto-Created

## Reproduction Steps

1. Ensure `~/.kaos-control/devops/kaos-control` does not exist (do not create it manually).
2. Open the kaos-control GUI and navigate to the DevOps Pipelines section.
3. Select a pipeline (e.g. `build`) and click Run.
4. Observe the UI — nothing appears to happen.
5. Click Cancel on the pipeline run.

## Expected Behaviour

- The `~/.kaos-control/devops/<project>` directory should be created automatically when a pipeline is first triggered (or on project initialisation), without requiring the user to create it manually.
- Clicking Run should start the pipeline and show visible progress or status feedback in the GUI.
- Clicking Cancel on an active run should cancel it gracefully.

## Actual Behaviour

- The `~/.kaos-control/devops/kaos-control` directory is never created automatically; the user had to create it manually.
- Clicking Run on a pipeline produces no visible effect — no run is started, no error is surfaced in the UI.
- Clicking Cancel returns the error: `no active run for pipeline: build`, confirming no run was ever initiated.

## Open Questions

1. **What is the specific frontend fix required?**
   This artifact is a defect description, not an implementation plan — there are no milestones, no list of files to change, and no description of what frontend behaviour is currently wrong beyond the symptom ("no visible effect"). The backend fix (`f6a1624 fix(devops-pipeline-run-no-op): correct devops log directory path`) has already been applied. After reviewing the existing frontend code (`web/src/components/devops/PipelineCard.vue`, `web/src/stores/devops.ts`, `web/src/api/devops.ts`), the error-handling and state-update paths appear correct for both success and failure cases. It is unclear whether any frontend change is still needed, and if so, exactly what it should be.

2. **Did the backend fix fully resolve the "no visible effect" symptom?**
   The defect labels both `backend` and `frontend`. If the root cause was solely the missing directory (now fixed server-side), the frontend may require no changes at all. Conversely, if there is a distinct frontend bug (e.g. the run API response is not reflected in the UI under certain conditions, or error toasts are not appearing), that specific scenario and the expected fix need to be spelled out before implementation can begin.

3. **Is there a separate frontend plan artifact that should be authored first?**
   The instruction to implement "milestone by milestone" implies a structured plan with milestones exists. None was found. Should a `frontend-plans/devops-pipeline-run-no-op-*-fe.md` plan artifact be written first (by the planning-analyst agent), with this defect re-queued once that plan is ready?
