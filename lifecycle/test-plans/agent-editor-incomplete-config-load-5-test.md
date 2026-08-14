---
title: 'Tests: Agent Editor Config Load and Round-Trip Preservation'
type: plan-test
status: draft
lineage: agent-editor-incomplete-config-load
parent: lifecycle/requirements/agent-editor-incomplete-config-load-2.md
release: KC-Release5
labels:
    - defect
    - agent
    - editor
    - test
---

# Tests: Agent Editor Config Load and Round-Trip Preservation

Verifies [[agent-editor-incomplete-config-load]] across both layers: the
backend read payload (`-3-be`) and the frontend load + merge-on-save
(`-4-fe`). Covers requirement NFR-2 (regression coverage) and NFR-3 (no secret
leakage), and the full Acceptance Criteria checklist.

## Design notes

- Go integration tests live in `tests/integration` (build tag `integration`,
  package `integration`); reuse the existing agent test scaffolding
  (`tests/integration/agent_helpers_test.go`, `ready_counts_test.go`) for a
  project whose `lifecycle/config.yaml` is written from a fixture.
- Web component/logic tests live in `tests/web` (vitest), alongside existing
  `AgentsRunsView.*` and agent-launch specs.
- Every failing test in this plan must fail against `main` (pre-fix) and pass
  after `-3-be` and `-4-fe` land, proving the defect and guarding the fix.

---

## Milestone 1 — API returns full non-secret config (backend integration)

**Description.** Stand up a project whose config defines an agent exercising
`timeout_minutes`, `git_identity`, `prompt_templates` (multiple keys),
`source_types`, `done_on_success`, `on_denial`, `observe_only`,
`bash_allowlist`, `bash_denylist`, and a second `claude-env` agent with
`auth_token`. Call `GET /api/p/{project}/agents` and assert the payload.

**Files to change**
- `tests/integration/agent_editor_config_test.go` (new)

**Acceptance criteria**
- Response for the rich agent includes every field above with the configured
  values, and `prompt_templates` contains all keys (no collapse).
- Response body for the `claude-env` agent contains neither `auth_token` nor
  its value (NFR-3).
- Test fails on pre-fix `handleListAgents` and passes after `-3-be`.

---

## Milestone 2 — Form populates every exposed field (web component)

**Description.** Mount `AgentConfigForm.vue` with an `initial` `AgentSummary`
that sets `timeout_minutes`, `git_identity.{name,email}`, and a 3-key
`prompt_templates` map (mirroring `idea-capture`). Assert the rendered inputs
hold those values, then trigger submit and assert the emitted `AgentFormData`
carries them, with all three template keys intact.

**Files to change**
- `tests/web/AgentConfigForm.load.spec.ts` (new)

**Acceptance criteria**
- Timeout input shows the configured number (not `0`); git name/email inputs
  are populated; all three templates are visible and keyed by role.
- Emitted submit payload round-trips the template map with no dropped or merged
  keys (FR-6).
- Creating (no `initial`) renders empty defaults (FR-7).

---

## Milestone 3 — Save merges and preserves non-exposed fields (web logic)

**Description.** Exercise the `handleAgentFormSubmit` merge behaviour. Prefer
extracting the merge into a small pure helper (e.g. `mergeAgentEntry(existing,
formData)`) so it is unit-testable without a live server; if it stays inline,
drive it through a mounted `AgentsRunsView` with mocked `configApi`. Feed a
parsed `config.yaml` whose target agent carries `active_status`,
`done_on_success`, `source_types`, `on_denial`, `observe_only`, `bash_*`,
`endpoint`, `base_url`, and `auth_token`. Simulate editing only `model`.

**Files to change**
- `tests/web/AgentConfigForm.mergeSave.spec.ts` (new)
- (if helper extracted) assert against `mergeAgentEntry` directly

**Acceptance criteria** (requirement AC)
- After save, the merged entry keeps `active_status`, `done_on_success`,
  `source_types`, `on_denial`, `observe_only`, `bash_allowlist`,
  `bash_denylist`, `endpoint`, `base_url`, and `auth_token` unchanged (FR-3,
  FR-5).
- Only `model` changed; open-then-save with no edits produces a
  semantically identical entry (FR-1 round-trip).
- Clearing all `allowed_write_paths` removes the key (FR-4); clearing
  `git_identity` fields removes the object.
- Create path emits only populated keys — no spurious empty values (FR-7).

---

## Milestone 4 — End-to-end round-trip preservation (backend integration)

**Description.** The cross-layer guard for NFR-2: write a config with a fully
populated agent (including `auth_token`), read it via `GET /agents`, apply a
single-field edit through the same merge rules the frontend uses, `PUT` the
resulting YAML via the config-update endpoint, reload, and re-read. Assert no
non-exposed field was lost and `auth_token` is unchanged on disk while never
appearing in either `GET /agents` response.

**Files to change**
- `tests/integration/agent_editor_config_test.go` (extend Milestone 1 file)

**Acceptance criteria**
- Post-save on-disk `config.yaml` retains every field of the original agent
  entry except the intentionally edited one.
- `auth_token` value on disk is byte-identical before and after; it appears in
  no `GET /agents` response.
- Test fails on pre-fix behaviour (replace-on-save drops fields) and passes
  after `-4-fe`.

---

## Milestone 5 — Companion test artifact

**Description.** After the tests are implemented, record a `test` artifact in
`lifecycle/tests/` summarising the scenarios and pointing at the concrete test
files, per the test-developer convention.

**Files to change**
- `lifecycle/tests/agent-editor-incomplete-config-load-6-test.md` (new `test`
  artifact; `parent:` this plan)

**Acceptance criteria**
- Artifact lists the five milestones' scenarios and links the test files in
  `tests/integration` and `tests/web`.
- Frontmatter: `type: test`, `status: draft`, `lineage:
  agent-editor-incomplete-config-load`, `parent:` this plan.
