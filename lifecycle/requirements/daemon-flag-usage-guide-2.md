---
title: Default to Usage Guide; Require -d/--daemon to Start the Server
type: requirement
status: approved
lineage: daemon-flag-usage-guide
created: "2026-06-27T14:30:00+10:00"
priority: normal
parent: lifecycle/ideas/daemon-flag-usage-guide.md
labels:
    - backend
    - go
    - usability
    - operability
    - onboarding
assignees:
    - role: product-owner
      who: agent
---

# Default to Usage Guide; Require -d/--daemon to Start the Server

## Problem

Today the `kaos-control` binary treats server startup as the default action: when invoked with no recognised subcommand (and no `serve` argument), `main()` falls through to `run()` and starts the HTTP server. There is no way to "just look at the tool" — a new user who runs `kaos-control` with no arguments unexpectedly launches a long-running daemon that binds a TCP port, opens project indexes, and starts background goroutines (watchers, reapers, scheduler, queue dispatcher).

This is poor onboarding and poor operability:

- New users get no orientation; they must read source or docs to discover the available commands and flags.
- A bare invocation has a side effect (starting a server) that is easy to trigger accidentally — e.g. in a shell, a script, or a process manager that was expected to print help.
- It diverges from conventional CLI design, where long-running or "destructive" modes opt in explicitly via a flag, and a no-argument invocation is safe and informative.

The existing `serve` subcommand and the implicit "fall through to serve" path both start the server, but neither is gated behind an explicit daemon opt-in.

## Goals / Non-goals

### Goals

1. **Safe, informative default** — Invoking `kaos-control` with no arguments prints a usage guide to stdout and exits `0`, with no server start and no side effects.
2. **Explicit daemon opt-in** — Starting the HTTP server (daemon mode) requires an explicit `-d` / `--daemon` flag (or the existing `serve` subcommand).
3. **Discoverable** — The usage guide lists the binary name, a one-line product description, the daemon flag, and the other top-level flags and subcommands, so a new user can orient themselves from a single bare invocation.
4. **Backwards-compatible subcommands** — Existing subcommands (`init`, `auth`, `devops`, `hook-helper`, `backfill-created`, `backfill`, `releases`) and the existing `--version`/`--help` flags continue to behave exactly as they do today.

### Non-goals

- Introducing a third-party CLI framework (e.g. cobra/urfave-cli). The standard library `flag` package and the existing hand-rolled dispatch are sufficient.
- Adding new server configuration flags beyond the daemon opt-in (config path, port, log level handling are unchanged).
- Changing the server's runtime behaviour once started.
- Daemonising in the Unix sense (forking, detaching, writing a PID file). "`-d`/`--daemon`" here means "run the foreground server process"; backgrounding remains the caller's responsibility (shell `&`, systemd, etc.).

## Detailed Requirements

### Functional

**F1 — No-argument invocation prints usage and exits 0**
- `kaos-control` with zero arguments writes the usage guide to **stdout** and exits with code **0**.
- It must **not** start the server, bind a listener, open projects, or start any background goroutine.

**F2 — Usage guide content**
The usage guide must include, at minimum:
- The binary name (`kaos-control`).
- A one-line description of the product (what the tool does).
- The daemon flag (`-d`, `--daemon`) and a short description.
- The other top-level flags: `--version`/`-V`, `--help`/`-h`, and `-config <path>`.
- The list of subcommands (`serve`, `init`, `auth`, `devops`, `hook-helper`, `backfill-created`, `backfill`, `releases`) with one-line descriptions.
- A pointer to per-command help (`kaos-control <command> --help`).

**F3 — Daemon mode requires explicit opt-in**
- Passing `-d` or `--daemon` starts the HTTP server (the current `run()` behaviour).
- The existing `serve` subcommand continues to start the server (unchanged), and is equivalent to the daemon flag for that purpose.
- `-d`/`--daemon` may be combined with `-config <path>` (e.g. `kaos-control -d -config /path/config.yaml` starts the server with that config).
- The server must **not** start unless either `-d`/`--daemon` or the `serve` subcommand is present.

**F4 — Help and version flags are unchanged**
- `--help`/`-h`/`-help` prints the usage guide and exits `0` (same content as F1/F2).
- `--version`/`-V`/`-version` prints the version/copyright/licence header and exits `0`.

**F5 — Existing subcommands unchanged**
- `init`, `auth`, `devops`, `hook-helper`, `backfill-created`, `backfill`, and `releases` dispatch and exit exactly as they do today, with their own argument parsing untouched.

**F6 — Unknown input handling**
- An unknown subcommand or unknown top-level flag writes an error plus the usage guide to **stderr** and exits with a **non-zero** code (current behaviour), rather than printing usage to stdout with exit 0. (This distinguishes "user asked for help" from "user made a mistake".)

### Non-functional

**NF1 — No new dependencies** — No new Go modules. The usage guide may be a static string or produced via the stdlib `flag` package's `Usage` hook.

**NF2 — Fast, side-effect-free help path** — The no-argument and `--help` paths must complete with no filesystem, network, or config-loading side effects, and return promptly (well under 100 ms).

**NF3 — Documentation consistency** — Any project documentation that tells users to run `kaos-control` (bare) to start the server must be updated to use `kaos-control -d` / `kaos-control serve`.

**NF4 — Stream/exit-code correctness** — Help requested by the user goes to stdout with exit 0; error diagnostics go to stderr with a non-zero exit. This must hold for both the help flags and the no-argument case.

## Acceptance Criteria

- [ ] `kaos-control` with no arguments prints the usage guide to stdout and exits `0`.
- [ ] The no-argument invocation starts no server, binds no port, opens no project, and starts no background goroutine.
- [ ] The usage guide includes the binary name, a one-line product description, the `-d`/`--daemon` flag, the `--version`/`--help`/`-config` flags, and all subcommands with descriptions.
- [ ] `kaos-control -d` starts the HTTP server (equivalent to the prior bare-invocation behaviour).
- [ ] `kaos-control --daemon` starts the HTTP server.
- [ ] `kaos-control -d -config <path>` starts the server using the given config file.
- [ ] `kaos-control serve` continues to start the server unchanged.
- [ ] `kaos-control --help` / `-h` prints the usage guide to stdout and exits `0`.
- [ ] `kaos-control --version` / `-V` prints the version/copyright/licence header and exits `0`.
- [ ] Each existing subcommand (`init`, `auth`, `devops`, `hook-helper`, `backfill-created`, `backfill`, `releases`) behaves exactly as before.
- [ ] An unknown subcommand or unknown top-level flag prints an error plus usage to stderr and exits non-zero.
- [ ] No new Go modules are introduced.
- [ ] Project documentation referencing a bare `kaos-control` server start is updated to `kaos-control -d` / `kaos-control serve`.
- [ ] Plans and tests in the [[daemon-flag-usage-guide]] lineage can reference this requirement.

## Resolved Questions

1. **Long/short flag wiring with the existing dispatch.** The current `main()` hand-rolls the first-argument switch and only `run()` calls `flag.Parse()`. Should `-d`/`--daemon` be handled in the top-level switch (so `kaos-control -d` is recognised before `flag.Parse`), and how should it compose with `-config` ordering (e.g. `-d -config x` vs `-config x -d`)? Recommendation: detect `-d`/`--daemon` in the top-level dispatch and then hand remaining args (including `-config`) to `run()`.

> Proceed as recommended

2. **Should `serve` be deprecated, aliased, or kept as a peer of `-d`?** The idea introduces `-d`/`--daemon` but `serve` already exists. Recommendation: keep `serve` as a documented equivalent; do not remove it. Confirm whether `serve` should still be advertised in the usage guide or quietly retained for compatibility.

> Keep serve as equivelent

3. **Exit code / stream for no-argument case — confirm stdout+0.** F1 specifies stdout and exit 0 (treating bare invocation as a help request). Some CLIs treat "no arguments" as a usage error (stderr, exit 2). The idea explicitly says "exit with code 0", so stdout+0 is assumed — confirm this is the intended convention.
