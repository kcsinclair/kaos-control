---
title: "Test plan: ignore patterns for lifecycle indexer"
type: plan-test
status: draft
lineage: ignore-readme-files-in-lifecycle-dir
parent: lifecycle/defects/ignore-readme-files-in-lifecycle-dir.md
---

# Test plan: ignore patterns for lifecycle indexer

Integration and unit tests to verify that configurable ignore patterns correctly prevent matching files from being indexed during startup scan, live watcher events, and direct `IndexFile` calls.

## Milestone 1 — Unit tests for `ShouldIgnore` helper

**Description:** Test the `ShouldIgnore` helper function in isolation with various pattern/path combinations to ensure correct matching semantics.

**Files to change:**
- `internal/config/config_test.go` (or a new `config_ignore_test.go` if the file is large) — add table-driven tests.

**Acceptance criteria:**
- [ ] `README.md` pattern matches `lifecycle/ideas/README.md` and `lifecycle/requirements/README.md`.
- [ ] `README.md` pattern does **not** match `lifecycle/ideas/my-readme.md`.
- [ ] Glob pattern `*.draft.md` matches `lifecycle/ideas/feature.draft.md`.
- [ ] Glob pattern `*.draft.md` does **not** match `lifecycle/ideas/feature.md`.
- [ ] Empty pattern list matches nothing.
- [ ] All tests pass with `go test ./internal/config/ -run ShouldIgnore`.

---

## Milestone 2 — Unit tests for `config.LoadProject` with ignore field

**Description:** Test that the `Ignore` field is correctly parsed from YAML, and that the default (`["README.md"]`) is applied when omitted.

**Files to change:**
- `internal/config/config_test.go` — add tests for `LoadProject` covering: field present, field absent (default), invalid glob syntax.

**Acceptance criteria:**
- [ ] A config YAML with `ignore: ["README.md", "CHANGELOG.md"]` loads both patterns.
- [ ] A config YAML without `ignore` produces the default `["README.md"]`.
- [ ] A config YAML with `ignore: ["[invalid"]` returns a validation error.

---

## Milestone 3 — Integration test: startup scan ignores matching files

**Description:** Set up a temporary project directory with a `lifecycle/config.yaml` containing ignore patterns, place a `README.md` inside a stage directory alongside a legitimate artifact, run `index.Open` + `Scan`, and assert the README is absent from the index.

**Files to change:**
- `tests/` — new test file, e.g. `tests/ignore_patterns_test.go`.

**Acceptance criteria:**
- [ ] After `Scan`, querying the index for `lifecycle/ideas/README.md` returns no result.
- [ ] The legitimate artifact (`lifecycle/ideas/login.md`) is present in the index.
- [ ] The scan log does not contain a warning for the ignored file.

---

## Milestone 4 — Integration test: watcher ignores matching files

**Description:** Start the watcher on a temporary project, create a `README.md` inside a watched stage directory, and confirm that no `file.changed` WebSocket event is broadcast and the file is not indexed.

**Files to change:**
- `tests/` — extend or add to the ignore-patterns test file.

**Acceptance criteria:**
- [ ] Writing a `README.md` to a watched directory does not result in a new row in the SQLite index after the debounce window.
- [ ] Writing a legitimate `.md` file to the same directory does result in indexing and a `file.changed` event.

---

## Milestone 5 — Integration test: `IndexFile` rejects ignored files directly

**Description:** Call `idx.IndexFile` with the absolute path of a file whose base name matches an ignore pattern and assert it returns an error without inserting a row.

**Files to change:**
- `tests/` — extend the ignore-patterns test file.

**Acceptance criteria:**
- [ ] `IndexFile` returns a non-nil error for an ignored file.
- [ ] The artifacts table has no row for the ignored file's path after the call.

---

## Milestone 6 — Integration test: API does not return ignored files

**Description:** Start the full server (or a test-scoped HTTP handler) with a project containing a `README.md` in a stage directory, and assert the artifact list and graph API endpoints do not include it.

**Files to change:**
- `tests/` — extend the ignore-patterns test file or add an API-level test.

**Acceptance criteria:**
- [ ] `GET /api/artifacts` response does not contain any artifact with path matching an ignored pattern.
- [ ] `GET /api/graph` response nodes do not include any ignored file.

---

## Cross-references

- [[ignore-readme-files-in-lifecycle-dir]] backend plan (index 2): tests in milestones 3–6 depend on the backend implementation being complete.
- [[ignore-readme-files-in-lifecycle-dir]] frontend plan (index 3): milestone 1 of the frontend plan (manual verification) is complemented by the API-level integration tests here.
