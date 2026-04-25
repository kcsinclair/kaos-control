# prompts used

## Initial prompt
```
I have written some high level requirements in high-level-requirements.md for an idea I have had, grill me on this document so you can write more detailed requirements.
```

### notes

high-level-requirements.md was really the idea!

The process of going from high level requirements (idea) helped me to flesh out the idea and guide the technical direction and identify any architectural changesd neede.

## going meta

Going to restructure the directory now and create initialise for CLAUDE.md

requirements-questions.md renamed to lifecycle/requirements/Innovation Maker - Making Releases from Ideas-questions.md
detailed-requirements.md renamed to lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md

Initialised Claude and they figured out the plan and wrote CLAUDE.md

### get Claude to maintain project plans in a plans folder.

Added this to CLAUDE.md
```
## Commit Conventions

- **Plans**: Every git commit must include an updated `plans/PROJECT_PLAN.md` reflecting what changed and the current project state. Update the "Recent Changes" section and any affected "Completed" or "Planned" items before committing.
- **Implementation plans**: When a Claude Code plan file (`~/.claude/plans/*.md`) was used for implementation, copy it into `plans/` with a descriptive name (e.g., `plans/geolite2-country-lookup.md`) and include it in the commit.
```

### prompted to get that done.

```
please save the plan you just had for creating CLAUDE.md, I found this in ~/.claude/plans/quiet-drifting-minsky.md, please check new instructions around plans in CLAUDE.md
```

## second prompt for the plans

Opus should read the requirements and create plans for backend, frontend and test

```
Please read the detailed requirements in the document "lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md".  Using these requirements you will create the development plans for three agents, an agent for backend, frontend and testing, the result should be three files:
- "lifecycle/backend-plans/Innovation Maker - Making Releases from Ideas-2-be.md"
- "lifecycle/frontend-plans/Innovation Maker - Making Releases from Ideas-3-fe.md"
- "lifecycle/test-plans/Innovation Maker - Making Releases from Ideas-4-test.md"
Please let me know if you have any questions.
```

### Opus response

1. Scope: full v1 of kaos-control per §1 goals. Roadmap items in §16 explicitly excluded.
2. Parent links: all three plans get parent: requirements/Innovation Maker - Making Releases from Ideas-1.md (siblings, parallel plans per §6.3). The 2/3/4 indices are unique identifiers within the lineage, not a serial chain.
3. Shared API contract: the backend plan owns the authoritative REST + WebSocket contract; the frontend plan consumes it; the test plan references both.
4. Plan structure: each plan is phased into milestones (M1, M2…) with acceptance criteria per milestone, so Sonnet agents can ship incrementally rather than implementing everything before any review.
5. QA framework for kaos-control's own tests: I'll recommend Playwright (spec leaves it "user decides" — we're the user for this repo). Best headless browser coverage + API testing in one tool.
6. Repo layout: single Go module at the repo root with cmd/kaos-control/, internal/..., web/ (Vite source embedded into the Go binary via embed.FS).

## development time

Then going to use a sonnet agent to start creating the code. 

### third prompt for the coding backend

Switch model to Sonnet

```
Please review the development plan in "lifecycle/backend-plans/Innovation Maker - Making Releases from Ideas-2-be.md"
Please proceed, and let me know if you have any questions.
```

### fourth prompt for the coding frontend

### fifth prompt for the coding tests


# Refining the roles prompt

lets talk about the roles, currently we have:
roles:
  - product-owner
  - backend-planner
  - frontend-planner
  - developer
  - qa
  - reviewer
  - approver

proposed roles after consideration:
  - product-owner
  - analyst
  - backend-developer
  - frontend-developer
  - test-developer
  - qa
  - reviewer
  - approver

So in the workflow:
  - product-owner -> Creates and curates ideas, deciding when to start the process
  - reviewer - reviews artefacts and makes comments or changes if needed
  - approver - approves artefacts.
  - analyst -> takes an idea, asks questions if needed, then write a detailed requirement for that idea.
  - lead-developer -> once detailed requirement is approved, writes the plans for backend, frontend and test.
  - backend-developer -> once backend plan is approved, takes the backend plan and does the development
  - frontend-developer -> once frontend plan is approved, takes the frontend plan and does the development
  - test-developer -> once test plan is approved, takes the test plan and does the development
  - qa -> runs the test plan and creates defects, and gets the right developer to fix the defect, retests and resolves the defect.

Right now we are in human oversight mode, with the product-owner is also the reviewer and approver and is running an agent when an artefact is ready.

Later reviewer and approver might be different humans or agents and perform that activity separately to the product-owner.

Later an agent could be automatically started when an artefact is approved.

Right now, we need an claude-code-cli agent for analyst, lead-developer, backend-developer, frontend-developer, testing-developer, qa

Later a backend-developer might right the code, and commit to GIT, which creates a pull-request, which is reviewed by the lead-developer and approved.

What changes do we need to the existing schema are needed.

Do you have any questions or suggestions on changes to this proposal?

An analyst reads requirements and write plans, a developer reads plans and writes code.  
