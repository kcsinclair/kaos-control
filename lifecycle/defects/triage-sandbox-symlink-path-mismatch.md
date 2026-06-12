---
title: 'Triage execute fails: sandbox resolves symlinks but ideasAbsDir uses raw ProjectRoot'
type: defect
status: done
lineage: triage-sandbox-symlink-path-mismatch
created: "2026-06-12T00:00:00+10:00"
labels:
    - defect
release: KC-Release3
assignees:
    - role: backend-developer
      who: agent
---

# Triage execute fails: sandbox resolves symlinks but ideasAbsDir uses raw ProjectRoot

## Reproduction Steps

1. Run the integration test suite on macOS:
   ```
   go test -tags integration ./tests/integration/... -run "TestTriageAPI_Success|TestTriageStartup|TestTriageWatcher"
   ```
2. Observe all triage execute paths fail immediately with:
   ```
   path "lifecycle/ideas/<slug>.md" is outside lifecycle/ideas/
   ```

## Expected Behaviour

Triage runs succeed when the artifact path is legitimately inside `lifecycle/ideas/`. Status transitions from `raw` to `draft`.

## Actual Behaviour

`triage/run.go:execute()` fails the path-boundary check for every artifact on macOS, even valid ones. The run is recorded as `failed` with `stderr_tail` = `path "lifecycle/ideas/X.md" is outside lifecycle/ideas/`.

## Root Cause

`internal/triage/run.go:57–63`:

```go
absPath, err := sandbox.Resolve(m.deps.ProjectRoot, relPath)
// ...
ideasAbsDir := filepath.Join(m.deps.ProjectRoot, "lifecycle", "ideas")
if !strings.HasPrefix(absPath, ideasAbsDir+string(filepath.Separator)) {
    return fmt.Errorf("path %q is outside lifecycle/ideas/", relPath)
}
```

`sandbox.Resolve` internally calls `filepath.EvalSymlinks(projectRoot)` before building the absolute path, so on macOS it returns a path rooted at `/private/var/folders/…`. But `ideasAbsDir` is built from the original (unresolved) `m.deps.ProjectRoot`, which on macOS is rooted at `/var/folders/…`. The `strings.HasPrefix` check therefore always fails.

## Failing Tests

- `TestTriageAPI_Success`
- `TestTriageFailure_MalformedJSON` (receives sandbox error instead of JSON parse error)
- `TestTriageStartup_SingleRawIdea`
- `TestTriageStartup_MultipleRawWithCap`
- `TestTriageWatcher_CreateRawIdea_TriageRuns`
- `TestTriageWatcher_RapidWrites_OneRun`
- `TestTriageWatcher_ReRunAfterStatusReset`

## Fix

In `triage/run.go`, resolve symlinks in `ProjectRoot` before constructing `ideasAbsDir`:

```go
resolvedRoot, err := filepath.EvalSymlinks(m.deps.ProjectRoot)
if err != nil {
    resolvedRoot = filepath.Clean(m.deps.ProjectRoot)
}
ideasAbsDir := filepath.Join(resolvedRoot, "lifecycle", "ideas")
```
