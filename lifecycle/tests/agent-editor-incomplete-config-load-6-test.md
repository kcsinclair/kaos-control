---
title: "Tests — Agent Editor Config Load and Round-Trip Preservation"
type: test
status: draft
lineage: agent-editor-incomplete-config-load
parent: lifecycle/test-plans/agent-editor-incomplete-config-load-5-test.md
---

# Tests — Agent Editor Config Load and Round-Trip Preservation

Automated coverage for [[agent-editor-incomplete-config-load]], implementing
the five milestones of
[lifecycle/test-plans/agent-editor-incomplete-config-load-5-test.md](../test-plans/agent-editor-incomplete-config-load-5-test.md).
Verified both the backend read payload (`-3-be`) and the frontend load +
merge-on-save fix (`-4-fe`), which were already `done` when this suite was
written; every test asserts the fixed behaviour and was checked against the
pre-fix `internal/http/agents.go` (commit `6d739a0b`) to confirm it fails
there (Milestone 1) before passing on the current tree.

## Milestone 1 — API returns full non-secret config (backend integration)

**File:** `tests/integration/agent_editor_config_test.go` —
`TestAgentEditorConfig_ListReturnsFullNonSecretConfig`

Stands up a project with a rich `claude-mediated` agent
(`full-config-agent`) exercising `timeout_minutes`, `git_identity`, a
3-key `prompt_templates` map, `source_types`, `done_on_success`,
`on_denial`, `observe_only`, `bash_allowlist`, `bash_denylist`, and a second
`claude-env` agent (`claude-env-agent`) carrying `auth_token`. Calls
`GET /api/p/testproject/agents` and asserts:

- Every listed field is present with its configured value; `prompt_templates`
  contains all three keys, uncollapsed.
- The raw response body contains neither the string `auth_token` nor the
  configured token value, for the `claude-env` agent.

## Milestone 2 — Form populates every exposed field (web component)

**File:** `tests/web/AgentConfigForm.load.spec.ts`

Mounts `AgentConfigForm.vue` with an `initial` `AgentSummary` mirroring
`idea-capture` (`timeout_minutes`, `git_identity.{name,email}`, a 3-key
`prompt_templates` map). Asserts:

- The timeout input shows the configured number, not `0`.
- Git name/email inputs are populated.
- All three prompt templates render, each labelled by role.
- Submitting with no edits emits an `AgentFormData` with the same
  `timeout_minutes`, `git_identity_*`, and all three `prompt_templates` keys
  intact (FR-6).
- Mounting with `initial: null` (create mode) renders empty/zero defaults
  (FR-7).

## Milestone 3 — Save merges and preserves non-exposed fields (web logic)

**File:** `tests/web/AgentConfigForm.mergeSave.spec.ts`

`handleAgentFormSubmit` (`AgentsRunsView.vue`) merges inline rather than
through an extracted helper, so this drives it through a mounted
`AgentsRunsView` with `@/api/config` mocked (`getConfig`/`updateConfig`
mocked; `parseConfigYaml`/`dumpConfigYaml` left as the real `js-yaml`
implementations so the captured `PUT` body is genuine YAML). The fixture
agent (`backend-developer`) carries `active_status`, `done_on_success`,
`source_types`, `on_denial`, `observe_only`, `bash_allowlist`,
`bash_denylist`, `endpoint`, `base_url`, and `auth_token` on the simulated
on-disk entry. Scenarios:

- Editing only `model` and saving preserves every other field, including
  `auth_token`, byte-for-byte.
- Open-then-save with no edits produces an entry deep-equal to the original
  (FR-1 round-trip).
- Clearing `allowed_write_paths` removes the key (FR-4).
- Clearing both `git_identity` fields removes the `git_identity` object
  (FR-4).
- Creating a new agent with only required fields (`name`, `roles`, `driver`)
  omits `model`, `allowed_write_paths`, `git_identity`, `prompt_templates`,
  and the ollama keys — no spurious empty values (FR-7). `timeout_minutes`
  is asserted present at `0`, since it is always emitted by design (0 =
  unlimited), not a spurious value.

## Milestone 4 — End-to-end round-trip preservation (backend integration)

**File:** `tests/integration/agent_editor_config_test.go` —
`TestAgentEditorConfig_RoundTripPreservesNonExposedFields`

The cross-layer guard for NFR-2: writes a config with a fully populated
`claude-env` agent (`round-trip-agent`) including `auth_token`, reads it via
`GET /agents`, parses the raw `config.yaml` via `GET /config`, applies a
single-field edit (`model`) using the same merge rules the frontend fix
applies (`mergeAgentEntryLikeEditor`, reproducing
`handleAgentFormSubmit`'s copy-then-overwrite-managed-keys behaviour), `PUT`s
the result, reloads, and re-reads. Asserts:

- `model` changed to the new value.
- `active_status`, `done_on_success`, `source_types`, `on_denial`,
  `observe_only`, `bash_allowlist`, `bash_denylist`, `endpoint`, `base_url`,
  and `auth_token` are unchanged on disk.
- `auth_token`'s value never appears in any `GET /agents` response, before or
  after the save.

## Milestone 5 — Companion test artifact

This document.

## Coverage note

Requirement AC checklist items are covered as follows: field-load items by
Milestone 2; the open→edit→save preservation and clearing items by Milestone
3; the `auth_token` non-leakage and full-field-preservation items by
Milestones 1 and 4 together; the create-path item by Milestone 3's create
scenario.
