---
title: "Frontend Plan — Surface Permission-Mode Precheck Failures in Agent Runs View"
type: plan-frontend
status: done
lineage: agent-permission-precheck
parent: lifecycle/requirements/agent-permission-precheck-2.md
created: "2026-05-12T15:45:00+10:00"
priority: high
labels:
    - agent
    - reliability
    - frontend
    - vue
release: KC-Release1
---

# Frontend Plan — Surface Permission-Mode Precheck Failures in Agent Runs View

Implements FR5 / AC7 from [[agent-permission-precheck-2]] on the SPA
side. Small surface area — one new component, one edit to the
existing agent-runs detail panel.

## Milestone 1 — Add `precheck_failure` to the AgentRun type

### Description

The structured failure payload from the backend (FR5) carries three
new fields: `reason`, `observed_permission_mode`, `remediation[]`.
Type these so the rest of the frontend can rely on them.

### Files to change

- **Edit** `web/src/types/api.ts` — add to the `AgentRun` type (or
  wherever it lives):

  ```ts
  export interface AgentRun {
    …
    /** Stable reason code on failure; null on success / pending. */
    failure_reason?: 'permission_mode_default' | 'precheck_timeout' | string | null
    /** Set when failure_reason === 'permission_mode_default'. */
    observed_permission_mode?: string | null
    /** Set on precheck-related failures; up to ~5 short remediation lines. */
    remediation?: string[] | null
  }
  ```

  And extend `WsEvent` payload shape for the `agent.failed` event to
  surface the same fields.

### Acceptance criteria

- `pnpm build` clean (vue-tsc).
- No type errors in existing `web/src/` callers.

---

## Milestone 2 — `RunFailureBanner` component

### Description

A reusable component that renders a structured failure block when a
run terminates with one of the precheck reason codes. Used in the
agent-runs detail panel (and could be reused elsewhere later).

### Files to change

- **New** `web/src/components/agent/RunFailureBanner.vue`:

  Props: `failureReason: string`, `observedMode?: string | null`,
  `remediation?: string[] | null`.

  Renders a coloured panel (existing `--color-error-subtle` /
  `--color-error` palette) with:

  - A heading derived from the reason code:
    - `permission_mode_default` → "Claude Code is in default permission mode"
    - `precheck_timeout` → "Claude Code did not start within the expected time"
    - other → "Run failed: {reason}"
  - A one-sentence body that includes the observed mode if present:
    "kaos-control needs Claude Code to run in `bypassPermissions`
    mode, but the agent run reported `default`."
  - An ordered list of `remediation` strings, with each step
    rendered with a step number and the inline `code` parts (`backtick`
    spans inside the strings) styled as code.
  - A small "What does this mean?" disclosure that expands to a
    paragraph explaining why the precheck exists (one tight paragraph
    cribbed from the requirement's "Why Claude Code ignores the flag"
    list).

### Acceptance criteria

- Vitest: `tests/web/RunFailureBanner.test.ts` covers:
  - Renders when `failureReason` is one of the known precheck codes.
  - Includes the observed mode in the body when present.
  - Renders each remediation step.
  - Renders a fallback heading for unknown reason codes.

---

## Milestone 3 — Render the banner in the agent runs detail panel

### Description

The agent-runs view already shows per-run detail (target, status,
elapsed, log link). Render `<RunFailureBanner>` near the top of the
detail panel when the selected run has `state === 'failed'` and
`failure_reason` is set.

### Files to change

- **Edit** `web/src/views/project/AgentsRunsView.vue` — find the
  detail-panel template block (the part that renders for the
  currently-selected run) and add:

  ```vue
  <RunFailureBanner
    v-if="selectedRun?.state === 'failed' && selectedRun.failure_reason"
    :failure-reason="selectedRun.failure_reason"
    :observed-mode="selectedRun.observed_permission_mode"
    :remediation="selectedRun.remediation"
  />
  ```

  Placement: directly under the run header line, above the existing
  log-stream / output panel.

### Acceptance criteria

- Vitest: extend the existing `AgentsRunsView` test (or add a focused
  one) to mount with a mocked store containing a failed run carrying
  the precheck payload and assert the banner is rendered.
- Vitest: a second case with a failed run whose `failure_reason` is
  null (a "regular" failure from before this feature shipped) — the
  banner does NOT render; existing failure UI is unaffected.

---

## Milestone 4 — WS event handler updates

### Description

The existing `agent.failed` WS event already updates the runs store.
Confirm the new payload fields pass through to `selectedRun` without
extra work; add a small unit test to lock that in.

### Files to change

- **Edit** `web/src/stores/agents.ts` — verify the `agent.failed`
  handler copies the new fields to the run row. The handler likely
  uses `Object.assign` already; if it whitelists fields, add the
  three new keys.

### Acceptance criteria

- Vitest: `tests/web/agentsStore.precheckFailure.test.ts` dispatches
  a synthetic `agent.failed` WS event with the precheck payload and
  asserts the matching run row in the store now has the three new
  fields populated.

---

## Verification (end-to-end)

1. `pnpm build` clean.
2. `pnpm test` clean (existing + new vitest cases pass).
3. Manual smoke:
   - Backend running on a machine that triggers the precheck failure
     (or with the test fixture that injects a `default` init event).
   - Launch an agent run.
   - Verify the agent-runs view shows the run as `failed` and renders
     the `RunFailureBanner` with the three remediation steps.
   - Verify the disclosure expands and the body text mentions the
     observed mode (`default`).

## Risk notes

- **Banner placement bloat.** If the detail panel header is already
  cluttered, the banner can push existing controls below the fold.
  If that's a problem in practice, consider collapsing the
  remediation list by default with a single-line summary and an
  "expand" affordance. Defer until visual review.

- **Future remediation strings.** The backend produces the strings;
  the frontend just renders them. If backticks inside the strings
  prove insufficient for code styling, switch to a small marked-up
  format later (e.g. each remediation as an object with `text` and
  optional `command` fields). Out of scope for KC-Release1.
