---
title: 'Backend: Agent-Read API Returns the Full Non-Secret AgentConfig'
type: plan-backend
status: draft
lineage: agent-editor-incomplete-config-load
parent: lifecycle/requirements/agent-editor-incomplete-config-load-2.md
release: KC-Release5
labels:
    - defect
    - agent
    - backend
---

# Backend: Agent-Read API Returns the Full Non-Secret AgentConfig

Implements the backend half of [[agent-editor-incomplete-config-load]]. The
frontend fix lives in the sibling plan [[agent-editor-incomplete-config-load]]
(`-4-fe`); its tests are in [[agent-editor-incomplete-config-load]] (`-5-test`).

## Problem Summary

`handleListAgents` (`internal/http/agents.go`) serialises a lossy
`agentSummary` projection: only `name`, `roles`, `driver`, `model`,
`active_status`, `allowed_write_paths`, `ollama_instance`, `ollama_endpoint`,
`base_url`, and `ready_count`. Every other field of `config.AgentConfig`
(`timeout_minutes`, `git_identity`, `prompt_templates`, `source_types`,
`done_on_success`, `endpoint`, `on_denial`, `observe_only`, `bash_allowlist`,
`bash_denylist`, `shell_command`) is dropped. The editor therefore has no way
to display those values (requirement FR-1, FR-2).

## Design decisions (resolving the requirement's Open Questions)

- **Read source (Q1):** extend the existing `GET /api/p/{project}/agents`
  response to carry the full **non-secret** `AgentConfig`. This is the cleaner
  option the requirement contemplates in FR-2 and makes `AgentSummary` the
  single source of truth the editor loads from. It does **not** change how
  save works: the frontend still merges onto the freshly-read raw
  `config.yaml` on disk (see the `-4-fe` plan), so secrets never round-trip
  through the client (FR-5).
- **Secrets (FR-5 / NFR-3):** `auth_token` is **never** added to the response
  struct. It is preserved on save only because save merges onto the on-disk
  entry.
- **Scope (Q2):** this defect is preserve-and-load only. No new configuration
  fields and no runtime changes (requirement Non-goals). `endpoint` is treated
  as a preserve-only legacy field (Q3): it is decoded in
  `config.go` but read nowhere else, so we surface it in the payload for
  completeness/preservation but add no behaviour around it.

---

## Milestone 1 — Expand the agent-read payload

**Description.** Replace the lossy `agentSummary` struct in `handleListAgents`
with one that mirrors every editable / preserve-relevant field of
`config.AgentConfig`, excluding secrets. Populate it from `p.Agents.Agents()`
exactly as today. Keep `ready_count` and the existing field names/JSON tags so
current consumers do not break.

New/added JSON fields (all `omitempty` except where a zero value is
meaningful):

| JSON key              | Source (`config.AgentConfig`) | Notes                                   |
|-----------------------|-------------------------------|-----------------------------------------|
| `timeout_minutes`     | `TimeoutMinutes`              | number; `0` = unlimited, always emitted |
| `git_identity`        | `GitIdentity{name,email}`     | nested object; omit when both empty     |
| `prompt_templates`    | `PromptTemplates`             | `map[string]string`, role → template    |
| `source_types`        | `SourceTypes`                 | already declared client-side; now sent  |
| `done_on_success`     | `DoneOnSuccess`               | bool                                     |
| `endpoint`            | `Endpoint`                    | legacy; preserve-only                    |
| `on_denial`           | `OnDenial`                    | already declared client-side; now sent  |
| `observe_only`        | `ObserveOnly`                 | already declared client-side; now sent  |
| `bash_allowlist`      | `BashAllowlist`               | already declared client-side; now sent  |
| `bash_denylist`       | `BashDenylist`                | already declared client-side; now sent  |
| `shell_command`       | `ShellCommand`                | shell-stub driver                        |

**Explicitly excluded:** `AuthToken` — no struct field, no JSON key, ever.

To guard against NFR-1 (schema drift silently reintroducing loss), add a code
comment on the response struct that names `auth_token` as the *only*
intentionally-omitted field, so a future field added to `AgentConfig` is a
visible decision to include or exclude here.

**Files to change**
- `internal/http/agents.go` — `handleListAgents` response struct + population.

**Acceptance criteria**
- `GET /api/p/{project}/agents` returns, for an agent that sets them, all of:
  `timeout_minutes`, `git_identity.name`, `git_identity.email`,
  `prompt_templates` (all keys), `source_types`, `done_on_success`,
  `on_denial`, `observe_only`, `bash_allowlist`, `bash_denylist`, `endpoint`,
  `shell_command`.
- The response body for an agent configured with `auth_token` (driver
  `claude-env`) does **not** contain the string `auth_token` nor its value.
- Previously-present keys (`name`, `roles`, `driver`, `model`,
  `active_status`, `allowed_write_paths`, `ollama_instance`,
  `ollama_endpoint`, `base_url`, `ready_count`) remain present and unchanged in
  shape.
- `go build ./...` and `go vet ./...` pass.

---

## Milestone 2 — Regression test for payload completeness and secret redaction

**Description.** Add a unit test alongside the existing handler tests that
constructs a project whose config includes one agent exercising every field
(including `auth_token` via a `claude-env` agent), calls `handleListAgents`,
and asserts (a) every non-secret field is present with the configured value,
and (b) `auth_token` is absent from the serialised JSON. This is the backend
leg of requirement NFR-2 / NFR-3; the cross-layer round-trip lives in the
`-5-test` plan.

**Files to change**
- `internal/http/agents_test.go` — new test (e.g.
  `TestListAgents_ReturnsFullNonSecretConfig`).

**Acceptance criteria**
- Test fails against the current lossy struct and passes after Milestone 1.
- Test asserts presence + value of each field in the Milestone 1 table.
- Test asserts the raw JSON response contains neither `auth_token` nor the
  configured token value.
- `go test ./internal/http/... -run TestListAgents` passes.

---

## Out of scope

- No changes to `handleUpdateConfig` (save path). Save remains a raw-YAML
  write driven by the client; preservation is a frontend merge concern handled
  in the `-4-fe` plan.
- No new editor-only fields, drivers, or runtime behaviour (requirement
  Non-goals).
