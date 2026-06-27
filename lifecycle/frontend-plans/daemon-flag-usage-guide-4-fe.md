---
title: "Frontend Plan: Default to Usage Guide; Require -d/--daemon to Start the Server"
type: plan-frontend
status: in-development
lineage: daemon-flag-usage-guide
parent: lifecycle/requirements/daemon-flag-usage-guide-2.md
---

# Frontend Plan: Default to Usage Guide; Require -d/--daemon to Start the Server

Implements the frontend slice of [[daemon-flag-usage-guide]] (requirement
`daemon-flag-usage-guide-2`).

## Scope: no SPA code changes required

This requirement is a **CLI/process-startup** change, entirely contained in the
Go entry point `cmd/kaos-control/main.go` (see the backend plan
[[daemon-flag-usage-guide]] `-3-be`). It changes **how the binary is invoked**,
not anything the embedded Vue SPA renders or requests.

Evidence gathered while planning:

- The SPA contains no reference to how the server is started — a search of
  `web/src` for `kaos-control serve`, `-d`, `--daemon`, "usage guide", or
  "daemon" returns nothing. The SPA never tells a user to run the binary.
- The usage guide (F2) is printed by the Go process to a terminal stream
  (stdout/stderr). It is **not** an HTTP response and not surfaced in the SPA.
- Once the server is started (via `-d`/`--daemon`/`serve`), it serves the
  **identical** embedded SPA as before — `run()`'s server body is unchanged, so
  every existing frontend API contract (including `GET` for version via
  `web/src/api/version.ts` → `web/src/stores/app.ts`) is unaffected.

Therefore there is **no Vue/TypeScript code to add or modify**. The honest
deliverable for this slice is *verification that the SPA is unaffected*, plus a
guard against the one realistic regression: a developer changing the dev-server
launch path and breaking the loop that builds and serves the SPA.

## Milestone 1 — Verify the SPA is served unchanged in daemon mode

### Description

Confirm that starting the server the new way (`kaos-control -d`, and
`kaos-control serve`) serves the embedded SPA exactly as a pre-change build did:
the app loads, the router mounts, and the version banner (the one piece of
top-level metadata the SPA fetches) renders. No code change — this is an
explicit regression checkpoint so the slice is not silently skipped.

### Files to change

- None. (`web/src/**` is intentionally untouched.)

### Acceptance criteria

- [ ] With the binary started via `kaos-control -d`, the SPA loads at the configured listen address and the main view renders (no blank page, no console errors attributable to this change).
- [ ] With the binary started via `kaos-control serve`, the SPA behaves identically to `-d`.
- [ ] `GET` of the version endpoint still succeeds and the SPA's version display (`web/src/stores/app.ts`) populates as before.
- [ ] No file under `web/src/` is modified by this lineage.

## Milestone 2 — Keep the dev build/serve loop working with the daemon flag

### Description

`make run` (and any documented dev loop that launches the SPA against a live
backend) must still start the server after the backend plan adds the required
`-d` flag to the `run:` target. This milestone is the frontend-developer's
checkpoint that the local "edit Vue → see it in the running app" loop is intact;
the actual `Makefile` edit is owned by the backend plan (Milestone 4) to keep a
single source of truth.

### Files to change

- None directly. Coordinates with backend plan `-3-be` Milestone 4 (`Makefile`
  `run:` target gains `-d`).

### Acceptance criteria

- [ ] After `make build-web` + `make run`, the server starts and serves the freshly built SPA from `web/dist/`.
- [ ] The dev instructions a frontend developer follows no longer reference a bare `kaos-control` start (covered by backend plan Milestone 4 doc sweep).

## Cross-links

- [[daemon-flag-usage-guide]] — originating idea and requirement (`-2`).
- Backend plan `daemon-flag-usage-guide-3-be` — owns all code and documentation changes, including the `Makefile` `run:` target this plan depends on.
- Test plan `daemon-flag-usage-guide-5-test` — the "SPA served unchanged" checks here are covered by the test plan's smoke check that `-d` brings up a serving instance.
