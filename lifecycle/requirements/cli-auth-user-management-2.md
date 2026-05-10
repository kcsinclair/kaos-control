---
title: CLI Auth User Management and Secured API
type: requirement
status: approved
lineage: cli-auth-user-management
created: "2026-05-10"
priority: high
parent: lifecycle/ideas/cli-auth-user-management.md
labels:
    - feature
    - backend
    - security
    - go
    - onboarding
    - operability
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

# CLI Auth User Management and Secured API

## Problem

The kaos-control server currently exposes its REST API without requiring authentication, meaning any network-reachable client can read and mutate project data. There is also no way to create the first user account without an already-running and already-authenticated server — a bootstrap deadlock. Operators need an offline, CLI-based mechanism to provision users and a server-side enforcement layer that rejects unauthenticated requests.

## Goals / Non-goals

### Goals

1. Provide a `kaos-control auth` CLI subcommand that can create, list, and delete user accounts directly against the auth SQLite database — without requiring the HTTP server to be running.
2. Support a first-run "bootstrap admin" flow so that initial deployment is self-contained.
3. Secure every REST and WebSocket endpoint behind authentication (session cookie or bearer token), rejecting unauthenticated requests with `401 Unauthorized`.
4. Ensure `kaos-control --help` discovers and documents `auth` (and all other top-level subcommands) with a one-line synopsis.

### Non-goals

- OAuth / OIDC / SSO federation (future work).
- Role-based access control beyond the single "authenticated user" gate (RBAC is a separate lineage).
- A web-based user-management UI (may follow later).
- Password-reset or email-verification flows.

## Detailed Requirements

### Functional

| ID | Requirement |
|----|-------------|
| F1 | `kaos-control auth create-user --email <email> --name <display> [--admin]` prompts interactively for a password (or accepts `--password-stdin`), hashes it with argon2id, and inserts a row into the auth store. |
| F2 | `kaos-control auth list-users` prints a table of email, display name, admin flag, and created-at for all accounts. |
| F3 | `kaos-control auth delete-user --email <email>` removes the account and invalidates all associated sessions. |
| F4 | `kaos-control auth reset-password --email <email>` prompts for a new password and updates the stored hash. |
| F5 | If no users exist in the auth store at server startup, the server logs a warning with the exact `auth create-user` command to run. |
| F6 | An HTTP middleware on the chi router rejects any request lacking a valid session cookie or `Authorization: Bearer <token>` header with HTTP 401 and a JSON error body. |
| F7 | The middleware exempts only: `POST /api/login`, `GET /health`, and static asset paths (`/assets/*`, `/favicon.ico`, `/index.html`, `/`). |
| F8 | Bearer tokens are generated per-user via `kaos-control auth create-token --email <email> [--expires <duration>]` and stored hashed in the auth DB. |
| F9 | `kaos-control --help` lists all top-level subcommands (`serve`, `auth`, `init`, etc.) with a one-line description for each. |

### Non-functional

| ID | Requirement |
|----|-------------|
| NF1 | Password hashing uses argon2id with the parameters already defined in `internal/auth` (time=2, mem=64 MB, threads=4, keyLen=32). |
| NF2 | Bearer tokens are cryptographically random (≥32 bytes), stored as argon2id hashes — never in plaintext. |
| NF3 | Auth middleware adds ≤1 ms p99 latency to authenticated requests (single SQLite lookup or in-memory session cache). |
| NF4 | CLI auth commands work against the auth DB file without importing or starting the HTTP server or watcher packages. |
| NF5 | All new code must pass `make lint` and `make test-unit`. |

## Acceptance Criteria

- [ ] Running `kaos-control auth create-user` with valid flags creates a user retrievable by `auth list-users`.
- [ ] Running `kaos-control auth create-user` with a duplicate email returns a non-zero exit code and a clear error message.
- [ ] Running `kaos-control auth delete-user` removes the user and their active sessions/tokens.
- [ ] Running `kaos-control auth reset-password` updates the hash; old password no longer authenticates.
- [ ] Running `kaos-control auth create-token` prints a token that authenticates API requests via `Authorization: Bearer <token>`.
- [ ] An unauthenticated `GET /api/artifacts` returns HTTP 401 with `{"error":"unauthorized"}`.
- [ ] An authenticated request (valid session cookie) to `GET /api/artifacts` succeeds (HTTP 200).
- [ ] An authenticated request (valid bearer token) to `GET /api/artifacts` succeeds (HTTP 200).
- [ ] `GET /health` remains accessible without credentials.
- [ ] Static SPA assets remain accessible without credentials.
- [ ] `kaos-control --help` output includes an `auth` line with synopsis.
- [ ] Integration tests in `tests/` cover the happy-path and rejection scenarios above.
- [ ] Related: [[cli-init-scaffold]] (shares CLI subcommand infrastructure).

## Resolved Questions

1. Should bearer tokens support scoping (read-only vs read-write) in this iteration, or is a single "full access" scope sufficient for now?

> Full access for now.

2. What is the desired session/token TTL default? (Current `SessionTTL` field exists but its default is not specified in the idea.)

> 1 month

3. Should the `--admin` flag on `create-user` gate any behaviour now, or is it purely a forward-looking field for future RBAC?

> forward-looking.
