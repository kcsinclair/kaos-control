---
title: DevOps Pipeline Run Does Nothing; Directory Not Auto-Created
type: defect
status: done
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
release: KC-Feature-Sprint
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
