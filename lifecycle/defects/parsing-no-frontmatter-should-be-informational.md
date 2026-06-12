---
title: Parsing errors should treat missing frontmatter as informational, not an error
type: defect
status: approved
lineage: parsing-no-frontmatter-should-be-informational
created: "2026-06-11T11:06:00+10:00"
priority: normal
labels:
    - defect
    - backend
    - artifacts
    - reliability
release: KC-Release3
assignees:
    - role: backend-developer
      who: agent
---

# Parsing errors should treat missing frontmatter as informational, not an error

## Reproduction Steps

1. Place a markdown file under `lifecycle/` that contains no YAML frontmatter block (no leading `---` delimiter).
2. Start the server or trigger a re-index (e.g. via the rehydrate-from-disk button or by saving the file).
3. Observe the server logs.

## Expected Behaviour

The indexer detects the absence of a frontmatter block before attempting to parse it and emits an informational log entry (e.g. `INFO skipping file — no frontmatter detected: <path>`). No error is recorded against the file, and the rest of the index scan continues normally.

## Actual Behaviour

The parser attempts to process the file and emits a parsing error (e.g. `ERROR failed to parse frontmatter: <path>`), which pollutes error logs with noise for files that are legitimately frontmatter-free (e.g. READMEs, scratch notes, partial drafts).
