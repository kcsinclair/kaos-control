---
title: TestOnboard_ExistingMode_AlreadyInitialised fails due to invalid config fixture and wrong status assertion
type: defect
status: draft
lineage: new-project-init-directory-options
parent: lifecycle/tests/new-project-init-directory-options-6-test.md
labels:
    - defect
created: "2026-08-20T19:48:00+10:00"
assignees:
    - role: test-developer
      who: agent
---

# TestOnboard_ExistingMode_AlreadyInitialised fails due to invalid config fixture and wrong status assertion

## Reproduction Steps

1. Run `go test -tags integration ./tests/integration/... -run TestOnboard_ExistingMode_AlreadyInitialised -v`
2. Observe test failure at `project_init_directory_options_test.go:283`.

## Expected Behaviour

`TestOnboard_ExistingMode_AlreadyInitialised` tests the onboarding flow when pointed at an existing directory that is already initialised as a kaos-control project. It should:
1. Seed a valid initialised project fixture (e.g. using `initcmd.ScaffoldProject` or writing valid `lifecycle/config.yaml` with required stages and roles).
2. Issue `POST /api/projects` with `mode: "existing"`.
3. Expect HTTP `201` (`http.StatusCreated`, which `handleCreateProject` returns on success) with `alreadyInitialised: true` in the response payload.
4. Verify that pre-existing configuration files remain unmutated on disk.

## Actual Behaviour

1. The test seeds an invalid dummy `lifecycle/config.yaml` fixture containing only `stages: []\n`.
2. When `POST /api/projects` processes the request, `handleCreateProject` runs `initcmd.ScaffoldProject`, which calls `directives.Generate` to establish agent directives.
3. `directives.Generate` calls `config.LoadProject(projectRoot)` to determine configured agent drivers. Because `stages` is empty, `config.LoadProject` rejects the file with `"project config: stages must not be empty"`.
4. `ScaffoldProject` fails, causing `handleCreateProject` to return HTTP `500` (`scaffold_failed: generating agent directives: loading project config: project config: stages must not be empty`).
5. Additionally, line 283 asserts `requireCRUDStatus(t, resp, 200)` instead of expecting HTTP `201` (`http.StatusCreated`).

## Logs / Output

```
=== RUN   TestOnboard_ExistingMode_AlreadyInitialised
2026/08/20 19:45:58 INFO kaos-control started addr=127.0.0.1:57411 version=dev
2026/08/20 19:45:58 INFO http method=GET path=/api/health status=200 bytes=28 duration=217.5µs request_id=loki.local/LaYM0gdPpT-000001
2026/08/20 19:45:58 INFO http method=POST path=/api/auth/login status=200 bytes=116 duration=36.327916ms request_id=loki.local/LaYM0gdPpT-000002
skipped: lifecycle/config.yaml (already exists; use --force to overwrite)
2026/08/20 19:45:58 INFO http method=POST path=/api/projects status=500 bytes=143 duration=5.863583ms request_id=loki.local/LaYM0gdPpT-000003
    project_init_directory_options_test.go:283: want status 200, got 500: {"error":{"code":"scaffold_failed","message":"generating agent directives: loading project config: project config: stages must not be empty"}}
--- FAIL: TestOnboard_ExistingMode_AlreadyInitialised (0.09s)
FAIL
FAIL	github.com/kaos-control/kaos-control/tests/integration	1.506s
```
