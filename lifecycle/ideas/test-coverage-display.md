---
title: Test Coverage Display in the Testing Page
type: idea
status: draft
lineage: test-coverage-display
created: "2026-05-10T09:22:35+10:00"
priority: normal
labels:
    - feature
    - frontend
    - testing
    - vue
---

# Test Coverage Display in the Testing Page

The testing page currently lacks visibility into test coverage metrics, making it difficult for developers and QA to understand how well the codebase is exercised by the existing test suite. Adding a test coverage display would surface this information directly within the lifecycle tool's UI.

The feature would integrate coverage data (e.g. from Go's `go test -cover` output or a coverage report file) and present it in the testing page, showing overall coverage percentages and ideally per-package or per-file breakdowns. This gives the test-developer and qa roles immediate feedback on coverage gaps without leaving the tool.

The display should update alongside test run results, and could highlight files or packages falling below a configurable coverage threshold, supporting the project's quality gates and making coverage a first-class signal in the workflow.
