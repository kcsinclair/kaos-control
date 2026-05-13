---
title: "Test Suite — Inline Status Transition Dropdown"
type: test
status: in-qa
lineage: artefact-inline-status-change
parent: lifecycle/test-plans/artefact-inline-status-change-5-test.md
---

# Test Suite — Inline Status Transition Dropdown

Integration tests covering the full stack for the inline status transition dropdown feature: REST API → backend transition logic → WebSocket broadcast → disk persistence.

## Test files

- `tests/integration/transition_allowed_targets_test.go` — Milestone 1
- `tests/integration/transition_execute_test.go` — Milestone 2
- `tests/integration/transition_ws_test.go` — Milestone 3
- `tests/integration/transition_roles_test.go` — Milestone 4
- `tests/integration/transition_edge_cases_test.go` — Milestone 5

## Scenarios covered

### Milestone 1 — allowed-targets endpoint (`-run TestAllowedTargets`)

| Test | Scenario |
|------|----------|
| `TestAllowedTargetsDraftAnalyst` | analyst role receives "clarifying" in targets for a draft artifact |
| `TestAllowedTargetsDraftProductOwner` | product-owner sees full superset including clarifying, rejected, abandoned, blocked |
| `TestAllowedTargetsNoMatchingRoles` | user with empty project roles receives an empty targets array |
| `TestAllowedTargetsNotFound` | non-existent artifact path returns 404 |
| `TestAllowedTargetsUnauthenticated` | unauthenticated request returns 401 |

### Milestone 2 — transition execution (`-run TestTransitionExecute`)

| Test | Scenario |
|------|----------|
| `TestTransitionExecuteDraftToClarifying` | analyst user transitions draft → clarifying; response contains status "clarifying" |
| `TestTransitionExecuteForbiddenRole` | dev user cannot do draft → clarifying; 403 with error.code "forbidden" and allowed_targets hint |
| `TestTransitionExecuteInvalidTarget` | target status not in workflow graph returns 403 |
| `TestTransitionExecuteNotFound` | non-existent artifact returns 404 |
| `TestTransitionExecuteDiskUpdate` | artifact file on disk has `status: clarifying` after successful transition |
| `TestTransitionExecuteGitCommit` | git commit message matches format `transition(<lineage>): <from> → <to>` |

### Milestone 3 — WebSocket broadcast (`-run TestTransitionWebSocket`)

| Test | Scenario |
|------|----------|
| `TestTransitionWebSocketArtifactIndexed` | hub client receives `artifact.indexed` event with path, action "transitioned", from, to |
| `TestTransitionWebSocketFeedNew` | hub client receives `feed.new` event with event_type "status_transition", non-zero id, summary, timestamp |

Both tests register via `env.proj.Hub.Register` and defer `Unregister` to avoid leaks.

### Milestone 4 — Product-owner override and multi-role (`-run TestTransitionRoles`)

| Test | Scenario |
|------|----------|
| `TestTransitionRolesProductOwnerDraftToApproved` | product-owner skips draft → approved (no rule in matrix); non-owner gets 403 |
| `TestTransitionRolesMultiRoleUnionSuccess` | [analyst, backend-developer] user can do both draft→clarifying and in-development→in-qa |
| `TestTransitionRolesMultiRoleCannotDoNeither` | [analyst, backend-developer] user cannot do approved→done (approver only) |
| `TestTransitionRolesAllowedTargetsMultiRole` | allowed-targets returns union of analyst + backend-developer targets; excludes approver-only "done" |

Multi-role tests use `multiRoleCfgYAML` (defined in `transition_roles_test.go`) which remaps `qa@test.local` to `[analyst, backend-developer]`.

### Milestone 5 — Error resilience and edge cases (`-run TestTransitionEdgeCases`)

| Test | Scenario |
|------|----------|
| `TestTransitionEdgeCasesConcurrent` | second request with same target fails 403 after first succeeds (status changed) |
| `TestTransitionEdgeCasesDeletedArtifact` | artifact deleted before request returns 404 or 500, not a panic |
| `TestTransitionEdgeCasesRequiredPlansGate` | non-product-owner blocked by required_plans gate; 409 with gate_not_ready and missing list |
| `TestTransitionEdgeCasesProductOwnerBypassesGate` | product-owner advances planning → in-development with no approved plans |

Edge-case gate tests reuse `approverOnlyCfgYAML` from `required_plans_test.go`.

## Run commands

```sh
# All transition tests
go test ./tests/... -tags integration -run "TestAllowedTargets|TestTransitionExecute|TestTransitionWebSocket|TestTransitionRoles|TestTransitionEdgeCases"

# Per-milestone
go test ./tests/... -tags integration -run TestAllowedTargets
go test ./tests/... -tags integration -run TestTransitionExecute
go test ./tests/... -tags integration -run TestTransitionWebSocket
go test ./tests/... -tags integration -run TestTransitionRoles
go test ./tests/... -tags integration -run TestTransitionEdgeCases
```
