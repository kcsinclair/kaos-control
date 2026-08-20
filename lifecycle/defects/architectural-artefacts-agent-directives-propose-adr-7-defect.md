---
created: "2026-08-15T11:53:03+10:00"
title: Design and build agents missing propose ADR on deviation directive
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

# Design and build agents missing propose ADR on deviation directive

## Resolution (2026-08-15)

Added a "propose a new ADR in `lifecycle/architecture/decisions/` (type: adr)
on deviation" directive to the prompt templates of `requirements-analyst`,
`backend-developer`, `frontend-developer`, `test-developer`, and `qa` in
[lifecycle/config.yaml](../config.yaml) (planning-analyst already had it), and
the matching write path (see the sibling
[adr-write-path defect](architectural-artefacts-agent-directives-adr-write-path-7-defect.md)).
The scaffold template
[internal/initcmd/templates/config.yaml.tmpl](../../internal/initcmd/templates/config.yaml.tmpl)
got a concise version of the same directive so new projects are correct.
`TestAgentDirectives_ProposeADROnDeviation` passes.

**Answering the open questions:** (1/3) the entire fix is config data with no
`internal/**` code component and no backend plan needed — it was mis-assigned
to backend-developer; it belongs to a config-owning role/human, and developer
agents must not be granted write access to `lifecycle/config.yaml` (it defines
their own guardrails). (2) yes — this and the write-path defect are one fix on
the same file, resolved together.

Integration test `TestAgentDirectives_ProposeADROnDeviation` fails because multiple design and build agent configurations in `lifecycle/config.yaml` (`requirements-analyst`, `backend-developer`, `frontend-developer`, `test-developer`, and `qa`) do not include prompt directives instructing the agent to propose an Architectural Decision Record (ADR) under `lifecycle/architecture/decisions/` when deviating from architecture, violating requirement FR-22.

## Reproduction Steps

1. Run the integration test from the repository root:
   ```bash
   go test -tags integration -v ./tests/integration/ -run '^TestAgentDirectives_ProposeADROnDeviation$'
   ```
2. Observe failures for `requirements-analyst`, `backend-developer`, `frontend-developer`, `test-developer`, and `qa`.

## Expected Behaviour

Every design/build agent's prompt template in `lifecycle/config.yaml` should direct it to propose an ADR in `lifecycle/architecture/decisions/` rather than deviate silently or get stuck without proposing an ADR.

## Actual Behaviour

Only `planning-analyst` contains the propose-ADR directive. The remaining five agents (`requirements-analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`) lack directives instructing them to propose an ADR under `lifecycle/architecture/decisions/`.

## Logs / Output

```
=== RUN   TestAgentDirectives_ProposeADROnDeviation
    architecture_directives_test.go:121: agent "requirements-analyst" prompt template(s) missing a "propose an ADR in lifecycle/architecture/decisions/" directive (FR-22)
    architecture_directives_test.go:121: agent "backend-developer" prompt template(s) missing a "propose an ADR in lifecycle/architecture/decisions/" directive (FR-22)
    architecture_directives_test.go:121: agent "frontend-developer" prompt template(s) missing a "propose an ADR in lifecycle/architecture/decisions/" directive (FR-22)
    architecture_directives_test.go:121: agent "test-developer" prompt template(s) missing a "propose an ADR in lifecycle/architecture/decisions/" directive (FR-22)
    architecture_directives_test.go:121: agent "qa" prompt template(s) missing a "propose an ADR in lifecycle/architecture/decisions/" directive (FR-22)
--- FAIL: TestAgentDirectives_ProposeADROnDeviation (0.00s)
```

## Resolved Questions (answered in Resolution above)

Raised by backend-developer, 2026-08-15. Read this defect as a backend implementation plan and traced what
`TestAgentDirectives_ProposeADROnDeviation` (`tests/integration/architecture_directives_test.go:111-124`)
actually asserts before writing any code. Stopping short of a code change
because the fix does not have a code component within this agent's scope.

1. **The entire fix is data, not code, and lives outside `internal/**` and
   `cmd/**`.** `combinedPromptText(ag)` and `hasProposeADRDirective(text)`
   (same test file) read the `prompt_template` / `system_prompt` strings
   already loaded from `lifecycle/config.yaml` via the existing
   `internal/config` loader — there is no missing Go feature to build. The
   fix is entirely: add a "propose an ADR in
   lifecycle/architecture/decisions/" sentence to the
   `requirements-analyst`, `backend-developer`, `frontend-developer`,
   `test-developer`, and `qa` prompt templates in `lifecycle/config.yaml`,
   matching what `planning-analyst` already has. That file is outside this
   agent's write scope (`internal/**`, `cmd/**` only; lifecycle artifacts are
   explicitly out of scope except for blocking this defect). Should this
   defect be reassigned to whichever role owns `lifecycle/config.yaml`
   edits (`planning-analyst`/`requirements-analyst`, or handled directly by
   product-owner), rather than backend-developer?
2. Related: `TestAgentDirectives_ADRAuthoringWritePath` (same test file,
   line 129) also fails/would fail for the same five agents on
   `allowed_write_paths` missing `lifecycle/architecture/decisions` — same
   `lifecycle/config.yaml` file, same out-of-scope situation. Is that the
   same defect/fix, or does it need its own artifact?
3. This defect has no `## Milestone` breakdown (unlike a typical
   `backend-plans/*-be.md` plan), and was assigned directly to
   backend-developer. Was a backend plan supposed to exist for this
   lineage first (with milestones scoped to `internal/**`), or is this
   defect meant to be actioned directly against `lifecycle/config.yaml` by
   a different role?

No code or config changes made pending answers.
