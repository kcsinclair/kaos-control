---
title: CLI Auth User Management — Integration & Unit Test Suite
type: test
status: draft
lineage: cli-auth-user-management
parent: lifecycle/test-plans/cli-auth-user-management-5-test.md
---

# CLI Auth User Management — Integration & Unit Test Suite

Tests for the `kaos-control auth` CLI subcommands, the `internal/auth` store, and the global HTTP authentication middleware. Covers requirements F1–F9 and NF1–NF5 from [[cli-auth-user-management-2]].

## Test Files

### `internal/auth/auth_test.go`

Package `auth_test`. Unit tests executed against a temporary SQLite DB (`t.TempDir()` per test). No `net/http`, `cmd/`, or project-level packages are imported (NF4).

Run with: `go test ./internal/auth/ -v`

### `tests/cli_auth_test.go`

Package `cli_test`, build tag `//go:build integration`. Integration tests invoking the compiled binary. `TestMain` (from `cli_init_test.go`) builds the binary once; each test sets `XDG_CONFIG_HOME` to an isolated temp dir with a `kaos-control/config.yaml` pointing `data_dir` at a separate temp dir.

Run with: `go test ./tests/ -tags integration -run TestAuth -v`

### `tests/auth_middleware_test.go`

Package `cli_test`, build tag `//go:build integration`. Integration tests that start the compiled binary as a subprocess on a random free port and exercise the HTTP auth/CSRF middleware layer via plain `net/http` requests.

Run with: `go test ./tests/ -tags integration -run 'TestUnauthenticated|TestSession|TestBearer|TestExpired|TestHealth|TestStatic|TestLogin|TestWebSocket|TestNoUser' -v`

### `tests/helpers_test.go`

Shared test helpers (`newBinCmd`, `runBin`) for the `cli_test` package.

## Scenarios Covered

### Milestone 1 — Auth Store: User CRUD (F1, F2, F3, F5, NF4)

| Test | Requirement |
|---|---|
| `TestCreateUser_AdminFlag` | F1 — create user with admin flag; retrieve and verify `Admin == true` | 
| `TestListUsers` | F3 — list users; assert all three present, fields populated |
| `TestDeleteUser` | F4 — delete user; assert `GetUser` and `GetSession` both return nil |
| `TestResetPassword` | F5 — reset password; old password fails, new succeeds |
| `TestCreateUser_DuplicateEmail` | F1 — duplicate email must return error |
| `TestSchemaIdempotency` | NF3 — `Open()` called twice on same file must not error |

### Milestone 2 — Auth Store: Bearer Tokens (F6, F7)

| Test | Requirement |
|---|---|
| `TestCreateToken` | F6 — token plaintext ≥64 hex characters |
| `TestValidateToken_Valid` | F7 — valid token resolves to correct user |
| `TestValidateToken_Invalid` | F7 — unknown token returns nil |
| `TestValidateToken_Expired` | F7 — expired token returns nil |
| `TestDeleteTokensForUser` | F7 — revoked tokens no longer validate |
| `TestDeleteUser_CascadesToTokens` | F4, F7 — user deletion cascades to tokens |

### Milestone 3 — CLI Subcommands (F1–F6, NF1)

| Test | Requirement |
|---|---|
| `TestAuthCreateUser` | F1 — exit 0, confirmation output contains email |
| `TestAuthCreateUser_DuplicateEmail` | F1 — exit 1, error message on duplicate |
| `TestAuthListUsers` | F3 — both emails and header row present in tabular output |
| `TestAuthDeleteUser` | F4 — deleted email absent from subsequent list-users |
| `TestAuthResetPassword` | F5 — new password authenticates via direct store call |
| `TestAuthCreateToken` | F6 — token printed to stdout, ≥64 chars |
| `TestAuthHelp` | NF1 — `auth --help` lists all five subcommands |
| `TestTopLevelHelp` | NF1 — top-level `--help` lists `serve`, `init`, `auth` |

### Milestone 4 — HTTP Auth Middleware (F8, F9, NF2, NF5)

| Test | Requirement |
|---|---|
| `TestUnauthenticatedRequest_Returns401` | F8 — 401 + `{"error":"unauthorized"}` body |
| `TestSessionCookieAuth_Returns200` | F8 — session cookie grants access |
| `TestBearerTokenAuth_Returns200` | F8 — bearer token grants access |
| `TestExpiredSession_Returns401` | F8 — expired session → 401 |
| `TestExpiredToken_Returns401` | F7, F8 — expired token → 401 |
| `TestHealthEndpoint_NoAuth` | F9 — `/api/health` exempt from auth |
| `TestStaticAssets_NoAuth` | F9 — `/`, `/index.html`, `/assets/*` exempt |
| `TestLoginEndpoint_NoAuth` | F9 — `/api/auth/login` exempt |
| `TestWebSocketAuth_Rejected` | F8 — unauthenticated WS upgrade → 401 |
| `TestBearerAuth_SkipsCsrf` | NF2 — bearer POST with no CSRF token must not return 403 |
| `TestSessionAuth_RequiresCsrf` | NF2 — session POST without CSRF token → 403 |

### Milestone 5 — No-User Startup Warning (NF5)

| Test | Requirement |
|---|---|
| `TestNoUserWarning` | NF5 — warning containing `auth create-user` logged at startup when no users exist |
| `TestNoWarningWithUsers` | NF5 — warning absent when ≥1 user exists at startup |

## Requirements Coverage Summary

| ID | Description | Tests |
|---|---|---|
| F1 | Create user (email, name, password, admin flag) | `TestCreateUser_AdminFlag`, `TestCreateUser_DuplicateEmail`, `TestAuthCreateUser`, `TestAuthCreateUser_DuplicateEmail` |
| F2 | List users ordered by created_at | `TestListUsers`, `TestAuthListUsers` |
| F3 | Delete user + cascade sessions/tokens | `TestDeleteUser`, `TestDeleteUser_CascadesToTokens`, `TestAuthDeleteUser` |
| F4 | Reset password | `TestResetPassword`, `TestAuthResetPassword` |
| F5 | Authenticate (email + password) | `TestResetPassword`, `TestAuthResetPassword` |
| F6 | Create bearer token | `TestCreateToken`, `TestAuthCreateToken` |
| F7 | Validate/revoke bearer tokens | `TestValidateToken_Valid`, `TestValidateToken_Invalid`, `TestValidateToken_Expired`, `TestDeleteTokensForUser`, `TestDeleteUser_CascadesToTokens`, `TestExpiredToken_Returns401` |
| F8 | Auth middleware (401 / session / bearer) | `TestUnauthenticated*`, `TestSession*`, `TestBearer*`, `TestExpired*`, `TestWebSocketAuth_Rejected` |
| F9 | Exempt endpoints | `TestHealthEndpoint_NoAuth`, `TestStaticAssets_NoAuth`, `TestLoginEndpoint_NoAuth` |
| NF1 | CLI help text | `TestAuthHelp`, `TestTopLevelHelp` |
| NF2 | CSRF enforcement / bearer exemption | `TestBearerAuth_SkipsCsrf`, `TestSessionAuth_RequiresCsrf` |
| NF3 | Schema idempotency | `TestSchemaIdempotency` |
| NF4 | Unit tests import no http/cmd packages | `internal/auth/auth_test.go` (verified by import list) |
| NF5 | No-user startup warning | `TestNoUserWarning`, `TestNoWarningWithUsers` |
