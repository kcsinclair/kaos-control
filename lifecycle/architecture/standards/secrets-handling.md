---
title: Secrets handling
type: doc
status: approved
lineage: standard-secrets-handling
created: "2026-08-21T11:51:00+10:00"
labels:
    - standard
    - security
    - secrets
---

# Standard: Secrets handling

Secrets — agent auth tokens, API keys, password hashes — must never leak into
API responses, logs, the index, or the frontend.

## Rules

- **Never return secrets from the API.** Agent auth tokens are excluded from
  `GET /agents` responses; only non-secret fields (e.g. a driver `base_url`) are
  exposed. New fields default to *not* serialised unless proven non-secret.
- **Passwords are hashed with argon2id** (`internal/auth/`), never stored or
  compared in plaintext. Sessions are opaque tokens.
- **Keep secrets out of logs.** Do not log tokens, keys, or full auth headers.
  Redact before logging request/agent detail.
- **Secrets are config/runtime inputs, not artifacts.** Do not write secrets into
  `lifecycle/` markdown, ADRs, or standards, and do not embed them in the
  binary. App config lives outside the repo (`~/.kaos-control/`).
- **Frontend never receives secret material.** If the UI needs to show that a
  credential exists, send a boolean/label, not the value.
- **Local trusted-auth is loopback-gated and spoof-resistant** — see
  [[adr-0001-no-header-based-client-ip-trust]]. Do not reintroduce header-derived
  client identity.
