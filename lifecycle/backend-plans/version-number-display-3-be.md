---
title: "Backend Plan — Version Number Display"
type: plan-backend
status: done
lineage: version-number-display
parent: lifecycle/requirements/version-number-display-2.md
created: "2026-05-10T00:00:00+10:00"
---

# Backend Plan — Version Number Display

This plan implements the server-side changes for [[version-number-display]]: a `VERSION` file as the single source of truth, a dedicated `GET /api/version` endpoint, and updated build integration.

## Milestone 1 — VERSION file and build integration

### Description

Create a `VERSION` file at the repository root containing `0.1.0`. Update the Makefile so that `make build` reads this file and injects its contents via `-ldflags` instead of relying on `git describe`. The existing `cmd/kaos-control/main.go` version variable remains the linker target; `internal/http.Version` continues to be set from it.

### Files to change

- `VERSION` (new) — single line: `0.1.0`
- `Makefile` — change `VERSION ?= $(shell git describe ...)` to `VERSION ?= $(shell cat VERSION)` so the file is authoritative. Keep the `?=` so CI can still override.

### Acceptance criteria

- [ ] `VERSION` exists at the repo root, contains exactly `0.1.0`, no leading `v`.
- [ ] `make build` produces a binary where `/api/health` returns `"version": "0.1.0"`.
- [ ] Running `go run ./cmd/kaos-control` without ldflags still returns `"version": "dev"`.
- [ ] No new Go modules introduced.

## Milestone 2 — GET /api/version endpoint

### Description

Add a public (unauthenticated) `GET /api/version` endpoint that returns `{"version": "<value>"}`. Register it in the router alongside `/api/health`, outside any auth middleware group.

### Files to change

- `internal/http/server.go`
  - Add `handleVersion()` handler (~5 lines) that writes `{"version": Version}` with `Content-Type: application/json`.
  - Register `r.Get("/api/version", s.handleVersion)` next to the existing `/api/health` route (around line 83), ensuring it is outside the authenticated route group.

### Acceptance criteria

- [ ] `GET /api/version` returns HTTP 200 with body `{"version": "0.1.0"}` (or `"dev"` in dev mode).
- [ ] The endpoint requires no authentication (no session cookie needed).
- [ ] `GET /api/health` continues to work and returns the same version value.
- [ ] Response time under 5 ms (static string, no DB or I/O).
- [ ] No new Go modules introduced.

## Milestone 3 — Ensure /health parity

### Description

Verify that `/api/health` and `/api/version` return the same version value. Both already read from `internal/http.Version`, so this milestone is a validation step — confirm by inspection and by the [[version-number-display]] test plan.

### Files to change

- None expected. If the two endpoints diverge, fix whichever is inconsistent.

### Acceptance criteria

- [ ] `/api/health` response includes `"version"` with the same value as `/api/version`.
- [ ] Confirmed by integration tests defined in the [[version-number-display]] test plan.
