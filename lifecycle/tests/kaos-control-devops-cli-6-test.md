---
title: "Tests — DevOps CLI with Linux-User Identity Mapping"
type: test
status: draft
lineage: kaos-control-devops-cli
parent: lifecycle/test-plans/kaos-control-devops-cli-5-test.md
---

# Tests — DevOps CLI with Linux-User Identity Mapping

Integration and unit tests for the `kaos-control devops` subcommand group and the
loopback-trusted local identity middleware. Tests cover Milestones 1–6 of the test plan.

## Test files

| File | Kind | Milestone |
|------|------|-----------|
| `internal/config/config_test.go` | Go unit (extended) | 1 |
| `internal/http/local_identity_test.go` | Go unit (new) | 2 |
| `tests/cli_devops_test.go` | Go integration (new) | 3, 4, 5 |
| `tests/manual/kaos-control-devops-cli-6-test.md` | Manual | 6 |

---

## Milestone 1 — Config mapping and identity helper (`config_test.go`)

Run with: `go test ./internal/config/ -run TestLinuxUser -v`

### Scenarios covered

- **linux_user round-trip** — `TestLinuxUserBinding_RoundTrip`: writes a project config
  with `linux_user: alice` and verifies the field survives `LoadProject`.
- **EmailForLinuxUser — bound** — `TestEmailForLinuxUser`: returns the bound email and
  `true` for a mapped username.
- **EmailForLinuxUser — unmapped** — `TestEmailForLinuxUser`: returns `("", false)` for a
  name not in any binding.
- **EmailForLinuxUser — empty input** — `TestEmailForLinuxUser`: returns `("", false)` for
  the empty string (NF3 guard).
- **Duplicate linux_user error** — `TestLinuxUserDuplicate_Error`: two bindings sharing the
  same `linux_user` value cause `LoadProject` to return an error naming the duplicate value.

---

## Milestone 2 — Loopback-trusted local identity (`local_identity_test.go`)

Run with: `go test ./internal/http/ -run TestLocalIdentity -v`

### Scenarios covered

- **Loopback + known email → authenticated** —
  `TestLocalIdentity_LoopbackKnownEmail_Authenticated`: 127.0.0.1 + valid email in
  X-Kaos-Local-User header → user is placed in context, handler returns 200.
- **IPv6 loopback (::1)** — `TestLocalIdentity_LoopbackIPv6_Authenticated`: same check for
  the `[::1]` loopback address.
- **Non-loopback → header ignored** — `TestLocalIdentity_NonLoopback_HeaderIgnored`: header
  is silently dropped when RemoteAddr is not loopback; caller stays unauthenticated (401).
- **Unknown email → unauthenticated** — `TestLocalIdentity_UnknownEmail_Unauthenticated`:
  an email not in the auth store is not elevated; caller stays unauthenticated (401).
- **Session wins over header** — `TestLocalIdentity_SessionWinsOverHeader`: a valid session
  cookie takes precedence over X-Kaos-Local-User (no silent elevation, NF3).
- **Bearer token wins over header** — `TestLocalIdentity_BearerWinsOverHeader`: a valid
  bearer token takes precedence over X-Kaos-Local-User (no silent elevation, NF3).

---

## Milestones 3–5 — CLI integration (`tests/cli_devops_test.go`)

Run with: `go test -tags integration ./tests/ -run TestDevops -v`

The test file starts a real server subprocess, registers a project with `linux_user`
bindings for the current OS user, seeds artifacts and pipeline fixtures, and pre-populates
the auth store.

### Milestone 3 — Help, project selection, output shape

- **Top-level help contains devops** — `TestDevops_TopLevelHelp_ContainsDevops`
- **devops --help lists subcommands and exit codes** —
  `TestDevops_SubcommandHelp_ListsOperations` (list, status, run; exit codes 0/1/3/4)
- **--project flag selects registered project** — `TestDevops_ProjectFlag_SelectsRegistered`
- **Unknown --project exits non-zero** — `TestDevops_ProjectFlag_UnknownErrors`
- **Cwd inference from project root** — `TestDevops_CwdInference_InProject`
- **Cwd outside project exits non-zero** — `TestDevops_CwdInference_NotInProject`
- **devops list --json emits valid JSON array** — `TestDevops_List_JSON`
- **--type filter narrows artifacts** — `TestDevops_List_FilterByType`
- **--json stdout parseable regardless of stderr** —
  `TestDevops_List_JSON_StdoutSeparatedFromStderr` (F12)

### Milestone 4 — Identity resolution

- **--token authenticates** — `TestDevops_Identity_BearerToken` (F6a)
- **KAOS_CONTROL_TOKEN env var authenticates** — `TestDevops_Identity_BearerTokenViaEnv`
- **Mapped linux_user resolves to email** — `TestDevops_Identity_MappedLinuxUser` (F6c):
  project config binds `linux_user: <test-runner-username>` to admin; no token supplied.
- **Unmapped linux_user exits 3** — `TestDevops_Identity_UnmappedExitsThree` (F7, NF3):
  project has no linux_user binding for the test runner; expects exit 3 and
  "identity not resolved" in stderr.
- **Token never in output** — `TestDevops_Identity_TokenNotInOutput` (NF2): token value
  absent from both stdout and stderr.

### Milestone 5 — Run, role gating, and parity

- **devops run prints run ID** — `TestDevops_Run_StartsAndPrintsRunID` (F4)
- **--follow streams log to completion** — `TestDevops_Run_Follow_StreamsLog` (F4)
- **Unknown task exits 1** — `TestDevops_Run_UnknownTask_ExitsOpFailed`
- **Insufficient role exits 4** — `TestDevops_Run_InsufficientRoleExitsFour` (F8)
- **Mapped linux user with insufficient role exits 4** —
  `TestDevops_Run_MappedLinuxUser_InsufficientRole` (F8 acceptance bullet)
- **Attribution via KC_API_TOKEN** — `TestDevops_Attribution_ViaKCToken` (F11): the
  `env-check` pipeline prints `TOKEN_LEN=<n>`; asserts n > 0, confirming the server issued
  a token tied to the resolved user. Full `RunRecord.TriggeredBy` tracking is a pending
  server enhancement.
- **CLI/HTTP authz parity** — `TestDevops_Parity_CLIAndHTTP`: qa role denied by both HTTP
  (403) and CLI (exit 4) for the same devops run operation.

### Known gaps

- **F9 allowed_write_paths for devops runs**: `handleRunPipeline` checks `requireRole`
  only; pipeline shell steps have no `allowed_write_paths` enforcement. This acceptance
  criterion cannot be integration-tested until the server enforces path restrictions for
  pipeline runs.
- **F11 RunRecord.TriggeredBy**: `RunRecord` has no `triggered_by` field; the run history
  endpoint does not return who triggered a run. Attribution is verified indirectly via the
  KC_API_TOKEN mechanism. Full run-history attribution requires a server-side enhancement.

---

## Milestone 6 — Frontend regression

Manual verification steps documented at:
`tests/manual/kaos-control-devops-cli-6-test.md`

Covers:
- Project users view renders `linux_user` for mapped bindings and cleanly handles absent
  values.
- CLI-triggered pipeline run appears in RunHistory.vue attributed to the resolved user.
- No browser console errors; `pnpm build` clean.
