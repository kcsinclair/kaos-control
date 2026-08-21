---
title: Embed the Vue SPA in the Go binary via embed.FS (single-binary distribution)
type: adr
status: approved
lineage: adr-embedded-spa-single-binary
created: "2026-08-21T11:46:00+10:00"
labels:
    - adr
    - architecture
    - frontend
    - distribution
related_to:
    - adr-0003-pure-go-sqlite-index
---

# ADR-0004: Embed the Vue SPA in the Go binary via embed.FS

## Context

kaos-control has two halves: a Go server (`cmd/`, `internal/`) and a Vue 3 SPA
(`web/`). The product promise is that a user runs **one binary** and gets the
whole tool — no separate web server, no static-asset hosting, no version-skew
between an API and a separately deployed frontend.

## Decision

Build the SPA (`make build-web` → `web/dist/`) and **embed it into the Go
binary with `embed.FS`**, served by the HTTP layer. The compiled frontend is a
build input to the binary, not a runtime dependency.

Consequently the frontend and backend **version together**: a given binary
always serves the exact SPA it was built with. `web/dist/` is git-ignored and
produced by the build (`make build-web` before `make build`); it is not the
source of truth.

## Consequences

- One artifact to ship, run, and roll back. The API and UI can never be
  mismatched at runtime.
- The build has an ordering constraint: `make build-web` must run before
  `make build` (the embed reads `web/dist/`). CI and the Makefile encode this.
- No CDN, no external asset host, no CORS between UI and API — the SPA is
  same-origin with the API it calls.
- Binary size carries the built frontend. Acceptable for a single-node,
  self-hosted tool; the alternative (separate deploys) trades that for
  operational and version-skew complexity we explicitly reject.
- Reinforces the pure-Go, `CGO_ENABLED=0` single-binary story
  ([[adr-0003-pure-go-sqlite-index]]).
