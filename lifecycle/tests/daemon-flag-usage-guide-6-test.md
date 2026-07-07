---
title: "Test Suite: Default to Usage Guide; Require -d/--daemon to Start the Server"
type: test
status: approved
lineage: daemon-flag-usage-guide
parent: lifecycle/test-plans/daemon-flag-usage-guide-5-test.md
---

# Test Suite: Default to Usage Guide; Require -d/--daemon to Start the Server

Integration tests for the `daemon-flag-usage-guide` feature, covering the CLI
argument dispatch, output streams, exit codes, and TCP listener behaviour
described in the test plan (`daemon-flag-usage-guide-5-test`).

## Test File

### `tests/cli_daemon_flag_test.go`

Package `cli_test`, build tag `//go:build integration`. All tests invoke the
compiled binary (built once by `TestMain` in `cli_init_test.go`) as a
subprocess and assert on stdout, stderr, exit code, and TCP listener state.
Server-start tests write a minimal `config.yaml` to a `t.TempDir()` with a
free port and a throwaway `data_dir`/`projects_dir`, ensuring tests never
touch the developer's real `~/.kaos-control`.

Run with:

```sh
go test ./tests/ -tags integration -run 'TestNoArgs|TestDaemon|TestHelp|TestUsage|TestVersion|TestSubcommands|TestSPA|TestUnknown|TestDoc' -v
```

## Scenarios Covered

### Milestone 1 — No-argument invocation (F1 / Resolved Q3)

- `TestNoArgs_UsageToStderr_ExitTwo` — bare invocation writes the usage guide
  to **stderr**, leaves **stdout** empty, and exits **2**.
- `TestNoArgs_NoSideEffects` — bare invocation with an isolated `HOME` creates
  no files under that directory (no config write, no DB creation).
- `TestNoArgs_CompletesQuickly` — bare invocation exits in under 1 s, showing
  no config load or server start occurs (NF2).

### Milestone 2 — Daemon opt-in starts the server (F3 / Resolved Q1/Q2)

- `TestDaemon_DashD_StartsServer` — `-d -config <tmp>` starts the server;
  TCP connection to the configured address succeeds; SIGTERM shuts it down.
- `TestDaemon_DoubleDashDaemon_StartsServer` — `--daemon -config <tmp>` starts the server.
- `TestDaemon_ReverseOrder_StartsServer` — `-config <tmp> -d` (reverse order)
  starts the server, proving the daemon token is stripped before `flag.Parse`.
- `TestDaemon_Serve_StartsServer` — `serve -config <tmp>` starts the server
  (Resolved Q2: `serve` is a documented peer of `-d`).
- `TestDaemon_ConfigWithoutD_DoesNotStart` — `-config <tmp>` without `-d` does
  not start the server; exits **2** with usage on stderr; nothing listens on
  the configured address.
- `TestDaemon_NoDaemonFlagLeak` — `-d`, `--daemon`, and reverse-order
  invocations produce no "flag provided but not defined: -d" error on any
  stream (three sub-tests).

### Milestone 3 — Usage content and help/version flags (F2, F4)

- `TestHelp_StdoutExit0` — `--help`, `-h`, `-help` each write the usage guide
  to **stdout**, leave stderr empty, and exit **0** (three sub-tests).
- `TestUsageContent_AllRequiredElements` — the `--help` output contains:
  binary name, daemon flags (`-d`/`--daemon`), version flags (`--version`/`-V`),
  help flags (`--help`/`-h`), `-config`, all eight subcommands (`serve`,
  `init`, `auth`, `devops`, `hook-helper`, `backfill-created`, `backfill`,
  `releases`), the `serve`-equivalent note, and the per-command help pointer.
- `TestUsage_Identical` — the usage text written to **stderr** on a bare
  invocation is byte-identical to the **stdout** of `--help` (single source
  of truth).
- `TestVersion_StdoutExit0` — `--version`, `-V`, `-version` each write the
  version header to **stdout**, leave stderr empty, and exit **0** (three
  sub-tests).
- `TestVersion_NoServerStart` — `--version` exits in under 1 s; no server starts.

### Milestone 4 — Existing subcommands unchanged (F5) and unknown-input handling (F6)

- `TestSubcommands_DispatchToOwnHandler` — `init`, `auth`, `devops`,
  `hook-helper`, `backfill-created`, `backfill`, `releases` each invoked with
  `--help` exit without starting a server and produce their own output (not the
  top-level usage guide) (seven sub-tests).
- `TestUnknownSubcommand_StderrExit1` — unknown subcommand `bogus` prints an
  error plus usage to stderr, leaves stdout empty, and exits **1**.
- `TestUnknownFlag_StderrExit1` — unknown flag `--nope` prints an error plus
  usage to stderr, leaves stdout empty, and exits **1**.
- `TestSPA_ServedWhenDaemon` — when started via `-d`, `GET /` returns HTTP 200
  with `text/html` (frontend plan smoke check).

### Milestone 5 — Documentation guards (NF3)

- `TestDoc_MakefileRunIncludesDaemonFlag` — the Makefile `run:` target body
  contains `-d` or `serve`, ensuring `make run` still starts the server.
- `TestDoc_NoBareStartInstructions` — `README.md` and `docs/*.md` contain no
  lines where `./kaos-control` or `./dist/kaos-control` appears as the first
  command with nothing (no flag, no subcommand) following it.
