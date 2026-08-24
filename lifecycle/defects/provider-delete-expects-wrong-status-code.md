---
title: "TestProviderAPI_Delete expects 200 OK but the API contract is 204 No Content"
type: defect
status: draft
lineage: open-provider-support
parent: lifecycle/tests/open-provider-support-6-test.md
labels:
    - defect
created: "2026-08-25T09:58:00+10:00"
assignees:
    - role: test-developer
      who: agent
---

# TestProviderAPI_Delete expects 200 OK but the API contract is 204 No Content

## Reproduction Steps

1. From the repository root, run:
   ```
   go test -tags=integration ./tests/integration/... -run TestProviderAPI_Delete -v
   ```
2. Observe the assertion failure on the response status code.

## Expected Behaviour

`tests/integration/provider_api_test.go:361-363` asserts
`DELETE /api/providers/{name}` returns `200 OK`.

## Actual Behaviour

`internal/http/providers.go:276` (`handleDeleteProvider`) returns
`http.StatusNoContent` (204) with an empty body on successful delete. This is
consistent with:
- other no-body delete handlers in the same package
  (`handleDeleteSchedulerJob` at `internal/http/scheduler.go:24`,
  `handleDeleteArtifact` at `internal/http/write.go:49`), and
- the frontend API client, which explicitly special-cases 204
  (`web/src/api/client.ts:57`, `if (res.status === 204) return undefined as T`)
  and the `deleteProvider` wrapper (`web/src/api/providers.ts:28-31`), which
  never reads a response body and synthesizes its own `{ ok, deleted }` return
  value.

Neither `lifecycle/backend-plans/open-provider-support-3-be.md` nor
`lifecycle/test-plans/open-provider-support-5-test.md` mandates a specific
status code for this endpoint, and no architecture standard governs REST
status-code conventions here, so this is not an architecture deviation — the
backend and frontend already agree with each other; only the integration test
disagrees with both.

```
provider_api_test.go:362: expected 200 OK, got 204
--- FAIL: TestProviderAPI_Delete (0.12s)
```

## Suggested Fix

Update the assertion in `tests/integration/provider_api_test.go:361-363` to
expect `http.StatusNoContent` (204) instead of `http.StatusOK`, matching the
actual and frontend-verified handler contract.

## Logs / Output

```
=== RUN   TestProviderAPI_Delete
2026/08/25 09:56:53 INFO http method=DELETE path=/api/providers/deletable-prov status=204 bytes=0 duration=486.667µs request_id=loki.local/kngQ5oxl7R-000055
    provider_api_test.go:362: expected 200 OK, got 204
--- FAIL: TestProviderAPI_Delete (0.12s)
```
