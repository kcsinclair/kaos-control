---
title: Fixture config.yaml uses `roles:` instead of `role:` for tech-writer agent
type: defect
status: in-development
lineage: end-to-end-smoke-tests
release: KC-Release2
assignees:
    - role: backend-developer
      who: agent
---

# Fixture config.yaml uses `roles:` instead of `role:` for tech-writer agent

## Reproduction Steps

1. Build `dist/kaos-control` with `make build`.
2. Run the E2E suite: `cd tests/e2e && pnpm test`.
3. Observe that 14 of 19 tests fail.

All failing tests that use the `kctest` or `loggedInPage` fixture share the same
error: the main content area renders `"project not found: testproject"` instead of
any project UI.

## Expected Behaviour

The server loads `testproject` on startup and all project-scoped API calls
(`/api/p/testproject/...`) return valid data. Tests depending on the project
(flows 01–05, 06 TC1–TC3, 07 TC3, 08 TC1–TC3, 09 TC1–TC3) exercise the
application rather than hitting a 404.

## Actual Behaviour

`project.Open` fails during server startup because `config.LoadProject` calls
`validateProject`, which returns:

```
project config: agent "tech-writer" has no roles
```

`main.go` logs the error and skips the project (`continue`), so `testproject` is
never added to the live projects map. Every subsequent API call to
`/api/p/testproject/...` returns HTTP 404 with body
`{"code":"project_not_found","message":"project not found: testproject"}`.

The sidebar still shows "Project testproject" because `GET /api/projects` reads
the registration file from disk rather than the runtime map, but all data API
calls fail.

## Root Cause

`internal/config/config.go` defines `AgentConfig.Roles` with the struct tag
`yaml:"role"` (singular):

```go
// internal/config/config.go
type AgentConfig struct {
    Name  string   `yaml:"name"`
    Roles []string `yaml:"role"`   // ← yaml key is "role", not "roles"
    ...
}
```

`tests/e2e/fixtures/lifecycle/config.yaml` uses `roles:` (plural) for the
`tech-writer` agent:

```yaml
  - name: tech-writer
    roles:           # ← WRONG — should be "role:"
      - tech-writer
    driver: shell-stub
```

`gopkg.in/yaml.v3` silently ignores unknown keys, so `Roles` remains empty and
`validateProject` raises the error above.

The `stub-agent` entry in the same file correctly uses `role:` (singular) and
loads without error.

## Affected Tests

All 14 project-dependent tests:

- `flows/01-login.spec.ts` — loggedInPage lands on dashboard with non-zero Lifecycle Total
- `flows/02-edit-save.spec.ts` — saves content to disk and fires file.changed WS event
- `flows/03-transition.spec.ts` — transitions artifact status, writes frontmatter, commits git
- `flows/04-agent-run.spec.ts` — stub agent runs and reaches done status without Claude Code
- `flows/05-graph-click.spec.ts` — map view renders expected node count and clicking a node navigates
- `flows/06-doc-request.spec.ts` — TC1, TC2, TC3
- `flows/07-doc-new.spec.ts` — TC3: standalone doc creation flow
- `flows/08-doc-queue.spec.ts` — TC1, TC2, TC3
- `flows/09-doc-graph.spec.ts` — TC1, TC2, TC3

## Logs / Output

```
Running 19 tests using 4 workers

  ✘   2 flows/02-edit-save.spec.ts:9:3 › Flow 02 — Edit and save artifact › saves content to disk and fires file.changed WS event (8.5s)
  ✘   5 flows/04-agent-run.spec.ts:4:3 › Flow 04 — Agent run › stub agent runs and reaches done status without Claude Code (8.2s)
  ✘   3 flows/03-transition.spec.ts:9:3 › Flow 03 — Status transition › transitions artifact status, writes frontmatter, commits git (10.5s)
  ✘   6 flows/01-login.spec.ts:9:3 › Flow 01 — Login and project access › loggedInPage lands on project dashboard with non-zero Lifecycle Total (10.3s)
  ✘   8 flows/06-doc-request.spec.ts:9:3 › Flow 06 — "Request docs" button (FR1) › TC1 (10.2s)
  ✘  10 flows/08-doc-queue.spec.ts:9:3 › Flow 08 — Queue Work for doc artifacts (NFR3) › TC1 (10.2s)
  ✘   7 flows/05-graph-click.spec.ts:4:3 › Flow 05 — Graph node click › map view renders expected node count and clicking a node navigates (15.2s)
  ...
  15 failed
```

Error context from `03-transition`:
```
- main:
  - button "← artifacts"
  - text: "project not found: testproject"
```

Error context from `08-doc-queue TC3`:
```
Error: expect(received).toBe(expected) // Object.is equality
Expected: 200
Received: 404
```
(The `GET /api/p/testproject/agents` returns 404 because the project was not registered.)

## Fix

In `tests/e2e/fixtures/lifecycle/config.yaml`, change `roles:` to `role:` for the
`tech-writer` agent entry:

```yaml
  - name: tech-writer
    role:            # ← was "roles:"
      - tech-writer
    driver: shell-stub
```
