---
title: "Devops List subcommand missing type flag"
type: defect
status: approved
lineage: kaos-control-devops-cli
parent: lifecycle/tests/kaos-control-devops-cli-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

## Reproduction Steps

1. Execute the integration test `TestDevops_List_FilterByType` or run the CLI command:
   ```bash
   kaos-control devops list --type=idea
   ```

## Expected Behaviour

The CLI subcommand `devops list` should support filtering the listed pipelines by their type using the `--type` flag. The command should run successfully and exit with code 0.

## Actual Behaviour

The CLI subcommand rejects the `--type` flag as undefined and exits with code 1.

## Logs / Output

```
stderr: flag provided but not defined: -type
Usage of devops list:
  -as string
    	assert identity as this email
  -json
    	emit JSON output
  -project string
    	project name (default: infer from cwd)
  -token string
    	bearer API token (overrides KAOS_CONTROL_TOKEN)
```
