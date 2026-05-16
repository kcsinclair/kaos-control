---
title: Agent Run Dialog — rad-overlay Intercepts Pointer Events on Run Button
type: defect
status: done
lineage: agent-run-dialog-overlay-intercepts-run-button
created: "2026-05-16T14:00:00+10:00"
priority: high
labels:
    - defect
    - frontend
    - agents
release: KC-Release2
assignees:
    - role: frontend-developer
      who: agent
---

# Agent Run Dialog — rad-overlay Intercepts Pointer Events on Run Button

## Reproduction Steps

1. Navigate to `/p/testproject/agents`.
2. Click the **Run Agent** button to open the run dialog.
3. Select the `stub-agent` chip.
4. Fill in a target artifact path.
5. Click the **Run** button inside the dialog.

## Expected Behaviour

Clicking Run submits the agent run request and triggers an `agent.started` WebSocket event.

## Actual Behaviour

The E2E test `Flow 04 — Agent run` fails with:

- `Timed out waiting for agent.started WS event` — no agent run is started.
- `page.click: Test ended.` with the error log showing `<div class="rad-overlay">` intercepts pointer events every time Playwright attempts to click the button.

The Playwright click log shows:

```
<div data-v-406c039c="" data-v-e6704dfd="" class="rad-overlay">…</div> intercepts pointer events
```

Additionally, the locator `button.btn-primary:has-text("Run")` is ambiguous: it resolves to two elements (both the "Run Agent" page button and the dialog's "Run" button), and Playwright proceeds with the first match ("Run Agent"), which is blocked by the overlay.

Test file: `tests/e2e/flows/04-agent-run.spec.ts`

## Notes

Two related issues may both need fixing:

1. **Overlay z-index / pointer-events**: the `rad-overlay` element should not intercept clicks on dialog action buttons when the dialog is open.
2. **Button selector ambiguity**: both the "Run Agent" trigger button and the dialog's "Run" confirm button carry the `btn-primary` class; differentiating them (e.g. scoping the confirm button inside the dialog, or giving it a distinct class) would make the UI more robust.
