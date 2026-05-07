---
title: Dashboard Home Screen
type: idea
status: done
lineage: dashboard-home-screen
created: "2026-05-06T14:48:54+10:00"
priority: normal
labels:
    - feature
    - frontend
    - vue
release: May2026
---

# Dashboard Home Screen

Introduce a new Dashboard page as the default home screen for each project. It should appear as the first item in the left navigation panel and be the landing page when a project is opened. The dashboard provides an at-a-glance summary of project health and activity without requiring the user to navigate into specific artifact views.

The left side of the dashboard should display visual summaries of artifact status. This includes a time-series graph showing how many artifacts have transitioned to 'done' over time (sourced from the activity feed), and a pie chart breaking down the current status distribution of all tickets that are not yet done. Additional summary widgets could include counts by type, in-progress vs blocked items, or recent agent activity.

The right side of the dashboard should display the project activity feed, giving users immediate visibility into recent changes, transitions, and agent outputs. The layout should be responsive and the dashboard should be extensible so that new widgets can be added as the product evolves.
