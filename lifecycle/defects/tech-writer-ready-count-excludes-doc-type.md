---
title: tech-writer Agent ready_count Returns 0 for Approved Doc Artifacts
type: defect
status: done
lineage: tech-writer-ready-count-excludes-doc-type
created: "2026-05-16T14:00:00+10:00"
priority: normal
labels:
    - defect
    - backend
    - agents
    - tech-writer
release: KC-Release2
assignees:
    - role: backend-developer
      who: agent
---

# tech-writer Agent ready_count Returns 0 for Approved Doc Artifacts

## Reproduction Steps

1. Start the kaos-control binary with the E2E fixtures that include `lifecycle/docs/smoke-doc-approved.md` (status: `approved`).
2. Log in and call `GET /api/p/testproject/agents`.
3. Inspect the `ready_count` field for the `tech-writer` agent entry.

## Expected Behaviour

`ready_count` for the `tech-writer` agent is ≥ 1 because `smoke-doc-approved.md` (type `doc`, status `approved`) is a valid ready artifact for that agent.

## Actual Behaviour

The E2E test `Flow 08 TC3 — ready count endpoint includes approved doc for tech-writer agent` fails:

```
Error: expect(received).toBeGreaterThanOrEqual(expected)
Expected: >= 1
Received:    0
```

`GET /api/p/testproject/agents` returns `ready_count: 0` for `tech-writer` even though an approved `doc` artifact exists in the project.

Test file: `tests/e2e/flows/08-doc-queue.spec.ts:59`

## Notes

The ready-count logic likely filters candidate artifacts by type and does not include `doc` as a type that `tech-writer` can work on. The agent configuration in `lifecycle/config.yaml` (or the matching logic in the backend) should be verified to ensure `doc` artifacts in `approved` status are counted as ready work for the `tech-writer` role.
