---
title: Release Artefacts in Markdown
type: idea
status: done
lineage: release-artefacts
priority: high
release: KC-Release3
parent: lifecycle/ideas/inherit-priority-and-release.md
---

## Raw Idea

For each release maintain a simple markdown file, which kaos-control can use to keep the SQLite DB up to date.  This means that someone running KC on another computer will automatically get all the releases loaded. If database table is empty, reload from files.  On DB change, sync to disk.  etc.
