# kaos-control

A single-binary lifecycle management tool that turns ideas into shipped releases. Markdown artifacts on disk are the source of truth; a Go server indexes them into SQLite and serves a Vue 3 SPA with a 3D graph, an editor, and an agent runner that drives Claude Code subprocesses to produce the next artifact in the lineage.

> **Status:** active development. Comprehensive documentation will follow — this README is a quick start only.

## What you get

- **Lifecycle directory** (`lifecycle/`) — markdown files with YAML frontmatter, organised by stage (`ideas`, `requirements`, `backend-plans`, `frontend-plans`, `test-plans`, `tests`, `defects`, `releases`, `sprints`).
- **Lineage tracking** — every artifact in a chain shares a slug and carries a monotonic index across stages.
- **Workflow state machine** — role-gated transitions (e.g. only `approver` can move a ticket from `planning` to `in-development`); plan-completion gates.
- **Agents** — pluggable LLM runners (currently `claude-code-cli`) bound to roles, with sandboxed write paths.
- **DevOps pipelines** — declarative YAML pipelines in `lifecycle/devops/` (build, deploy, release, …) that the product owner can trigger from the UI; per-step output streams to the browser over WebSocket and is persisted to `~/.kaos-control/devops/<project>/`.
- **Web UI** — 3D / 2D graph, artifact editor with markdown preview, agent run dialog with live progress, DevOps page with live pipeline runs, parse-error view, project config editor.
- **Distribution** — one Go binary with the frontend embedded.

## Tech stack

- **Backend**: Go 1.25, `chi`, `goldmark`, `modernc.org/sqlite` (pure-Go), `go-git`, `coder/websocket`, `fsnotify`.
- **Frontend**: Vue 3, Vite 5, TypeScript, Pinia, `markdown-it`, `3d-force-graph` + three.js, Cytoscape.js + fcose, CodeMirror 6.

## Getting started

### Prerequisites

| | Why |
|---|---|
| **Go 1.25+** | Builds the server binary. |
| **Node.js 20+ & pnpm** | Builds the embedded SPA. |
| **Git 2.30+** | The server commits artifact changes to your project's git repo. |
| **Claude Code CLI** *(optional)* | Required only if you want to run agents. `npm install -g @anthropic-ai/claude-code` and `claude auth login`. |

### 1. Build the binary

```sh
git clone https://github.com/kcsinclair/kaos-control.git
cd kaos-control
make all          # builds web/dist + ./dist/kaos-control
```

The Go binary embeds the SPA via `embed.FS`, so `./dist/kaos-control` is a single self-contained executable.

### 2. First run

```sh
./dist/kaos-control
```

On first launch, kaos-control writes a default `~/.kaos-control/config.yaml` and starts listening on `:8042`. Open <http://localhost:8042>.

The first user can be created without authentication (bootstrap). Use the in-app sign-up form, or:

```sh
curl -X POST http://localhost:8042/api/admin/users \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","display_name":"You","password":"choose-a-strong-password"}'
```

After the first account exists, this endpoint requires authentication.

### 3. Bootstrap a project

The fastest path is the CLI scaffolder:

```sh
cd /path/to/your/project        # any directory; an existing git repo or a fresh one
kaos-control init               # creates lifecycle/, lifecycle/config.yaml, CLAUDE.md
```

`kaos-control init` creates the standard `lifecycle/` directory tree (`ideas/`, `requirements/`, `backend-plans/`, `frontend-plans/`, `test-plans/`, `tests/`, `defects/`, `releases/`, `sprints/`, `prototypes/`), a skeleton `lifecycle/config.yaml` with the standard role and agent definitions, and a `CLAUDE.md` to guide agent runs in this project.

Then register the project with kaos-control:

```sh
mkdir -p ~/.kaos-control/projects
cat > ~/.kaos-control/projects/myproject.yaml <<EOF
name: myproject
path: /path/to/your/project
description: <one-line description>
owner: you@example.com
EOF
```

Restart the server (or wait for it to pick up the new entry on the next scan) and the project will appear in the picker.

### 4. Use it

- Open <http://localhost:8042>, sign in, choose your project.
- Create an idea, work it through `clarifying → planning → in-development → in-qa → approved → done`.
- Configure agents per-role in `<project>/lifecycle/config.yaml` to have them produce the next artifact in the lineage.
- Wire DevOps pipelines into `<project>/lifecycle/devops/*.yaml` to trigger build/test/release from the UI.

## Where things live

| Path | Purpose |
|---|---|
| `~/.kaos-control/config.yaml` | App-level config (server, auth, agents, ollama) |
| `~/.kaos-control/projects/*.yaml` | One file per registered project |
| `~/.kaos-control/data/<project>/index.db` | Per-project SQLite cache (rebuilt from disk on startup) |
| `~/.kaos-control/devops/<project>/<run_id>.log` | DevOps pipeline run logs |
| `<project>/lifecycle/config.yaml` | Per-project: roles, agents, plan gates, dashboard tracked types |
| `<project>/lifecycle/devops/*.yaml` | DevOps pipeline definitions for this project |
| `<project>/lifecycle/{ideas,requirements,…}/` | Artifacts (markdown + YAML frontmatter) |

## Repository layout

```
cmd/kaos-control/   Go binary entry point
internal/           Backend packages (agent, artifact, http, index, …)
web/                Vue 3 SPA (built into web/dist/, embedded by Go)
lifecycle/          This project's own artifacts (the meta-bootstrap)
tests/              Integration test code
plans/              Project plan + per-change implementation plans
```

## Documentation

- [CLAUDE.md](CLAUDE.md) — guidance for Claude Code agents working in this repo (commit conventions, lineage rules, build commands).
- [lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md](lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md) — authoritative product spec.
- [plans/PROJECT_PLAN.md](plans/PROJECT_PLAN.md) — living state-of-the-project document.

## Licence

**[GNU AGPLv3](LICENSE)** — copyleft with a network-use clause. If you run a
modified version of kaos-control as a network service, you must publish your
modifications under the same licence.

Commercial licences for organisations that cannot accept AGPL terms may be
available on request — open an issue to start the conversation.

## Contributing

Contributions are welcome. The project uses the
[Developer Certificate of Origin](https://developercertificate.org/) — sign
off on every commit with `git commit -s`. See [CONTRIBUTING.md](CONTRIBUTING.md)
for the full workflow.
