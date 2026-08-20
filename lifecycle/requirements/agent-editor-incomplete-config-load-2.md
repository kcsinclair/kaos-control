---
created: "2026-07-14T19:34:44+10:00"
title: Agent Editor Loads and Preserves All config.yaml Fields
type: requirement
status: blocked
lineage: agent-editor-incomplete-config-load
parent: lifecycle/defects/agent-editor-incomplete-config-load.md
assignees:
    - role: product-owner
      who: agent
---

# Agent Editor Loads and Preserves All config.yaml Fields

## Problem

When a user opens an existing agent in the agent editor, only a partial
subset of that agent's configuration is loaded into the form. Fields such as
`timeout_minutes`, `git_identity`, and `prompt_templates` show as blank or
zero even when they are populated in `lifecycle/config.yaml`, and other
configured fields (`active_status`, `source_types`, `done_on_success`,
`bash_allowlist`, `bash_denylist`, `on_denial`, `observe_only`, `endpoint`)
are not represented in the editor at all.

The gap spans three layers:

1. **Backend list endpoint** — `GET /api/p/{project}/agents`
   (`internal/http/agents.go`, `handleListAgents`) serialises only
   `name`, `roles`, `driver`, `model`, `active_status`,
   `allowed_write_paths`, `ollama_instance`, `ollama_endpoint`, `base_url`,
   and `ready_count`. Every other field of `config.AgentConfig` is dropped
   from the response.
2. **Client type** — `AgentSummary` (`web/src/types/api.ts`) declares some
   fields the backend does not send and still omits `timeout_minutes`,
   `git_identity`, and `prompt_templates`.
3. **Editor form** — `AgentConfigForm.vue` initialises `timeoutMinutes`,
   `gitIdentityName`, `gitIdentityEmail`, and `promptTemplatesRaw` to
   empty/zero rather than from `props.initial`.

The consequence is **silent data loss on save**. `handleAgentFormSubmit`
(`web/src/views/project/AgentsRunsView.vue`) reads the full `config.yaml`,
builds a fresh entry from *only* the form's fields, and **replaces** the
matched agent entry wholesale (`agents[idx] = entry`). Any field the form did
not surface — or surfaced but failed to populate — is erased from
`config.yaml` when the user saves an unrelated edit.

## Goals / Non-goals

**Goals**

- Opening an existing agent for editing loads every editable field currently
  present in that agent's `config.yaml` entry.
- Saving an agent never discards a field that was present in `config.yaml`
  before the edit, whether or not the editor surfaces it.
- The full round-trip (open → save with no changes) is a no-op on the agent's
  YAML content, modulo key ordering and formatting.

**Non-goals**

- No change to the agent runtime, driver behaviour, or the set of supported
  drivers.
- No new agent-configuration fields are introduced; this defect is about
  faithfully loading and preserving the existing schema.
- Redesign of the prompt-template editor UX beyond correctly loading and
  round-tripping existing templates.
- Secret material handling policy beyond the non-echo / preserve rule stated
  below.

## Detailed Requirements

### Functional

**FR-1 — Editable fields load from config.** When an existing agent is opened
in the editor, the form is populated from that agent's `config.yaml` entry for
every field the form exposes, at minimum: `name`, `roles`, `driver`, `model`,
`allowed_write_paths`, `timeout_minutes`, `git_identity.name`,
`git_identity.email`, `prompt_templates`, `ollama_instance`, and
`ollama_endpoint`. No exposed field that is set in config may render blank or
default when config holds a value.

**FR-2 — Data source carries all fields.** The editor must have access to the
complete configured value of each field it loads. This may be satisfied by
extending the agent-read API to return the full (non-secret) `AgentConfig`, or
by the editor reading the parsed `config.yaml` entry directly. The chosen
source must include the fields listed in FR-1 and FR-3.

**FR-3 — Non-loss of unedited fields on save.** Saving an agent must preserve
every field present in that agent's pre-edit `config.yaml` entry that the
editor does not expose, including but not limited to: `active_status`,
`done_on_success`, `source_types`, `endpoint`, `bash_allowlist`,
`bash_denylist`, `on_denial`, `observe_only`, `shell_command`, and `base_url`.
Save must **merge** the form's values onto the existing entry rather than
replacing the entry.

**FR-4 — Empty vs unset.** Clearing a field the editor exposes and saving must
persist that field as cleared/removed. Preservation under FR-3 applies only to
fields the editor does **not** expose; it must not resurrect a value the user
deliberately cleared in an exposed field.

**FR-5 — Secret fields.** `auth_token` (and any field marked secret) must
never be sent to the client. It must nonetheless survive an edit-and-save
round trip unchanged, since save merges onto the on-disk entry rather than a
client-supplied copy.

**FR-6 — Prompt templates round-trip.** All role→template entries in
`prompt_templates` load into the editor and, if unchanged, serialise back to
config byte-for-byte equivalent (ignoring YAML formatting/quoting). Multi-role
template maps must not collapse to a single template.

**FR-7 — Create unaffected.** Creating a new agent continues to work; fields
left blank are simply omitted from the new entry (no phantom empty values that
would alter runtime defaults).

### Non-functional

**NFR-1 — No schema drift.** The set of fields loaded and preserved is derived
from `config.AgentConfig` in `internal/config/config.go`; adding a future
field there must not silently reintroduce the loss (see AC checklist).

**NFR-2 — Regression coverage.** The round-trip preservation is covered by an
automated test so this class of defect cannot regress unnoticed.

**NFR-3 — No secret leakage.** `auth_token` must not appear in any HTTP
response body or client-held state; verified by test.

## Acceptance Criteria

- [ ] Opening an agent whose config sets `timeout_minutes` shows that value
      (not `0`) in the editor.
- [ ] Opening an agent whose config sets `git_identity.name` and
      `git_identity.email` shows both values in the editor.
- [ ] Opening an agent whose config sets one or more `prompt_templates` shows
      all of them in the editor, keyed by role.
- [ ] Editing one exposed field and saving leaves `active_status`,
      `done_on_success`, `source_types`, `endpoint`, `bash_allowlist`,
      `bash_denylist`, `on_denial`, and `observe_only` unchanged in
      `config.yaml`.
- [ ] Saving an agent that has an `auth_token` in config leaves the
      `auth_token` value unchanged, and the token never appears in the
      agent-read API response.
- [ ] Open-then-save with no field changes produces a semantically identical
      agent entry (same keys and values, formatting aside).
- [ ] Clearing an exposed field (e.g. removing all `allowed_write_paths`) and
      saving removes/clears that field in `config.yaml`.
- [ ] Creating a new agent with only required fields produces an entry with no
      spurious empty-valued keys.
- [ ] An automated test exercises the open→edit→save round trip and asserts
      no non-exposed field is lost (satisfies NFR-2), aligned with
      [[end-to-end-smoke-tests]] and [[test-everything]] coverage.
- [ ] Behaviour is consistent with the fields introduced by
      [[claude-hooks-driver]] (`bash_allowlist`, `bash_denylist`, `on_denial`,
      `observe_only`) and [[ollama-agent-support]] (`ollama_instance`,
      `ollama_endpoint`) — none of these are dropped on save.

## Open Questions

- **Read source:** Should the fix extend `GET /agents` (or add a single-agent
  read endpoint) to return the full non-secret `AgentConfig`, or should the
  editor rely on parsing `config.yaml` client-side (as `handleAgentFormSubmit`
  already does on save)? The merge-on-save requirement (FR-3) is achievable
  purely client-side, but surfacing fields for editing (FR-1) is cleaner with
  a fuller API payload.
- **Field surfacing scope:** This requirement mandates *preserving* all
  fields but only *displaying/editing* the currently-exposed subset. Is it in
  scope to also add editor controls for `active_status`, `source_types`,
  `done_on_success`, and the claude-mediated permission fields, or should
  those remain preserve-only until a separate enhancement?
- **Endpoint field:** `AgentConfig.Endpoint` (`endpoint`) is distinct from
  `ollama_endpoint`. Is it still in use, and should it be preserved silently
  or exposed?
