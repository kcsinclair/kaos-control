---
title: Lifecycle indexer does not ignore README.md files; no configurable ignore patterns in YAML config
type: defect
status: in-development
lineage: ignore-readme-files-in-lifecycle-dir
priority: normal
labels:
    - defect
    - backend
    - watcher
    - artefacts
---

# Lifecycle indexer does not ignore README.md files; no configurable ignore patterns in YAML config

## Reproduction Steps

1. Place a `README.md` file anywhere within the `lifecycle/` directory tree (e.g. `lifecycle/README.md` or `lifecycle/ideas/README.md`).
2. Start the kaos-control server (or allow the fsnotify watcher to pick up the file).
3. Observe the SQLite index and the artifact list returned by the API.

## Expected Behaviour

`README.md` files (and ideally other configurable ignore patterns) within the `lifecycle/` directory tree should be silently skipped during both the startup full-scan and the live fsnotify-driven re-index. The set of ignored filenames or glob patterns should be specifiable in `lifecycle/config.yaml` (e.g. an `ignore` or `exclude_files` list), so that projects can customise which files are excluded without code changes.

## Actual Behaviour

`README.md` files are treated as regular markdown artifacts and indexed alongside legitimate lifecycle artifacts. There is no `ignore` / `exclude` configuration option in `lifecycle/config.yaml`, so there is no way to suppress indexing of these files without modifying the source code.
