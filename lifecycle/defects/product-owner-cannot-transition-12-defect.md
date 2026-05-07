---
title: "GET /allowed-targets returns 404 — endpoint not implemented"
type: defect
status: done
lineage: innovation-maker
parent: lifecycle/tests/product-owner-cannot-transition-11-test.md
labels:
    - defect
assignees:
    - role: backend-developer
      who: agent
release: May2026
---

# GET /allowed-targets returns 404 — endpoint not implemented

## Reproduction Steps

1. Start the server with `make run`.
2. Authenticate as any user (e.g. `admin@test.local`).
3. Seed an artifact at `lifecycle/requirements/po-targets.md` with `status: in-development`.
4. Issue `GET /api/p/testproject/artifacts/lifecycle/requirements/po-targets.md/allowed-targets`.
5. Observe the response.

## Expected Behaviour

HTTP 200 with a JSON body `{"targets": ["in-qa", "done", "approved", "rejected", "abandoned", "blocked"]}` (or the role-appropriate subset).

For unauthenticated requests, HTTP 401 Unauthorized.

## Actual Behaviour

HTTP 404 `{"error":{"code":"not_found","message":"artifact not found"}}` for all users, including unauthenticated.

The four Milestone 3 integration tests all fail:

- `TestAllowedTargetsProductOwnerGetsSuperSet` — expected 200, got 404
- `TestAllowedTargetsDevUserSubset` — expected 200, got 404
- `TestAllowedTargetsQAUserDoesNotIncludeInQA` — expected 200, got 404
- `TestAllowedTargetsUnauthenticatedReturns401` — expected 401, got 404

## Root Cause

The `GET /artifacts/*` dispatcher in `internal/http/server.go:90–97` only handles the `/history` suffix; all other GET paths fall through to `handleGetArtifact`. The path passed to the handler includes the trailing `/allowed-targets` segment, so the artifact is not found.

There is no `handleAllowedTargets` function in `internal/http/transition.go` and no routing entry for the `/allowed-targets` suffix.

## Logs / Output

```
--- FAIL: TestAllowedTargetsProductOwnerGetsSuperSet (0.13s)
    product_owner_transition_test.go:210: expected status 200, got 404: {"error":{"code":"not_found","message":"artifact not found"}}
--- FAIL: TestAllowedTargetsDevUserSubset (0.11s)
    product_owner_transition_test.go:247: expected status 200, got 404: {"error":{"code":"not_found","message":"artifact not found"}}
--- FAIL: TestAllowedTargetsQAUserDoesNotIncludeInQA (0.11s)
    product_owner_transition_test.go:290: expected status 200, got 404: {"error":{"code":"not_found","message":"artifact not found"}}
--- FAIL: TestAllowedTargetsUnauthenticatedReturns401 (0.09s)
    product_owner_transition_test.go:323: expected 401 for unauthenticated request, got 404
```

## Fix Guidance

1. Add a `handleAllowedTargets` handler to `internal/http/transition.go` that:
   - Returns 401 if the user is not authenticated.
   - Looks up the artifact by path (stripping `/allowed-targets` from the param).
   - Returns 404 if the artifact is not found.
   - Calls `p.Workflow.AllowedTargets(row.Status, userRoles)`.
   - Responds with `{"targets": [...]}`.
2. Wire it in `internal/http/server.go` inside the `GET /artifacts/*` dispatcher (add a `strings.HasSuffix(param, "/allowed-targets")` branch before the fallthrough to `handleGetArtifact`).
