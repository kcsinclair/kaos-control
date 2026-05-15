---
title: "Tech Writer Agent — Backend Plan"
type: plan-backend
status: approved
lineage: tech-writer-agent
parent: lifecycle/requirements/tech-writer-agent-2.md
---

# Tech Writer Agent — Backend Plan

Backend implementation for the tech-writer agent feature. Covers new artifact type, workflow transitions, config plumbing, docs-capture generation endpoint, and agent routing.

Cross-references: [[tech-writer-agent]] frontend plan for UI triggers (FR1, FR2), test plan for integration coverage.

---

## Milestone 1 — Register `doc` type and `docs` stage

### Description

Add `doc` to the artifact type vocabulary and wire the `docs` stage directory so the indexer, filename builder, and sandbox resolver all recognise `lifecycle/docs/` as a valid target.

### Files to change

- `internal/artifact/artifact.go` — add `"doc": true` to `KnownTypes`.
- `internal/http/write.go` — add `"docs": "doc"` to the `stageSuffix` map so `buildFilename` produces `<slug>-N-doc.md`.
- `internal/artifact/artifact.go` — add `"docs"` → `"doc"` case in `stageToType()` (if present).

### Acceptance criteria

- [ ] `artifact.KnownTypes["doc"]` is `true`.
- [ ] `buildFilename("foo", 4, "docs")` returns `"foo-4-doc.md"`.
- [ ] `buildFilename("foo", 0, "docs")` returns `"foo.md"` (originating doc, no suffix).
- [ ] `stageToType("docs")` returns `"doc"`.
- [ ] Existing types and stages are unchanged.

---

## Milestone 2 — Doc-specific workflow transitions

### Description

Add type-scoped rules to the default transition matrix so `doc` artifacts follow the medium-length pipeline: `draft → approved → in-development → in-qa → done`. The standard feature transitions (`draft → clarifying → planning → …`) must NOT apply to `doc` artifacts.

### Files to change

- `internal/workflow/workflow.go` — add type-scoped `defaultRules` entries for `doc`:
  - `{from: "draft", to: "approved", roles: ["product-owner"], types: ["doc"]}`
  - `{from: "approved", to: "in-development", roles: ["tech-writer"], types: ["doc"]}`
  - `{from: "in-development", to: "in-qa", roles: ["tech-writer"], types: ["doc"]}`
  - `{from: "in-qa", to: "done", roles: ["qa"], types: ["doc"]}`
  - `{from: "in-qa", to: "in-development", roles: ["qa"], types: ["doc"]}` (defect-loop: QA sends back)
- `internal/workflow/workflow.go` — update the existing generic rules (`draft → clarifying`, `clarifying → planning`, `planning → in-development`) to exclude `doc` type. The cleanest approach: modify `ruleMatchesType` or the rule entries so that these three generic rules gain an exclusion for `doc`. One option is an `excludeTypes []string` field on `rule`; alternatively, make the generic rules explicitly list their applicable types. Evaluate both during implementation — prefer the option with fewer downstream changes.

### Acceptance criteria

- [ ] `engine.CanTransition("draft", "approved", ["product-owner"], "doc")` → `true`.
- [ ] `engine.CanTransition("draft", "clarifying", ["analyst"], "doc")` → `false`.
- [ ] `engine.CanTransition("approved", "in-development", ["tech-writer"], "doc")` → `true`.
- [ ] `engine.CanTransition("in-development", "in-qa", ["tech-writer"], "doc")` → `true`.
- [ ] `engine.CanTransition("in-qa", "done", ["qa"], "doc")` → `true`.
- [ ] `engine.CanTransition("in-qa", "in-development", ["qa"], "doc")` → `true` (defect re-route).
- [ ] Existing transitions for `requirement`, `plan-backend`, `test`, etc. are unchanged.
- [ ] `AllowedTargets("draft", ["analyst"], "requirement")` still includes `"clarifying"`.

---

## Milestone 3 — Exempt `doc` from `required_plans` gate

### Description

Ensure `GateReady` does not apply to `doc` artifacts. The gate checks that a lineage has approved plans before allowing `planning → in-development`; since `doc` skips planning entirely this gate must never fire for docs.

### Files to change

- `internal/workflow/workflow.go` — in the handler that invokes `GateReady` (or in `GateReady` itself), skip the check when the artifact type is `doc`. Alternatively, the new doc-specific transition rules bypass the `planning` status entirely, so the gate naturally never fires. **Verify** this by tracing the call path: the transition handler in `internal/http/` likely calls `GateReady` only when `from == "planning"`. If so, this milestone is a verification-only step with no code change.

### Acceptance criteria

- [ ] A `doc` artifact can transition `approved → in-development` without any plans existing in the lineage.
- [ ] A `requirement` artifact still cannot transition `planning → in-development` without approved plans.

---

## Milestone 4 — `docs-capture` agent config and generate endpoint

### Description

Create the `docs-capture` agent (inline driver, like `idea-capture`) with a `doc-generate` prompt template. Extend the idea-generate HTTP handler and `ideachat.Generate` function to accept `type: "doc"` so the BrainDumpModal can produce doc proposals.

### Files to change

- `internal/ideachat/generate.go` — extend the `artifactType` validation switch to accept `"doc"`. When `artifactType == "doc"`, set `templateKey = "doc-generate"` and `targetDir = "lifecycle/docs"`.
- `internal/http/idea_generate.go` — extend the `templateKey` resolution to handle `type: "doc"`:
  ```go
  case "doc":
      templateKey = "doc-generate"
  ```
  Also update `targetDir` assignment: `if artifactType == "doc" { targetDir = "lifecycle/docs" }`.
- `internal/http/idea_generate.go` — the `resolveIdeaCaptureConfig` function must find the `docs-capture` agent config when `templateKey == "doc-generate"`. Either extend the lookup to search all inline agents for a matching template key, or add a second lookup keyed by agent name. The simpler approach: add `"doc"` as a known artifact type to `resolveIdeaCaptureConfig`, mapping it to the `docs-capture` agent name.

The `docs-capture` agent configuration itself lives in `lifecycle/config.yaml` (project-level, not Go code), but the backend must resolve it. See the config milestone below.

### Acceptance criteria

- [ ] `POST /api/p/:project/ideas/generate` with `{"input": "...", "type": "doc"}` returns a valid proposal with `target_dir: "lifecycle/docs"` and correct doc frontmatter shape.
- [ ] `POST /api/p/:project/ideas/generate` with `{"input": "...", "type": "doc", "source_lineage": "login", "source_path": "lifecycle/requirements/login-2.md"}` returns a proposal whose frontmatter includes `lineage: "login"` and `parent: "lifecycle/requirements/login-2.md"`.
- [ ] Requests with `type: "idea"` and `type: "defect"` remain unchanged.

---

## Milestone 5 — Support source-lineage context in doc generation

### Description

When a doc is requested from an existing artifact (FR1), the generate endpoint receives `source_lineage` and `source_path`. The `doc-generate` prompt template must include context from the source lineage (capped to the originating idea + the requirement, per resolved question 5). Extend `GenerateOptions` and the prompt builder to accept and use this context.

### Files to change

- `internal/ideachat/generate.go` — add `SourceLineage string` and `SourcePath string` fields to `GenerateOptions`. When populated, read the source artifact and the originating idea from the index, and append their bodies to the user message as context.
- `internal/http/idea_generate.go` — parse `source_lineage` and `source_path` from the request body and pass them through to `GenerateOptions`.

### Acceptance criteria

- [ ] When `source_lineage` and `source_path` are supplied, the LLM prompt includes the content of the source artifact and the originating idea as context sections.
- [ ] Context is capped to the originating idea + the requirement in that lineage (not all plans/tests/defects).
- [ ] When `source_lineage` is empty, no context is appended (standalone doc flow).

---

## Milestone 6 — Config additions: role, stage, agents

### Description

Add the `tech-writer` role, `docs` stage, and two agents (`docs-capture`, `tech-writer`) to `lifecycle/config.yaml`. Create the `lifecycle/docs/` directory.

### Files to change

- `lifecycle/config.yaml`:
  - Add `tech-writer` to the `roles:` list.
  - Add `{ name: docs, dir: docs }` to the `stages:` list.
  - Add `docs-capture` agent entry:
    ```yaml
    - name: docs-capture
      role: [product-owner]
      driver: inline
      model: claude-sonnet-4-6
      allowed_write_paths: [lifecycle/docs]
      prompt_templates:
        doc-generate: |
          <prompt template per FR3 — structured doc brief generation>
    ```
  - Add `tech-writer` agent entry:
    ```yaml
    - name: tech-writer
      role: [tech-writer]
      driver: claude-code-cli
      model: sonnet
      active_status: in-development
      source_types: [doc]
      allowed_write_paths: [lifecycle/docs, docs]
      git_identity:
        name: Tech Writer Agent
        email: tech-writer@kaos-control.local
      prompt_templates:
        tech-writer: |
          <prompt template per FR5 — expand brief into full documentation>
    ```
- Create empty directory `lifecycle/docs/` (add `.gitkeep` if needed).
- `internal/config/` — verify the config loader parses the new role and stage without error. No schema change expected (roles and stages are string lists), but confirm.

### Acceptance criteria

- [ ] `lifecycle/config.yaml` contains `tech-writer` in `roles`.
- [ ] `lifecycle/config.yaml` contains `{ name: docs, dir: docs }` in `stages`.
- [ ] `docs-capture` agent is configured with `driver: inline`, `model: claude-sonnet-4-6`, and `doc-generate` prompt template.
- [ ] `tech-writer` agent is configured with `driver: claude-code-cli`, `model: sonnet`, `active_status: in-development`, `source_types: [doc]`, `allowed_write_paths: [lifecycle/docs, docs]`.
- [ ] `lifecycle/docs/` directory exists.
- [ ] Application starts without config errors after these changes.

---

## Milestone 7 — QA routing for doc artifacts

### Description

When a `doc` artifact transitions `in-development → in-qa`, its `assignees` must be updated to `[{ role: qa, who: agent }]` so the QA agent picks it up. When QA raises defects against a doc, the defect's `assignees` must route to `tech-writer`. This follows the existing pattern used for defects against developer artifacts.

### Files to change

- `internal/http/transition.go` (or wherever the status-transition handler lives) — on `in-development → in-qa` for `type: doc`, set `assignees: [{ role: qa, who: agent }]` in the artifact's frontmatter before writing.
- `internal/http/agents.go` — in `handleGetReadyCounts`, ensure the `tech-writer` agent's ready count includes `doc` artifacts in `approved` status (this should work automatically via `source_types: [doc]`). Also verify `countAssignedDefects` counts defects assigned to `tech-writer`.

### Acceptance criteria

- [ ] Transitioning a `doc` artifact to `in-qa` sets its assignees to `[{ role: qa, who: agent }]`.
- [ ] The QA agent's ready count includes `doc` artifacts in `in-qa`.
- [ ] Defects filed against docs with `assignees: [{ role: tech-writer, who: agent }]` appear in the tech-writer's assigned-defect count.
- [ ] Existing QA routing for `test` artifacts is unaffected.
