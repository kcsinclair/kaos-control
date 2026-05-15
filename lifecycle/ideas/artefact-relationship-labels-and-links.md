---
title: Artefact Relationship Labels and Clickable Links
type: idea
status: approved
lineage: artefact-relationship-labels-and-links
created: "2026-05-10T09:35:28+10:00"
priority: medium
labels:
    - artefacts
    - frontend
    - enhancement
    - usability
release: KC-Release2
---

# Artefact Relationship Labels and Clickable Links

In the Artefact detail view, the relationship section currently labels both inbound and outbound parent relationships as "PARENT". This is ambiguous and misleading depending on direction. For inbound relationships, the label should read "PARENT OF" (the current artefact is the parent of the listed item). For outbound relationships, the label should read "CHILD OF" (the current artefact is a child of the listed item).

In addition to the label fix, all relationship entries (parent and child links) should be rendered as clickable links that navigate directly to the referenced artefact. Currently these are displayed as plain text, requiring the user to manually search for the related artefact.
