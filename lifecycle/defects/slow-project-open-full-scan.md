---
title: Project Open Takes ~44s Due to Unconditional Full File Scan
type: defect
status: in-development
lineage: slow-project-open-full-scan
created: "2026-04-28T09:14:56+10:00"
priority: normal
labels:
    - defect
    - backend
    - go
    - enhancement
assignees:
    - role: backend-developer
      who: agent
---

# Project Open Takes ~44s Due to Unconditional Full File Scan

## Reproduction Steps

1. Register a project with ~108 markdown artifacts under `lifecycle/`.
2. Open (or restart) the kaos-control server so the project is loaded.
3. Observe the server logs — two entries are emitted:
   - `opening project` at t=0
   - `scan complete files=108 duration=44230763667` (~44 seconds later)

## Expected Behaviour

- Project open should complete in well under 5 seconds for a ~100-file project.
- The indexer should skip re-parsing any file whose `mtime` has not advanced beyond the timestamp already stored in the SQLite cache, avoiding redundant disk reads and markdown parses.
- The `scan complete` log should report duration in human-readable seconds (e.g. `44.23s`) rather than raw nanoseconds, and should also log the number of files skipped vs re-indexed.

## Actual Behaviour

- Every project open triggers an unconditional full scan of all `lifecycle/**/*.md` files regardless of whether they have changed since the last index run.
- 108 files take ~44 seconds to scan (≈408 ms per file on average), making the tool effectively unusable until the scan finishes.
- Duration is logged as a raw nanosecond integer (`44230763667`), which is not human-readable.
- No indication is given of which files are the slowest to process or how many could have been skipped.
