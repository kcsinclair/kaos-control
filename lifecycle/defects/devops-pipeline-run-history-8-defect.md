---
title: E2E Test 'expanding a history row shows the inline log pane' fails to spawn server
type: defect
status: draft
lineage: devops-pipeline-run-history
parent: lifecycle/tests/devops-pipeline-run-history-6-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# E2E Test 'expanding a history row shows the inline log pane' fails to spawn server

## Reproduction Steps

1. Build the binary by running `make build`.
2. Go to the `tests/e2e` directory and install dependencies if necessary.
3. Run the Playwright E2E tests: `pnpm test flows/run-history.spec.ts`.
4. Observe that the test fails immediately during server startup.

## Expected Behaviour

The test harness (`tests/e2e/harness/kaos-control.ts`) should spawn the `kaos-control` server successfully. The server requires an explicit subcommand like `serve` or flag like `-d` to start.

## Actual Behaviour

The test harness spawns the binary with only `['-config', configPath]` as arguments. The binary rejects this and prints:
`error: no command given; use -d/--daemon or 'serve' to start the server`
This causes the test to fail with a timeout waiting for the health check.

## Logs / Output

```
  2) flows/run-history.spec.ts:80:3 › Run History smoke › expanding a history row shows the inline log pane 

    Error: Server startup failed:
    stdout: 
    stderr: error: no command given; use -d/--daemon or 'serve' to start the server

    kaos-control — lifecycle management for turning ideas into releases.

    Usage:
      kaos-control -d [-config <path>]      Start the HTTP server (daemon mode)
      kaos-control serve [-config <path>]   Start the HTTP server (equivalent to -d)
      kaos-control <command> [flags]
```
