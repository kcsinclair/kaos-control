---
title: CLI Auth User Management and Secured API — Test Plan
type: plan-test
status: in-development
lineage: cli-auth-user-management
parent: lifecycle/requirements/cli-auth-user-management-2.md
release: KC-Release0
assignees:
  - role: test-developer
    who: agent
---

# CLI Auth User Management and Secured API — Test Plan

Integration and unit tests for the `kaos-control auth` CLI subcommands ([[cli-auth-user-management-3-be]]) and the global authentication middleware. Tests exercise the auth store directly, the CLI binary, and the HTTP API. Frontend-specific auth flows ([[cli-auth-user-management-4-fe]]) are covered at the API boundary level — the SPA itself is verified manually.

## Milestone 1: Auth Store Unit Tests — User CRUD

### Description

Unit tests for the new `ListUsers`, `DeleteUser`, and `ResetPassword` methods, plus the extended `CreateUser` signature with the `admin` flag. These run in-process against a temporary SQLite DB.

### Files to change

- **`internal/auth/auth_test.go`** (extend existing test file)
  - `TestCreateUser_AdminFlag` — create a user with `admin=true`, retrieve via `GetUser`, assert `Admin == true`.
  - `TestListUsers` — create 3 users, call `ListUsers`, assert order by `created_at` and all fields populated.
  - `TestDeleteUser` — create a user and a session, delete the user, assert `GetUser` returns nil and `GetSession` returns nil.
  - `TestResetPassword` — create a user, reset password, assert old password fails `Authenticate` and new password succeeds.
  - `TestCreateUser_DuplicateEmail` — create the same email twice, assert second call returns error.
  - `TestSchemaIdempotency` — call `Open()` twice on the same DB file, assert no error.

### Acceptance criteria

- [ ] All tests pass with `go test ./internal/auth/ -v`.
- [ ] Tests use `t.TempDir()` for DB isolation — no shared state between tests.
- [ ] No test imports `internal/http` or `cmd/` packages (NF4 verification).

## Milestone 2: Auth Store Unit Tests — Bearer Tokens

### Description

Unit tests for token creation, validation, and revocation.

### Files to change

- **`internal/auth/auth_test.go`** (extend)
  - `TestCreateToken` — create a token, assert plaintext is ≥64 hex characters.
  - `TestValidateToken_Valid` — create a token, validate it, assert correct user returned.
  - `TestValidateToken_Invalid` — validate a random string, assert nil returned.
  - `TestValidateToken_Expired` — create a token with `expires` 1 second in the past, assert nil returned.
  - `TestDeleteTokensForUser` — create 2 tokens for a user, delete all, assert neither validates.
  - `TestDeleteUser_CascadesToTokens` — create user + token, delete user, assert token no longer validates.

### Acceptance criteria

- [ ] All token tests pass.
- [ ] Token plaintext is never stored in the DB (inspect the database rows in test if feasible).

## Milestone 3: CLI Subcommand Integration Tests

### Description

End-to-end tests that build the binary and invoke `kaos-control auth` subcommands against a temporary auth DB. These verify the full CLI path from argument parsing through store operations to output formatting.

### Files to change

- **`tests/cli_auth_test.go`** (new file)
  - Build the binary once in `TestMain` using `go build` to a temp directory.
  - `TestAuthCreateUser` — run `kaos-control auth create-user --email test@test.com --name Test --password-stdin --config <tmpconfig>` with password piped to stdin. Assert exit 0 and confirmation output.
  - `TestAuthCreateUser_DuplicateEmail` — create the same user twice. Assert exit 1 and error message contains "already exists" or similar.
  - `TestAuthListUsers` — create 2 users, run `list-users`, assert output contains both emails in a tabular format.
  - `TestAuthDeleteUser` — create a user, delete them, run `list-users`, assert email is absent.
  - `TestAuthResetPassword` — create a user, reset password via CLI, then authenticate programmatically with the new password to confirm it works.
  - `TestAuthCreateToken` — create a user and a token, assert token is printed to stdout and is non-empty.
  - `TestAuthHelp` — run `kaos-control auth --help`, assert output lists all subcommands.
  - `TestTopLevelHelp` — run `kaos-control --help`, assert output includes `auth` with a synopsis.

  Each test creates a temporary config file pointing `data_dir` at `t.TempDir()` so tests are isolated.

### Acceptance criteria

- [ ] All CLI tests pass with `go test ./tests/ -run TestAuth -v`.
- [ ] Tests do not require a running HTTP server.
- [ ] Each test is isolated (own temp directory, own auth DB).
- [ ] `TestTopLevelHelp` output includes `serve`, `init`, and `auth` subcommands.

## Milestone 4: HTTP Auth Middleware Integration Tests

### Description

Integration tests that start the HTTP server and verify the global auth middleware correctly rejects unauthenticated requests and accepts authenticated ones (session cookies and bearer tokens).

### Files to change

- **`tests/auth_middleware_test.go`** (new file)
  - Use `httptest.Server` or start the real server on a random port with a temp config.
  - Set up: create a user and obtain a session cookie via `POST /api/auth/login`. Create a bearer token via the auth store directly.

  - `TestUnauthenticatedRequest_Returns401` — `GET /api/p/{project}/artifacts` with no credentials. Assert `401` and body `{"error":"unauthorized"}`.
  - `TestSessionCookieAuth_Returns200` — same request with the session cookie. Assert `200`.
  - `TestBearerTokenAuth_Returns200` — same request with `Authorization: Bearer <token>`. Assert `200`.
  - `TestExpiredSession_Returns401` — create a session, manually expire it in the DB, assert `401`.
  - `TestExpiredToken_Returns401` — create an expired token, assert `401`.
  - `TestHealthEndpoint_NoAuth` — `GET /health` without credentials. Assert `200`.
  - `TestStaticAssets_NoAuth` — `GET /`, `GET /index.html`, `GET /assets/somefile` without credentials. Assert these do not return `401`.
  - `TestLoginEndpoint_NoAuth` — `POST /api/auth/login` without credentials (but with valid body). Assert not `401` (may be `200` or `400` depending on credentials validity, but never `401` from middleware).
  - `TestWebSocketAuth_Rejected` — attempt WS upgrade to `/api/p/{project}/ws` without credentials. Assert the upgrade is rejected.
  - `TestBearerAuth_SkipsCsrf` — `POST` a mutating endpoint with bearer token and no `X-CSRF-Token`. Assert success (not a CSRF error).
  - `TestSessionAuth_RequiresCsrf` — `POST` a mutating endpoint with session cookie but no `X-CSRF-Token`. Assert `403` (CSRF failure).

### Acceptance criteria

- [ ] All middleware tests pass.
- [ ] Tests cover both cookie and bearer auth paths.
- [ ] Exempt endpoints are verified accessible without auth.
- [ ] CSRF enforcement is verified for session auth and skipped for bearer auth.

## Milestone 5: No-User Startup Warning Test

### Description

Verify that the server emits a warning log when no users exist in the auth DB at startup.

### Files to change

- **`tests/auth_middleware_test.go`** (extend)
  - `TestNoUserWarning` — start the server with an empty auth DB, capture log output (via a `slog.Handler` that writes to a buffer, or by reading stderr). Assert the log contains the `kaos-control auth create-user` command string.
  - `TestNoWarningWithUsers` — create a user before starting the server, assert the warning is absent.

### Acceptance criteria

- [ ] Warning is logged when auth DB has zero users.
- [ ] Warning is not logged when auth DB has ≥1 user.
- [ ] Warning text includes the exact `auth create-user` command.

## Milestone 6: Lifecycle Test Artifact

### Description

Create the test artifact in `lifecycle/tests/` that describes what the test code in `tests/` covers for this feature, following the project convention.

### Files to change

- **`lifecycle/tests/cli-auth-user-management-6-test.md`** (new file)
  - Frontmatter: `type: test`, `status: draft`, `lineage: cli-auth-user-management`, `parent: lifecycle/test-plans/cli-auth-user-management-5-test.md`.
  - Body: list each test file and the scenarios it covers, mapping back to requirement IDs (F1–F9, NF1–NF5).

### Acceptance criteria

- [ ] Artifact exists with correct frontmatter.
- [ ] Every requirement ID from the spec is mapped to at least one test.

## Cross-references

- [[cli-auth-user-management-3-be]] — Backend plan: all store methods and middleware being tested.
- [[cli-auth-user-management-4-fe]] — Frontend plan: 401 interceptor and login redirect (tested at API boundary in Milestone 4).
- [[cli-init-scaffold]] — CLI subcommand pattern; similar CLI integration tests may exist there as a reference.
