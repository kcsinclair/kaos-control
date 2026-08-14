---
title: 'Frontend: Agent Editor Loads Every Field and Merges on Save'
type: plan-frontend
status: draft
lineage: agent-editor-incomplete-config-load
parent: lifecycle/requirements/agent-editor-incomplete-config-load-2.md
release: KC-Release5
labels:
    - defect
    - agent
    - editor
    - frontend
---

# Frontend: Agent Editor Loads Every Field and Merges on Save

Implements the frontend half of [[agent-editor-incomplete-config-load]]. It
depends on the expanded read payload from the sibling backend plan
[[agent-editor-incomplete-config-load]] (`-3-be`); tests are specified in
[[agent-editor-incomplete-config-load]] (`-5-test`).

## Problem Summary

Two frontend faults combine into silent data loss (requirement Problem
section):

1. **Load gap.** `AgentConfigForm.vue` initialises `timeoutMinutes` (to `0`),
   `gitIdentityName`, `gitIdentityEmail`, and `promptTemplatesRaw` (to empty)
   instead of from `props.initial`. Even where the backend does send a value it
   is ignored; several fields were not sent at all (fixed in `-3-be`).
2. **Replace-on-save.** `handleAgentFormSubmit`
   (`web/src/views/project/AgentsRunsView.vue`) builds a fresh `entry` from
   only the form's fields and does `agents[idx] = entry`, erasing every
   config key the form does not manage (`active_status`, `source_types`,
   `done_on_success`, `on_denial`, `observe_only`, `bash_*`, `endpoint`,
   `shell_command`, `base_url`, `auth_token`).

## Design decisions (aligned with the requirement)

- The editor **loads** the currently-exposed subset only, but now loads it
  completely (FR-1): `name`, `roles`, `driver`, `model`,
  `allowed_write_paths`, `timeout_minutes`, `git_identity.*`,
  `prompt_templates`, `ollama_instance`, `ollama_endpoint`.
- Save **merges** onto the existing on-disk entry (FR-3), so non-exposed and
  secret fields survive untouched. No new editor controls for
  `active_status` / `source_types` / permission fields — that is a separate
  enhancement (requirement Non-goals, Q2 resolved to preserve-only).

---

## Milestone 1 — Complete the `AgentSummary` client type

**Description.** Extend `AgentSummary` (`web/src/types/api.ts`) to declare the
fields the `-3-be` payload now sends and the form needs to read:
`timeout_minutes?: number`, `git_identity?: { name?: string; email?: string }`,
`prompt_templates?: Record<string, string>`, `done_on_success?: boolean`,
`endpoint?: string`, `shell_command?: string`. (`source_types`, `observe_only`,
`bash_allowlist`, `bash_denylist`, `on_denial` are already declared.) Add a
comment noting `auth_token` is intentionally never present.

**Files to change**
- `web/src/types/api.ts`

**Acceptance criteria**
- Type compiles (`pnpm exec vue-tsc --noEmit`) and exposes the new optional
  fields.
- No `auth_token` field is added to `AgentSummary`.

---

## Milestone 2 — Populate the form from `props.initial`

**Description.** In `AgentConfigForm.vue`, initialise every exposed field from
`props.initial`:
- `timeoutMinutes` ← `props.initial?.timeout_minutes ?? 0`
- `gitIdentityName` ← `props.initial?.git_identity?.name ?? ''`
- `gitIdentityEmail` ← `props.initial?.git_identity?.email ?? ''`
- prompt templates ← `props.initial?.prompt_templates ?? {}` (see Milestone 3
  for the lossless representation)

Keep the existing initialisation for `name`, `roles`, `driver`, `model`,
`ollama_instance`, `ollama_endpoint`, `allowed_write_paths`.

**Files to change**
- `web/src/components/agent/AgentConfigForm.vue`

**Acceptance criteria** (requirement AC checklist)
- Opening an agent whose config sets `timeout_minutes` shows that value, not
  `0`.
- Opening an agent whose config sets `git_identity.name` / `.email` shows both.
- Creating a new agent (`props.initial == null`) still starts with empty/default
  fields (FR-7).
- `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Milestone 3 — Lossless prompt-template load and edit

**Description.** The current `promptTemplatesRaw` textarea + ad-hoc
`parsePromptTemplates` regex is lossy: it can drop keys and mangles multi-line
blocks, so a multi-template agent (e.g. `idea-capture` with `idea-capture`,
`idea-generate`, `defect-generate`) collapses. Replace it with a faithful
round-trip (requirement FR-6):
- On load, hold the original `prompt_templates` map from `props.initial`.
- Render all role→template pairs into the editor with a delimiter format the
  parser can re-read without loss (retain the `role: |` block textarea, but
  serialise **all** keys on load and parse **all** keys back on submit; verify
  round-trip for the 3-key `idea-capture` case). If the block textarea cannot
  be made reliably lossless, fall back to one labelled `<textarea>` per role.
- On submit, emit the full template map. If a template body is cleared the key
  is removed (FR-4); untouched keys are emitted unchanged.

**Files to change**
- `web/src/components/agent/AgentConfigForm.vue` (`parsePromptTemplates`, the
  prompt-templates section, and its initial state)

**Acceptance criteria**
- Opening `idea-capture` shows all three templates, each keyed by role.
- Open-then-submit with no edits emits a `prompt_templates` map equal to the
  input map (no dropped keys, no merged bodies).
- Clearing a single template's body removes only that key from the emitted map.

---

## Milestone 4 — Merge on save instead of replace

**Description.** Rewrite `handleAgentFormSubmit`
(`web/src/views/project/AgentsRunsView.vue`) so an **edit** merges the form's
managed fields onto the existing parsed entry rather than replacing it:
- Locate the existing entry: `const existing = idx >= 0 ? { ...agents[idx] } :
  {}`.
- Assign the form-managed keys onto a copy: `name`, `role`, `driver`, `model`,
  `allowed_write_paths`, `timeout_minutes`, `git_identity`, `prompt_templates`,
  and the ollama keys (only when `driver === 'ollama'`).
- **Clearing semantics (FR-4):** when the form value for an exposed field is
  empty, `delete` that key from the merged entry (e.g. no
  `allowed_write_paths`, no `git_identity`, `model` omitted) rather than
  writing an empty value.
- **Preserve (FR-3, FR-5):** every key present on `existing` that the form does
  not manage — `active_status`, `done_on_success`, `source_types`, `endpoint`,
  `on_denial`, `observe_only`, `bash_allowlist`, `bash_denylist`,
  `shell_command`, `base_url`, `auth_token` — is carried through untouched.
- For **create** (`idx < 0`), keep building a minimal entry with only
  populated keys (FR-7).
- Fix the `editAgent.value` success-message check so it reflects edit vs create
  correctly (it is read after `closeAgentForm()` resets it today).

**Files to change**
- `web/src/views/project/AgentsRunsView.vue` (`handleAgentFormSubmit`)

**Acceptance criteria** (requirement AC checklist)
- Editing one exposed field (e.g. `model`) on `backend-developer` and saving
  leaves `active_status`, `done_on_success`, and `source_types` unchanged in
  `config.yaml`.
- Saving a `claude-env` agent that has `auth_token` leaves the token value
  unchanged in `config.yaml` (the token is never in client state; it survives
  because it stays on `existing`).
- Open-then-save with no changes yields a semantically identical entry (same
  keys and values; ordering/formatting aside).
- Clearing all `allowed_write_paths` and saving removes the key from the entry.
- Creating a new agent with only required fields produces an entry with no
  spurious empty-valued keys.
- `pnpm exec vue-tsc --noEmit` and `pnpm build` pass.

---

## Out of scope

- Adding editor controls for preserve-only fields (`active_status`,
  `source_types`, `done_on_success`, permission fields) — separate enhancement.
- Any change to the backend save endpoint or agent runtime.
