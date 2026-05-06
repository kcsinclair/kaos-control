---
title: Releases and Roadmaps
type: idea
status: clarifying
lineage: releases-and-roadmaps
created: "2026-05-06T11:13:00+10:00"
priority: high
labels:
    - workflow
    - roadmaps
    - releases
release: May2026
---

# Releases and Roadmaps

Auto creating roadmaps based on releases.  Allowing simple tools for planning and visualisation of the work to be done, as well as sharing with others.

* in this model, you are always ready to release, idea to development, QA complete, release. 
* with more people involved, or when you get busier, more ideas than time to herd robots, the backlog increases
* You need to prioritise and manage what is needed and what is wanted and when.
* a roadmap is still relevant but implies some sort of timeline, it is really the higher level ideas, could be done using a label, or a new frontmatter tag, e.g. roadmap: Q3CY26
* Define a release, which has start and end dates, or start date and length (which auto calculates the end date) and the roadmap is automatically generated from the releases and the dates.
* An idea or defect is assigned to a release.  For small teams, this could be simple to, next-release, later-release, unscheduled-release or Release-1, Release-2, Future-Release, Unscheduled, or can be as granular as needed.
* Product owner can use future releases, e.g "CY28" with a date of 31 Dec 2028 for things to be done that year.
* One or more sprints go into a release, sprints not needed for smaller teams, a release could be a sprint.  No changes to sprints right now.
* The Roadmap will have two views, a gantt chart and the 2D/3D graphs.  The roadmap will focus on ideas and defects, other items are not included at this stage.
* The gannt chart view will be columns where the columns represent a time period, either week, month, quarter, half year or year.  A bar would be placed on the gantt chart in the correct column for the dates and a suitable length, the bar should include a summary of how many ideas and defects are included, clicking on a release should display a modal with cards for each idea and defect included.
* The Gannt chart will have options for managing releases, "Create Release" button, which shows a modal to enter start date and length.
* When viewing a release in the modal, you can click edit or delete, when editing you can change the release name, and the dates.  If a release is renamed, all artefacts using that release should be updated.  When you delete a release you only delete the release in the database, no artefacts are updated.
* The roadmap 2D and 3D view, which is the releases as larger items and the backbone of the map, linked based on the dates, with the ideas to be included in each release added to the map.  There should be a Roadmap left menu item which shows this view.  The code should reusing/leverage existing as much as possible.
* When editing an artefact, the release box should be a select list from the defined releases, optionally allowing the user to create new release.
* The kanban should filter by releases
