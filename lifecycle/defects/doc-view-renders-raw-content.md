---
title: Documentation View Renders Raw Base64 and Markdown Instead of Formatted Output
type: defect
status: done
lineage: doc-view-renders-raw-content
created: "2026-06-13T12:01:08+10:00"
priority: normal
labels:
    - defect
    - frontend
    - ux
    - usability
    - editor
release: KC-Release3
assignees:
    - role: frontend-developer
      who: agent
---

# Documentation View Renders Raw Base64 and Markdown Instead of Formatted Output

## Reproduction Steps

1. Open the documentation view for an artifact or file that contains PNG images or HTML content.
2. Observe the rendered output for image and HTML elements.
3. Open the documentation view for an artifact written in Markdown.
4. Observe the rendered output for the Markdown content.

## Expected Behaviour

- PNG images should be rendered as visible images, not as raw base64-encoded strings.
- HTML content should be rendered as formatted HTML, not displayed as raw HTML markup.
- Markdown content should be rendered as a formatted preview document (headings, bold, lists, code blocks, etc.), not as raw Markdown source text.

## Actual Behaviour

- PNG content is displayed as a raw base64 string (e.g. `data:image/png;base64,iVBORw0K...`).
- HTML content is displayed as raw HTML markup rather than being rendered.
- Markdown content is displayed as raw Markdown source (e.g. `## Heading`, `**bold**`) instead of a formatted preview document.

## Findings — 2026-06-13 (reopened: marked done by run 8fb98baaf72049ac but not resolved)

The frontend-developer run implemented the fix in `DocsEditorView.vue` but wrote
it against assumptions that don't match the backend response, and there was no
QA step that verified the rendered output against the running app. Live status:

- **PNG — fixed.** Backend `GET /docs/*` returns `body_base64` + `mime`
  (`image/png`) for non-markdown; `isImage` + `<img src="data:…">` renders.
- **HTML — still broken.** `isHtml` checks `mime === 'text/html'`, but
  `internal/http/docs.go` derives mime via `http.DetectContentType`, which returns
  `"text/html; charset=utf-8"` — the exact match fails, so HTML falls through to
  the "can't be edited inline" fallback. Additionally the iframe binds
  `:srcdoc="body"`, but the backend returns `body_base64` (not `body`) for
  non-markdown files, so the iframe would be empty even if the mime matched.
  Fix (frontend): `isHtml` should match the `text/html` prefix, and the iframe
  should render the decoded `body_base64`.
- **Markdown — still raw for editor roles.** `showPreview` defaults to
  `!canEdit`, so read-only users see the rendered preview but editors
  (product-owner/analyst/developer/QA) default to the CodeMirror source editor —
  i.e. raw markdown. The defect expects the documentation *view* to render a
  formatted preview by default; recommend defaulting to preview for all roles
  with an Edit/Preview toggle (editors click Edit to modify).

Process note: this defect went frontend-developer → done with no QA verification
against the running UI, which is why the gaps shipped as "done".
