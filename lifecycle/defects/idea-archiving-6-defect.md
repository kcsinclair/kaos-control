---
title: Dot-prefixed directories created at runtime are indexed due to watcher walk race condition
type: defect
status: draft
lineage: idea-archiving
parent: lifecycle/tests/idea-archiving-5-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

## Reproduction Steps

1. Run the Go integration test suite containing `TestDotDirExclusion_RuntimeCreation`:
   ```sh
   go test ./tests/integration -run TestDotDirExclusion_RuntimeCreation -tags=integration -count=1
   ```
2. Observe the test failure in `recursive_subdir_test.go`.

## Expected Behaviour

At runtime, if a dot-prefixed directory (e.g., `.trash`) and a markdown file (e.g., `dot.md`) are created together under the project's ideas folder, the watcher should skip indexing it because it is inside a dot-prefixed folder.

## Actual Behaviour

The file `.trash/dot.md` gets scanned and indexed.

## Logs / Output

```
--- FAIL: TestDotDirExclusion_RuntimeCreation (0.65s)
    recursive_subdir_test.go:376: .trash/dot.md should not be indexed (see KNOWN DEFECT comment above)
FAIL
```
