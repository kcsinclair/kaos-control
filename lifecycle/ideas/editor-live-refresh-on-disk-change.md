---
title: Auto-Refresh Editor on Disk Change
type: idea
status: clarifying
lineage: editor-live-refresh-on-disk-change
created: "2026-04-27T16:24:28+10:00"
priority: high
labels:
    - frontend
    - enhancement
    - watcher
    - usability
    - vue
---

# Auto-Refresh Editor on Disk Change

When a Markdown file is open in the editor and its content changes on disk, the editor should automatically reload the file and display it with the updated content — no manual refresh or navigation required.

A brief, unobtrusive notification (e.g. a toast or inline banner) should appear to inform the user that the file has been refreshed, so they are aware the content has changed underneath them rather than being silently replaced.

The watcher backend already emits `file.changed` WebSocket events on disk mutations; the frontend should subscribe to those events, detect when the currently-open file is affected, and trigger a re-fetch and re-render of the editor content.
