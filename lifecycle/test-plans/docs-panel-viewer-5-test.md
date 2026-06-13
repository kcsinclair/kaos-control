---
title: Test Plan — Documentation Panel Viewer
type: plan-test
status: in-development
lineage: docs-panel-viewer
created: "2026-06-12T00:00:00+10:00"
priority: normal
parent: lifecycle/requirements/docs-panel-viewer-2.md
labels:
    - test
    - feature
release: KC-Release3
assignees:
    - role: test-developer
      who: agent
---

# Test Plan — Documentation Panel Viewer

Tests cover the requirement in [[docs-panel-viewer-2]] and verify the implementations from [[docs-panel-viewer-3-be]] (backend) and [[docs-panel-viewer-4-fe]] (frontend). Tests live in `tests/integration/` (Go integration tests against a real HTTP server + tmpdir project) and in `web/src/**/__tests__/` (Vitest component / store tests already scheduled inside the frontend plan).

The integration tests below use the existing `newTestEnv` harness, which auto-logs in as `admin@test.local` (`product-owner, analyst, reviewer, approver`). Tests that need a read-only user re-login as `qa@test.local`. The convention follows [project: kaos-control codebase] — devops URL helpers return full URLs for `http.Get`; the run-log endpoint returns NDJSON.

---

## Milestone 1 — Backend integration tests: list + read

### Description

Cover the happy paths for `GET /docs` and `GET /docs/*path`, including subdirectory handling, the `docs_dir_present: false` case, the binary-file fallback, and the title/summary extraction fallbacks (frontmatter → H1 → filename).

### Files to change

- **New** `tests/integration/docs_list_test.go`:
  - Helper `seedDocs(t, projectRoot, files map[string]string)` writes a map of `relPath -> contents` under `<projectRoot>/docs/`.
  - `TestDocsList_EmptyWhenNoDocsDir`:
    - Seed a project with no `docs/`. `GET /api/p/{project}/docs` returns 200, `docs_dir_present: false`, `docs: []`.
  - `TestDocsList_ReturnsSortedEntries`:
    - Seed `docs/zeta.md` (frontmatter `title: Aardvark`), `docs/alpha.md` (no frontmatter, H1 `# Beta`), `docs/sub/gamma.md`.
    - List returns three entries sorted by lowered title: `Aardvark`, `Beta`, `gamma` (filename fallback). Ties broken by path.
    - `sub_dir` is `""` for the first two, `"sub"` for the third.
  - `TestDocsList_TitleFallbackChain`:
    - Frontmatter `title:` wins over H1; H1 wins over filename; filename-without-extension is used when neither is present.
  - `TestDocsList_SummaryTruncation`:
    - First paragraph of 500 chars is truncated to 200 runes + ellipsis. Frontmatter `summary:` overrides body extraction. Frontmatter `description:` is used when `summary:` is absent.
  - `TestDocsList_NonMarkdownEntry`:
    - Seed `docs/diagram.png` (random bytes); list entry has `is_markdown: false`, `title: "diagram.png"`, `summary: ""`.
  - `TestDocsList_NonUtf8Fallback`:
    - Seed `docs/binary.md` with bytes `\xff\xfe\x00\x01`; list entry has `summary: "(binary or non-text file — cannot preview)"`. Title falls back to the filename.
  - `TestDocsGet_HappyPath`:
    - `GET /api/p/{project}/docs/alpha.md` returns body, file_sha, `is_markdown: true`.
  - `TestDocsGet_Subdirectory`:
    - `GET /api/p/{project}/docs/sub/gamma.md` returns body. Path is preserved through the wildcard route.
  - `TestDocsGet_NotFound`:
    - `GET /api/p/{project}/docs/missing.md` returns 404 with `apiError("not_found", ...)`.

### Acceptance criteria

- `go test ./tests/integration/ -run TestDocsList` and `-run TestDocsGet` pass.
- Each test cleans its tmpdir (handled by `newTestEnv`).
- No reliance on the real `docs/` of the kaos-control repo — every test seeds its own tmpdir.

---

## Milestone 2 — Backend integration tests: write + concurrency + permissions

### Description

Cover `PUT /docs/*path` happy path, sha-mismatch concurrency, the 415 non-markdown rejection, the role gate from [[auth-role-checks-mutations]], and the synchronous `doc.changed` broadcast.

### Files to change

- **New** `tests/integration/docs_write_test.go`:
  - `TestDocsPut_HappyPath`:
    - Seed `docs/alpha.md` with `# Alpha`. `PUT` with body `"# Alpha\n\nedit"` and the current sha. Response is 200 with a new `file_sha`. Re-reading the file from disk shows the edit.
  - `TestDocsPut_ShaMismatch`:
    - `PUT` with a stale `expected_sha` returns 409 `apiError("sha_mismatch", ...)`. The file is not modified on disk.
  - `TestDocsPut_NotMarkdown`:
    - Seed `docs/diagram.png`. `PUT /api/p/{project}/docs/diagram.png` returns 415 `apiError("not_markdown", ...)`.
  - `TestDocsPut_CreateNotAllowed`:
    - `PUT /api/p/{project}/docs/brand-new.md` returns 404 (creation is out-of-band; resolved question in [[docs-panel-viewer-2]]).
  - `TestDocsPut_NoRoleForbidden`:
    - Per the resolution of Q1 below, `qa` IS a docs editor (it's in `RolesArtifactEditors`), so QA editing is allowed and is covered by the happy-path write tests. This test instead verifies the forbidden path with a user who has **no roles on the project**: create/log in as a zero-role user, `PUT /api/p/{project}/docs/alpha.md` returns 403 `apiError("forbidden", ...)`.
  - `TestDocsPut_QARoleAllowed`:
    - Log in as `qa@test.local` (role `[qa]`). `PUT /api/p/{project}/docs/alpha.md` returns 200 — QA can fix doc issues directly (see Q1 resolution).
  - `TestDocsPut_BroadcastsDocChanged`:
    - Open a WebSocket against `/api/p/{project}/ws`. Issue a `PUT` for `docs/alpha.md`. Within 500 ms (well below the watcher debounce), assert at least one `doc.changed` event with `payload.path = "alpha.md"` is delivered.

### Acceptance criteria

- All tests pass via `go test ./tests/integration/ -run TestDocsPut`.
- The WebSocket assertion uses a buffered channel + `time.After` (no `time.Sleep` polling).

---

## Milestone 3 — Backend integration tests: path traversal + symlink rejection

### Description

Lock down the security surface called out in §Non-functional 4 of [[docs-panel-viewer-2]].

### Files to change

- **New** `tests/integration/docs_security_test.go`:
  - `TestDocsGet_RejectsParentTraversal`:
    - `GET /api/p/{project}/docs/../README.md` returns 400 `apiError("path_traversal", ...)`. Repeat with URL-encoded `%2e%2e/README.md`.
  - `TestDocsGet_RejectsAbsolutePath`:
    - `GET /api/p/{project}/docs//etc/passwd` returns 400. (Note: chi wildcard normalisation may strip the leading slash; verify against the actual router behaviour and adjust the assertion accordingly.)
  - `TestDocsGet_RejectsEscapingSymlink`:
    - Seed a target file `<tmp>/outside.md` *outside* the project root. Create `docs/escape.md` as a symlink to it. `GET /api/p/{project}/docs/escape.md` returns 400.
  - `TestDocsPut_RejectsParentTraversal`:
    - `PUT /api/p/{project}/docs/../etc/foo.md` returns 400.

### Acceptance criteria

- All four tests pass.
- The symlink test is gated on `runtime.GOOS != "windows"` to avoid CI surprises.

---

## Milestone 4 — Watcher unit / integration tests for `doc.changed`

### Description

Verify that the fsnotify-driven watcher emits `doc.changed` for files added/modified/deleted under `<projectRoot>/docs/`, and that lifecycle changes still emit `file.changed` (no regression).

### Files to change

- **New** `internal/watcher/docs_test.go`:
  - Set up a tmpdir, create `docs/`, instantiate `Watcher` against it, start in a goroutine.
  - Subscribe to the hub via a buffered channel.
  - `TestWatcher_DocCreateEmitsDocChanged`: create `docs/a.md`; receive one `doc.changed` event within 500 ms with `payload.path = "a.md"`.
  - `TestWatcher_DocModifyEmitsDocChanged`: modify `docs/a.md`; receive at least one `doc.changed` event.
  - `TestWatcher_DocDeleteEmitsDocChanged`: delete `docs/a.md`; receive a `doc.changed` event. No index mutation occurs (the docs file was never indexed).
  - `TestWatcher_LifecycleStillEmitsFileChanged`: write a `lifecycle/ideas/x.md`; receive `file.changed`. No `doc.changed` for that path.
  - `TestWatcher_MissingDocsDirIsNonFatal`: `Start` against a project root with no `docs/` returns no startup error and continues watching `lifecycle/` normally.

### Acceptance criteria

- `go test ./internal/watcher/... -short` passes.
- Tests use the same channel-with-timeout pattern as existing watcher tests (no `time.Sleep`).

---

## Milestone 5 — Frontend Vitest coverage

### Description

The unit-test files in [[docs-panel-viewer-4-fe]] (the API client, the store, the view) are owned by the frontend agent. This milestone simply codifies the acceptance gate so that the test plan covers them as a single deliverable.

### Files to change

- Already enumerated in [[docs-panel-viewer-4-fe]]:
  - `web/src/api/__tests__/docs.test.ts`
  - `web/src/stores/__tests__/docs.test.ts`
  - `web/src/views/project/__tests__/DocsView.test.ts`

### Acceptance criteria

- `pnpm test` in `web/` exits zero with the new tests included.
- New tests run in ≤ 2 seconds combined.
- No `console.error`/`console.warn` from the suite.

---

## Milestone 6 — End-to-end smoke checklist

### Description

A manual smoke checklist that the QA agent runs against `make run` after both backend and frontend land. This is not automated for v1 — see Risk notes for the future automation hook.

### Files to change

- **New** `lifecycle/tests/docs-panel-viewer.md` (artifact describing what the test code covers, per the `lifecycle/tests/` convention):
  - Title: "Documentation Panel Viewer — coverage map".
  - Body sections: "Automated coverage" (lists each test file from Milestones 1–5) and "Manual smoke" (the checklist below).

Manual smoke checklist:

- [ ] Fresh login, navigate to `/p/kaos-control/docs`. Six existing cards render in ≤ 300 ms.
- [ ] Cards are sorted alphabetically (case-insensitive). Non-markdown entries (`architecture-diagram.html`, `architecture.png`) render as links, not buttons.
- [ ] Typing `arch` in the search box filters the list within 100 ms; `aria-live` count updates.
- [ ] Clearing the search restores the full list.
- [ ] Searching for `nomatch-xyz` shows the "No documents match 'nomatch-xyz'" empty state with a working "Clear search" action.
- [ ] Renaming `docs/` away on disk (`mv docs docs.bak`) triggers the panel's "No `docs/` folder" empty state without a page reload. Renaming back restores it.
- [ ] Opening `docs/architecture.md` loads the markdown editor with the file body.
- [ ] Saving an edit persists to disk; reopening the file shows the edit.
- [ ] Modifying the file on disk while the editor is open shows the "Disk version updated" indicator (no overwrite without explicit reload).
- [ ] Logging in as `qa@test.local` and opening a doc renders an **editable** editor and a `PUT` save succeeds (QA is a docs editor — see Q1 resolution). A user with no project roles instead gets a 403 on save.
- [ ] `PUT /api/p/{project}/docs/../escape.md` rejected with HTTP 400 (curl).
- [ ] Tab through the panel from the sidebar header: focus reaches "Documentation", Enter activates; on the panel, Tab advances through the search input then each card; Enter on a card opens the editor.

### Acceptance criteria

- The `lifecycle/tests/docs-panel-viewer.md` artifact exists with frontmatter `title`, `type: test`, `status: draft`, `lineage: docs-panel-viewer`.
- All twelve manual checklist items pass on the QA agent's smoke run before the requirement transitions to `done`.

---

## Risk notes

- **WebSocket flakiness in CI** — Milestone 2's WS test can flake if the hub broadcast races the subscription handshake. Mitigation: open the WS, send a `ping` and wait for the first echo before issuing `PUT`. The existing helper `wsOpenAndDrain(t, env, ...)` does exactly this — reuse it.
- **Watcher dedup** — fsnotify on macOS can emit multiple `WRITE` events per logical save. Tests assert "at least one" event, not "exactly one", to avoid flakes.
- **Manual smoke automation** — once Playwright E2E is in place project-wide (out of scope here), Milestone 6 should be converted to a `tests/e2e/docs_panel_test.ts`. Tracked separately.

## Verification (end-to-end)

1. `make lint` clean.
2. `make test-unit` clean.
3. `make test-integration` clean (Milestones 1–4 new files included).
4. `pnpm test` in `web/` clean (Milestone 5).
5. Manual smoke checklist (Milestone 6) executed by the QA agent — every item ticked before status transitions to `done`.

---

## Resolved Questions

### Q1 (RESOLVED 2026-06-13): Should `qa@test.local` receive 403 or 200 on `PUT /api/p/{project}/docs/*path`?

**Decision (product owner): QA *can* edit docs — resolution 2 below.** Rationale: in
practice a QA engineer who spots a problem in a doc should be able to fix a minor
issue directly rather than be blocked; larger changes are expected to be checked
with the team, but that's a team-process convention, not something the endpoint
enforces. (A docs review/approval workflow could be a future enhancement; it is
out of scope here.)

**Consequences:**
- **No backend change.** `handlePutDoc` keeps `RolesArtifactEditors` (which
  includes `RoleQA`) — the implementation is already correct.
- The Milestone 2 write tests above are updated accordingly:
  `TestDocsPut_QARoleAllowed` (QA → 200) and `TestDocsPut_NoRoleForbidden`
  (zero-role user → 403) replace the original `TestDocsPut_ReadOnlyRoleForbidden`.
- The Milestone 6 smoke item is updated: QA sees an editable editor, not
  read-only.

This unblocks the Milestone 2 write tests; the test plan returns to active
implementation.

<details><summary>Original question and analysis (for the record)</summary>

#### Should `qa@test.local` receive 403 or 200 on `PUT /api/p/{project}/docs/*path`?

**Blocking test:** `TestDocsPut_ReadOnlyRoleForbidden` (Milestone 2, `docs_write_test.go`).

**The contradiction:** The test plan specifies that logging in as `qa@test.local` and issuing a `PUT` to `/api/p/{project}/docs/alpha.md` should return `403 apiError("forbidden", ...)`. However, the current backend implementation at `internal/http/docs.go:142` uses `requireRole(w, r, p, RolesArtifactEditors...)` for the `handlePutDoc` handler, and `RolesArtifactEditors` is defined as:

```go
RolesArtifactEditors = []string{RoleProductOwner, RoleAnalyst, RoleBackendDeveloper,
    RoleFrontendDeveloper, RoleTestDeveloper, RoleQA}
```

`RoleQA = "qa"` is included. The project config in the test environment gives `qa@test.local` the `[qa]` role. Therefore, the backend currently returns **200 (allowed)** for QA on this endpoint — the test as written would fail.

**Two possible resolutions:**

1. **QA should NOT be able to edit docs** → change `handlePutDoc` to use `RolesArtifactAuthors` (which excludes QA) instead of `RolesArtifactEditors`. This aligns with the manual smoke checklist item ("Logging in as `qa@test.local` and opening a doc renders the editor in read-only mode"), which implies QA is read-only for docs. A test developer cannot make this backend change; it requires a backend-developer or product-owner decision.

2. **The test plan is wrong** → QA can legitimately edit docs (consistent with `RolesArtifactEditors`). In that case, `TestDocsPut_ReadOnlyRoleForbidden` should be rewritten to use a user with no project roles at all (e.g. a freshly created user with no role assignments), and the expected status code should be 403 for "unauthenticated or zero-role user".

**Action needed:** Product owner to decide which interpretation is correct and either (a) update the backend role check, or (b) update this test plan to reflect what the implementation actually does. Implementation of all Milestone 2 write tests is blocked until this is resolved.

</details>
