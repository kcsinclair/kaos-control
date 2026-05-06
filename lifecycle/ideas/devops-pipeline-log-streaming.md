---
title: DevOps Pipeline Log Streaming View
type: idea
status: approved
lineage: devops-pipeline-log-streaming
created: "2026-05-06T15:19:10+10:00"
priority: normal
labels:
    - feature
    - frontend
    - agent-runner
    - usability
---

# DevOps Pipeline Log Streaming View

When a devops pipeline is running, display its log output in a scrolling, real-time view similar to the existing agent run view. This gives users immediate visibility into pipeline progress without needing to navigate away or poll for status.

The log panel should appear on the bottom half of the screen, allowing the user to continue viewing pipeline configuration or other context in the top half while logs stream in below. The scrolling behaviour should auto-follow new output but allow the user to scroll up to review earlier lines without losing their position.

This could reuse the existing agent log streaming infrastructure (WebSocket broadcast of log lines) and the log display component from the agent view, applying them to pipeline runs as a consistent pattern across the tool.
