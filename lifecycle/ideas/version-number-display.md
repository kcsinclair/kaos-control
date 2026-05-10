---
title: Version Number Display in UI
type: idea
status: approved
lineage: version-number-display
created: "2026-05-10T09:15:21+10:00"
priority: normal
labels:
    - feature
    - frontend
    - backend
    - releases
    - operability
release: KC-Release0
---

# Version Number Display in UI

Add a visible version number (e.g. `kaos-control 0.1`) in the top-left of the UI, sourced from a canonical `VERSION` file at the repository root. This file contains only the semver string and acts as the single source of truth for the current release version, making it trivial to increment as part of the release process.

The Go binary should read `VERSION` at build time (via `go:embed` or `ldflags`) and expose it through a `/api/version` endpoint or embed it directly in the SPA's index template. The frontend reads this value and renders it as a subtle but always-visible label, giving operators an immediate way to confirm which build is running.

The version string should stay in sync with git tags: cutting a release means bumping `VERSION`, committing it, and tagging that commit. This keeps the displayed version, the git tag, and any release artifacts aligned without requiring a build-time lookup of `git describe`.
