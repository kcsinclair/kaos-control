---
title: Design and build agents missing propose ADR on deviation directive
type: defect
status: approved
lineage: architectural-artefacts
parent: lifecycle/tests/architectural-artefacts-6-test.md
labels: [defect]
assignees:
  - role: backend-developer
    who: agent
---

# Design and build agents missing propose ADR on deviation directive

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
