---
title: 'Test plan — defect-generate template fallback, self-repair, and graceful UI'
type: plan-test
status: approved
lineage: defect-generate-missing-template
parent: lifecycle/defects/defect-generate-missing-template.md
release: KC-Release5
---

# Test plan — "New Defect → Generate" missing-template fix

Verifies the backend fix ([[defect-generate-missing-template-2-be]]) and the
frontend degradation ([[defect-generate-missing-template-3-fe]]): defect
generation works on a fresh project, never hard-errors with
`... agent has no template "defect-generate"`, self-repairs missing config, and
degrades gracefully when a genuine misconfiguration remains.

Suites (per `CLAUDE.md` / config.yaml qa runner): Go unit (`make test-unit`),
Go integration (`make test-integration`), web vitest (`cd web && pnpm test`),
and Playwright e2e (`make test-e2e`).

---

## Milestone 1 — Backend unit: template fallback resolves defect-generate

**Description.** Prove `resolveIdeaCaptureConfig` never hard-errors for the
generation keys.

**Files to change.**
- `internal/http/idea_chat_test.go` (new or extend).

**Acceptance criteria.**
- Given an `idea-capture` agent with **no** `defect-generate` key,
  `resolveIdeaCaptureConfig(p, "defect-generate")` returns a non-empty
  `SystemPrompt` and `nil` error.
- Same asserted for `idea-generate` and `doc-generate`, and for the
  no-agent-configured case.
- The returned `defect-generate` default contains the `## Reproduction Steps`,
  `## Expected Behaviour`, `## Actual Behaviour` section instructions and the
  mandatory `defect` label instruction.
- An unknown key returns a non-nil error whose message does **not** contain the
  raw phrase `has no template` leaking to any HTTP client (that raw string is
  confined to internal resolution).

---

## Milestone 2 — Backend unit: config validate-and-repair

**Description.** Cover `ValidateAndRepair` in `internal/config`.

**Files to change.**
- `internal/config/config_test.go`.

**Acceptance criteria.**
- A config whose `idea-capture` agent lacks `defect-generate` is repaired: the
  runtime config gains a non-empty `defect-generate` template and exactly one
  `RepairNote` naming that agent + key.
- A config with a **custom** `defect-generate` is left unchanged and yields no
  repair note for that key (assert the custom string is preserved verbatim).
- A config with no `idea-capture` agent gains one with all three generation
  keys and correct defaults (`driver: inline`, `model: claude-sonnet-4-6`,
  `allowed_write_paths: [lifecycle/ideas]`).
- Structurally invalid configs (empty stages, agent missing driver) still cause
  `LoadProject` to error — repair does not mask them.

---

## Milestone 3 — Backend unit: init template + no-drift guard

**Description.** Guard the init template and single-source-of-truth
consolidation ([[defect-generate-missing-template-2-be]] Milestones 3 & 5).

**Files to change.**
- `internal/initcmd/initcmd_test.go`.

**Acceptance criteria.**
- A project scaffolded from the init template has an `idea-capture` agent whose
  `prompt_templates` includes `idea-capture`, `idea-generate`, and
  `defect-generate`.
- Loading the scaffolded config produces zero repair notes.
- Drift guard: the init template's `defect-generate` and the Go default template
  agree on the required `##` sections and the `defect` label contract.

---

## Milestone 4 — Backend integration: fresh project generates a defect

**Description.** End-to-end HTTP against a freshly scaffolded project — the
exact scenario from GitHub issue #16.

**Files to change.**
- `tests/` — new integration test (e.g. `defect_generate_test.go`) using the
  existing `testEnv` harness (admin auto-login; devops/URL helpers return full
  URLs — see project memory).
- `lifecycle/tests/` — artifact describing what this integration test covers.

**Acceptance criteria.**
- `POST /api/p/:project/ideas/generate` with body `{ "input": "<≥5 words
  describing a bug>", "type": "defect" }` against a project scaffolded from the
  init template returns `200` with a proposal whose `body` contains the three
  defect `##` sections and whose `frontmatter`/`labels` include `defect`;
  `target_dir` is `lifecycle/defects`.
- Against a config with `idea-capture` present but `defect-generate` stripped,
  the same request still returns `200` (runtime repair / default fallback), not
  `500`, and the response body never contains `has no template`.
- A genuinely unresolvable key path returns `422` with code
  `template_unavailable` (not `500`), asserted via a targeted request or a
  handler-level test if no request can trigger an unknown key.

---

## Milestone 5 — Web vitest: store + modal degradation

**Description.** Cover the frontend mapping and modal behaviour from
[[defect-generate-missing-template-3-fe]].

**Files to change.**
- `web/src/stores/__tests__/brainDump.spec.ts`.
- `web/src/components/idea/__tests__/BrainDumpModal.spec.ts` (new if absent).

**Acceptance criteria.**
- A mocked 422 `template_unavailable` response drives `store.error` to the
  actionable guidance string and sets `phase === 'input'`.
- A mocked generic error yields the generic message and no manual-entry action.
- The modal renders the actionable alert and exposes "Create defect manually"
  only for the config/template error class; the manual action posts a defect
  artifact and resolves.
- A test asserts the raw `has no template` string is never rendered.

---

## Milestone 6 — Playwright e2e: New Defect → Generate happy path

**Description.** Drive the real UI on a built binary + scaffolded project.

**Files to change.**
- e2e suite under the `make test-e2e` target (Playwright).

**Acceptance criteria.**
- Open the New Defect modal, enter a ≥5-word description, click **Generate**,
  and see a defect **preview** (with reproduction/expected/actual content) — no
  error banner, and specifically not the `idea-capture agent has no template
  "defect-generate"` message.
- Accept the proposal and confirm a defect artifact is created under
  `lifecycle/defects/`.

---

## Regression / exit criteria

- All four suites green: `make lint`, `make test-unit`,
  `make test-integration`, `cd web && pnpm test`, `make test-e2e`.
- The string `idea-capture agent has no template "defect-generate"` cannot reach
  an end user through any generation path.
- Fresh-project defect generation works without manual config edits.
