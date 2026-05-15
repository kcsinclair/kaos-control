---
title: "Frontend Plan: Mediated Claude Driver with Permission Hooks"
type: plan-frontend
status: done
lineage: claude-hooks-driver
parent: lifecycle/requirements/claude-hooks-driver-2.md
created: "2026-05-15T14:00:00+10:00"
---

# Frontend Plan: Mediated Claude Driver with Permission Hooks

Parent: [[claude-hooks-driver]].

This plan covers the Vue 3 / TypeScript frontend changes for FR20, FR21,
and the UI portions of FR14–FR17. Backend work is in
[[claude-hooks-driver-3-be]]; test coverage is in
[[claude-hooks-driver-5-test]].

---

## Milestone 1 — TypeScript Types & WS Event Handling

### Description

Add types for the new `agent.permission` WebSocket event and the
`denied_tool_calls` array on run completion events. Wire the new event
into the agents store.

### Files to change

- **`web/src/types/api.ts`**:
  - Add `'agent.permission'` to `WsEventType` union.
  - Add `PermissionDecision` interface:
    ```typescript
    interface PermissionDecision {
      run_id: string
      tool_name: string
      target_path?: string
      command?: string
      decision: 'allow' | 'deny'
      reason: string
      policy_rule: string
      timestamp: string
    }
    ```
  - Add `denied_tool_calls?: DenialRecord[]` to `AgentRunRow`.
  - Add `DenialRecord` interface:
    ```typescript
    interface DenialRecord {
      tool_name: string
      path?: string
      command?: string
      reason: string
      rule: string
    }
    ```
  - Add `'claude-mediated'` to the driver string comment on `AgentSummary`.
  - Add `observe_only?: boolean`, `bash_allowlist?: string[]`,
    `bash_denylist?: string[]`, `on_denial?: string` to `AgentSummary`.

- **`web/src/stores/agents.ts`**:
  - Add `permissionEvents: ref<Map<string, PermissionDecision[]>>` to state
    (keyed by `run_id`).
  - In `onWsEvent`, handle `agent.permission`:
    - Append to `permissionEvents.get(runId)`.
    - Also append a formatted line to `progressLines` so it appears inline
      in the live progress view (prefix with `[PERMISSION]` and colour-code
      in `formatEvent`).
  - In `agent.finished` / `agent.failed` handler: capture
    `payload.denied_tool_calls` onto the matching `AgentRunRow`.

- **`web/src/views/project/WorkspaceView.vue`**:
  - Add `agent.permission` to the WS fan-out switch so it reaches
    `agentsStore.onWsEvent`.

### Acceptance criteria

- [ ] `agent.permission` events are received and stored per run.
- [ ] Permission events appear inline in the live progress log.
- [ ] `denied_tool_calls` is captured on completed/failed run records.
- [ ] No regressions to existing `agent.progress` / `agent.finished` handling.

---

## Milestone 2 — Permission Event Rendering in Run Timeline

### Description

Render permission decisions inline in the expanded run detail row. Denied
calls are visually distinct (red icon/badge) per AC8.

### Files to change

- **`web/src/views/project/AgentsRunsView.vue`**:
  - In the expanded `run-detail` `<tr>`, between the live progress block
    and the stderr block, add a **Permission Events** section that renders
    when `agentsStore.permissionEvents.get(run.run_id)?.length > 0`.
  - Each event rendered as a compact row:
    ```
    [allow|deny icon] tool_name  target_path/command  reason  timestamp
    ```
  - Use existing `status-chip` pattern with `data-status="allow"` (green
    via `--badge-done-*`) and `data-status="deny"` (red via
    `--badge-blocked-*`).
  - For live runs, new permission events append in real-time (reactive
    via store ref).

- **`web/src/components/agent/RunDetailModal.vue`**:
  - Add the same permission events section below the run metadata.
  - Fetch permission events from the store (they persist for the session)
    or from the run log if the store doesn't have them (for historical
    runs opened after page load — deferred to a later milestone if needed).

### Acceptance criteria

- [ ] Permission events appear in the expanded run row (AC8).
- [ ] Denied calls show a red badge/chip; allowed calls show green (AC8).
- [ ] Events update in real-time for running agents.
- [ ] The section is hidden when there are no permission events.

---

## Milestone 3 — Denied-calls Summary on Run Completion

### Description

When a run completes with denials, render a prominent summary (FR21, AC9).

### Files to change

- **`web/src/components/agent/RunDenialSummary.vue`** (new component):
  - Props: `denials: DenialRecord[]`.
  - Renders a warning card with a red/amber border and a list of denied
    tool calls:
    ```
    ⚠ N tool calls were denied during this run
    ─────────────────────────────────────────
    • Write to "internal/http/server.go" — denied: outside allowed paths
    • Bash "sudo rm -rf /" — denied: matches denylist
    ```
  - Uses `--badge-blocked-bg` / `--badge-blocked-text` CSS variables for
    the card background.
  - Includes a note: "Auto-commit was skipped. Queue is paused."

- **`web/src/views/project/AgentsRunsView.vue`**:
  - In the expanded run detail, render `<RunDenialSummary>` when
    `run.denied_tool_calls?.length > 0`, positioned above `RunSummaryCard`.

- **`web/src/components/agent/RunDetailModal.vue`**:
  - Same placement of `<RunDenialSummary>`.

- **`web/src/components/agent/RunFailureBanner.vue`**:
  - Extend to also show when `run.denied_tool_calls?.length > 0` even if
    the run status is `done` (a run can complete with denials if
    `on_denial: continue`). Use a distinct message: "This run had N denied
    tool calls. Auto-commit was skipped."

### Acceptance criteria

- [ ] Denied-calls summary renders prominently on the run detail view (AC9).
- [ ] Summary shows tool name, target path/command, and denial reason.
- [ ] The banner appears for both `done` and `failed` runs with denials.
- [ ] The component is not rendered when there are no denials.

---

## Milestone 4 — Driver Badge for `claude-mediated`

### Description

Add visual identification for the new driver type in the agent panel and
run list.

### Files to change

- **`web/src/components/agent/AgentPanelRow.vue`**:
  - Add `data-driver="claude-mediated"` CSS rule:
    ```css
    .driver-badge[data-driver="claude-mediated"] {
      background: #fef3c7;  /* amber-100 */
      color: #92400e;       /* amber-800 */
    }
    ```
    Amber distinguishes it from the purple `claude-code-cli` badge.

- **`web/src/views/project/AgentsRunsView.vue`**:
  - The existing driver badge rendering already uses `data-driver` binding,
    so it will pick up the new CSS rule automatically.
  - Add the same CSS rule to the view's scoped styles.

### Acceptance criteria

- [ ] `claude-mediated` agents show an amber driver badge.
- [ ] Existing driver badges (`claude-code-cli` purple, `ollama` blue) unchanged.

---

## Milestone 5 — Observe-only Indicator

### Description

When an agent is configured with `observe_only: true`, surface this in the
UI so operators know enforcement is not active.

### Files to change

- **`web/src/components/agent/AgentPanelRow.vue`**:
  - When `agent.observe_only` is true, render a small "observe" badge
    next to the driver badge (e.g. outline style, amber text).

- **`web/src/views/project/AgentsRunsView.vue`**:
  - In the permission events section, when the agent is in observe-only
    mode, show a banner: "Observe-only mode — all tool calls were allowed.
    Decisions shown are what would have been enforced."

- **`web/src/components/agent/RunDenialSummary.vue`**:
  - If the run was observe-only, adjust the message: "N tool calls would
    have been denied (observe-only mode — no enforcement)."

### Acceptance criteria

- [ ] Observe-only agents have a visible indicator in the agent panel.
- [ ] Permission event rendering clarifies observe-only mode.
- [ ] Denial summary adjusts its language for observe-only runs.

---

## Milestone 6 — Queue Pause Indicator

### Description

When the agent queue is paused due to denials (FR16), surface this state
in the UI.

### Files to change

- **`web/src/views/project/AgentsRunsView.vue`**:
  - If the queue is paused (existing `queue` store state or a new WS
    event), show a banner at the top of the runs view:
    "Agent queue is paused due to denied tool calls. Review the denied
    calls and resume the queue."
  - Include a "Resume Queue" button that calls the existing queue resume
    API endpoint.

### Acceptance criteria

- [ ] Queue pause state is reflected in the runs view.
- [ ] Operator can resume the queue from the UI.
- [ ] Banner disappears when the queue is resumed.
