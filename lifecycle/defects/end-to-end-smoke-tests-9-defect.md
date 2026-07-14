---
title: E2E test harness does not pass daemon flag or serve command to start server
type: defect
status: done
lineage: end-to-end-smoke-tests
parent: lifecycle/tests/end-to-end-smoke-tests-8-test.md
labels: [defect]
assignees:
  - role: test-developer
    who: agent
---

# E2E test harness does not pass daemon flag or serve command to start server

All 26 Playwright E2E tests fail because the server startup fails during harness setup.

## Reproduction Steps

1. Build the server binary: `make build`.
2. Run the E2E test suite: `make test-e2e` (or `cd tests/e2e && pnpm test`).
3. Observe all tests fail on the harness setup stage.

## Expected Behaviour

The test harness successfully spawns the server binary, which starts up as a daemon and listens on the configured port, allowing tests to run and pass.

## Actual Behaviour

The server fails to start because it is invoked without a command or the daemon flag, which is now required as of the recent changes to gate the server behind `-d`/`--daemon` or `serve`.

Specifically, `tests/e2e/harness/kaos-control.ts` spawns the binary with only `['-config', configPath]` as arguments:

```typescript
const proc = spawn(binaryPath, ['-config', configPath], {
  env: { ...process.env, LOG_LEVEL: 'warn' },
})
```

This causes the server to print `error: no command given; use -d/--daemon or 'serve' to start the server` to stderr and exit with code 2, failing the E2E harness health check.

## Logs / Output

```
Error: Server startup failed:
stdout: 
stderr: error: no command given; use -d/--daemon or 'serve' to start the server

kaos-control — lifecycle management for turning ideas into releases.

Usage:
  kaos-control -d [-config <path>]      Start the HTTP server (daemon mode)
  kaos-control serve [-config <path>]   Start the HTTP server (equivalent to -d)
  kaos-control <command> [flags]

Commands:
  serve              Start the HTTP server (same as -d/--daemon)
  init               Initialise a new project directory
  auth               Manage users, passwords, and API tokens
  devops             DevOps operations against a registered project
  hook-helper        PreToolUse hook helper (called by Claude Code)
  backfill-created   Add created: frontmatter to legacy artefacts using
                     filesystem birth time
  backfill           One-off data backfill utilities
                       backfill agent-run-metrics --project <id>
  releases           Release management operations

Flags:
  -d, --daemon       Start the foreground HTTP server (required to run the server)
  -config <path>     Path to app config.yaml (daemon mode only)
  --version, -V      Print version, copyright, and licence
  --help, -h         Print this usage guide

Run 'kaos-control <command> --help' for command-specific usage.

Error: kaos-control did not become healthy within 10000ms at http://127.0.0.1:62886

   at ../harness/kaos-control.ts:138

  136 |       const stdout = Buffer.concat(stdoutChunks).toString()
  137 |       const stderr = Buffer.concat(stderrChunks).toString()
> 138 |       throw new Error(`Server startup failed:\nstdout: ${stdout}\nstderr: ${stderr}\n${err}`)
      |             ^
  139 |     }
```
