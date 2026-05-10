---
title: CLI Auth User Management and Secured API — Backend Plan
type: plan-backend
status: draft
lineage: cli-auth-user-management
parent: lifecycle/requirements/cli-auth-user-management-2.md
release: KC-Release0
assignees:
  - role: backend-developer
    who: agent
---

# CLI Auth User Management and Secured API — Backend Plan

Implements the `kaos-control auth` CLI subcommand family and hardens the HTTP server so that every non-exempt endpoint requires authentication via session cookie or bearer token. Builds on the existing `internal/auth` package (argon2id hashing, sessions, SQLite store) and the manual subcommand dispatch pattern established by [[cli-init-scaffold]].

## Milestone 1: Extend Auth Store — User CRUD and Schema Migration

### Description

Add the `admin` column to the `users` table, and implement `ListUsers`, `DeleteUser`, and `ResetPassword` on `auth.Store`. The `admin` flag is forward-looking (stored but not gated) per resolved question 3 in the requirement.

### Files to change

- **`internal/auth/auth.go`**
  - Alter DDL: add `admin INTEGER NOT NULL DEFAULT 0` to `users` table. Add `ALTER TABLE users ADD COLUMN admin INTEGER NOT NULL DEFAULT 0` as a migration step executed in `Open()` (ignore "duplicate column" error for idempotency).
  - Extend `User` struct: add `Admin bool` field with JSON tag `"admin"`.
  - Extend `CreateUser` signature to `CreateUser(email, displayName, password string, admin bool) error`. Update INSERT to include admin column.
  - Add `func (s *Store) ListUsers() ([]User, error)` — `SELECT email, display_name, admin, created_at FROM users ORDER BY created_at`.
  - Add `func (s *Store) DeleteUser(email string) error` — `DELETE FROM users WHERE email = ?` then `DELETE FROM sessions WHERE user_email = ?` (also delete tokens once the token table exists — see Milestone 2).
  - Add `func (s *Store) ResetPassword(email, newPassword string) error` — hash new password, `UPDATE users SET password_hash = ? WHERE email = ?`, return error if no row matched.
  - Update `GetUser` and `GetSession` queries to also select `admin`.

### Acceptance criteria

- [ ] `CreateUser` inserts a row with the correct `admin` value.
- [ ] `ListUsers` returns all users ordered by `created_at`.
- [ ] `DeleteUser` removes the user row and all their sessions.
- [ ] `ResetPassword` updates the hash; old password fails `Authenticate`, new one succeeds.
- [ ] Schema migration is idempotent — calling `Open()` twice does not error.
- [ ] `make test-unit` passes for `internal/auth`.

## Milestone 2: Bearer Token Storage and Validation

### Description

Add a `tokens` table and the store methods to create, validate, and revoke bearer tokens (requirements F8, NF2). Tokens are cryptographically random (≥32 bytes), stored as argon2id hashes. Each token has an optional expiry.

### Files to change

- **`internal/auth/auth.go`**
  - Add DDL for `tokens` table:
    ```sql
    CREATE TABLE IF NOT EXISTS tokens (
        id         TEXT PRIMARY KEY,
        user_email TEXT NOT NULL,
        token_hash TEXT NOT NULL,
        expires_at INTEGER,          -- NULL = no expiry
        created_at INTEGER NOT NULL
    );
    CREATE INDEX IF NOT EXISTS idx_tokens_user ON tokens(user_email);
    ```
  - Add `func (s *Store) CreateToken(userEmail string, expires *time.Time) (plaintext string, err error)`:
    - Generate 32 random bytes → hex-encode as the plaintext token.
    - Hash with `hashPassword` (reuses argon2id).
    - Generate a short random ID for the `id` PK.
    - INSERT into `tokens`.
    - Return the plaintext (only time it is available).
  - Add `func (s *Store) ValidateToken(plaintext string) (*User, error)`:
    - Scan all non-expired token rows (or add a prefix/hint column to avoid full scan — see NF3 perf note below).
    - For each, `verifyPassword(plaintext, token_hash)`. On match, return the associated `User`.
    - If expired or no match, return nil.
  - Add `func (s *Store) DeleteTokensForUser(email string) error` — `DELETE FROM tokens WHERE user_email = ?`.
  - Update `DeleteUser` to also call `DeleteTokensForUser`.

**Performance note (NF3):** argon2id verification per-token is expensive if there are many tokens. To keep p99 ≤1 ms, store a non-secret **token prefix** (first 8 hex chars) in a `prefix` column and filter candidates by prefix before running argon2id. This limits verification to O(1) rows in practice.

### Acceptance criteria

- [ ] `CreateToken` returns a hex string ≥64 characters (32 bytes).
- [ ] `ValidateToken` with the returned plaintext resolves to the correct user.
- [ ] `ValidateToken` with a wrong token returns nil.
- [ ] `ValidateToken` with an expired token returns nil.
- [ ] `DeleteTokensForUser` invalidates all tokens for that user.
- [ ] Token plaintext is never stored in the database (only hash + prefix).

## Milestone 3: Auth CLI Subcommand Package

### Description

Create `cmd/kaos-control/authcmd/` following the pattern of `cmd/kaos-control/initcmd/`. Each CLI command opens the auth DB directly (requirement NF4 — no HTTP server dependency), performs its operation, and exits.

### Files to change

- **`cmd/kaos-control/authcmd/authcmd.go`** (new file)
  - `func Run(args []string) int` — dispatches to sub-subcommands: `create-user`, `list-users`, `delete-user`, `reset-password`, `create-token`.
  - Prints usage on `--help` or unknown subcommand.
  - Resolves config path (reuse `defaultConfigPath()` logic or accept `--config` flag) to derive `DataDir` and open the auth DB.

- **`cmd/kaos-control/authcmd/create_user.go`** (new file)
  - Parses `--email`, `--name`, `--admin` flags.
  - Reads password: if `--password-stdin` flag is set, reads from `os.Stdin`; otherwise prompts interactively using `term.ReadPassword` (from `golang.org/x/term`).
  - Calls `store.CreateUser(...)`.
  - Prints success message with email; exits 1 with clear error on duplicate email.

- **`cmd/kaos-control/authcmd/list_users.go`** (new file)
  - Calls `store.ListUsers()`.
  - Prints a formatted table: `EMAIL | DISPLAY NAME | ADMIN | CREATED AT`.
  - Uses `text/tabwriter` for alignment.

- **`cmd/kaos-control/authcmd/delete_user.go`** (new file)
  - Parses `--email` flag (required).
  - Calls `store.DeleteUser(...)`.
  - Prints confirmation.

- **`cmd/kaos-control/authcmd/reset_password.go`** (new file)
  - Parses `--email` flag.
  - Prompts for new password (same logic as create-user).
  - Calls `store.ResetPassword(...)`.

- **`cmd/kaos-control/authcmd/create_token.go`** (new file)
  - Parses `--email` and `--expires` (duration string, e.g., `720h` for 30 days) flags.
  - Calls `store.CreateToken(...)`.
  - Prints the plaintext token to stdout once with a warning that it cannot be recovered.

### Acceptance criteria

- [ ] `kaos-control auth create-user --email a@b.com --name "Alice" --password-stdin` creates a user (pipe password via echo).
- [ ] `kaos-control auth create-user` with duplicate email exits 1 with descriptive error.
- [ ] `kaos-control auth list-users` displays a table of all users.
- [ ] `kaos-control auth delete-user --email a@b.com` removes the user and sessions/tokens.
- [ ] `kaos-control auth reset-password --email a@b.com` updates the password.
- [ ] `kaos-control auth create-token --email a@b.com --expires 720h` prints a bearer token.
- [ ] All commands work without the HTTP server running (NF4).
- [ ] `make lint` passes.

## Milestone 4: Wire Auth Subcommand into Main and Update Help

### Description

Register the `auth` subcommand in `main.go`'s dispatch logic and ensure `--help` lists all top-level subcommands with synopses (requirements F9).

### Files to change

- **`cmd/kaos-control/main.go`**
  - Add case for `os.Args[1] == "auth"` → `os.Exit(authcmd.Run(os.Args[2:]))`.
  - Refactor the usage/help output to list all subcommands with one-line descriptions:
    ```
    Usage: kaos-control <command> [flags]

    Commands:
      serve    Start the HTTP server (default)
      init     Initialise a new project directory
      auth     Manage users, passwords, and API tokens

    Run 'kaos-control <command> --help' for command-specific usage.
    ```

### Acceptance criteria

- [ ] `kaos-control auth --help` prints auth subcommand usage.
- [ ] `kaos-control --help` lists `serve`, `init`, and `auth` with synopses.
- [ ] Unknown subcommands print usage and exit 1.

## Milestone 5: Global Auth Middleware with Bearer Token Support

### Description

Harden the chi router so that all non-exempt endpoints require authentication (requirement F6). Extend the existing `sessionMiddleware` to also check `Authorization: Bearer <token>` headers. Apply `requireAuth` globally instead of per-route-group, with explicit exemptions for the paths listed in F7.

### Files to change

- **`internal/http/auth.go`**
  - Extend `sessionMiddleware`: after checking the session cookie, if no user was found, check for an `Authorization` header with scheme `Bearer`. If present, call `store.ValidateToken(token)`. If valid, inject the `*auth.User` into context.
  - Bearer-authenticated requests skip CSRF enforcement (tokens are not vulnerable to CSRF).

- **`internal/http/server.go`**
  - Move `requireAuth` from per-group `r.With(requireAuth)` to a global middleware applied after `sessionMiddleware`.
  - Add an exemption list inside `requireAuth`:
    - `POST /api/auth/login`
    - `GET /health`
    - Static asset paths: `/`, `/index.html`, `/favicon.ico`, `/assets/*`
    - `POST /api/admin/users` when `UserCount() == 0` (bootstrap — already exists).
  - Remove the per-route `r.With(requireAuth)` wrappers that are now redundant.
  - Ensure the 401 response body is `{"error":"unauthorized"}` (matching requirement AC).

- **`internal/http/server.go`** (WebSocket upgrade)
  - Ensure WebSocket upgrade requests at `/api/p/{project}/ws` also pass through the global auth middleware. The existing `sessionMiddleware` already runs before the WS handler; confirm bearer tokens work too.

### Acceptance criteria

- [ ] `GET /api/artifacts` without credentials returns `401` with `{"error":"unauthorized"}`.
- [ ] `GET /api/artifacts` with valid session cookie returns `200`.
- [ ] `GET /api/artifacts` with valid `Authorization: Bearer <token>` returns `200`.
- [ ] `GET /health` returns `200` without credentials.
- [ ] Static SPA assets (`/`, `/assets/main.js`, etc.) are served without credentials.
- [ ] `POST /api/auth/login` is accessible without credentials.
- [ ] Bearer-authenticated requests bypass CSRF checks.
- [ ] WebSocket connections require authentication.

## Milestone 6: No-User Startup Warning

### Description

At server startup, after opening the auth DB, check `store.UserCount()`. If zero, log a warning with the exact CLI command to create the first user (requirement F5).

### Files to change

- **`cmd/kaos-control/main.go`** (in `run()`)
  - After `auth.Open(...)`, call `store.UserCount()`.
  - If count == 0, log at `slog.Warn` level:
    ```
    No users found. Create the first admin user with:
      kaos-control auth create-user --email <email> --name <name> --admin
    ```

### Acceptance criteria

- [ ] Starting the server with an empty auth DB logs the warning at WARN level.
- [ ] Starting the server with ≥1 user does not log the warning.

## Milestone 7: Session TTL Default Update

### Description

Update the default `SessionTTL` from 24 hours to 30 days (720h) per resolved question 2 in the requirement.

### Files to change

- **`internal/config/config.go`**
  - Change default `SessionTTL` from `24 * time.Hour` to `30 * 24 * time.Hour`.

### Acceptance criteria

- [ ] Default config (no config file) uses 30-day session TTL.
- [ ] Explicit `session_ttl` in config file overrides the default.

## Cross-references

- [[cli-auth-user-management-4-fe]] — Frontend plan: login page 401 handling, CSRF flow adjustments.
- [[cli-auth-user-management-5-test]] — Test plan: integration tests for CLI commands and middleware.
- [[cli-init-scaffold]] — Established the subcommand dispatch pattern reused here.
