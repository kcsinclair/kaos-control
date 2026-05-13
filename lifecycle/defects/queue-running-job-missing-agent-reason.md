---
title: 'Queue Page: Agent and Reason Fields Empty While Job is Running'
type: defect
status: approved
lineage: queue-running-job-missing-agent-reason
created: "2026-05-13T16:11:30+10:00"
priority: normal
labels:
    - defect
    - queue
    - frontend
    - usability
release: KC-Release1
assignees:
    - role: frontend-developer
      who: agent
---

# Queue Page: Agent and Reason Fields Empty While Job is Running

## Reproduction Steps

1. Navigate to the Queue page.
2. Trigger or wait for a job to start running.
3. Observe the queue table while the job status is active/running.

## Expected Behaviour

While a job is running, the Agent column should display the name of the agent executing the job, and the Reason column should display the relevant reason or description associated with the job.

## Actual Behaviour

While a job is running, the Agent column is empty (blank) and the Reason column always displays "-" instead of the actual reason value.
