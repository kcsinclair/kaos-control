---
title: "handlePatchRelease returns 422 instead of 404 for non-existent artifact"
type: defect
status: in-development
lineage: inline-release-display-edit
parent: lifecycle/tests/inline-release-display-edit-8-test.md
labels:
  - defect
assignees:
  - role: backend-developer
    who: agent
---

# handlePatchRelease returns 422 instead of 404 for non-existent artifact

`PATCH /api/p/:project/artifacts/*path/release` validates the release name
against the database **before** checking whether the artifact file exists.
When a non-existent artifact path is combined with any release name value,
the release validation fires first and returns `422 Unprocessable Entity`
instead of the expected `404 Not Found`.

## Reproduction Steps

1. Start the server (or run the integration test below) with an empty project
   (no artifacts seeded).
2. Send:
   ```
   PATCH /api/p/testproject/artifacts/lifecycle/ideas/does-not-exist.md/release
   Content-Type: application/json

   {"release": "v1.0"}
   ```
3. Observe the response status code.

## Expected Behaviour

`404 Not Found` — the artifact path does not exist; the release name should
never be validated until the artifact is confirmed to exist.

## Actual Behaviour

`422 Unprocessable Entity`:
```json
{"error":{"code":"invalid_release","message":"release not found: v1.0"}}
```

## Root Cause

In `internal/http/write.go:530` (`handlePatchRelease`), the release name
lookup against the database (lines 554–565) happens before the
`os.ReadFile(absPath)` call (lines 573–581) that performs the artifact
existence check.

The fix is to move the artifact existence check (sandbox resolve + file read)
to before the release validation block.

## Logs / Output

```
=== RUN   TestReleasePatch_ArtifactNotFound
...
INFO http method=PATCH path=/api/p/testproject/artifacts/lifecycle/ideas/does-not-exist.md/release status=422 ...
    patch_release_test.go:209: expected status 404, got 422: {"error":{"code":"invalid_release","message":"release not found: v1.0"}}
--- FAIL: TestReleasePatch_ArtifactNotFound (0.15s)
FAIL    github.com/kaos-control/kaos-control/tests/integration  0.802s
```
