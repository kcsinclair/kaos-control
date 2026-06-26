---
title: DevOps CLI with Linux-User Identity Mapping — Test Plan
type: plan-test
status: draft
lineage: kaos-control-devops-cli
parent: lifecycle/requirements/kaos-control-devops-cli-2.md
release: KC-Release4
assignees:
    - role: test-developer
      who: agent
---

# DevOps CLI with Linux-User Identity Mapping — Test Plan

Covers the acceptance criteria of [[kaos-control-devops-cli-2]], implemented by
[[kaos-control-devops-cli-3-be]] (backend) and [[kaos-control-devops-cli-4-fe]] (frontend).

Tests split into: (1) unit tests for the new config field/helper and the loopback identity
middleware; (2) build-tagged CLI integration tests that invoke the compiled binary against a
running server, following the established `tests/cli_*_test.go` pattern (binary built once by
`TestMain`, isolated temp config/data dirs per test). Related lineages exercised:
[[auth-role-checks-mutations]], [[devops-pipelines]], [[devops-pipeline-log-streaming]],
[[agent-usage-analytics-report]].

## Milestone 1 — Unit tests: config mapping and identity helper

### Description

Test `linux_user` parsing, `EmailForLinuxUser`, and ambiguous-mapping validation in
isolation (backend Milestone 1).

### Files to change

- **`internal/config/config_test.go`** (extend)
  - `linux_user` on a binding round-trips through `LoadProject`.
  - `EmailForLinuxUser("alice")` → bound email + `true`; unmapped → `("", false)`;
    `""` input → `("", false)`.
  - Duplicate `linux_user` across two bindings → `LoadProject` errors.

### Acceptance criteria

- [ ] All cases above pass under `make test-unit`.

## Milestone 2 — Unit tests: loopback-trusted local identity

### Description

Test the `X-Kaos-Local-User` middleware path (backend Milestone 2) with `httptest`,
asserting the loopback gate and the no-silent-elevation guarantee (NF3, F7).

### Files to change

- **`internal/http/auth_test.go`** (extend) or new `internal/http/local_identity_test.go`
  - Loopback RemoteAddr + known email → request authenticated as that user; role-gated
    handler allows/denies per `RolesFor`.
  - **Non-loopback** RemoteAddr + same header → header ignored → 401 (channel is
    loopback-only).
  - Unknown email in header → 401, no fallback identity.
  - Header present alongside a valid session/bearer → existing auth still wins, no
    regression.

### Acceptance criteria

- [ ] All cases pass; `make lint` + `make test-unit` clean for `internal/http`.

## Milestone 3 — CLI integration: help, project selection, output shape

### Description

Build-tagged integration tests invoking the compiled binary (F1, F5, F12).

### Files to change

- **New** `tests/cli_devops_test.go` (build tag `integration`, package `cli_test`)
  - `kaos-control --help` output contains `devops`; `kaos-control devops --help` lists
    `list`, `status`, `run` and documents exit codes (F1, NF5).
  - `--project <name>` selects a registered project; running inside a registered project
    root with no `--project` infers it; an ambiguous/unregistered cwd errors non-zero (F5).
  - `devops list --json` against a seeded project emits a valid JSON array matching the
    index; `--type`/`--status`/`--lineage` filters narrow the set (F2).
  - `--json` writes only JSON to stdout while diagnostics go to stderr (F12) — assert stdout
    parses as JSON even when a warning is emitted.

### Acceptance criteria

- [ ] All assertions above pass under `go test -tags integration ./tests/...`.

## Milestone 4 — CLI integration: identity resolution

### Description

Cover the three identity paths and the hard-fail-on-unresolved guarantee (F6, F7, NF3).

### Files to change

- **`tests/cli_devops_test.go`** (extend)
  - **Token**: `KAOS_CONTROL_TOKEN` / `--token` authenticates a CI invocation as the
    token's user; role gates still apply (acceptance bullet, F6a).
  - **Mapped Linux user**: with a `linux_user` mapping matching the test's OS user (resolve
    via `os/user.Current()` and write that into the fixture config), a read command succeeds
    with no token (F6c). *Note for developer: bind the fixture to the actual test runner's
    username so the mapping resolves in CI.*
  - **Unmapped Linux user, no token**: exits `3` with "identity not resolved" and performs
    no action (F7, NF3).
  - **Token redaction**: `--token` value never appears in stdout/stderr, including any
    verbose mode (NF2).

### Acceptance criteria

- [ ] Each path produces the documented exit code and behaviour; the unmapped case mutates
      nothing.

## Milestone 5 — CLI integration: role-gated `run`, attribution, and HTTP parity

### Description

The core security tests (F8, F9, F11) plus the allow/deny parity check against the HTTP API.

### Files to change

- **`tests/cli_devops_test.go`** (extend)
  - `devops run <task>` as a `product-owner`/`devops` user starts a run and prints a run id;
    `--follow` streams the NDJSON log and exits with the run's terminal status (F4) — uses a
    trivial seeded pipeline per [[devops-pipelines]] / [[devops-pipeline-log-streaming]].
  - A mapped Linux user with an **insufficient** role is rejected by `devops run` with exit
    `4` and "role required", despite filesystem read access (F8, acceptance bullet).
  - A run that would write outside the role's `allowed_write_paths` is refused before any
    file changes (F9).
  - **Attribution**: after a CLI-triggered run, the run history / analytics record the
    resolved kaos-control email, not the raw Linux username (F11) —
    cross-check [[agent-usage-analytics-report]].
  - **Parity**: for the same identity+operation, assert the CLI's allow/deny outcome matches
    the HTTP API's (drive both the CLI and a direct authenticated HTTP call, compare exit
    code vs status code class). Covers the parity acceptance bullet.

### Acceptance criteria

- [ ] All cases pass; CLI and HTTP authz outcomes agree for matched identity+operation.
- [ ] `make lint` and `make test-unit` pass; `go test -tags integration ./tests/...` green.

## Milestone 6 — Frontend regression check

### Description

Lightweight verification of the two FE seams in [[kaos-control-devops-cli-4-fe]].

### Files to change

- **`tests/web/`** (extend, if a component/e2e harness exists) — otherwise document as a
  manual check in `tests/manual/`:
  - Project users view shows `linux_user` for a mapped binding and renders cleanly when
    absent.
  - A CLI-triggered pipeline run appears in `RunHistory.vue` attributed to the resolved
    user and streams to completion in the log view.

### Acceptance criteria

- [ ] The two FE behaviours are verified (automated where a harness exists, else a recorded
      manual check); no console errors and `pnpm build` clean.
