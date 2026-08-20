---
created: "2026-08-14T15:59:04+10:00"
title: 'Frontend plan — graceful defect-generation errors and config-health guidance'
type: plan-frontend
status: done
lineage: defect-generate-missing-template
parent: lifecycle/defects/defect-generate-missing-template.md
release: KC-Release5
---

# Frontend plan — "New Defect → Generate" graceful degradation

The New Defect flow calls `generateIdea(project, input, 'defect', …)` from
`web/src/stores/brainDump.ts` (`generate`, L42-66). When the backend returns a
config/template error, the store currently surfaces the raw `ApiError.message`
(`brainDump.ts:60`) straight into the modal — so the user literally sees
`idea-capture agent has no template "defect-generate"` with no next step.

Once the backend (see [[defect-generate-missing-template-2-be]]) supplies a
default template, this path should normally succeed. This plan makes the failure
modes that remain **actionable** rather than raw, and adds an optional config-
health hint. Test coverage is in [[defect-generate-missing-template-4-test]].

---

## Milestone 1 — Map generation errors to actionable guidance

**Description.** Translate backend error codes into human, actionable messages
in the brain-dump store instead of echoing the raw server string. Handle the new
`template_unavailable` (422) code from
[[defect-generate-missing-template-2-be]] Milestone 4, and provide a sensible
generic fallback for any other `config`/`template` error code.

**Files to change.**
- `web/src/stores/brainDump.ts` — in `generate` (and `createDoc`) `catch`
  blocks, branch on `ApiError.code`. For `template_unavailable` /
  `config_error`, set `error.value` to guidance like: "Defect generation isn't
  configured for this project. Ask an admin to add a `defect-generate` template
  to the idea-capture agent, or edit the defect manually." Keep the existing
  generic message for unknown errors.
- `web/src/api/client.ts` — confirm `ApiError` exposes `.code` (add if missing)
  so the store can branch on a stable code rather than message text.
- `web/src/api/ideaChat.ts` — no contract change expected; verify it propagates
  `ApiError` with `.code` intact.

**Acceptance criteria.**
- A 422 `template_unavailable` response yields the actionable guidance string in
  `store.error`, never the raw `has no template` text.
- The store branches on `ApiError.code`, not on substring matching of the
  message.
- On error, `phase` returns to `'input'` (unchanged behaviour) so the user can
  retry or switch to manual entry.

---

## Milestone 2 — Modal presentation: actionable error + manual-entry escape hatch

**Description.** In the New Defect / brain-dump modal, render the mapped error
distinctly (not as a stack-trace-looking blob) and, when generation is
unavailable, offer a one-click path to create the defect manually so the user is
never dead-ended.

**Files to change.**
- `web/src/components/idea/BrainDumpModal.vue` — present `store.error` in a
  dismissible inline alert. When the error is the config/template class, show a
  secondary action (e.g. "Create defect manually") that seeds a minimal defect
  frontmatter/body from the current `input` and posts it via the existing
  `POST /p/:project/artifacts` path (reuse the `createDoc`-style flow, adapted
  for `type: defect`, `stage: defects`).
- Add a small store action (e.g. `createDefectManually(project)`) in
  `web/src/stores/brainDump.ts` mirroring `createDoc` but writing a defect
  artifact (frontmatter `type: defect`, `status: raw`, `lineage: <slug>`,
  `labels: ['defect']`).

**Acceptance criteria.**
- When generation fails with a config/template error, the modal shows the
  actionable message and a visible "Create defect manually" action.
- Triggering the manual action creates a defect artifact under
  `lifecycle/defects/` and closes the modal on success.
- The generic (non-config) error path shows the alert **without** the manual
  escape hatch (retry only), preserving current behaviour.

---

## Milestone 3 — Optional config-health hint

**Description.** Surface the backend config-health/repair signal (from
[[defect-generate-missing-template-2-be]] Milestone 4) so admins learn that the
runtime auto-filled a missing template and should persist it in `config.yaml`.

**Files to change.**
- `web/src/api/` — add a typed client call for the config-health endpoint
  (`GET /p/:project/config/health`) returning `{ repairs: RepairNote[] }`.
- `web/src/types/api.ts` — add `RepairNote` and the response type.
- A lightweight banner/toast component (reuse existing notification/banner
  pattern; check `web/src/components/project/` and any global toast store) shown
  to admin roles when `repairs` is non-empty, worded as a non-blocking hint:
  "kaos-control auto-filled N missing agent template(s) at startup. Add them to
  lifecycle/config.yaml to make it permanent."

**Acceptance criteria.**
- When `repairs` is non-empty, an admin sees a single non-blocking hint naming
  the repaired template(s); non-admins see nothing.
- When `repairs` is empty, no banner renders.
- The hint is dismissible and does not block defect generation or any other
  flow.

---

## Milestone 4 — Frontend tests

**Description.** Cover the store error-mapping and the modal's actionable-error
+ manual-entry behaviour with vitest. Detailed matrix lives in
[[defect-generate-missing-template-4-test]]; this milestone is the frontend
harness for it.

**Files to change.**
- `web/src/stores/__tests__/brainDump.spec.ts` — add cases: 422
  `template_unavailable` → actionable message + `phase === 'input'`; generic
  error → generic message; `createDefectManually` success path.
- `web/src/components/idea/__tests__/BrainDumpModal.spec.ts` (new or existing) —
  assert the actionable alert renders and the manual-entry action appears only
  for the config/template error class.

**Acceptance criteria.**
- `cd web && pnpm test` passes with the new cases.
- Tests assert on `ApiError.code` branching, not on raw message strings.
- A test fails if the raw `has no template` string is ever shown to the user.
