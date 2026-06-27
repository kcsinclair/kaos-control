---
title: "Backend Plan: Default to Usage Guide; Require -d/--daemon to Start the Server"
type: plan-backend
status: in-development
lineage: daemon-flag-usage-guide
parent: lifecycle/requirements/daemon-flag-usage-guide-2.md
---

# Backend Plan: Default to Usage Guide; Require -d/--daemon to Start the Server

Implements [[daemon-flag-usage-guide]] (requirement
`daemon-flag-usage-guide-2`). The entire change lives in the CLI entry point
`cmd/kaos-control/main.go` plus project documentation. There are no changes to
`run()`'s server-startup body, to any `internal/` package, or to the SPA — see
the frontend plan ([[daemon-flag-usage-guide]] `-4-fe`) for why the SPA is
untouched, and the test plan (`-5-test`) for coverage.

## Decisions inherited from the requirement (read first)

The requirement body (F1, several acceptance criteria, NF4) originally
specified the **no-argument** case as *stdout + exit 0*. **Resolved Question 3
overrides this**: the product owner chose to treat a bare invocation as a
**usage error → stderr, exit 2**. This plan follows the resolved decision. The
distinction the resolved answer draws is:

| Invocation | Stream | Exit | Rationale |
|---|---|---|---|
| `kaos-control` (no args) | **stderr** | **2** | user gave no instruction → usage error (Resolved Q3) |
| `kaos-control --help` / `-h` / `-help` | **stdout** | **0** | help explicitly requested (F4) |
| `kaos-control --version` / `-V` | **stdout** | **0** | version explicitly requested (F4) |
| unknown flag / subcommand | **stderr** | **1** | mistake, current behaviour (F6) |

`--help` stays stdout+0 (F4 is unchanged); only the *no-arg* stream/exit moves.

From Resolved Question 1: detect `-d`/`--daemon` in the top-level hand-rolled
dispatch, strip it, and hand the remaining args (including `-config`) to
`run()`. From Resolved Question 2: keep `serve` as a documented **peer** of
`-d` (not deprecated, still advertised in the usage guide).

A consequence of F3 ("the server must **not** start unless `-d`/`--daemon` or
`serve` is present") is that the **current implicit "fall through to serve via
`-config`" path is removed**: `kaos-control -config <path>` *without* `-d` must
no longer start the server. It becomes a usage error (stderr + usage, exit 2).
This is a deliberate behaviour change entailed directly by F3; it is called out
again in Milestone 2 and verified by the test plan.

---

## Milestone 1 — No-argument invocation prints usage to stderr and exits 2

### Description

Today, when `len(os.Args) == 1`, `main()` skips the dispatch switch entirely and
falls through to `run()`, which starts the server (`cmd/kaos-control/main.go:64`,
`:124`). Intercept the no-argument case at the very top of `main()` and emit the
usage guide to **stderr**, then `os.Exit(2)`. This must happen before any
logging setup, config load, listener bind, or project open — satisfying F1 and
NF2 (side-effect-free, sub-100 ms path).

### Files to change

- `cmd/kaos-control/main.go` — at the start of `main()`, before the existing
  `if len(os.Args) > 1` block:

  ```go
  if len(os.Args) == 1 {
      // No command given. Treat as a usage error (Resolved Q3): write the
      // usage guide to stderr and exit non-zero, rather than starting the
      // server. No side effects occur on this path.
      fmt.Fprint(os.Stderr, usage)
      os.Exit(2)
  }
  ```

### Acceptance criteria

- [ ] `kaos-control` with no arguments writes the usage guide to **stderr** and exits **2**.
- [ ] The no-argument path performs no logging setup, no `config.LoadApp`, no `net.Listen`, no `project.Open`, and starts no goroutine (verified by inspection and by the test plan's process-level checks).
- [ ] `run()` is not reached when no arguments are supplied.
- [ ] The path returns well under 100 ms (NF2).

---

## Milestone 2 — Gate server start behind `-d`/`--daemon`; compose with `-config`; remove implicit `-config` serve

### Description

Add `-d` / `--daemon` as the explicit daemon opt-in, handled in the top-level
hand-rolled dispatch (Resolved Q1) so it is recognised before `run()` calls
`flag.Parse()` (the `flag` package does not know about `-d`). Requirements F3
demands both orderings work: `kaos-control -d -config x` **and**
`kaos-control -config x -d`. Because the daemon flag may appear in any position,
scan the arguments, record whether a daemon flag is present, and strip it from
`os.Args` before calling `run()` (mirroring how `serve` is already stripped at
`cmd/kaos-control/main.go:97-99`). Accept `-d`, `--daemon`, and `-daemon`
(single- and double-dash, matching Go `flag` conventions and the hand-rolled
`--config`/`-config` handling already present).

Then enforce F3: the server starts **only** when a daemon flag is present **or**
the `serve` subcommand was used. The existing implicit path where
`kaos-control -config <path>` (with no `-d`) falls through to `run()` and starts
the server (`cmd/kaos-control/main.go:106-118`) is **removed** — a bare
`-config` with no daemon opt-in becomes a usage error.

Recommended shape (keep the existing subcommand `case`s untouched; restructure
only the flag-handling and fall-through):

```go
// After the no-arg guard (Milestone 1) and after the subcommand switch has
// had its chance to handle init/auth/devops/hook-helper/backfill*/releases/
// serve/--help/--version, decide whether to start the daemon.

daemon := false
filtered := os.Args[:1] // keep argv[0]
for _, a := range os.Args[1:] {
    switch a {
    case "-d", "--daemon", "-daemon":
        daemon = true // strip: do not pass to flag.Parse
    default:
        filtered = append(filtered, a)
    }
}
os.Args = filtered

if !daemon {
    // No -d/--daemon and not the `serve` subcommand (serve is handled in the
    // switch, which returns or sets a sentinel). Anything left here — e.g. a
    // bare `-config <path>` — is a usage error: the server requires opt-in.
    fmt.Fprintf(os.Stderr, "error: no command given; use -d/--daemon or 'serve' to start the server\n\n%s", usage)
    os.Exit(2)
}
```

The developer has latitude on exactly how to thread `serve` vs `-d` through the
existing switch (e.g. a small `start bool` set in the `serve` case and in the
daemon scan, with a single `if start { run() }` at the end). The required
**observable** behaviour is the acceptance criteria below; the control-flow
shape is a recommendation. Keep `unknown flag`/`unknown subcommand` handling
(F6) exactly as today: stderr + usage + exit 1.

### Files to change

- `cmd/kaos-control/main.go` — restructure the dispatch in `main()` so that:
  - `-d` / `--daemon` (any position, combinable with `-config`) starts `run()`;
  - `serve` continues to start `run()` (unchanged semantics, Resolved Q2);
  - the daemon flag is stripped from `os.Args` before `flag.Parse()` in `run()`;
  - a no-daemon, no-`serve` invocation that is not help/version/a subcommand is a usage error.
  - `run()` itself (`:130` onward) is **not** modified.

### Acceptance criteria

- [ ] `kaos-control -d` starts the HTTP server (equivalent to the prior bare-invocation behaviour).
- [ ] `kaos-control --daemon` starts the HTTP server.
- [ ] `kaos-control -d -config <path>` starts the server using the given config file.
- [ ] `kaos-control -config <path> -d` (reverse order) also starts the server using the given config file.
- [ ] `kaos-control serve` continues to start the server unchanged, and `serve -config <path>` continues to work.
- [ ] `kaos-control -config <path>` **without** `-d` does **not** start the server; it prints usage to stderr and exits non-zero (consequence of F3).
- [ ] The stripped daemon token never reaches `flag.Parse()` (no "flag provided but not defined: -d" error in any ordering).
- [ ] Unknown top-level flag or unknown subcommand still prints an error plus usage to **stderr** and exits **1** (F6, unchanged).
- [ ] `init`, `auth`, `devops`, `hook-helper`, `backfill-created`, `backfill`, `releases` dispatch and exit exactly as before (F5) — their `os.Args[2:]` parsing is untouched.
- [ ] `go build ./...` and `go vet ./...` pass; no new Go modules added (NF1).

---

## Milestone 3 — Usage guide content (F2)

### Description

Update the `usage` const (`cmd/kaos-control/main.go:34-53`) so a single bare or
`--help` invocation orients a new user. The current banner is **missing** the
`devops` and `releases` subcommands and has no `-d`/`--daemon` flag or product
description. F2 requires, at minimum: binary name; a one-line product
description; the `-d`/`--daemon` flag with a short description; the other
top-level flags (`--version`/`-V`, `--help`/`-h`, `-config <path>`); all eight
subcommands (`serve`, `init`, `auth`, `devops`, `hook-helper`,
`backfill-created`, `backfill`, `releases`) each with a one-line description;
and the per-command help pointer (already present).

Keep `serve` listed as a peer with a note that it is equivalent to `-d`
(Resolved Q2). Suggested banner:

```text
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
```

The exact wording of the one-line product description can be refined to match
README/`Innovation Maker` framing; the **structural** requirements are the
acceptance criteria below.

### Files to change

- `cmd/kaos-control/main.go` — expand the `usage` const to satisfy F2.

### Acceptance criteria

- [ ] The usage guide includes the binary name `kaos-control`.
- [ ] The usage guide includes a one-line product description.
- [ ] The usage guide lists the `-d`/`--daemon` flag with a short description.
- [ ] The usage guide lists `--version`/`-V`, `--help`/`-h`, and `-config <path>`.
- [ ] The usage guide lists all eight subcommands (`serve`, `init`, `auth`, `devops`, `hook-helper`, `backfill-created`, `backfill`, `releases`), each with a one-line description.
- [ ] The usage guide notes that `serve` is equivalent to `-d`/`--daemon` (Resolved Q2).
- [ ] The usage guide includes the per-command help pointer.
- [ ] `kaos-control --help` / `-h` / `-help` prints this guide to **stdout** and exits **0** (F4, unchanged).
- [ ] `kaos-control --version` / `-V` / `-version` prints the version/copyright/licence header to stdout and exits 0 (F4, unchanged).
- [ ] The same `usage` string is reused for the no-arg (Milestone 1), unknown-input (F6), and help paths — single source of truth.

---

## Milestone 4 — Documentation consistency (NF3)

### Description

Update every piece of project documentation that tells a user to run
`kaos-control` (bare) to start the server, so it uses `kaos-control -d` (or
`kaos-control serve`). Known touchpoints found in the repo:

- `README.md:266` — the fenced command `./dist/kaos-control` → `./dist/kaos-control -d`.
- `README.md:181` and `README.md:269` — prose that says a bare run "starts on `:8042`" / "starts listening on `:8042`"; reword to reflect that `-d` (or `serve`) is now required.
- `Makefile:45-46` — the `run:` target runs `go run … ./cmd/kaos-control` with no args; append `-d` so `make run` still starts the server in dev.
- Sweep `docs/*.md` (e.g. `docs/architecture.md`, `docs/end-to-end-smoke-tests.md`, `docs/ollama-claude-code-driver.md`) and `docs/architecture-diagram.html` / `docs/architecture-summary.md` for any bare-start instruction and update consistently.

Per CLAUDE.md commit conventions, also bump `plans/PROJECT_PLAN.md`
"Recent Changes" when this change is committed.

### Files to change

- `README.md` (lines ~181, ~266, ~269 and any other bare-start references).
- `Makefile` (`run:` target).
- `docs/*.md` and `docs/architecture-diagram.html` where a bare `kaos-control` start is documented.
- `plans/PROJECT_PLAN.md` (Recent Changes log) at commit time.

### Acceptance criteria

- [ ] No remaining project documentation instructs the user to start the server with a bare `kaos-control` / `./dist/kaos-control` invocation.
- [ ] `make run` starts the server (its target passes `-d`).
- [ ] `grep -rnE "kaos-control *$|dist/kaos-control *$" README.md docs Makefile` returns no bare-start instructions (allowing for legitimate non-start mentions).
- [ ] `plans/PROJECT_PLAN.md` reflects the change in its Recent Changes log.

---

## Cross-links

- [[daemon-flag-usage-guide]] — originating idea and the requirement (`-2`) this plan implements.
- Frontend plan `daemon-flag-usage-guide-4-fe` — establishes that the SPA needs no code changes; the daemon still serves the identical embedded SPA once started.
- Test plan `daemon-flag-usage-guide-5-test` — exercises every acceptance criterion above at the process/CLI level, including the Resolved-Q3 stderr+exit-2 no-arg behaviour and the removed implicit-`-config` path.
