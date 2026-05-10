---
title: Version Number Display
type: requirement
status: done
lineage: version-number-display
created: "2026-05-10T00:00:00+10:00"
priority: normal
parent: lifecycle/ideas/version-number-display.md
labels:
    - feature
    - frontend
    - backend
    - releases
    - operability
release: KC-Release0
assignees:
    - role: product-owner
      who: agent
---

# Version Number Display

## Problem

Operators and developers running kaos-control have no immediate visual confirmation of which version they are using. The version string is currently compiled into the Go binary via `-ldflags` and exposed only inside the `/health` endpoint's JSON response — it is invisible in the UI and requires a deliberate API call to discover. This makes it harder to confirm deployments, triage bugs ("which build are you on?"), and correlate behaviour with release notes.

There is also no canonical `VERSION` file in the repository. The version is only set when a build explicitly passes `-ldflags`, meaning development builds default to `"dev"` and there is no single, human-readable source of truth for the current release number.

## Goals / Non-goals

### Goals

1. **Single source of truth** — Introduce a `VERSION` file at the repository root containing only the semver string (e.g. `0.1.0`). All build and display paths derive the version from this file.
2. **Backend exposure** — Expose the version through a dedicated `GET /api/version` endpoint that returns the version string. Continue including it in the existing `/health` response.
3. **Frontend display** — Render the version as a persistent, unobtrusive label in the UI so any user can see the running version at a glance without inspecting API responses.
4. **Release alignment** — The `VERSION` file, git tags, and displayed version must stay in sync by convention: cutting a release means bumping `VERSION`, committing, and tagging that commit.

### Non-goals

- Automatic version bumping or release automation tooling.
- Git-describe-based version computation at build time.
- Displaying build metadata beyond the semver string (e.g. commit SHA, build date) — these can be added later but are out of scope.
- Versioning the API itself (this is about displaying the *product* version).

## Detailed Requirements

### Functional

**F1 — VERSION file**
- A plain-text file named `VERSION` at the repository root.
- Contains exactly one line: a valid semver string (e.g. `0.1.0`), with no leading `v` prefix, no trailing newline beyond the final line terminator.
- The Makefile or build process reads this file and injects its contents into the Go binary at compile time (via `go:embed` or `-ldflags`).

**F2 — Backend version endpoint**
- `GET /api/version` returns a JSON response: `{"version": "<semver>"}`.
- This endpoint requires no authentication (it is safe to expose the product version publicly).
- The existing `/health` endpoint continues to include `"version"` in its response, sourced from the same value.

**F3 — Frontend version display**
- The SPA displays the version string in a fixed, always-visible location. The idea specifies "top-left"; the final placement should be consistent with the existing layout (e.g. in the sidebar header or app toolbar).
- The version is fetched from `GET /api/version` (or from the `/health` response) once at app startup.
- Format: `kaos-control <version>` (e.g. `kaos-control 0.1.0`).
- The label must be visually subtle — small font, muted colour — so it does not compete with primary navigation or content.

**F4 — Build integration**
- `make build` reads the `VERSION` file and passes its contents to the Go linker so that the compiled binary reports the correct version.
- Development builds (`make run` or `go run`) should fall back to `"dev"` if the linker variable is not set.

### Non-functional

**NF1 — No new dependencies** — This feature must not introduce any new Go modules or npm packages.

**NF2 — Performance** — The `/api/version` endpoint must respond in under 5 ms (it returns a static string).

**NF3 — Accessibility** — The version label in the UI must meet WCAG 2.1 AA contrast requirements against its background.

## Acceptance Criteria

- [ ] A `VERSION` file exists at the repo root containing a valid semver string.
- [ ] `make build` produces a binary that reports the correct version (not `"dev"`).
- [ ] Running the binary without ldflags (e.g. `go run`) falls back to `"dev"`.
- [ ] `GET /api/version` returns `{"version": "<value from VERSION>"}` with status 200.
- [ ] `GET /health` continues to include `"version"` with the same value.
- [ ] The SPA displays `kaos-control <version>` in a fixed, always-visible location.
- [ ] The displayed version matches the value returned by `/api/version`.
- [ ] The version label is visually subtle and does not obscure navigation elements.
- [ ] The version label meets WCAG 2.1 AA contrast requirements.
- [ ] No new Go modules or npm packages are introduced.
- [ ] [[version-number-display]] lineage artifacts (plans, tests) can reference this requirement.

## Resolved Questions

1. Should the `/api/version` endpoint be completely unauthenticated, or should it sit behind the existing auth middleware like other `/api/*` routes? The idea says "operators" which implies authenticated users, but version info is generally harmless to expose publicly and useful for health-check tooling.

> Unauthenticated for this API is OK.

2. Should the `VERSION` file use a leading `v` prefix (e.g. `v0.1.0`) to match git tag convention, or bare semver (e.g. `0.1.0`) with the `v` added only when tagging? The idea implies bare semver.

> bare semver

3. What is the initial version number to seed in the `VERSION` file? Suggested: `0.1.0`.

> 0.1.0 is good.
