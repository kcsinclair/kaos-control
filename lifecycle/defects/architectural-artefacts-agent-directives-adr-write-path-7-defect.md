---
title: Analyst and developer agents missing allowed write path for ADR decisions
type: defect
status: approved
lineage: architectural-artefacts
parent: lifecycle/tests/architectural-artefacts-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# Analyst and developer agents missing allowed write path for ADR decisions

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
