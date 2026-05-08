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

Later a backend-developer might write the code, and commit to GIT, which creates a pull-request, which is reviewed by the lead-developer and approved.

What changes do we need to the existing schema are needed.

Do you have any questions or suggestions on changes to this proposal?

An analyst reads requirements and write plans, a developer reads plans and writes code.  

# optionally split the lifecycle from the code, 

allow different base directories for the different agents to work in.  allows obsidian integration for project management, but code not in obsidian.

# benefit of all this is

Work where you want when you want, keep things moving, with Visual Studio Code and Obsidian, while your robot waits for work.

# action buttons

When viewing an artefact, standard quick action buttons, e.g. approve this, etc.

# 3. Interactivity — the harder one

## how bad is this?
claude --dangerously-skip-permissions -p <prompt>

## more
Claude Code's -p mode is strictly non-interactive. There's no input channel after launch. The agent finishes, success or fail, with whatever it produced. This is a fundamental constraint of the current driver, not a missing feature.

What works today:

Agent writes uncertainty into the artifact itself (the analyst-requirements prompt already says "list specific clarifying questions in an Open Questions section — do NOT guess"). Human reads the artifact, edits in answers, reruns the agent.
The artifact lock + reviewer/approver flow is the human checkpoint.
What doesn't work today:

Mid-run "wait, should I do X or Y?" — agent has to commit to one.
Long-running back-and-forth.
Real solutions are bigger pieces of work:

1. agent.question event type — agent prints a structured marker on stdout (e.g. [[ASK]] should X be Y or Z?); supervisor pauses the agent, broadcasts a WS event, the UI prompts the human, the answer is piped back to stdin. Requires keeping the subprocess alive and writable, and a new prompt convention.
2. Switch to Anthropic SDK directly with a tool-use loop including an ask_human tool. More flexibility, but you'd be replacing claude-code-cli with your own Anthropic API integration.
3. MCP driver — already named in spec §7.2 as roadmap. The agent is itself an MCP server; tighter back-and-forth.

For the immediate term, the lowest-effort improvement is to lean harder on the "agent writes questions into the artifact, human answers, agent re-runs" loop. Update the prompt templates so every agent has an explicit "If unsure, write to Open Questions and stop" instruction.


# Marketing

When flow is on its own you want to capture your ideas and get them down onto the pixels onto the screen so that you can work on them later at the flow idea is just flow. You just want to get them into the system so that you can work on them later later you’ve got time when you’re sitting somewhere that you can work through with an agent to flush them out and do whatever you need to do but to get the idea down is the critical thing so get the idea captured and then start working on it when you have time sitting on a train sitting on a bus whatever you’re doing you can just work it through the code that soul is a problem.

You want to be able to dump everything when you can and then work through it while you are on the train.

You may not want it to be a factory, endlessly coding and using up tokens.  You may want to spread out the work and develop progressively and affordably.

## Flexible framework

This is a fixable framework that lets you work with a developer bots but it could also be humans. It runs on your computer and can use local or cloud LLM to run agents.  The concept could also apply to other workflows, like writing, marketing, etc. It could also be marketing instead of being developer could be designed. Could be anything that you need to get done. You can use this framework and the workflow to get things done.

Its about getting things done, its about idea capture and innovating rapidly.

You don't have a dev team now, use bots, as you grow, add more people and keep using the same workflow.


## additional thoughts.

I have been thinking its about my long history with software development and that I have software still in production after 20 years.  

kaos-control translates what I have learned, how to get ideas from people and get them into running code.  This process encapsulates that and has the benefit of using humans or robots to get the work done.  

The earlier idea I had before "Innovation Maker - Making Releases from Ideas.md" was how to organise my ideas about the kaos-core software and figure out which features to do first and which to wait on, and how to get coding agents working better for me.  That idea evolved into kaos-control.

**Having ideas isn't hard, staying on track, avoiding perfection and rabbit holes and making things work is hard.**
