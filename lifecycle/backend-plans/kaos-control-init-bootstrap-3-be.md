---
title: 'Backend Plan: Init Bootstrap — Config Defaults, DevOps Scaffold, and Pipeline Creation API'
type: plan-backend
status: approved
lineage: kaos-control-init-bootstrap
parent: lifecycle/requirements/kaos-control-init-bootstrap-2.md
---

# Backend Plan: Init Bootstrap — Config Defaults, DevOps Scaffold, and Pipeline Creation API

This plan implements the backend changes for [[kaos-control-init-bootstrap]]: expanding the init scaffold to include a `devops/` directory and a sample pipeline, enriching the config template with idea-capture, kanban, and dashboard defaults, and adding a `POST` endpoint for creating pipeline definitions from the UI.

---

## Milestone 1: Add `devops/` directory to scaffold

### Description

Add `devops` to the list of directories created by `kaos-control init` so that new projects have a place for pipeline YAML files from the start.

### Files to change

- `internal/initcmd/scaffold.go` — append `"devops"` to the `lifecycleDirs` slice (after `"tests"`).

### Acceptance criteria

- [ ] `scaffoldDirs()` creates a `devops/` directory with a `.gitkeep` file at the project root.
- [ ] Running `kaos-control init` twice does not error or duplicate the `devops/` directory.
- [ ] `go build` and `go vet` pass.

---

## Milestone 2: Add sample pipeline seed file

### Description

Create an embedded template `sample-pipeline.yaml.tmpl` and register it as a seed file so that `kaos-control init` writes a working sample pipeline into `devops/sample.yaml`. The sample pipeline should have `type: build` and a single step that runs `echo "hello world"`.

### Files to change

- `internal/initcmd/templates/sample-pipeline.yaml.tmpl` — **new file**. Contents:
  ```yaml
  name: Sample Pipeline
  type: build

  steps:
    - name: Hello world
      description: A sample step to verify pipeline execution
      command: echo "hello world"
      timeout: 30s
  ```
- `internal/initcmd/seedfiles.go` — add a new `seedFileSpec` entry:
  - `tmpl`: `"sample-pipeline.yaml.tmpl"`
  - `relPath`: `"devops/sample.yaml"`
  - `force`: tied to `ForceFlags.Config` (or a new `ForceFlags.Devops` field if appropriate; prefer reusing `.Config` for simplicity).

### Acceptance criteria

- [ ] After `kaos-control init`, `devops/sample.yaml` exists and is valid YAML parseable by `devops.Discover()`.
- [ ] Re-running `init` without `--force` skips the file if it already exists.
- [ ] `go build` and `go vet` pass.

---

## Milestone 3: Expand `config.yaml.tmpl` with idea-capture, kanban, and dashboard defaults

### Description

Add three new blocks to the generated `config.yaml.tmpl` so that new projects get a working idea-capture agent, kanban board, and dashboard out of the box. The defaults should be functionally equivalent to those in this project's own `lifecycle/config.yaml`.

### Files to change

- `internal/initcmd/templates/config.yaml.tmpl` — add the following blocks:

  **Dashboard block** (after the `required_plans` section):
  ```yaml
  dashboard:
    tracked_types:
      - requirement
      - idea
      - defect
  ```

  **Kanban block** (after `dashboard`):
  ```yaml
  kanban:
    columns:
      - name: Backlog
        statuses:
          - draft
      - name: In-Progress
        statuses:
          - clarifying
          - planning
          - in-development
          - in-qa
      - name: Done
        statuses:
          - done
          - abandoned
          - rejected
    uncategorised: true
    card_fields:
      - title
      - type
      - priority
      - labels
      - age
  ```

  **Idea-capture agent** (append to the `agents` list):
  ```yaml
    - name: idea-capture
      role:
        - product-owner
      driver: inline
      model: claude-sonnet-4-6
      allowed_write_paths:
        - lifecycle/ideas
      prompt_templates:
        idea-capture: |
          <idea-capture prompt from lifecycle/config.yaml>
        idea-generate: |
          <idea-generate prompt from lifecycle/config.yaml>
  ```

  Use the full prompt text from this project's `lifecycle/config.yaml` (lines 360–393+) as the default, ensuring Go template syntax (`{{` / `}}`) is properly escaped if needed.

### Acceptance criteria

- [ ] `kaos-control init` in a fresh directory produces a `lifecycle/config.yaml` containing an `idea-capture` agent block, a `kanban:` block with default columns, and a `dashboard:` block with `tracked_types`.
- [ ] The generated config is valid YAML and parses without error by the config loader.
- [ ] `go build` and `go vet` pass.

---

## Milestone 4: Add `POST /api/p/{project}/devops/pipelines` endpoint

### Description

Add an endpoint that creates a new pipeline definition file in `devops/`. The endpoint must validate the slug, parse the YAML to ensure it's a valid pipeline, reject duplicates, and enforce role restrictions.

### Files to change

- `internal/http/devops.go` — add `handleCreatePipeline(w, r)`:
  1. Authenticate and check `product-owner` or `devops` role (same pattern as existing handlers).
  2. Decode JSON body: `{ "slug": string, "definition": string }`.
  3. Validate slug against the pattern `^[a-z0-9][a-z0-9\-]*[a-z0-9]$|^[a-z0-9]$`.
  4. Check that no file `devops/{slug}.yaml` already exists → `409 Conflict` if it does.
  5. Validate that `definition` is valid YAML by attempting to parse it with `devops.parsePipelineFile` logic (or unmarshal to `pipelineYAML` and validate required fields).
  6. Write the definition to `devops/{slug}.yaml` with mode `0644`.
  7. Return `201 Created` with pipeline metadata (`slug`, `name`, `type`, `step_count`).

- `internal/http/server.go` — register the new route:
  ```go
  r.Post("/devops/pipelines", s.handleCreatePipeline)
  ```
  (at line ~222, alongside the existing devops routes.)

- `internal/devops/discovery.go` — export a `ValidateDefinition(data []byte) (*Pipeline, error)` function that reuses the existing parsing and validation logic from `parsePipelineFile` but accepts raw bytes instead of a file path. Refactor `parsePipelineFile` to call this internally to avoid duplication.

### Acceptance criteria

- [ ] `POST /api/p/{project}/devops/pipelines` with valid JSON `{"slug":"ci","definition":"name: CI\ntype: build\nsteps:\n  - name: test\n    command: go test ./..."}` returns `201` and creates `devops/ci.yaml`.
- [ ] Duplicate slug returns `409 Conflict`.
- [ ] Invalid YAML or missing required fields returns `400 Bad Request` with a descriptive error.
- [ ] Invalid slug (uppercase, special chars) returns `400 Bad Request`.
- [ ] Unauthenticated request returns `401`; wrong role returns `403`.
- [ ] `go build`, `go vet`, and `staticcheck` pass.

---

## Milestone 5: Ensure idempotency of the full init flow

### Description

Verify and fix any edge cases where running `kaos-control init` on an already-initialised project could fail or overwrite data. The `devops/` directory and its seed pipeline must follow the same skip-if-exists pattern as other seed files.

### Files to change

- `internal/initcmd/scaffold.go` — confirm `devops` is handled like other dirs (MkdirAll + gitkeep skip). No change expected if Milestone 1 used the existing `lifecycleDirs` pattern.
- `internal/initcmd/seedfiles.go` — confirm the sample pipeline seed spec skips correctly.
- `internal/initcmd/initcmd.go` — review `Run()` to ensure no ordering issues between scaffold dirs and seed files (seed files depend on `devops/` existing).

### Acceptance criteria

- [ ] Running `kaos-control init` on a fresh dir, then again, produces no errors and does not overwrite any files.
- [ ] A project with an existing `devops/` directory and custom pipelines is unaffected by a second `init`.
- [ ] `go build` and unit tests (`make test-unit`) pass.

---

## Cross-references

- The **Create Pipeline UI** in [[kaos-control-init-bootstrap-4-fe]] depends on the `POST /devops/pipelines` endpoint from Milestone 4.
- The **integration tests** in [[kaos-control-init-bootstrap-5-test]] will exercise the init scaffold changes (Milestones 1–3) and the create-pipeline API (Milestone 4).
