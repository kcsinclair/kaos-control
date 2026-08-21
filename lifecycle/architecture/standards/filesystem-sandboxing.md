---
title: Filesystem access is sandboxed and traversal-safe
type: doc
status: approved
lineage: standard-filesystem-sandboxing
created: "2026-08-21T11:52:00+10:00"
labels:
    - standard
    - security
    - filesystem
---

# Standard: Filesystem access is sandboxed and traversal-safe

kaos-control resolves user- and agent-supplied paths against project roots
constantly (artifact reads/writes, promote, agent tool calls). Every such
resolution must be confined to its intended root.

## Rules

- **Resolve paths through the sandbox resolver** (`internal/sandbox/`), which
  rejects `..` traversal (`ErrPathTraversal`) and absolute-path escapes
  (`ErrAbsolutePath`). Never `filepath.Join` untrusted input onto a root and use
  it directly.
- **Confine to the project root.** Path resolution is computed against the
  symlink-resolved project root so firmlinks/symlinks cannot widen scope.
- **Agent writes are scope-enforced,** not just declared: file-mutating tool
  calls are checked against the agent's `allowed_write_paths` through the same
  resolver — see [[adr-0006-mediated-agent-driver-permission-model]].
- **Fail closed.** An unresolvable or out-of-root path is an error to reject, not
  a path to best-effort clean up and proceed with.
- **The app config dir is off-limits to project operations** (NFR guard):
  project-directory operations must not resolve into `~/.kaos-control/`.
