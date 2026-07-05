---
title: Test Suite: Recursive Subdirectory Support for Artifact Directories
type: test
status: draft
lineage: idea-archiving
parent: lifecycle/test-plans/idea-archiving-5-test.md
---

# Test Suite: Recursive Subdirectory Support for Artifact Directories

This test suite covers the implementation of recursive subdirectory support for artifact directories as specified in [idea-archiving-2](../requirements/idea-archiving-2.md).

## Covered Test Cases

The following test cases are implemented:

1. **Milestone 1 - Unit: path-component parsing & rel_path derivation**
   - Flat paths (`lifecycle/ideas/login.md`) → `Stage=="ideas"`, `RelPath=="login.md"`
   - Single-nested paths (`lifecycle/ideas/done/login.md`) → `Stage=="ideas"`, `RelPath=="done/login.md"`
   - Deeply-nested paths (`lifecycle/ideas/2026/q3/release-x.md`) → `Stage=="ideas"`, `RelPath=="2026/q3/release-x.md"`
   - Cross-platform separators work correctly
   - Type/status/lineage/parent are unchanged by moving between folders

2. **Milestone 2 - Integration: recursive scan, index, and API exposure**
   - Seeding nested artifacts makes them appear in API and index
   - API artifacts carry correct `rel_path` with forward slashes
   - Backward compatibility maintained for flat artifacts
   - Nested artifacts are editable via PUT and appear in graph endpoint

3. **Milestone 3 - Integration: live watcher recursion & new-subdir detection**
   - Writing a new `*.md` into an existing nested dir at runtime is picked up
   - Creating a brand-new subdirectory and artifact in one operation results in indexing and watching the directory
   - Moving an artifact between folders preserves identity

4. **Milestone 4 - Integration: dot-dir exclusion, path-safety, watch cap**
   - Hidden paths (`.md` under `lifecycle/ideas/.trash/`, `.hidden.md`) are not indexed
   - POST with `subdir` parameter creates files in subdirectories correctly
   - Path traversal attempts outside the root are rejected

5. **Milestone 5 - Integration: cross-folder lineage uniqueness**
   - `NextIndexForLineage("login")` returns same next index regardless of folder location
   - Cross-folder collisions are detected and surfaced identically to flat collisions

6. **Milestone 7 - E2E smoke: archive round-trip**
   - End-to-end flow through the running app confirming archive use-case
   - Seed artifact, move it into `archive/`, confirm in list with path, then back to root

## Test Files

- `tests/integration/recursive_subdir_test.go` - Main integration tests for recursive subdirectory support