---
title: DevOps CLI with Linux-User Identity Mapping — Backend Plan
type: plan-backend
status: in-development
lineage: kaos-control-devops-cli
parent: lifecycle/requirements/kaos-control-devops-cli-2.md
release: KC-Release4
assignees:
    - role: backend-developer
      who: agent
---

# DevOps CLI with Linux-User Identity Mapping — Backend Plan

Implements the `kaos-control devops` subcommand group from [[kaos-control-devops-cli-2]].

## Architecture decision (resolved from requirement)

Per the requirement's *Resolved Questions*, the CLI **talks to the running server** over
the local HTTP API (Q4) and never reimplements workflow/authz logic (non-goal §1, NF1).
The server already enforces the role matrix and workflow gates via `requireRole` /
`RolesFor` and authenticates bearer tokens via `auth.Store.ValidateToken`
(see [[auth-role-checks-mutations]]). The CLI is therefore an HTTP client plus an
**identity resolver**; the only new server-side surface is a mechanism to assert a
*locally-trusted Linux identity* to the server.

Identity precedence (F6): `--token`/`KAOS_CONTROL_TOKEN` → `--as <email>` → Linux user
from `os/user` mapped via project config. The first two map to existing token auth; the
third needs the local-identity path in Milestone 2.

`devops run` targets **devops pipelines/tasks only** (Q3), reusing the existing pipeline
endpoints (see [[devops-pipelines]] and [[devops-pipeline-log-streaming]]). Agent runs are
explicitly out of scope for `run`.

The frontend counterpart is [[kaos-control-devops-cli-4-fe]]; tests are
[[kaos-control-devops-cli-5-test]].

---

## Milestone 1 — `linux_user` config field and lookup helper

### Description

Add the Linux-username → kaos-control-email mapping to the project config. Per Q1 the
mapping is a field on the project's user binding (alongside `email`), so it lives in
`lifecycle/config.yaml`. Add a lookup helper mirroring `RolesFor`.

### Files to change

- **`internal/config/config.go`**
  - Extend `UserBinding`:
    ```go
    type UserBinding struct {
        Email     string   `yaml:"email"`
        Roles     []string `yaml:"roles"`
        LinuxUser string   `yaml:"linux_user,omitempty"`
    }
    ```
  - Add `func (p *Project) EmailForLinuxUser(linuxUser string) (string, bool)` — iterates
    `p.Users`, returns the bound `Email` for the first binding whose `LinuxUser` equals
    `linuxUser` (exact match, case-sensitive). Returns `("", false)` when unmapped or when
    `linuxUser == ""`.
  - Add validation in the project-config validate path: if two bindings share the same
    non-empty `LinuxUser`, return an error (ambiguous mapping must fail loudly, not pick
    arbitrarily).

- **`lifecycle/config.yaml`** (this project's own config) — add `linux_user:` to the
  `keith@sinclair.org.au` binding as a worked example for the integration fixtures.

### Acceptance criteria

- [ ] A project config with `linux_user: alice` on a binding round-trips through
      `LoadProject` with the field populated.
- [ ] `EmailForLinuxUser("alice")` returns the bound email and `true`; an unmapped name
      returns `("", false)`; `EmailForLinuxUser("")` returns `("", false)`.
- [ ] Two bindings with the same `linux_user` cause `LoadProject` to error.
- [ ] `make test-unit` passes for `internal/config`.

---

## Milestone 2 — Loopback-trusted local identity on the server

### Description

The Linux-user path (F6c, F7) needs the CLI to assert a resolved email to the server
*without* a password or token. Per Q5 the OS-reported username is trusted on the host, and
Q4 fixes the server as the execution point. Add a **loopback-only** trusted-identity path
to the auth middleware: a request carrying `X-Kaos-Local-User: <email>` is authenticated as
that user **iff** (a) the connection originates from loopback (`127.0.0.0/8`, `::1`) and
(b) the email exists in the auth store. Role authority still derives from `RolesFor` per
project — this is an *identity* mechanism, not new authz (NF1, non-goal §1). It is never a
silent elevation: an unrecognised email is a hard 401, matching NF3/F7.

### Files to change

- **`internal/http/auth.go`**
  - In the auth middleware (`authMiddleware`/`requireAuth`), after the session and
    `Authorization: Bearer` checks and before the unauthenticated fall-through, add:
    1. Read `X-Kaos-Local-User`. If empty, skip.
    2. Confirm the remote address is loopback (`net.ParseIP(host).IsLoopback()`); if not,
       ignore the header entirely (do **not** 401 differently — treat as no header so the
       channel can't be probed remotely).
    3. `user, _ := s.cfg.Auth.GetUser(email)`; if non-nil, attach `user` to the request
       context exactly as the bearer path does. Mark the request as local-identity (a new
       `localIdentityContextKey`) and treat it as CSRF-exempt like bearer auth, since these
       are non-browser callers.
    4. If the email is unknown, leave the request unauthenticated → downstream 401.
  - Add `isLocalIdentity(ctx)` helper alongside `isBearerAuth`.

### Acceptance criteria

- [ ] A loopback request with `X-Kaos-Local-User: <known-email>` is authenticated as that
      user and passes role-gated endpoints exactly as a session would.
- [ ] The same header on a **non-loopback** RemoteAddr is ignored (request treated as
      unauthenticated → 401), proving the channel is loopback-only.
- [ ] An unknown email in the header yields 401, never a fallback to any other identity.
- [ ] Existing session and bearer auth tests are unaffected.
- [ ] `make lint` and `make test-unit` pass for `internal/http`.

---

## Milestone 3 — `devops` subcommand scaffold and identity resolution

### Description

Add the `kaos-control devops` subcommand group using the same manual-dispatch pattern as
`auth` (see `cmd/kaos-control/authcmd/`) and `cli-init-scaffold`. Wire it into `main.go`.
Implement identity resolution (F6/F7), project selection (F5), `--json` plumbing (F12), and
the documented exit-code scheme (NF5).

### Files to change

- **`cmd/kaos-control/main.go`**
  - Add `case "devops": os.Exit(devopscmd.Run(os.Args[2:]))` to the dispatch switch.
  - Add a `devops    DevOps operations against a registered project` line to the top-level
    usage text so `kaos-control --help` lists it (F1).

- **New** `cmd/kaos-control/devopscmd/devopscmd.go`
  - `func Run(args []string) int` — parses the leading subcommand (`list`, `status`,
    `run`, `--help`) and dispatches; unknown subcommand prints usage and returns the
    "operation failed" code. The `devops --help` text lists every subcommand with a
    one-line synopsis and documents the exit codes (F1, NF5).
  - Define exit-code constants (NF5):
    ```go
    const (
        exitOK             = 0
        exitOpFailed       = 1
        exitIdentityUnresolved = 3
        exitForbidden      = 4
    )
    ```

- **New** `cmd/kaos-control/devopscmd/identity.go`
  - `resolveIdentity(flags)` returns an `authMode` describing how to authenticate the HTTP
    client, in precedence order (F6):
    1. `--token`/`KAOS_CONTROL_TOKEN` → bearer mode carrying the token.
    2. `--as <email>` → only honoured when the *base* resolved caller is already privileged
       (the CLI sets `X-Kaos-Local-User` to the `--as` email; the server's loopback path +
       `RolesFor` decide authority — Q2). For non-privileged callers the server rejects it.
    3. `os/user.Current().Username` → `Project.EmailForLinuxUser`; on hit, local-identity
       mode carrying the resolved email. On miss with no token, return
       `exitIdentityUnresolved` with `identity not resolved: linux user %q has no mapping
       and no --token/KAOS_CONTROL_TOKEN supplied` (F7, NF3).
  - Never log or echo token values; redact them in any verbose output (NF2).

- **New** `cmd/kaos-control/devopscmd/project.go`
  - `selectProject(flags)` — `--project <name>` selects from the registry
    (`config.LoadProjectRegistry`); when omitted, infer by matching `os.Getwd()` against
    registered project roots, requiring an unambiguous match else
    `exitOpFailed` with a clear message (F5).

### Acceptance criteria

- [ ] `kaos-control --help` lists `devops`; `kaos-control devops --help` lists `list`,
      `status`, `run` with synopses and documents exit codes 0/1/3/4 (F1, NF5).
- [ ] Identity resolution honours the precedence token → `--as` → Linux user.
- [ ] Unmapped Linux user with no token exits `3` with an "identity not resolved" message
      and performs no action (F7).
- [ ] `--project` selects explicitly; omitted `--project` infers from cwd and errors on
      ambiguity (F5).
- [ ] `--token` values never appear in stdout/stderr/verbose output (NF2).
- [ ] `make lint` passes (`go vet` + staticcheck) for the new package.

---

## Milestone 4 — Authenticated HTTP client and `list` / `status`

### Description

Implement the read commands (F2, F3, F10) against existing endpoints. A small client wraps
the resolved identity: it sets `Authorization: Bearer <token>` (bearer mode) or
`X-Kaos-Local-User: <email>` (local-identity mode) on every request to the server's listen
address. Read commands require a resolved identity but no write role (F10).

### Files to change

- **New** `cmd/kaos-control/devopscmd/client.go`
  - `newClient(appCfg, identity)` — base URL from `App.Server.Listen` (loopback host +
    port; honour `public_host`/TLS if set). `do(method, path, body)` attaches the identity
    headers, returns body + status. Maps `401`→`exitIdentityUnresolved`, `403`→
    `exitForbidden` (parsing the server's `role required:` message through to stderr),
    other non-2xx → `exitOpFailed` (NF5, F8).
  - Diagnostics go to **stderr**; only command output goes to stdout so `--json` stays
    parseable (F12).

- **New** `cmd/kaos-control/devopscmd/list.go`
  - `kaos-control devops list [--type] [--status] [--lineage] [--json]` → `GET
    /api/p/{project}/artifacts` with query params forwarded to the existing
    `handleListArtifacts` filters. Human output is an aligned table; `--json` emits the raw
    artifact array to stdout (F2).

- **New** `cmd/kaos-control/devopscmd/status.go`
  - `kaos-control devops status [--json]` composes project health from existing endpoints:
    artifact status-distribution (`handleGetStatusDistribution`), active/queued agent runs
    (agent-runs list), and lock state (locks endpoint). Human output is a summary block;
    `--json` emits one combined object (F3). Document in code that this mirrors the HTTP
    `status` view for the parity test in [[kaos-control-devops-cli-5-test]].

### Acceptance criteria

- [ ] `devops list --json` returns a JSON array matching the index; `--type`/`--status`/
      `--lineage` narrow the set correctly (F2).
- [ ] `devops status` reports status counts, active-run, and lock state consistent with the
      HTTP views (F3).
- [ ] Both commands succeed for any resolved identity and require no write role (F10).
- [ ] `403`/`401` responses map to exit codes `4`/`3` with messages on stderr; stdout under
      `--json` remains valid JSON (F12, NF5).
- [ ] Read commands complete in ≤1s against a warm index for ≤1000 artifacts (NF4).

---

## Milestone 5 — `run` for devops tasks with `--follow`

### Description

Implement `devops run <task>` against devops pipelines only (Q3). It triggers a pipeline
run via the existing endpoint, prints the run id, and with `--follow` streams the NDJSON run
log to completion, exiting with the run's terminal status. Authz, workflow gates, and
allowed-write-path enforcement happen server-side via `requireRole` (F8, F9, NF1); the CLI
adds no parallel checks. Attribution is automatic: the server records the run against the
resolved user from the identity headers, not the raw Linux name (F11) — relevant to
[[agent-usage-analytics-report]].

### Files to change

- **New** `cmd/kaos-control/devopscmd/run.go`
  - `kaos-control devops run <task> [--json] [--follow]`:
    1. `POST /api/p/{project}/devops/pipelines/{task}/run` → parse `run_id`. Print the run
       id (human) or `{"run_id": "..."}` (`--json`).
    2. A `403` from the server (insufficient role / write-path violation) surfaces the
       server's `role required:` message to stderr and exits `4` — the CLI does not
       pre-judge (F8, F9).
    3. With `--follow`: poll/stream `GET /api/p/{project}/devops/runs/{run_id}` (NDJSON, per
       [[devops-pipeline-log-streaming]]), writing log lines to stdout (or, under `--json`,
       passing through the NDJSON events), and exit `0` on success / `1` on a failed
       terminal status (F4).
  - Unknown task slug → `exitOpFailed` with the server's 404 message.

### Acceptance criteria

- [ ] `devops run <task>` as a correctly-roled (`product-owner`/`devops`) user starts a run
      and prints a run id (F4).
- [ ] `--follow` streams output and exits with the run's terminal status (F4).
- [ ] A mapped Linux user with an **insufficient** role is rejected with exit `4` and a
      "role required" message, even with filesystem read access (F8 — acceptance bullet).
- [ ] A write outside the role's `allowed_write_paths` is refused server-side before any
      mutation; the CLI exits `4` (F9).
- [ ] The triggered run is attributed to the resolved kaos-control user in run history /
      analytics, not the Linux username (F11).
- [ ] `make lint` and `make test-unit` pass; integration coverage in
      [[kaos-control-devops-cli-5-test]] is green.
