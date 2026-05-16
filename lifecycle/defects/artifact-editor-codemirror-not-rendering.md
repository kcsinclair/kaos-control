---
title: Artifact Editor — CodeMirror Not Rendered; file.changed WS Event Not Received After Save
type: defect
status: in-development
lineage: artifact-editor-codemirror-not-rendering
created: "2026-05-16T14:00:00+10:00"
priority: high
labels:
    - defect
    - frontend
    - editor
    - websocket
release: KC-Release2
assignees:
    - role: frontend-developer
      who: agent
---

# Artifact Editor — CodeMirror Not Rendered; file.changed WS Event Not Received After Save

## Reproduction Steps

1. Start the kaos-control binary with the E2E test fixtures.
2. Log in as admin and navigate to an artifact detail page (e.g. `/p/testproject/artifacts/lifecycle/requirements/smoke-req-01.md`).
3. Wait up to 10 s for the CodeMirror editor (`.cm-content`) to appear.
4. If the editor does appear, click inside it, append text, and press the Save button.
5. Observe whether a `file.changed` WebSocket event is broadcast on `/api/p/testproject/ws` within 8 s of saving.

## Expected Behaviour

- The CodeMirror editor renders (`.cm-content` element visible) within 10 s of navigation.
- Clicking Save writes the content to disk and the server broadcasts a `file.changed` WS event within 8 s.

## Actual Behaviour

The E2E test `Flow 02 — Edit and save artifact` fails with two errors:

1. `Timed out waiting for file.changed WS event` — the WS event is never received.
2. `locator('.cm-content').first()` — element not found, indicating the editor did not render.

Test file: `tests/e2e/flows/02-edit-save.spec.ts`

## Notes

Both the editor render failure and the missing WS event may share the same root cause (broken route or component mounting error in the artifact detail view). Investigate in order: (1) verify the artifact detail route renders the editor component, (2) verify the save handler triggers the server-side file-watcher broadcast.
