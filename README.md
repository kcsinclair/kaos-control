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

## Quick start

```sh
# Build everything (frontend + backend)
make all

# Run in dev mode (default config at ~/.kaos-control/config.yaml)
make run
```

App config: `~/.kaos-control/config.yaml`. Projects are registered as YAML files in `~/.kaos-control/projects/*.yaml`. Per-project config (roles, agents, plan gates) lives at `<project>/lifecycle/config.yaml`. DevOps pipelines live at `<project>/lifecycle/devops/*.yaml`; their run logs are written to `~/.kaos-control/devops/<project>/`.

Open <http://localhost:8080> to see the project picker.

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
