---
title: "Test Plan: Recursive Subdirectory Support for Artifact Directories"
type: plan-test
status: done
lineage: idea-archiving
parent: lifecycle/requirements/idea-archiving-2.md
created: "2026-07-05T16:00:00+10:00"
priority: high
labels:
    - test
    - watcher
    - index
    - artifacts
---

# Test Plan: Recursive Subdirectory Support for Artifact Directories

Verifies [idea-archiving-2](../requirements/idea-archiving-2.md) across the
backend ([[idea-archiving-3-be]]) and frontend ([[idea-archiving-4-fe]]) plans.
Extends the existing watcher/index refresh patterns in
[[editor-live-refresh-on-disk-change]] and the E2E harness in
[[end-to-end-smoke-tests]].

## Context from the code

- Go integration harness: `tests/integration/helpers_test.go`
  (`//go:build integration`). `newTestEnv(t, seeds)` (helpers_test.go:46) boots a
  temp-dir project with lifecycle stages, git init, a running server, watcher +
  lock reaper, and auto-logins as admin. `seedArtifact{relPath, content}`
  (helpers_test.go:296) seeds files; `makeArtifact` (helpers_test.go:302) builds
  frontmatter+body; `doRequest`/`readJSON`/`requireStatus`
  (helpers_test.go:397/428/450) drive the API. `env.proj.Idx.Get(relPath)` reads
  the in-memory index directly.
- Canonical watcher-refresh pattern: `tests/integration/external_edit_test.go`
  `TestExternalEditPickedUp` (external_edit_test.go:19) — write a file on disk,
  poll `proj.Idx.Get` up to ~2 s until re-indexed.
- Artifact-parse unit tests live beside `internal/artifact/artifact.go`.
- Frontend Vitest suites live in `tests/web/` (e.g. `useExternalChange.test.ts`,
  `ArtifactListView.*.test.ts`, `LineageBreadcrumb.test.ts`); config
  `tests/web/vitest.config.ts`. Playwright E2E lives in `tests/e2e/`.

Every Acceptance Criterion bullet in the requirement maps to at least one test
below.

---

## Milestone 1 — Unit: path-component parsing & rel_path derivation

**Description.** Cover `artifact.parsePathComponents` / `Parse` for flat,
single-nested, and deeply-nested paths, plus cross-platform separators
(req #5, #6, #14; backend M1).

**Files to change / add.**
- `internal/artifact/artifact_test.go` (add table-driven cases).

**Acceptance criteria.**
- `lifecycle/ideas/login.md` → `Stage=="ideas"`, `RelPath=="login.md"`.
- `lifecycle/ideas/done/login.md` → `Stage=="ideas"`, `RelPath=="done/login.md"`,
  identical `Slug`/`Index` to the flat case.
- `lifecycle/ideas/2026/q3/release-x.md` → `Stage=="ideas"`,
  `RelPath=="2026/q3/release-x.md"`.
- A path with OS-native `\` separators still yields forward-slash `RelPath`.
- Type/status/lineage/parent are byte-identical regardless of folder or folder
  name (covers req #5 and AC "unchanged by moving between folders").

---

## Milestone 2 — Integration: recursive scan, index, and API exposure

**Description.** Prove startup scan indexes nested artifacts, they appear in the
API with correct `rel_path`, and flat behaviour is byte-identical (req #1, #4,
#12; ACs 1, 5, 9-flat-equivalence).

**Files to change / add.**
- `tests/integration/recursive_subdir_test.go` (new).

**Acceptance criteria.**
- Seeding `lifecycle/ideas/done/archived.md` makes it appear in
  `GET /p/{project}/artifacts` and `env.proj.Idx.Get("lifecycle/ideas/done/archived.md")`
  is non-nil (AC 1).
- Each API artifact carries `rel_path` with forward slashes; flat → bare
  filename, nested → `done/archived.md` (AC 5).
- A project seeded with **no** subdirectories produces the same index rows as
  before, with `rel_path` == filename for every artifact (AC 9 — backward
  compatibility). Assert the flat `rel_path` equals `filepath.Base(path)`.
- The nested artifact is editable via `PUT` and appears in the graph endpoint
  keyed by its full path (AC 2).

---

## Milestone 3 — Integration: live watcher recursion & new-subdir detection

**Description.** Extend the `external_edit_test.go` pattern to nested dirs and
runtime-created subdirectories (req #2, #3; ACs 3, 4). This is the direct
extension called out in [[editor-live-refresh-on-disk-change]].

**Files to change / add.**
- `tests/integration/recursive_subdir_test.go` (add cases) or a sibling
  `recursive_watch_test.go`.

**Acceptance criteria.**
- Writing a new `*.md` into an existing nested dir at runtime is picked up:
  poll `proj.Idx.Get(relPath)` until non-nil within the debounce+margin window
  (~2 s), mirroring `TestExternalEditPickedUp` (AC 3).
- Creating a brand-new subdirectory **and** an artifact inside it in one
  operation at runtime results in the artifact being indexed and the directory
  watched (a subsequent edit to that file is also picked up, proving the dir is
  watched) (AC 4).
- Moving an artifact from the root into `archive/` and back updates its
  `rel_path` while preserving identity: after the move, exactly one indexed row
  exists at the new path with unchanged type/status/lineage/index/parent, the
  old path is gone, and no duplicate lineage row appears (ACs 6, 7).

---

## Milestone 4 — Integration: dot-dir exclusion, path-safety, watch cap

**Description.** Verify hidden paths are skipped, nested writes are
traversal-safe, and the create-with-subdir path works (req #8, #10, #11, #13;
ACs 8, 9-nested-writes).

**Files to change / add.**
- `tests/integration/recursive_subdir_test.go` (add cases).
- `internal/sandbox/sandbox_test.go` (nested-path + traversal cases if not
  already covered).

**Acceptance criteria.**
- A `.md` file under `lifecycle/ideas/.trash/` and a dotfile
  `lifecycle/ideas/.hidden.md` are **not** indexed and their dir is not watched
  (AC 8). `.git` under a root is likewise ignored.
- `POST /artifacts` with `subdir: "archive"` creates the file under
  `lifecycle/<stage>/archive/` and it is indexed and returned (AC 9-writes).
- `PUT`/`POST`/`GET` to a nested path succeeds; a request whose path contains
  `..` escaping the root is rejected with the sandbox error (AC 9-traversal,
  covers the `handleGetArtifact` sandbox fix).
- (If feasible in CI) a synthetic count above the 5000-dir cap logs the warning
  and the server keeps serving — otherwise cover the cap logic with a targeted
  `internal/watcher` unit test.

---

## Milestone 5 — Integration: cross-folder lineage uniqueness

**Description.** Confirm lineage-index allocation and collision surfacing span
the whole root regardless of folder (req #9; AC 10).

**Files to change / add.**
- `tests/integration/recursive_subdir_test.go` (add cases) or
  `internal/index/index_test.go`.

**Acceptance criteria.**
- `NextIndexForLineage("login")` returns the same next index whether the
  lineage's existing members are in the root or scattered across subfolders.
- Two artifacts sharing the same lineage+index placed in
  `lifecycle/ideas/a/` and `lifecycle/ideas/b/` are surfaced as a collision by
  the same mechanism (and to the same degree) as two flat colliding files (AC 10).

---

## Milestone 6 — Frontend: types, breadcrumb, list, live-refresh

**Description.** Cover the frontend `rel_path` consumption, folder breadcrumb,
list display, and nested live-refresh (frontend M1-M4; ACs 2, 5).

**Files to change / add.**
- `tests/web/LineageBreadcrumb.test.ts` — folder-segment rendering from
  `rel_path` (flat → no folder segment; `archive/foo.md` → `archive` segment;
  `2026/q3/foo.md` → ordered `2026`,`q3`).
- `tests/web/ArtifactListView.*.test.ts` — nested row shows `rel_path`; flat row
  shows bare filename; sort/paginate unaffected.
- `tests/web/useExternalChange.test.ts` — extend path-filtering scenarios so a
  WS `file.changed` for a nested path triggers auto-refresh / conflict banner
  identically to a flat path.

**Acceptance criteria.**
- Breadcrumb renders correct ordered folder segments for flat, single-nested,
  and deeply-nested `rel_path`, and renders safely while `rel_path` is absent
  (loading state).
- List row shows `rel_path`; existing list specs still pass.
- `useExternalChange` matches nested paths; no regression in its existing 8
  scenarios.

---

## Milestone 7 — E2E smoke: archive round-trip

**Description.** One end-to-end flow through the running app confirming the
archive use-case, extending [[end-to-end-smoke-tests]] (AC 1, 2, 6).

**Files to change / add.**
- `tests/e2e/flows/` — new spec `NN-archive-subdir.spec.ts` using the existing
  `harness/kaos-control.ts` server harness and `fixtures/seed-helpers.ts`.

**Acceptance criteria.**
- Seed an artifact, move it (via API or on-disk) into `archive/`, and confirm
  the app lists it with its `archive/` path, opens it in the editor with a
  folder breadcrumb, and shows no duplicate — then move it back and confirm
  parity.

---

## Coverage traceability

| Requirement AC | Covered by |
| --- | --- |
| Startup scan indexes nested (AC1) | M2 |
| Editable/graph parity (AC2) | M2, M6, M7 |
| Runtime nested create indexed + WS (AC3) | M3 |
| New subdir at runtime watched (AC4) | M3 |
| rel_path exposed, fwd slash, flat=filename (AC5) | M1, M2, M6 |
| Move preserves identity/edges/history (AC6) | M3, M7 |
| Type/status/lineage/parent folder-invariant (AC7) | M1, M3 |
| Dot-dirs skipped (AC8) | M4 |
| Nested write ok / traversal rejected (AC9) | M4 |
| Flat repo byte-identical (backward compat) | M2 |
| Cross-folder collision surfacing (AC10) | M5 |
| Extend watcher/index refresh patterns (AC11) | M3, M6, M7 |

## Run commands

- Backend units: `make test-unit` (`go test ./... -short`).
- Go integration: `go test -tags integration ./tests/integration/...`.
- Frontend units: `cd tests/web && pnpm test`.
- E2E: per `tests/e2e/playwright.config.ts`.
