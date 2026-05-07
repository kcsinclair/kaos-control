---
title: Roadmap Graph Returns 404 and Release Artifact Links Navigate to /undefined
type: defect
status: done
lineage: roadmap-graph-404-and-release-artifact-undefined-links
created: "2026-05-07T11:00:29+10:00"
priority: normal
labels:
    - defect
    - frontend
    - backend
    - roadmaps
    - releases
    - artefacts
release: May2026
---

# Roadmap Graph Returns 404 and Release Artifact Links Navigate to /undefined

## Reproduction Steps

1. Open the application at http://127.0.0.1:8042.
2. Navigate to the roadmap graph view for the `kaos-control` project.
3. Observe the "Not Found" message displayed on screen and the browser console error: `Failed to load resource: the server responded with a status of 404 (Not Found)` for `http://127.0.0.1:8042/api/p/kaos-control/roadmap/graph`.
4. Navigate to a release artifact detail page.
5. Observe empty boxes in the Artifacts section.
6. Click one of the empty artifact boxes and observe the browser navigates to `http://127.0.0.1:8042/p/kaos-control/artifacts/undefined`.

## Expected Behaviour

- The roadmap graph view should successfully fetch graph data from `/api/p/kaos-control/roadmap/graph` and render the graph.
- The release detail page should display linked artifacts with their correct titles and IDs, and clicking an artifact should navigate to its correct detail URL (e.g. `/p/kaos-control/artifacts/<valid-id>`).

## Actual Behaviour

- The `/api/p/kaos-control/roadmap/graph` endpoint returns HTTP 404, causing the roadmap view to display a "Not Found" error with no graph rendered.
- On the release detail page, artifact entries are rendered as empty boxes with no title or identifier. Clicking them navigates to `/p/kaos-control/artifacts/undefined`, indicating the artifact ID property is `undefined` at the point of link construction.
