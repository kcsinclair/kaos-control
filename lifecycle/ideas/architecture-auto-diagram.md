---
title: Auto-Generate Architecture Diagram from the Codebase
type: idea
status: draft
lineage: architecture-auto-diagram
created: "2026-08-14T12:30:00+10:00"
priority: normal
labels:
    - architecture
    - visualization
    - enhancement
parent: lifecycle/ideas/architectural-artefacts.md
---

# Auto-Generate Architecture Diagram from the Codebase

Run an analysis against the project's actual code and generate an
architecture diagram of the system as built — components and the
interactions between them — rather than (only) the intended architecture
chosen in the wizard. The idea is that you can visualise and understand the
real architecture, and see how the tech is interacting with the architecture.

The generated diagram would be stored as an artifact under
`lifecycle/architecture/` (a document/diagram for humans and robots, per
[[architectural-artefacts]]) and displayed in the
[[architecture-overview-view]] alongside the chosen architecture, making
drift between *intended* and *actual* visible at a glance. Regeneration could
be run on demand or as a devops pipeline step.

Not scheduled for KC-Release5 — captured here so the KC-Release5 overview
view leaves a slot for it. Candidate approaches (static analysis of imports /
package structure, an agent run with a diagramming prompt, or an existing
architecture-diagramming tool) to be evaluated at requirements time.
