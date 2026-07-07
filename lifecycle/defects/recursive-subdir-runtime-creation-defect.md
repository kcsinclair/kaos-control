---
title: .trash/dot.md indexed during runtime creation (TestDotDirExclusion_RuntimeCreation failure)
type: defect
status: done
lineage: idea-archiving
parent: lifecycle/tests/idea-archiving-5-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

## Resolution (2026-07-07)

Fixed. `watcher.shouldProcess` only checked the filename's leading dot, so files
inside a dot-directory created at runtime (`.trash/dot.md`) slipped through — the
runtime dir-create WalkDir indexed them even though `addDirRecursive` skipped
*watching* the dot-dir. `shouldProcess` now rejects any path with a dot-prefixed
directory ancestor, mirroring the startup scan's SkipDir-on-dotdir behaviour.
`TestDotDirExclusion_RuntimeCreation` passes; watcher units green. Canonical
defect for this bug — `idea-archiving-6` is a duplicate.

## Reproduction Steps

1. Run the integration test suite: `go test ./tests/integration -tags integration -run TestDotDirExclusion_RuntimeCreation -v`
2. The test creates a directory `.trash` and a file inside it `dot.md` under `lifecycle/ideas/` at runtime.
3. The internal watcher processes the runtime creation and indexes `.trash/dot.md`.
4. Observe the test fails because the directory is dot-prefixed and should have been excluded from indexing.

## Expected Behaviour

Files under any dot-prefixed directory (e.g., `.trash`) created at runtime should be ignored by the file watcher and not indexed.

## Actual Behaviour

The file `.trash/dot.md` is indexed when the `.trash` directory is created at runtime, because `internal/watcher/watcher.go`'s new-directory race-handling fallback loop (`filepath.WalkDir`) walks and indexes pre-existing files under the newly-created directory without validating whether the directory itself or its ancestors are dot-prefixed.

## Logs / Output

```
--- FAIL: TestDotDirExclusion_RuntimeCreation (0.65s)
    recursive_subdir_test.go:376: .trash/dot.md should not be indexed (see KNOWN DEFECT comment above)
```
