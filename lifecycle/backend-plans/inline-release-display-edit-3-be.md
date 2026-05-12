---
title: "Backend Plan: Inline Release Display and Editing"
type: plan-backend
status: done
lineage: inline-release-display-edit
parent: lifecycle/requirements/inline-release-display-edit-2.md
---

## Overview

Add a `PATCH /api/p/:project/artifacts/*/release` endpoint that updates an artifact's `release` frontmatter field on disk, validates the release name against the project's release store, re-indexes the artifact, and broadcasts a WebSocket event. The handler mirrors the existing `handlePatchPriority` pattern in `internal/http/write.go`.

## Milestone 1: Route Registration

**Description:** Register the new PATCH sub-route for `/release` alongside the existing `/priority` route.

**Files to change:**
- `internal/http/server.go` (~line 193-199): Add a `strings.HasSuffix(param, "/release")` branch inside the existing `r.Patch("/artifacts/*", ...)` handler, calling `s.handlePatchRelease(w, r)`.

**Acceptance criteria:**
- [ ] `PATCH /api/p/:project/artifacts/<path>/release` routes to the new handler.
- [ ] Unknown PATCH sub-routes still return 404.
- [ ] Existing `/priority` route is unaffected.

## Milestone 2: Handler Implementation

**Description:** Implement `handlePatchRelease` in `internal/http/write.go`, following the `handlePatchPriority` pattern (lines 437-525).

**Files to change:**
- `internal/http/write.go`: New function `handlePatchRelease`.

**Logic:**
1. Extract project from context; require appropriate role (`RolesPriorityEditors` or a new `RolesReleaseEditors` — use the same set as priority for now since release assignment is a triage action).
2. Parse the `*` URL param, trim the `/release` suffix to get `relPath`.
3. Decode JSON body: `{ "release": "<name>" | null }`. Use `*string` for the field so `null` is distinguishable from missing.
4. **Validate the release name** (if non-null): instantiate `release.NewStore(p.Idx.DB())` and call `store.GetByName(p.Entry.Name, *req.Release)`. If not found, return 422 with `apiError("invalid_release", "release not found: <name>")`.
5. Resolve path via `sandbox.Resolve`, read file, parse artifact.
6. Check lineage lock (same pattern as priority handler — 423 if locked by another user).
7. Set `a.FM.Release = releaseName` (empty string when null to clear it, matching the `omitempty` YAML tag behaviour).
8. Rebuild markdown via `buildMarkdown`, write file, re-index, broadcast `artifact.indexed` event.
9. Return `{ "artifact": ArtifactRow }` with 200.

**Acceptance criteria:**
- [ ] Valid release name → 200 with updated artifact; `release` field written to disk.
- [ ] `null` release → 200; `release` field removed from frontmatter.
- [ ] Non-existent release name → 422 with `invalid_release` error code.
- [ ] Locked lineage by another user → 423.
- [ ] Missing/invalid JSON body → 400.
- [ ] Artifact not found → 404.
- [ ] WebSocket `artifact.indexed` event broadcast after successful write.
- [ ] `go vet ./...` and `go build ./...` pass.

## Milestone 3: Permission Constant

**Description:** Add a `RolesReleaseEditors` permission constant (or reuse `RolesPriorityEditors`) so that release assignment permissions are explicit and can be adjusted independently later.

**Files to change:**
- `internal/http/permissions.go`: Add `RolesReleaseEditors` with the same initial value as `RolesPriorityEditors` (`[RoleProductOwner, RoleAnalyst]`).

**Acceptance criteria:**
- [ ] `RolesReleaseEditors` is defined and used by `handlePatchRelease`.
- [ ] No other handlers are affected.

## Cross-links

- The [[inline-release-display-edit]] frontend plan depends on this endpoint being available.
- The [[inline-release-display-edit]] test plan will exercise this endpoint directly.
