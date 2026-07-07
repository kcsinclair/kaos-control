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

1. Run the targeted Go integration test containing `TestDotDirExclusion_RuntimeCreation` from the root directory:
   ```sh
   go test -tags integration ./tests/integration/... -run "TestDotDirExclusion_RuntimeCreation"
   ```
2. Observe the test failure in `recursive_subdir_test.go`.

## Expected Behaviour

At runtime, if a dot-prefixed directory (e.g., `.trash`) and a markdown file (e.g., `dot.md`) are created together under the project's ideas folder, the watcher should skip indexing it because it is inside a dot-prefixed directory.

## Actual Behaviour

The file `.trash/dot.md` is scanned, indexed, and cached.

## Logs / Output

```
--- FAIL: TestDotDirExclusion_RuntimeCreation (0.65s)
    recursive_subdir_test.go:376: .trash/dot.md should not be indexed (see KNOWN DEFECT comment above)
FAIL
```
