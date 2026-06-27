---
title: 'devops CLI: add list and status subcommands'
type: defect
status: approved
lineage: devops-cli-list-and-status-commands
created: "2026-06-27T14:27:39+10:00"
priority: normal
labels:
    - defect
    - devops
    - feature
    - backend
    - go
release: KC-Release4
assignees:
    - role: backend-developer
      who: agent
---

# devops CLI: add list and status subcommands

## Reproduction Steps

1. Run `kaos-control devops list`
2. Run `kaos-control devops status build`

## Expected Behaviour

`kaos-control devops list` should enumerate all available devops jobs defined under `lifecycle/devops/`, displaying each job's short name and description.

`kaos-control devops status <job-name>` (e.g. `kaos-control devops status build`) should display the status of the most recent run for the named job, including relevant metadata such as start time, finish time, exit status, and any output summary.

## Actual Behaviour

Neither subcommand exists. Running `kaos-control devops list` or `kaos-control devops status build` produces an error or unknown-command output. There is currently no CLI interface for browsing available devops jobs or inspecting run history.
