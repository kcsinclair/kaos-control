---
title: Child Artifacts Not Linked After Agent Run
type: defect
status: done
lineage: child-artifacts-not-linked
created: "2026-06-11T08:51:31+10:00"
priority: normal
labels:
    - defect
    - artifacts
    - agent-runner
    - workflow
---

# Child Artifacts Not Linked After Agent Run

## Reproduction Steps

1. Submit an idea or requirement for processing through the agent pipeline.
2. Observe that the agent run completes and child artifacts are generated (e.g. job 661055f8, artifact `release-artefacts.md`).
3. Inspect the generated child artifacts for `parent:` frontmatter linking back to their originating artifact.
4. Navigate to the graph UI and observe the lineage connections between parent and child artifacts.

## Expected Behaviour

Each generated child artifact should include a `parent:` field in its YAML frontmatter pointing to the previous artifact in the lineage. The graph UI should display edges connecting parent and child artifacts, reflecting the correct lineage chain.

## Actual Behaviour

Child artifacts are created by the agent run (job `661055f8`, artifact `release-artefacts.md`) but the `parent:` frontmatter link is absent or incorrect. As a result, artifacts are not connected in the lineage graph and the lifecycle chain is broken.
