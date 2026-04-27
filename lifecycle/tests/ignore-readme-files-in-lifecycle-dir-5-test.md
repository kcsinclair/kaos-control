---
title: "Tests: ignore patterns for lifecycle indexer"
type: test
status: in-qa
lineage: ignore-readme-files-in-lifecycle-dir
parent: lifecycle/test-plans/ignore-readme-files-in-lifecycle-dir-4-test.md
---

# Tests: ignore patterns for lifecycle indexer

Integration and unit tests that verify configurable ignore patterns prevent
matching files from being indexed during startup scan, live watcher events,
and direct `IndexFile` calls.

## Scenarios covered

### Unit tests (`internal/config/config_test.go`)

**TestShouldIgnore** — table-driven tests for the `ShouldIgnore` helper:

- `README.md` pattern matches `lifecycle/ideas/README.md`.
- `README.md` pattern matches `lifecycle/requirements/README.md`.
- `README.md` pattern does **not** match `lifecycle/ideas/my-readme.md`.
- Glob `*.draft.md` matches `lifecycle/ideas/feature.draft.md`.
- Glob `*.draft.md` does **not** match `lifecycle/ideas/feature.md`.
- Empty pattern list (`[]string{}`) matches nothing.
- Nil pattern list matches nothing.
- Second pattern in a multi-pattern list still triggers a match.

Run with: `go test ./internal/config/ -run ShouldIgnore`

**TestLoadProjectIgnoreField** — verifies YAML loading for `Project.Ignore`:

- Explicit `ignore: ["README.md", "CHANGELOG.md"]` loads both patterns.
- Missing `ignore` key in YAML produces the default `["README.md"]`.
- `ignore: ["[invalid"]` (bad glob syntax) causes `LoadProject` to return a validation error.

Run with: `go test ./internal/config/ -run TestLoadProject`

### Integration tests (`tests/integration/ignore_patterns_test.go`)

**TestIgnorePatterns_StartupScan** (Milestone 3):

Seeds `lifecycle/ideas/README.md` and `lifecycle/ideas/login.md` into a temp
project. After `project.Open` (which runs the startup scan), asserts:

- `README.md` is absent from the SQLite index.
- `login.md` is present in the index.

**TestIgnorePatterns_WatcherSkipsIgnored** (Milestone 4):

Registers a hub channel to capture WebSocket events, then:

1. Writes `README.md` to a live-watched directory, waits 400 ms past the 150 ms
   debounce, and asserts no index row and no `file.changed` event for that path.
2. Writes `lifecycle/ideas/new-feature.md`, waits up to 2 s for a `file.changed`
   event, and asserts both the event and the index row are present.

**TestIgnorePatterns_IndexFileRejectsIgnored** (Milestone 5):

Creates `lifecycle/ideas/README.md` on disk, calls `proj.Idx.IndexFile` directly,
and asserts:

- The call returns a non-nil error.
- No row exists in the artifacts table for the ignored path.

**TestIgnorePatterns_APIExcludesIgnored** (Milestone 6):

Seeds `README.md` alongside `login.md`, then:

- `GET /api/p/testproject/artifacts` — asserts `README.md` absent, `login.md` present.
- `GET /api/p/testproject/graph` — asserts graph nodes include `login.md` but not `README.md`.

## Test files

| File | Purpose |
|---|---|
| `internal/config/config_test.go` | Unit tests for `ShouldIgnore` and `LoadProject` (Milestones 1–2) |
| `tests/integration/ignore_patterns_test.go` | Integration tests (Milestones 3–6); build tag: `integration` |
