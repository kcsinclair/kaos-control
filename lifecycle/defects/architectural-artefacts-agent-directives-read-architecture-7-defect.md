---
created: "2026-08-15T11:32:40+10:00"
title: QA agent prompt template missing read lifecycle/architecture directive
type: defect
status: done
lineage: architectural-artefacts
parent: lifecycle/tests/architectural-artefacts-6-test.md
labels:
    - defect
release: KC-Release5
assignees:
    - role: backend-developer
      who: agent
---

# QA agent prompt template missing read lifecycle/architecture directive

Integration test `TestAgentDirectives_ReadArchitectureFirst` fails because the `qa` agent configured in `lifecycle/config.yaml` does not have a prompt template directing it to read `lifecycle/architecture/` before performing work, violating requirement FR-21.

## Reproduction Steps

1. Run the integration test from the repository root:
   ```bash
   go test -tags integration -v ./tests/integration/ -run '^TestAgentDirectives_ReadArchitectureFirst$'
   ```
2. Observe failure reporting missing directive text for agent `qa`.

## Expected Behaviour

Every design/build agent configuration in `lifecycle/config.yaml` (`requirements-analyst`, `planning-analyst`, `backend-developer`, `frontend-developer`, `test-developer`, `qa`) must include a prompt directive instructing the agent to read `lifecycle/architecture/` before starting work.

## Actual Behaviour

The `qa` agent prompt template in `lifecycle/config.yaml` lacks any reference to reading `lifecycle/architecture/`.

## Logs / Output

```
=== RUN   TestAgentDirectives_ReadArchitectureFirst
    architecture_directives_test.go:103: agent "qa" prompt template(s) missing a "read lifecycle/architecture/" directive (FR-21)
--- FAIL: TestAgentDirectives_ReadArchitectureFirst (0.00s)
```
