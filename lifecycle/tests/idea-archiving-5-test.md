---
title: 'Test Suite: Recursive Subdirectory Support for Artifact Directories'
type: test
status: approved
lineage: idea-archiving
---

# Test Suite: Recursive Subdirectory Support for Artifact Directories

Covers [idea-archiving-2](../requirements/idea-archiving-2.md) per the plan at
[idea-archiving-5-test.md](../test-plans/idea-archiving-5-test.md). All seven
milestones now have coverage; a prior partial run (agent run
`30c6973d9ee0a7e4`, killed on timeout) had left `recursive_subdir_test.go`
with four of its seven tests actually failing — those are fixed below, not
just extended.

## Scope note: file locations vs. the test plan

This suite's write scope is `tests/**` and `lifecycle/tests/**` only — it does
not touch `internal/**`. The test plan named `internal/artifact/artifact_test.go`,
`internal/sandbox/sandbox_test.go`, and `internal/index/index_test.go` as
Milestone 1/4/5 targets; those unit-test unexported functions from inside
their own package, which isn't reachable from outside `internal/`. Where the
function under test is exported, this suite exercises it directly from
`tests/integration/` instead (see Milestone 1 and 4 below). Milestone 5 turned
out to already be fully covered by `internal/index/lineage_folder_test.go`,
added separately by the backend-developer's own commit (`acc2d15b`) — this
suite does not duplicate it.

## Covered Test Cases

1. **Milestone 1 — Unit: path-component parsing & rel_path derivation**
   `tests/integration/artifact_parse_test.go` (new). Drives the exported
   `artifact.Parse` (which internally calls the unexported
   `parsePathComponents`) rather than the unit function directly, per the
   scope note above:
   - Flat, single-nested, and deeply-nested paths → correct `Stage`/`RelPath`.
   - `filepath.Join` + `filepath.ToSlash` (the pattern every real caller uses)
     still yields forward-slash `RelPath`.
   - Type/status/lineage/parent/slug/index are byte-identical for the same
     content parsed at three different folder depths.

2. **Milestone 2 — Integration: recursive scan, index, and API exposure**
   `tests/integration/recursive_subdir_test.go`, `TestRecursiveScan` (fixed)
   and `TestFlatOnlyProjectBackwardCompat` (new):
   - Nested artifact from startup scan is indexed, in the artifacts API with
     `rel_path == "done/archived.md"`, and appears in `GET /graph` keyed by
     full path.
   - PUT edits a nested artifact; the update round-trips through GET.
   - A project with **no** subdirectories produces `rel_path == filepath.Base(path)`
     for every row (backward compatibility, AC9).

3. **Milestone 3 — Integration: live watcher recursion & new-subdir detection**
   `TestRuntimeNestedCreate` (fixed — was failing: wrote into a nested dir
   that was never created) and `TestMovePreservesIdentity` (strengthened):
   - A new file written into an *already-watched* nested dir is picked up.
   - A brand-new subdir + file created together is indexed, and the dir is
     confirmed watched by a follow-up edit.
   - Moving root → subfolder → root preserves type/status/lineage/index/slug/parent,
     updates `rel_path`, leaves no row at the old path, and leaves exactly one
     row for the lineage at each step (no duplicate).

4. **Milestone 4 — Integration: dot-dir exclusion, path-safety**
   - `TestDotDirExclusion` (fixed — its flat-file assertion checked a path
     that was never seeded) + new `TestDotfileExclusion_Runtime`: a dotfile
     or dot-dir present at startup, or a dotfile written at runtime into an
     already-watched dir, is never indexed.
   - `TestNestedWriteSafety` (fixed — asserted HTTP 200 for a create that
     correctly returns 201): `POST .../artifacts` with `subdir` creates and
     indexes under the nested folder; a path containing `..` is rejected.
   - `tests/integration/sandbox_nested_test.go` (new): exercises the exported
     `sandbox.Resolve` directly — nested existing/non-existent paths resolve
     inside the root, `..` traversal and absolute paths are rejected, and a
     symlink that escapes the root is rejected.
   - **Not covered**: the 5000-directory watch cap (`maxWatchedDirs` in
     `internal/watcher/watcher.go`). Simulating 5000 real directories is
     impractical for this suite, and the plan's fallback (a targeted
     `internal/watcher` unit test) is out of this suite's write scope.

5. **Milestone 5 — Integration: cross-folder lineage uniqueness**
   Already fully covered by `internal/index/lineage_folder_test.go`
   (`TestNextIndexForLineage_CrossFolder`, `TestLineageIndexCollision_AcrossFolders`),
   added by the backend-developer alongside the feature. `TestCrossFolderUniqueness`
   in `recursive_subdir_test.go` (from the prior run) is a lighter integration-level
   check of the same behaviour and is left in place, not duplicated further.

6. **Milestone 6 — Frontend: types, breadcrumb, list, live-refresh**
   - `tests/web/LineageBreadcrumb.test.ts`: new describe block
     `folder segments from rel_path` — flat/single-nested/deeply-nested
     `rel_path` render the right ordered folder segments; absent `rel_path`
     falls back to splitting the full path without crashing.
   - `tests/web/ArtifactListView.relPath.test.ts` (new): flat rows show the
     bare filename, nested/deep rows show `rel_path` unabridged, a stale row
     with empty `rel_path` falls back to the full path, and sorting/pagination
     are unaffected by mixed flat/nested rows.
   - `tests/web/useExternalChange.test.ts`: new describe block — nested and
     deeply-nested paths trigger auto-refresh / conflict-banner identically to
     flat paths, and a sibling file in the same folder is correctly ignored.
   - (Note: `web/src/components/artifact/__tests__/LineageBreadcrumb.spec.ts`
     and `web/src/views/project/__tests__/ArtifactListView.spec.ts` already
     cover this same behaviour as colocated component specs, written by the
     frontend-developer alongside the feature; the files above are the
     designated `tests/web/` regression-suite counterparts.)

7. **Milestone 7 — E2E smoke: archive round-trip**
   `tests/e2e/flows/11-archive-subdir.spec.ts` (new): seeds a flat idea on
   disk, moves it into `lifecycle/ideas/archive/`, confirms the list shows it
   at its new path with no duplicate row and the editor breadcrumb shows an
   `archive` folder segment, then moves it back and confirms parity. Disk
   moves don't trigger the frontend's `artifact.indexed` WS auto-refresh
   (that event only fires on API writes), so each assertion polls the API
   directly until the watcher has caught up before navigating.

## Known defect found while testing (not fixed — out of this suite's scope)

`TestDotDirExclusion_RuntimeCreation` in `recursive_subdir_test.go`
reproducibly **fails**. Creating a dot-directory and a `*.md` file inside it
together at runtime (`mkdir .trash && write .trash/dot.md`, matching the
"directory created together with a file already inside it" race the
watcher's own comments describe) gets indexed, even though it should be
excluded like any other dot-dir.

Root cause: `internal/watcher/watcher.go`'s fsnotify event loop, right after
calling `addDirRecursive` for a newly-created directory, does a second
`filepath.WalkDir` over that directory to catch files that raced the watcher
(around watcher.go:206). `addDirRecursive` correctly skips dot-prefixed
directories, but this second walk does not re-check the directory name before
indexing files it finds — only `shouldProcess`'s basename check runs, which
doesn't see the dot-prefixed ancestor. The equivalent *startup-scan* path
(`internal/index/index.go`) and the equivalent *runtime dotfile-in-a-normal-dir*
path (`shouldProcess`'s basename check) are both correct; only this one
fallback walk is missing the check. This is left as-is pending a fix in
`internal/watcher/watcher.go`, which is outside this suite's write scope
(`tests/**`, `lifecycle/tests/**`).

## Test Files

- `tests/integration/artifact_parse_test.go` — Milestone 1
- `tests/integration/recursive_subdir_test.go` — Milestones 2, 3, 4 (fixed + extended)
- `tests/integration/sandbox_nested_test.go` — Milestone 4
- `tests/web/LineageBreadcrumb.test.ts` — Milestone 6
- `tests/web/ArtifactListView.relPath.test.ts` — Milestone 6
- `tests/web/useExternalChange.test.ts` — Milestone 6
- `tests/e2e/flows/11-archive-subdir.spec.ts` — Milestone 7

## Run commands

- Go integration: `go test -tags integration ./tests/integration/...`
- Frontend units: `cd tests/web && pnpm vitest run`
- E2E: `cd tests/e2e && pnpm playwright test flows/11-archive-subdir.spec.ts`
  (requires `make build-web && make build` first so the embedded frontend is current)
