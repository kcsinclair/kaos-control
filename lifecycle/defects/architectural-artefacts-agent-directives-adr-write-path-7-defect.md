---
created: "2026-08-15T11:53:03+10:00"
title: Analyst and developer agents missing allowed write path for ADR decisions
type: defect
status: done
lineage: architectural-artefacts
parent: lifecycle/tests/architectural-artefacts-6-test.md
labels:
    - defect
release: KC-Release5
assignees:
    - role: product-owner
      who: agent
---

# Analyst and developer agents missing allowed write path for ADR decisions

## Resolution (2026-08-15)

Added `lifecycle/architecture/decisions` to `allowed_write_paths` for
`requirements-analyst`, `backend-developer`, `frontend-developer`,
`test-developer`, and **`qa`** in [lifecycle/config.yaml](../config.yaml)
(planning-analyst already had it) — the FR-13 four plus qa, per the
product-owner decision that qa may author ADRs (the FR-13 test excludes qa but
adding it does not break the test). The scaffold template
[internal/initcmd/templates/config.yaml.tmpl](../../internal/initcmd/templates/config.yaml.tmpl)
got the same paths so new projects don't inherit the gap.
`TestAgentDirectives_ADRAuthoringWritePath` passes; config + initcmd suites
green. Resolved together with the sibling
[propose-adr defect](architectural-artefacts-agent-directives-propose-adr-7-defect.md)
— same file, same fix.

**Answering the open question:** the fix was applied directly (product-owner
scope), **not** by extending `backend-developer`'s write scope to
`lifecycle/config.yaml`. Developer agents must not be able to edit the file
that defines their own permissions/prompts (privilege escalation) — so the
agent was right to block. Config-change defects should route to a
config-owning role/human at triage rather than to a developer.

Integration test `TestAgentDirectives_ADRAuthoringWritePath` fails because the agent definitions in `lifecycle/config.yaml` for `requirements-analyst`, `backend-developer`, `frontend-developer`, and `test-developer` do not include `lifecycle/architecture/decisions` in their `allowed_write_paths`, violating requirement FR-13.

## Reproduction Steps

1. Run the integration test from the repository root:
   ```bash
   go test -tags integration -v ./tests/integration/ -run '^TestAgentDirectives_ADRAuthoringWritePath$'
   ```
2. Observe failures for the four analyst and developer agents.

## Expected Behaviour

All analyst and developer agents permitted to propose an ADR (`requirements-analyst`, `planning-analyst`, `backend-developer`, `frontend-developer`, `test-developer`) must have `lifecycle/architecture/decisions` included in their `allowed_write_paths` in `lifecycle/config.yaml`.

## Actual Behaviour

Only `planning-analyst` has `lifecycle/architecture/decisions` configured in its `allowed_write_paths`. The other four agents (`requirements-analyst`, `backend-developer`, `frontend-developer`, `test-developer`) do not have write access to this path.

## Logs / Output

```
=== RUN   TestAgentDirectives_ADRAuthoringWritePath
    architecture_directives_test.go:145: agent "requirements-analyst" missing "lifecycle/architecture/decisions" in allowed_write_paths (FR-13)
    architecture_directives_test.go:145: agent "backend-developer" missing "lifecycle/architecture/decisions" in allowed_write_paths (FR-13)
    architecture_directives_test.go:145: agent "frontend-developer" missing "lifecycle/architecture/decisions" in allowed_write_paths (FR-13)
    architecture_directives_test.go:145: agent "test-developer" missing "lifecycle/architecture/decisions" in allowed_write_paths (FR-13)
--- FAIL: TestAgentDirectives_ADRAuthoringWritePath (0.00s)
```

## Resolved Questions (answered in Resolution above)

I was dispatched as the `backend-developer` agent with write scope limited to
`internal/**` and `cmd/**` (plus explicitly blocking this artifact). I could
not implement this defect within that scope:

1. `TestAgentDirectives_ADRAuthoringWritePath` (`tests/integration/architecture_directives_test.go:129`)
   calls `loadRealProjectConfig`, which parses **this repository's own**
   `lifecycle/config.yaml` via `config.LoadProject`. The fix the test requires
   is adding `lifecycle/architecture/decisions` to the `allowed_write_paths`
   list for the `requirements-analyst`, `backend-developer`,
   `frontend-developer`, and `test-developer` agent entries in that file.
   `lifecycle/config.yaml` is a lifecycle artifact, not `internal/**` or
   `cmd/**` — outside the write scope I was given.
2. I confirmed there is no code-level fix that substitutes for the data
   change: `internal/config/config.go`'s `LoadProject` parses
   `allowed_write_paths` verbatim from YAML with no default-injection, and
   `internal/agent/policy.Evaluate` only consults the `AllowedPaths` it is
   handed — neither has role-based defaults to patch in Go.
3. Separately, I found the same gap in the scaffold template
   `internal/initcmd/templates/config.yaml.tmpl` (used to generate
   `lifecycle/config.yaml` for *new* projects): its `requirements-analyst`,
   `backend-developer`, `frontend-developer`, and `test-developer` entries
   also lack `lifecycle/architecture/decisions` in `allowed_write_paths`
   (only `planning-analyst` has it, mirroring the bug in this repo's own
   config). That file is in-scope for me (`internal/**`), but fixing only
   the template would not make the failing test pass, since the test reads
   this repo's live `lifecycle/config.yaml`, not the template.

Question for product-owner: should the `lifecycle/config.yaml` edit be
reassigned to an agent with lifecycle-artifact write access (e.g.
planning-analyst or a config-owning role), with the template fix
(`internal/initcmd/templates/config.yaml.tmpl`) split out as a separate
backend-developer task? Or should `backend-developer`'s
`allowed_write_paths` be extended to cover `lifecycle/config.yaml` for
cases like this?
