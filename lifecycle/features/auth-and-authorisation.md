---
title: Auth & authorisation
type: feature
status: approved
lineage: feature-auth-and-authorisation
created: "2026-08-21T15:14:00+10:00"
summary: Argon2id local accounts with session-cookie + CSRF protection, and per-project role bindings that gate workflow transitions.
function: Auth & access
labels:
    - feature
    - auth
related_to:
    - lifecycle/ideas/auth-role-checks-mutations.md
    - lifecycle/requirements/auth-role-checks-mutations-2.md
    - lifecycle/ideas/cli-auth-user-management.md
    - lifecycle/requirements/cli-auth-user-management-2.md
    - lifecycle/ideas/websocket-origin-check.md
    - lifecycle/requirements/websocket-origin-check-2.md
---

# Auth & authorisation

Local accounts, sessions, and per-project role bindings that the workflow
state machine and agent write-scoping both depend on.

## What it does

- **Local accounts.** Argon2id-hashed passwords in
  `~/.kaos-control/data/auth.db`.
- **Bootstrap-friendly.** First user can be created without authentication;
  subsequent users require a logged-in session.
- **Session cookies + CSRF.** HTTP-only session cookie + matching CSRF token
  on every mutating request. WebSocket connections are origin-checked.
- **CLI user management.** Create/manage users from the `kaos-control auth`
  CLI subcommand, independent of a running server.
- **Per-project role bindings.** `users:` block in `lifecycle/config.yaml`
  binds each email to a list of roles (`product-owner`, `analyst`,
  `backend-developer`, `frontend-developer`, `test-developer`, `qa`,
  `tech-writer`, `reviewer`, `approver`, `devops`).

See also [[workflow-state-machine]] for how roles gate transitions, and
[[agent-permission-mediation]] for how agents are bounded independently of
user roles.
