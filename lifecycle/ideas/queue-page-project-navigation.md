---
title: Queue Page Project Navigation
type: idea
status: clarifying
lineage: queue-page-project-navigation
created: "2026-05-13T11:27:58+10:00"
priority: normal
labels:
    - queue
    - frontend
    - vue
    - usability
    - enhancement
release: KC-Release1
---

# Queue Page Project Navigation

The Queue page currently lacks navigation context, making it difficult for users to switch between projects or return to the current project's view. Two navigation improvements are needed: a left sidebar menu listing all registered projects for quick switching, and a clickable current project name in the output area that links back to the project detail view.

The left menu should mirror the project list available elsewhere in the app, allowing users to jump directly to any project's queue without navigating back through the main dashboard. This is especially useful when monitoring multiple concurrent agent runs.

The current project name displayed in the queue output header should be rendered as a link that navigates back to the project's main view, providing a clear breadcrumb-style escape hatch for users who want to review artifacts or trigger new runs after observing queue activity.
