# **Innovation Maker - Making Releases from Ideas**

# Concept
A golang web application which allows you to create and edit markdown or yaml files in a directory.  These files are the project files which are then grouped together to deliver a release.

Use the 3d-force-graph (https://github.com/vasturiano/3d-force-graph) library to create maps of the items.

This allows the product owner to better see the relationships between the requirements and the work needed.

You jump between the raw tickets and the mapping tool.  Then a tool gets all the tickets and meta data and makes a map of the tickets, with links to labels and to eachother, so that you can see dependancies between tickets and map it out.

Then you can build the roadmap easily.  Includes visualisation of the EPICS and higher value things.

The visualisation tool can filter the objects by type or labels or sprints so you can focus on the smaller pieces or the whole picture.

AI agents will read the tickets and create new items with what they are going to do.

A simple idea can be fleshed out into a more complex application, feature or release.

# Usage
The web application is accessible from anywhere with connectivity, e.g. reverse proxy if needed.

You access the application and the open a project which is a directory on a computer.

You add requirements, review and approve various stages in the development cycle.

When documents are ready for next step in the process, you can trigger an agent to perform its task and create the next artifact.

This is then reviewed and next step is done.

The agents all work in the same directory


# Capturing Requirements and Ideas
* The product owner creates requirements or ideas which need to be delivered.
* Create individual items/tickets which are requirement/idea.
* Label them with the type of work they are.
* Once ready they are commited to GIT

# Innovation Process 
* An AI Agent can read the files or access remotely using MCP.
* The first agent reads the requirement/idea and fleshes it out.
* It might create related clarifying questions about the idea.
* The clarifying questions can be answered and saved into the questions document.
* The plan phase is next.  Two parts to the plan, User Interface and Backend
* The Backend Agent reviews the tickets and documents its backend plan into related items.
* The Frontend Agent reviews the tickets creates a User Interface plan and a UI prototype created to be reviewed.
* The backend and frontend are reviewed, then rejected (rejected for replan), abonded, updated, approved
* Once the related plans are approved, the requirements and plans are handed over to a developer agent, the developer agent will be responsible for code and unit tests.
* The approved plans will be handed to a QA Agent which will review the requirement, the plans and create a QA test plan, once approved it will create any needed code for the QA testing.  This will be extending any existing QA testing.  QA testing includes headless browser robotic testing.

# JIRA Integration Feature
All of this could be an alternate engine for JIRA, any part of the process could be using JIRA, but this can also work standalone.
Put all the tickets into a project and tagged accordingly.
