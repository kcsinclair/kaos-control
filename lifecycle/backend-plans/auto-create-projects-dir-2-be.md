---
title: "Backend Plan: Auto-Create Projects Directory on First Run"
type: plan-backend
status: draft
lineage: auto-create-projects-dir
parent: lifecycle/ideas/auto-create-projects-dir.md
---

# Backend Plan: Auto-Create Projects Directory on First Run

This plan ensures the `~/.kaos-control/projects/` directory is created automatically during startup, before any config or project loading occurs. Currently `LoadProjectRegistry` silently returns an empty list when the directory is missing (line 184–186 of `internal/config/config.go`), which is safe but hides a misconfigured or fresh-install state. The fix is small and surgical.

## Milestone 1 — Ensure projects directory in `LoadApp`

### Description

Add an `os.MkdirAll` call for `ProjectsDir` inside `config.LoadApp()`, immediately after the config is loaded/defaulted and before the function returns. This guarantees the directory exists before `run()` calls `LoadProjectRegistry`.

### Files to change

- `internal/config/config.go` — in `LoadApp()`, after `validateApp()` succeeds and after `DataDir` is resolved (around line 107), add:

```go
if err := os.MkdirAll(cfg.ProjectsDir, 0o700); err != nil {
    return nil, fmt.Errorf("creating projects dir %s: %w", cfg.ProjectsDir, err)
}
```

Use permission `0o700` (owner-only) since this directory holds per-project registration files which may contain local paths. This is consistent with the idea's suggestion of 0700/0750.

### Acceptance criteria

- [ ] `LoadApp()` creates `~/.kaos-control/projects/` if it does not exist.
- [ ] `LoadApp()` succeeds if the directory already exists (idempotent).
- [ ] Permission on newly created directory is `0o700`.
- [ ] `go build ./...` and `go vet ./...` pass.
- [ ] No changes to `cmd/kaos-control/main.go` are required — the directory is ready before `LoadProjectRegistry` is called.

## Milestone 2 — Ensure data directory in `LoadApp`

### Description

Apply the same treatment to `DataDir`. While not explicitly requested by [[auto-create-projects-dir]], `DataDir` has the same first-run gap and is set in the same code path. Creating it here avoids a separate future fix.

### Files to change

- `internal/config/config.go` — immediately after the `ProjectsDir` mkdir, add:

```go
if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
    return nil, fmt.Errorf("creating data dir %s: %w", cfg.DataDir, err)
}
```

### Acceptance criteria

- [ ] `LoadApp()` creates `~/.kaos-control/data/` if it does not exist.
- [ ] `LoadApp()` succeeds if the directory already exists (idempotent).
- [ ] `go build ./...` and `go vet ./...` pass.

## Cross-links

- [[auto-create-projects-dir]] — originating idea.
- The frontend plan (`auto-create-projects-dir-3-fe`) has no direct dependency on this change; the backend change is invisible to the SPA.
- The test plan (`auto-create-projects-dir-4-test`) will verify both milestones via integration tests.
