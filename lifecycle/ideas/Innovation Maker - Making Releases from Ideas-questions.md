---
title: "Innovation Maker — Requirements Clarification Q&A"
priority: normal
type: idea
status: done
lineage: innovation-maker
parent: ideas/Innovation Maker - Making Releases from Ideas.md
labels:
    - artefacts
    - og
    - feature
---

# Requirements Clarification Questions

Answer inline under each question. Leave blank (or write `TBD`) for anything you haven't decided yet.

---

## 1. Users & collaboration

**1.1** Single-user or team? "Accessible from anywhere" suggests team, but "directory on a computer" suggests personal. Which is it?

> Initially this will be a single user system, but it is expected that it will be a multi-user system, an instance of the server will run on a directory and multiple users might be accessing it.

**1.2** If multi-user: auth (local accounts? SSO? none)? What happens when two people edit the same ticket at once — last-write-wins, locking, CRDT, or "resolve via git merge"?

> local accounts initially, sso in the roadmap
> when editing tickets in the GUI, the GUI should lock the ticket

**1.3** Roles: is "product owner" the only human role, or are there separate reviewers/approvers/devs with different permissions?

> There should be many roles and one or more humans will fulfil those roles, it might be deterimined later that an agent could perform the roles as well.  
> So go with multiple roles and a mapping between users and roles, and agents and roles.
> Treat all the roles as human or agent, there could be human developers and agentic developers for example.

---

## 2. Ticket file format (biggest unanswered question)

**2.1** Markdown **or** YAML? Or markdown-with-YAML-frontmatter? Pick one canonical shape — it drives everything (parser, graph builder, agent contracts, JIRA mapping).

> markdown-with-YAML-frontmatter

**2.2** What are the **required fields** on a ticket? (id, title, type, status, labels, links, assignee, sprint, parent, depends_on, …?)

> keep it simple and flexible, initially tickets will be single things as more people get involved we will need more structured tickets
> For now we could use markdown headings as the field names and the text under as the content.  That would be really flexible.  YAML frontmatter would also be handy for name value pairs.

**2.3** How are **relationships** encoded — `depends_on: [TICKET-12]` in frontmatter, wiki-style `[[links]]` in the body, or a separate graph file? This is the input to the 3d-force-graph, so it has to be unambiguous.

> We should support hard links in the files, lets support both yaml frontmatter and wiki-style links.

**2.4** Filenames: human-readable slugs, ULID-style IDs, or both (`AUTH-001-login-flow.md`)? Who assigns the ID?

> human-readable slugs

---

## 3. Project / release / epic hierarchy

**3.1** You mention ticket, EPIC, release, sprint, label. What's the hierarchy — release → epic → ticket, with labels and sprints orthogonal? Or are all of these just "items" with different `type:` values and everything linked via relationships?

> EPIC's are things which will be across multiple releases.
> A release will handle multiple tickets
> Sprints are time based and will contain tickets, normally a sprint should result in a release but not always.

**3.2** Can a ticket belong to multiple releases/epics, or exactly one?

> a ticket is the thing can be resolved in a single release.  If it spans multiple releases it is an epic.

---

## 4. Workflow state machine (needs to be precise)

**4.1** Exact states per ticket. I'm reading: `draft → clarifying → plan-BE + plan-FE (parallel) → dev → qa → done`, with `rejected / abandoned / updated / approved` as plan sub-states. Is that right?

> sounds good as the basis.

**4.2** Are BE and FE plans both required for every ticket, or optional per type (e.g., backend-only tickets skip FE)?

> optional, sometimes only backend or only frontend

**4.3** Who can transition states — only humans, or can agents auto-advance on success?

> the transition can be done with the human or agent with the right role.

**4.4** Is state stored in the file (frontmatter `status:`) or derived from directory location / git branch?

> good question, history is important, so the original idea/requirement should be fairly intact, then a new ticket is created, which links to the original and the status would be recorded.
> all tickets involved in the original idea would have links back to it, so you can see how big the idea was or wasn't

---

## 5. Agents

**5.1** Which LLM(s)? Claude only, or pluggable? Who supplies API keys — the app operator or per-user?

> Pluggable
> Different roles might be different agents using different LLMs, some cloud some local, etc.
> These would be setup at the beginning, and will be updated over the lifecycle of a project.

**5.2** Where do agents run — in-process in the Go app, spawned as subprocesses (Claude Code CLI?), or remote API calls?

> Both, the Go app will spawn agents using CLI for first version and in the roadmap we will support other methods of using agents, like remore API calls.

**5.3** You said "AI Agent can read the files **or** access remotely using MCP." Does that mean **this app exposes an MCP server** so external agents (Claude Code, Cursor) can drive it? Or that **this app consumes MCP** to reach external tools? Both?

> Both.  The priority would be to support MCP for agents to get information.  This feature should be on the roadmap for the idea to be fleshed out.

**5.4** Agent output: does the agent **write files directly** to the project dir, or stage a proposed diff that the human accepts/rejects?

> the agent will write files directly, a human or agent will review the files or review the GIT requests to accept.

**5.5** Long-running agents: progress visible in UI? Cancelable? What if the agent crashes mid-way — is the partial artifact kept?

> Definitely need visibility of long running agents.  The idea of this whole system to make breaking up the project to small enough chunks that there should be limited long running agents.
> Lets start simple, long running agents visible and when they started and current status and an option to kill an agent if needed.
> Roadmap for future enhancements as requirements evolve.

**5.6** Concurrency: can multiple agents run at once on the same project? Same ticket?

> Multiple agents can run at the same time.
> One ticket one agent.  HARD RULE!  

---

## 6. Git integration

**6.1** Is git **mandatory** or optional? If a project dir isn't a git repo, does the app refuse, or work without?

> Git is mandatory, branches and pull requests can be used as part of the workflow.

**6.2** Who commits — the human clicking "save", the app on state transitions, or the agent after producing output? Commit message conventions?

> When a new output or artifact is ready, it is committed to GIT by whoever creates it.

**6.3** Branching: everyone on `main`, or one branch per ticket/agent-run? PRs?

> a branch per ticket which is merged when completed, that will keep all the history together.

**6.4** Remote required, or local-only OK?

> Definitely local, remote is optional.

---

## 7. Developer and QA agents (biggest scope item)

**7.1** "Developer agent will be responsible for code and unit tests" — does that code live in **the same project directory** as the requirements, or a **separate code repo** this app is aware of? If separate, how is the mapping declared?

> there will be a project directory which is a regular structure required for the code, with the source, libraries, etc as per the language conventions.  
> A directory called lifecycle will be used and under that directories for ideas, requirements, specifications, test plans, etc

**7.2** If same dir: how do you keep requirements (.md/.yaml) from mixing messily with application source code?

> the requirements and all the related artifacts will be in one or more sub-directories

**7.3** QA agent: which headless browser framework — Playwright, Puppeteer, Cypress? Does the app ship it, or does the user bring it?

> User decides

**7.4** "Extending existing QA testing" — does the app detect an existing test harness and append to it, or generate a fresh one?

> User tells the app which harness to use, if it is existing or not.

---

## 8. 3D graph specifics

**8.1** What's a **node** — only tickets, or also labels, sprints, releases, agents?

> All the concepts would be nodes of different colours

**8.2** What's an **edge** — `depends_on`, `parent_of`, `related_to`, `blocks`? Different colors per edge type?

> Yes those are all edges and should be different colours, and should be directed

**8.3** Expected scale — 50 tickets? 5,000? (3d-force-graph chokes past a few thousand without clustering.)

> I expect this will be 100's of tickets

**8.4** Click a node → opens the file where? In-app editor, or shell out to `$EDITOR` / VS Code?

> when a node is clicked it is previewed in a modal, in the modal are action buttons to then do something else with the node.   the actions should be at the top.

---

## 9. JIRA integration

**9.1** Bidirectional sync or one-way mirror? If bidirectional, **who wins** on conflict — filesystem or JIRA?

> Roadmap the JIRA feature for future planning.

**9.2** Cloud, Server/DC, or both?

> Roadmap the JIRA feature for future planning.

**9.3** Is JIRA a **swap-in replacement** for the file storage (i.e., "use JIRA instead of a directory"), or an **addition** (files + mirror to JIRA)? Your doc hints at both — pick one.

> Roadmap the JIRA feature for future planning.

---

## 10. Editing experience

**10.1** In-app markdown editor (WYSIWYG? textarea + preview?), or does the app just expect you to edit files in your own IDE and re-read from disk?

> A simple in app editor will be good.  I expect users will also be using an IDE in the same directory.

**10.2** If in-app: live preview? Live updates across users?

> Monitor files for changes and if a file is open for viewing, prompt user to refresh.

---

## 11. Tech stack

**11.1** Go web framework preference (stdlib `net/http`, chi, gin, echo)?

> _answer:_

**11.2** Frontend: plain JS + 3d-force-graph, or a framework (React/Vue/Svelte)?

> Plain JS + 3d-force-graph + cytoscape for 2D + Vue.js for advanced UI interactions.

**11.3** Any database (SQLite index for fast graph queries over the file tree), or pure filesystem reads every time?

> For first version SQLite index with files on disk

**11.4** Realtime updates: polling, SSE, or websockets?

> Websockets

---

## 12. Security / deployment

**12.1** "Reverse proxy if needed" — does the app handle TLS itself, auth itself, or rely entirely on the proxy?

> The APP should be able to offer Auth and TLS natively, it will always handle auth, but user may run with HTTP and use reverse proxy with TLS to allow outside access.

**12.2** Is there any sandbox on which directories can be opened, or can the user point it at `/`?

> The application would be installed in a directory, e.g. /home/keith/kaos-control
> Each project directory would have a configuration file, e.g. my-new-project.yaml for /home/keith/Projects/my-new-project
> When the application runs it can access each project directory based on Unix file/directory permissions, for multi-user it would rely on group permissions
> The user logs in and picks the project they want to work on and it shows them that project directory.  The project name should form part of the URL.
> The application should only open files in the directory tree of the project it is working on.
> Docker could be used to provide sandbox if that is required.

---

## Priority

If you only answer three before I start drafting: **1.1** (single vs multi-user), **2.1–2.3** (ticket schema), and **5.2–5.4** (agent execution model).

---

# Round 2 — Follow-ups from your answers

## Critical (shape of the spec depends on these)

**A. Lifecycle model (follow-up to 4.4)** — Your answer suggested each stage creates a **new linked file** rather than editing the original. So one idea becomes: `ideas/login.md` → `requirements/login.md` → `specifications/login-be.md` + `specifications/login-fe.md` → `dev/login.md` → `test-plans/login.md`, each linking back. Is that the model? Or does the original ticket stay put and get updated in place, with a frontmatter `status:` field tracking where it is?

> Multiple files across directories, linked together.
> Wondering if unique filenames would be less confusing, should append an index to each new file, so it would be: `ideas/login.md` → `requirements/login-2.md` → `specifications/login-3-be.md` + `specifications/login-4-fe.md` → `dev/login-5.md` → `test-plans/login-6.md`

**B. Canonical role list (follow-up to 1.3 & 4.3)** — Proposal: `product-owner`, `backend-planner`, `frontend-planner`, `developer`, `qa`, `reviewer`, `approver`. Each role maps to specific transitions (e.g. only `reviewer` can reject, only `approver` can approve). Is that the right list? Can one human/agent hold multiple roles?

> List looks good.
> One human or agent can hold multiple roles.

**C. Project directory layout (follow-up to 7.1)** —
- Does the app ever read/write the **code tree**, or is it strictly scoped to `lifecycle/`? (The developer agent presumably needs to write code *somewhere*.)
- Canonical list of `lifecycle/` subdirectories. Proposal: `ideas/`, `requirements/`, `specifications/`, `plans/backend/`, `plans/frontend/`, `dev/`, `test-plans/`, `tests/`, `prototypes/`. Amend as needed.

> Do we need that many?  I think we can go from requirements to plans, and dev folder would be plans.  Lets go with `ideas/`, `requirements/`, `backend-plans/`, `frontend-plans/`, `dev-plans/`, `test-plans/`, `tests/`, `prototypes/`
> Ideally this should be flexible and the sub-directories in lifecycle reflect the process a team uses, they may want different names or structures, lets start with the above and consider making it configurable.

**D. Go framework (11.1 was left blank)** — My recommendation: stdlib `net/http` + `chi` router — tiny, idiomatic, no magic, good fit. OK, or prefer gin/echo/fiber?

> Will go with your recommendation.

**E. Frontend build pipeline (follow-up to 11.2)** — You listed "plain JS + 3d-force-graph + cytoscape + Vue.js". Vue usually implies a bundler. Which did you mean:
- (i) Vue via CDN, no build step, or
- (ii) Vite build pipeline producing static assets the Go server serves?

> Build pipeline is the go.

---

## Important (I'd pick a default but worth your input)

**F. Agent git identity (follow-up to 6.2)** — Proposal: each agent config has a git `name` + `email` (e.g. `dev-agent <dev@innovation-maker.local>`) and the app ensures commits use it. OK?

> Agreed

**G. Branch workflow (follow-up to 6.3)** — Proposal: branch `ticket/<slug>` created on ticket creation; every artifact commit for that ticket lands on that branch; merge to `main` on state=done via PR if remote exists, else fast-forward locally. OK?

> Sounds good.  Is it possible to make this flexible for different naming schemes?

**H. GUI-vs-IDE edit conflict (follow-up to 10.2)** — If GUI has the ticket open for editing (locked) and the file changes on disk (from an IDE), what wins? Proposal: disk change wins, GUI shows "file changed externally — reload and lose your edits?" OK?

> Agreed

**I. Project discovery / config (follow-up to 12.2)** — You said `my-new-project.yaml` points to `/home/keith/Projects/my-new-project`. Where does the config file live — in the app install dir (e.g. `/home/keith/kaos-control/projects/*.yaml`), or somewhere else? How does a user register a new project — via UI, or by dropping a file?

> It would be in `/home/keith/kaos-control/projects/*.yaml`, a UI to CRUD these would be great.

**J. Concurrency vs GUI lock (follow-up to 5.6 & 1.2)** — "One ticket, one agent — HARD RULE" is clear. Does the GUI edit-lock **also** block agents, and vice versa? Assumption: yes, same unified lock.

> Agreed

---

## Minor — default assumptions (say "yes defaults" or call out ones you want changed)

- **Slug collisions & renames:** renaming a ticket file rewrites all inbound links automatically; slug collision on create = error.
- **Lock timeout:** GUI edit lock expires after 5 min of inactivity or on disconnect.
- **Epic promotion:** a ticket can be converted to an epic later; file-type change preserves history.
- **Labels/types:** free-form, user-defined, stored in YAML frontmatter; app auto-collects all values for filter UI.
- **UI prototype format from FE agent:** static HTML/CSS under `lifecycle/prototypes/<slug>/`, opened in a new tab.
- **Node modal actions:** Edit, Change state, Run agent, Open in IDE, View git history, Delete.
- **WebSocket events:** `file.changed`, `agent.started`, `agent.progress`, `agent.finished`, `agent.failed`, `lock.acquired`, `lock.released`.

> yes defaults.

# Round 3 questions.

1. The indexed filename scheme (A) — the index appears to be monotonic across the entire lineage (1 for login.md, then 2, 3, 4, 5, 6 across stages). I'll spec it that way: first file has no suffix, every subsequent artifact in the same lineage increments a shared counter, regardless of stage. If a plan is rejected and re-planned, the new plan gets the next index and supersedes the old one (which stays in git history). Shout if you wanted per-stage numbering instead.

> Sounds good.

2. Configurability (C and G) — you flagged two things you want flexible: lifecycle subdirectory names, and git branch naming scheme. I'll put both under a project-level config.yaml with defaults matching what you specified, overridable per project. That keeps v1 shippable with sane defaults and opens the door for teams to customise later.

> Perfect!

# drafted requirements

Drafted detailed-requirements.md — 17 sections covering goals, roles, lineage model, file format, directory layout, state machine, agents, git, visualisation, UI, events, stack, config, security, defaults, roadmap, and parked open questions.

A few things I made concrete decisions on (flagging so you can challenge if wrong):

* Release/Sprint modelled as their own artifacts in lifecycle/releases/ and lifecycle/sprints/, not as config. Lets them appear as nodes in the graph with member_of edges.
* Rejected plans get the next index and supersede — old artifact stays in place and git history; no in-place editing.
* Clarifying questions are the one exception to the "new file per stage" rule — questions.md updates in place because the interaction is conversational.
* Transition authorisation matrix with sensible defaults, overridable in config.yaml.
argon2id for password hashing, pure-Go SQLite (modernc.org/sqlite), Vite + Vue 3 SFCs.
* Parked open questions section at the end for things like styling system, prompt template location, and SQLite migration strategy — decide during implementation, not now.
