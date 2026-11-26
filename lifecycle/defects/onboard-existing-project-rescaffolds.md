---
title: "Onboarding an existing project re-scaffolds it and 500s when its config is invalid"
type: defect
status: draft
lineage: onboard-existing-project-rescaffolds
created: "2026-08-28T12:22:00+10:00"
priority: medium
labels:
    - defect
    - onboarding
    - initcmd
    - directives
---

## Summary

Registering a project in "existing" mode runs the full scaffold even when the
directory is already initialised. Two consequences:

1. It **writes into a directory it should leave alone** — `EnsureArchitectureScaffold`
   creates the architecture catalog, and `directives.Generate` writes AGENTS.md,
   CLAUDE.md and GEMINI.md ([internal/initcmd/initcmd.go:112-136](../../internal/initcmd/initcmd.go#L112-L136)).
   Neither is gated on whether the project was already initialised.

2. It **hard-fails with HTTP 500** when the existing `lifecycle/config.yaml` does
   not pass validation, because `directives.Generate` loads the project config:

   ```
   POST /api/projects -> 500
   {"error":{"code":"scaffold_failed",
             "message":"generating agent directives: loading project config:
                        project config: stages must not be empty"}}
   ```

Pointing kaos-control at a directory whose config is incomplete or hand-edited
therefore fails registration outright, with an error naming an internal step
rather than telling the operator their config is invalid.

## Impact

The intended contract (NFR2) is that onboarding an already-initialised directory
reports `alreadyInitialised: true` and leaves the directory unmodified. Neither
holds: files are written, and an invalid config aborts the whole request.

## Detected by

`TestOnboard_ExistingMode_AlreadyInitialised` — expects 200 with
`alreadyInitialised: true` and the config file byte-identical afterwards; gets
500. The test is correct.

## Proposed fix

Short-circuit the scaffold when the project is detected as already initialised —
return `alreadyInitialised: true` without calling `EnsureArchitectureScaffold` or
`directives.Generate`. Refreshing directives on an existing project is what
`init --refresh-directives` is for, and should stay an explicit action.

Separately, when a config does fail to load during registration, the API should
return 400 with the validation message rather than 500 with a wrapped internal
step name.
