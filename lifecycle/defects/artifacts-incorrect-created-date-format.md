---
title: Artifacts Incorrect Created Date Format
type: defect
status: done
lineage: artifacts-incorrect-created-date-format
priority: normal
labels:
    - defect
    - artefacts
    - backend
    - agent-runner
---

When the analyst is creating requirements, the date in the created field in the Frontmatter is not in the correct format, see lifecycle/requirements/artifact-breadcrumb-remove-broken-links-2.md as example.  They are 
```
created: "2026-04-27"
```

Instead of
```
created: "2026-05-06T00:00:00+10:00"
```