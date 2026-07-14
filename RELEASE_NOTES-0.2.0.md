# kaos-control 0.2.0

*Released from v0.1.3 (2026-06-14) — a large release (~405 commits).*

Renumbered from the originally-planned `0.1.4`: this release adds several
features and some behaviour changes, which is a **minor** bump in 0.x, not a
patch.

## Highlights

- **Artifacts in subdirectories** — organise `lifecycle/` into nested folders.
- **Guided Open Questions resolution** — answer agent questions in a modal instead of hand-editing markdown.
- **DevOps run history + a real CLI** — see past pipeline runs, and drive them from the terminal.
- **`claude-env` driver** — point agents at local or alternative model endpoints.
- **Major reliability & security hardening** — including 13 `x/crypto` CVEs.

## New features

### Recursive subdirectory support
Artifacts can now live in nested folders under any stage. The artifact list
shows a path chip and a folder breadcrumb, creation supports a target subdir
(sandbox-checked against path traversal), and the indexer/watcher were hardened
for deep trees, dot-directories, and a watched-directory cap.

### Open Questions — guided resolution
When an agent blocks on questions, you now get:
- a per-artefact **banner + Resolve action**,
- a **guided modal** to answer questions one at a time,
- a **menu-bar badge** with a live "awaiting your answers" count,
- a **blocked/awaiting** filter in the artifact list.

Answers are written back in a configurable format, the `## Open Questions`
heading becomes `## Resolved Questions`, and the artefact is unblocked and routed
onward — no more hand-editing markdown and manually renaming headings.

### DevOps pipeline run history
Pipeline runs are persisted with:
- an **expandable per-row run log**,
- **latest-run summary badges** on pipeline cards and group headers,
- **retention / pruning** of old run records.

### DevOps CLI
`kaos-control devops list | status <job> | run <pipeline> [--follow]`, with
loopback-trusted **Linux-user identity mapping** so local runs need no token,
and CLI-originated runs rendering correctly in run history.

### `claude-env` agent driver
Run Claude-style agents against a custom `ANTHROPIC_BASE_URL` / auth token — for
local models, gateways, or alternative endpoints — with the auth token **never
exposed to the browser**.

### Priority & release inheritance
Plans, rejection artifacts, and agent-generated children now inherit their
parent's `priority` and `release` fields.

### Daemon flag
A bare `kaos-control` invocation prints usage and exits 2; the server only starts
behind `-d` / `--daemon` / `serve` — safer, clearer invocation.

### Resilient agents & live queue
- Transient auth (401) failures **fail fast and re-enqueue** without pausing the
  whole queue.
- `queue.added` (on enqueue) and `queue.cancelled` (on cancel) now broadcast, so
  every connected client's queue view updates in real time.

## Reliability & fixes

- **Approve now sticks** — fixed an index/disk status desync where a content-hash
  guard locked in a stale status.
- **Watcher correctness** — runtime-created dot-directories (e.g. `.trash/`) are
  no longer indexed.
- **No more false "truncated stream"** — fixed a `cmd.Wait()` / stdout-scanner
  race that wrongly failed fast-exiting runs and dropped their metrics.
- **Smarter auto-block** — a placeholder / `None` Open Questions section no
  longer blocks an artefact.
- **Test-runner** now runs every suite synchronously (no parking on deferred
  wakeups) and files deduped defects.
- Frontend test fetch-leaks guarded; multiple integration/build repairs from
  concurrent agent work.

## Security

- **`golang.org/x/crypto` 0.51 → 0.52** — clears **13 CVEs (7 critical)** across
  `x/crypto/ssh` (auth bypass, FIDO/U2F presence-check bypass, key-constraint
  bypasses, DoS/deadlocks).
- Dependency bumps: `vite` 6.4.3, `echarts` 6.1, `js-yaml` 4.3,
  `golang.org/x/net` 0.55, `markdown-it` 14.2, `undici` override.
- Go toolchain → **1.25.12**; `make lint` fully green across all five stages
  (go vet, staticcheck, govulncheck, gosec, gitleaks).

## Notes

- See [road-to-1.0.md](plans/road-to-1.0.md) for the versioning rationale and the
  criteria that would earn a 1.0.
