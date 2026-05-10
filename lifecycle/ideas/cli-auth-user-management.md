---
title: CLI Auth User Management and Secured API
type: idea
status: draft
lineage: cli-auth-user-management
created: "2026-05-10T16:14:11+10:00"
priority: normal
labels:
    - feature
    - backend
    - security
    - go
    - onboarding
    - operability
---

# CLI Auth User Management and Secured API

Add a `kaos-control auth` subcommand to bootstrap and manage users from the command line, covering operations such as creating the initial admin user and adding additional users without requiring the server to be running. This makes first-time setup self-contained and removes any dependency on a pre-existing authenticated session to create the first account.

Secure all REST API endpoints to require either a valid session token (from a login flow) or a bearer token, so that unauthenticated access is rejected by default. This closes the gap where the HTTP API is currently reachable without credentials, which is a prerequisite for any production or multi-user deployment.

Ensure `kaos-control --help` enumerates all top-level subcommands — including `auth` and any others — with a brief description of each, so operators can discover available functionality without consulting external documentation.
