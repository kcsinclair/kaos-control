---
title: Documentation View Renders Raw Base64 and Markdown Instead of Formatted Output
type: defect
status: in-development
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
