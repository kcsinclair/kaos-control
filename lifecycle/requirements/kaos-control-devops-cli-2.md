---
title: DevOps CLI with Linux-User Identity Mapping
type: requirement
status: blocked
lineage: kaos-control-devops-cli
created: "2026-06-26T00:00:00+10:00"
priority: high
parent: lifecycle/ideas/kaos-control-devops-cli.md
labels:
    - devops
    - backend
    - security
    - feature
    - tooling
    - go
release: KC-Release4
assignees:
    - role: product-owner
      who: agent
---

# DevOps CLI with Linux-User Identity Mapping

Parent: [[kaos-control-devops-cli]].

## Problem

Operational tasks against a kaos-control project — listing artifacts, triggering
agent runs, checking status, running test suites — are only reachable through the
web UI today. Operators working from a terminal and CI pipelines have no
scriptable, first-class interface. A naïve CLI would also be a security regression:
the HTTP API enforces authentication and role-based authz (see
[[auth-role-checks-mutations]]), and a CLI that bypassed those gates would become a
privileged back door on any machine with filesystem access to the project.

We need a `kaos-control devops` subcommand group that is (a) ergonomic from a shell,
(b) usable non-interactively in CI, and (c) subject to the **exact same** workflow
gates, role checks, and allowed-write-path restrictions that the HTTP API enforces —
attributing every action to a resolved kaos-control user.

## Goals / Non-goals

### Goals

1. Add a `kaos-control devops` subcommand group exposing read and action operations
   (`list`, `status`, `run`) against a registered project.
2. Resolve the invoking **Linux user** (`os/user`) to an existing kaos-control user
   account so role-based authz applies without a separate interactive login.
3. Reuse the **same authorization layer** as the HTTP API (role matrix, workflow
   gates, allowed-write-path checks) so the CLI is a first-class interface, never a
   back door.
4. Support **non-interactive / CI** use via an explicit API token or service-account
   identity, independent of the Linux-user mapping.
5. Make every command return a meaningful **exit code** and support
   machine-readable (`--json`) output for scripting.

### Non-goals

- A new transport or business-logic layer: the CLI must call the existing
  domain/service packages (or a loopback to the running server), not reimplement
  workflow or authz logic.
- Managing user accounts or tokens — that is owned by [[cli-auth-user-management]].
- OS-level privilege escalation, sudo integration, or changing filesystem
  permissions. Filesystem permissions are respected as-is by the OS.
- A long-running daemon or interactive TUI.
- SSO/OIDC identity (see [[sso-oauth-integration]]).

## Detailed Requirements

### Functional

| ID | Requirement |
|----|-------------|
| F1 | `kaos-control devops --help` lists the subcommand group and every subcommand with a one-line synopsis; `kaos-control --help` lists `devops` as a top-level subcommand. |
| F2 | `kaos-control devops list [--type <t>] [--status <s>] [--lineage <slug>] [--json]` prints the project's artifacts, filterable by type/status/lineage, as an aligned table or JSON array. |
| F3 | `kaos-control devops status [--json]` prints project health: artifact counts by status, active/queued agent runs, and lock state. |
| F4 | `kaos-control devops run <agent-or-task> [--artifact <path>] [--json]` triggers an agent run (or a named task such as `test-all`) and reports the resulting run id; `--follow` streams run output until completion. |
| F5 | `--project <name>` selects the target registered project; when omitted, the project is inferred from the current working directory (matching a registered project root) and an unambiguous match is required. |
| F6 | The CLI resolves the invoking identity, in precedence order: (a) `--token <t>` flag or `KAOS_CONTROL_TOKEN` env var → bearer-token user; (b) `--as <email>` for an explicit service-account (only honoured for already-privileged callers, see F8); (c) the Linux username from `os/user` mapped to a kaos-control account. |
| F7 | Linux-user → kaos-control-user mapping is read from project/app config (e.g. a `linux_user:` field or `os_user → email` map). If the invoking Linux user has no mapping and no token is supplied, the command exits non-zero with a clear "identity not resolved" error and does **not** perform the action. |
| F8 | Every action command enforces the **same role matrix and workflow gates** as the equivalent HTTP endpoint (per [[auth-role-checks-mutations]]). A caller lacking the required role is rejected with a non-zero exit and a "role required: <roles>" message, regardless of filesystem access. |
| F9 | Action commands (those that write artifacts or trigger runs) respect each role's `allowed_write_paths`; an attempted write outside the allowed paths is refused before any disk mutation. |
| F10 | Read commands (`list`, `status`) require a resolved, authenticated identity but no write role. |
| F11 | Run attribution: the resulting agent run / artifact change is recorded against the resolved kaos-control user (not the raw Linux user), so audit and analytics ([[agent-usage-analytics-report]]) attribute correctly. |
| F12 | All commands emit human-readable output by default and strict JSON to stdout under `--json`, with diagnostics on stderr so stdout stays parseable. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NF1 | Authorization decisions reuse the existing helpers (`RolesFor`, `hasAnyRole`, `workflow.CanTransition`) — no new or parallel authz primitives. |
| NF2 | Tokens are never logged or echoed; `--token` values are redacted from any verbose/debug output. |
| NF3 | CLI commands operate without requiring an interactive password prompt, so they are safe in CI; failure to resolve identity is a hard error, never a silent fallback to an elevated identity. |
| NF4 | Read commands complete in ≤1s for a project of ≤1000 artifacts on a warm index. |
| NF5 | Exit codes are conventional: `0` success, non-zero on any error, with distinct codes for "identity unresolved", "forbidden", and "operation failed" documented in `--help`. |
| NF6 | All new code passes `make lint` and `make test-unit`; integration coverage lives in `tests/`. |

## Acceptance Criteria

- [ ] `kaos-control devops --help` and `kaos-control --help` both list `devops` with a synopsis.
- [ ] `kaos-control devops list --json` against a seeded project returns a JSON array of artifacts matching the index, and `--status`/`--type`/`--lineage` filters narrow the set correctly.
- [ ] `kaos-control devops status` reports artifact-status counts and active-run/lock state consistent with the HTTP `status` view.
- [ ] `kaos-control devops run <agent>` as a correctly-roled user starts a run and prints a run id; `--follow` streams output and exits with the run's terminal status.
- [ ] A Linux user mapped to a kaos-control account with an **insufficient** role is rejected by `devops run` with a non-zero exit and "role required" message — even though they can read the files on disk.
- [ ] A Linux user with **no** mapping and no token gets a non-zero "identity not resolved" error and the action does not execute.
- [ ] `KAOS_CONTROL_TOKEN`/`--token` authenticates a CI invocation as the token's user, and the same role gates apply.
- [ ] An action that would write outside the resolved role's `allowed_write_paths` is refused before any file is modified.
- [ ] A triggered run is attributed to the resolved kaos-control user in run history / analytics, not to the raw Linux username.
- [ ] CLI authz outcomes (allow/deny) match the HTTP API for the same identity+operation in a parity test.
- [ ] Integration tests in `tests/` cover identity resolution (token, mapped Linux user, unmapped), role-gated allow/deny, and `--json` output shape.
- [ ] `make lint` and `make test-unit` pass; related: [[auth-role-checks-mutations]], [[cli-auth-user-management]], [[devops-pipelines]].

## Open Questions

1. **Linux-user mapping location & shape.** Should the `os_user → email` map live in
   per-project `config.yaml`, in app-level `~/.kaos-control/config.yaml`, or as a
   field on each user record? The idea implies per-user association but does not fix
   the storage location.
2. **Service-account (`--as`) authority.** Should `--as <email>` impersonation be
   available at all in this iteration, and if so, which role(s) may use it
   (`product-owner` only)? Or should CI rely solely on per-account bearer tokens?
3. **Task vocabulary for `run`.** Is `run` limited to configured agents, or does it
   also expose named composite tasks like `test-all`? If the latter, where are those
   tasks defined — config, or a fixed built-in set?
4. **Loopback vs in-process.** Should the CLI talk to a running server over the
   local API (requiring the server to be up) or operate in-process against the index
   and agent runner directly (working offline)? This affects how `--follow`
   streaming and live runs behave.
5. **Trust model for `os/user`.** Mapping trusts the OS-reported username on a
   shared host. Is that acceptable for the target deployment, or must the CLI also
   require a token even for mapped Linux users in multi-tenant environments?
