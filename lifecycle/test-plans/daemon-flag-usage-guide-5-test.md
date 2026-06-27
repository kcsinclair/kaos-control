---
title: "Test Plan: Default to Usage Guide; Require -d/--daemon to Start the Server"
type: plan-test
status: draft
lineage: daemon-flag-usage-guide
parent: lifecycle/requirements/daemon-flag-usage-guide-2.md
---

# Test Plan: Default to Usage Guide; Require -d/--daemon to Start the Server

Verifies [[daemon-flag-usage-guide]] (requirement `daemon-flag-usage-guide-2`),
as implemented by the backend plan ([[daemon-flag-usage-guide]] `-3-be`). The
behaviour under test is **CLI/process-level**: argument dispatch, output
streams, exit codes, and whether a TCP listener is bound. The frontend plan
(`-4-fe`) contributes only the "SPA still served" smoke check (Milestone 4 here).

## Testing approach

`main()` calls `os.Exit`, so the behaviour cannot be asserted by calling it
in-process. Use a **subprocess strategy**: build the binary once per package
test run (`go build -o <tmp>/kaos-control ./cmd/kaos-control`, or reuse
`dist/kaos-control` if present), then run it with `os/exec.Command` and assert on
captured **stdout**, **stderr**, and **exit code**. This matches the existing
integration-test style under `tests/` (artifacts in `lifecycle/tests/`).

For each daemon-start case, point the binary at a **throwaway config**
(`-config <tmpdir>/config.yaml` with a free port / `:0` or an unused high port)
and a temporary `HOME`/`XDG_CONFIG_HOME` so tests never touch the developer's
real `~/.kaos-control`. Detect "server started" by polling the listen address
for a successful TCP connect (or `GET /` returning HTTP), then terminate the
child (SIGTERM) and assert it shuts down cleanly. Detect "server did NOT start"
by asserting the process exits promptly **and** that nothing is listening on the
configured address.

Critically distinguish the three help/error conventions per the Resolved-Q3
decision recorded in the backend plan:

| Case | stdout | stderr | exit |
|---|---|---|---|
| no args | empty | usage guide | 2 |
| `--help` / `-h` / `-help` | usage guide | empty | 0 |
| `--version` / `-V` | version header | empty | 0 |
| unknown flag / subcommand | empty | error + usage | 1 |

## Milestone 1 — No-argument invocation: usage to stderr, exit 2, no side effects

### Description

Assert F1 as amended by Resolved Q3: a bare invocation is a usage error.

### Files to change

- `tests/cli_daemon_flag_test.go` (new) — subprocess harness + `TestNoArgs*`.
- `lifecycle/tests/daemon-flag-usage-guide-cli.md` (new) — artifact describing this CLI test coverage.

### Acceptance criteria

- [ ] Running the binary with **no arguments** prints the usage guide to **stderr** (stdout is empty) and exits with code **2**.
- [ ] No TCP listener is bound during the no-argument run (a connect attempt to the config's listen address fails / refused).
- [ ] The no-argument run completes quickly (well under 100 ms wall time, allowing for process spawn), evidencing no config load / project open / goroutine startup (NF2).
- [ ] The no-argument run creates no files under the temporary `HOME` (no config written, no data/auth/queue DBs).

## Milestone 2 — Daemon opt-in starts the server; ordering and `serve` parity

### Description

Assert F3 and Resolved Q1/Q2: `-d`, `--daemon`, `serve`, and `-config`
combinations start the server; the daemon token never reaches `flag.Parse()`;
and a bare `-config` without opt-in does **not** start the server.

### Files to change

- `tests/cli_daemon_flag_test.go` (extend).

### Acceptance criteria

- [ ] `kaos-control -d -config <tmp>` starts the server and serves on the configured address; SIGTERM shuts it down cleanly (exit 0 / no error).
- [ ] `kaos-control --daemon -config <tmp>` starts the server.
- [ ] `kaos-control -config <tmp> -d` (reverse order) starts the server using the given config — proving the daemon token was stripped before `flag.Parse()` and `-config` was honoured.
- [ ] `kaos-control serve -config <tmp>` starts the server (parity with `-d`).
- [ ] `kaos-control -config <tmp>` **without** `-d`/`serve` does **not** start the server: nothing listens on the configured address, and the process exits non-zero with usage on stderr (consequence of F3).
- [ ] No run produces a `flag provided but not defined: -d` (or `-daemon`/`--daemon`) error on any stream.
- [ ] In every "server started" case, the configured listen address accepts a TCP connection (or `GET /` returns an HTTP status) before termination.

## Milestone 3 — Usage guide content (F2) and help/version flags (F4)

### Description

Assert the usage guide is complete (F2) and that the help/version flags keep
stdout+0 semantics (F4), using the same usage string across paths.

### Files to change

- `tests/cli_daemon_flag_test.go` (extend) — `TestUsageContent`, `TestHelpFlags`, `TestVersionFlags`.

### Acceptance criteria

- [ ] `kaos-control --help`, `-h`, and `-help` each print the usage guide to **stdout** and exit **0**.
- [ ] The help output contains: the binary name `kaos-control`; a non-empty product description line; the `-d`/`--daemon` flag; `--version`/`-V`, `--help`/`-h`, and `-config`; and all eight subcommands — `serve`, `init`, `auth`, `devops`, `hook-helper`, `backfill-created`, `backfill`, `releases`.
- [ ] The help output indicates `serve` is equivalent to `-d`/`--daemon` (Resolved Q2).
- [ ] The help output includes the per-command help pointer (`kaos-control <command> --help`).
- [ ] The usage text produced on the no-arg path (Milestone 1, stderr) is byte-identical to the `--help` path (stdout) — single source of truth.
- [ ] `kaos-control --version`, `-V`, and `-version` each print the version/copyright/licence header to **stdout** and exit **0**, and do **not** start the server.

## Milestone 4 — Existing subcommands unchanged (F5) and unknown-input handling (F6)

### Description

Regression-guard the compatibility surface (F5) and the mistake path (F6).

### Files to change

- `tests/cli_daemon_flag_test.go` (extend) — `TestSubcommandsUnchanged`, `TestUnknownInput`.

### Acceptance criteria

- [ ] Each of `init`, `auth`, `devops`, `hook-helper`, `backfill-created`, `backfill`, `releases` invoked with `--help` (or a benign no-op form) dispatches to its own handler — i.e. produces that subcommand's own output, not the top-level usage guide — and none of them start the HTTP server.
- [ ] An **unknown subcommand** (e.g. `kaos-control bogus`) prints an error **plus** the usage guide to **stderr** and exits **non-zero (1)**; stdout is empty; no server starts.
- [ ] An **unknown top-level flag** (e.g. `kaos-control --nope`) prints an error plus usage to **stderr** and exits **non-zero (1)**; no server starts.
- [ ] A smoke check confirms that when started via `-d`, the embedded SPA is served (e.g. `GET /` returns HTML / 200), satisfying the frontend plan's ([[daemon-flag-usage-guide]] `-4-fe`) "SPA served unchanged" criterion.

## Milestone 5 — Documentation and dependency guards (NF1, NF3)

### Description

Lightweight guards that the no-new-dependency and documentation constraints hold.

### Files to change

- `tests/cli_daemon_flag_test.go` (extend) or a small `tests/docs_daemon_flag_test.go`.

### Acceptance criteria

- [ ] No new module appears in `go.mod`/`go.sum` attributable to this change (NF1) — verified by `go build ./...` succeeding with the existing module set and by review of the diff.
- [ ] A documentation guard asserts there is no remaining bare-start instruction (`./dist/kaos-control` / `kaos-control` with no command) in `README.md`, `docs/*.md`, and the `Makefile` `run:` target (NF3); the `run:` target includes `-d`.
- [ ] `go vet ./...` and `make test-unit` pass with the new tests included.

## Cross-links

- [[daemon-flag-usage-guide]] — originating idea and requirement (`-2`).
- Backend plan `daemon-flag-usage-guide-3-be` — the implementation these tests verify; Milestone numbering here maps onto its F1–F6 / NF1–NF4 and Resolved Questions.
- Frontend plan `daemon-flag-usage-guide-4-fe` — its "SPA served unchanged" checkpoint is covered by Milestone 4's smoke check.
