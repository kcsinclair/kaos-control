# kaos-control

> **Self-hosted orchestrator for Claude Code agents.** Drive ideas through a role-gated SDLC — plan → build → test → release — with per-agent sandboxing, live run streaming, and markdown-on-disk as the source of truth. Single Go binary.

[![Watch the demo](https://img.youtube.com/vi/xSg7I4zPC84/maxresdefault.jpg)](https://youtu.be/xSg7I4zPC84)

📺 **[Watch the 5-minute walkthrough](https://youtu.be/xSg7I4zPC84)** — set up kaos-control, add a project, and build a website end-to-end.

🌐 **[kaos-control.io](https://kaos-control.io)** · 📦 **[Releases](https://github.com/kcsinclair/kaos-control/releases)** · 📄 [AGPLv3](LICENSE)

> **Status:** v0.1.x — actively developed, breaking changes possible before 1.0. Working releases ship from this repo's own lifecycle (dogfooded).

---

## Why this exists

Claude Code is brilliant until you want to run it unattended on real work. The choices today are bad: grant blanket permissions with `--dangerously-skip-permissions` and hope, or babysit every tool call. Either way, three things stay broken:

- **No structure around the agent.** Where did this ticket come from? What plan is it executing? What's the acceptance test? The agent writes code; the *why* lives in your head or a Notion doc that drifts.
- **No sandbox you can audit.** Bypass mode is all-or-nothing. There's no per-agent allowlist, no record of which tool calls were made on which artefact.
- **No cost or privacy lever.** Every prompt is your architecture leaving your machine, and the token bill arrives at the end of the month with no breakdown of *what* spent it.

kaos-control gives you a middle path:

1. **Structured artefacts as the unit of work** — every idea becomes a markdown file with YAML frontmatter, walks a configurable lifecycle (`ideas → requirements → plans → tests → releases`), and carries a lineage slug so every line of shipped code traces back to the ask.
2. **A mediated Claude driver** that routes every tool call through a `PreToolUse` hook, checked against per-agent path and bash allow/deny lists. Hard sandbox, full audit trail, no bypass mode required.
3. **Per-role model assignment** — Opus where thinking quality matters, Haiku for QA, local Ollama for code that should never leave your machine. Token spend logged per run.

Single Go binary with the SPA embedded. Self-hosted. Your artefacts live in your git repo. Pull the network cable and the workflow keeps going.

---

## What you get

- **Lifecycle directory** (`lifecycle/`) — markdown files with YAML frontmatter, organised by stage (`ideas`, `requirements`, `backend-plans`, `frontend-plans`, `test-plans`, `tests`, `prototypes`, `defects`, `releases`, `sprints`).
- **Lineage tracking** — every artefact in a chain shares a slug and carries a monotonic index across stages. Every deployed line traces back to the original idea.
- **Workflow state machine** — role-gated transitions (e.g. only `approver` can move a ticket from `planning` to `in-development`); plan-completion gates.
- **Two Claude drivers** — pick `claude-code-cli` for speed on a trusted machine, or `claude-mediated` for a hard sandbox with audited tool calls (mandatory in Claude Enterprise environments).
- **Pluggable agents** — `claude-code-cli`, `claude-mediated`, `ollama`, plus a `shell-stub` for test scaffolding. Bound to roles, sandboxed write paths, per-agent permission policy on the mediated driver.
- **DevOps pipelines** — declarative YAML in `lifecycle/devops/` (build, deploy, release). Triggered from the UI; per-step output streams to the browser over WebSocket and persists to `~/.kaos-control/devops/<project>/`.
- **Web UI** — 3D and 2D graph views, Kanban, Gantt, roadmap, artefact editor with markdown preview, agent run dialog with live progress, DevOps page with live pipeline runs, parse-error view, project config editor.
- **Distribution** — one Go binary (~250 MB) with the frontend embedded. macOS (Intel + ARM), Linux (x86-64 + arm64), Windows (x86-64).

---

## Who it's for

- **Lead developers** introducing agents into a real team workflow without giving up code review, tests, or release discipline.
- **CTOs and tech leaders** who want to approve documents rather than tinker with an IDE — the discipline is encoded in `CLAUDE.md` and inherited per project.
- **Founders running a one-person dev shop** who need structure to track what's built and plan what's next.
- **Product owners** who want vague ideas captured faithfully, traceable through to release.

If your reaction to "I unleashed Claude Code on the repo overnight" is *"…and what exactly did it do?"* — this is for you.

---

## Tech stack

- **Backend**: Go 1.25, `chi`, `goldmark`, `modernc.org/sqlite` (pure-Go), `go-git`, `coder/websocket`, `fsnotify`. Local-model agents talk to Ollama over plain `net/http` — no extra library dependency.
- **Frontend**: Vue 3, Vite 6, TypeScript, Pinia, `markdown-it`, `3d-force-graph` + three.js, Cytoscape.js + fcose, CodeMirror 6.

---

## Claude Code permissions

kaos-control ships two Claude drivers, picked per-agent in `lifecycle/config.yaml` via the `driver:` field:

- **`claude-code-cli`** — runs `claude --dangerously-skip-permissions -p ...` as a headless subprocess. Fast and simple, ideal for personal use on a trusted machine. **Requires Claude to be in bypass-permissions mode** on every machine that runs kaos-control (one-time setup, below).
- **`claude-mediated`** — runs `claude` in default permission mode and routes every tool call through kaos-control's `PreToolUse` hook for allow/deny against per-agent path and bash allow/deny lists. Mandatory in environments where bypass mode is blocked (e.g. Claude Enterprise) or when you want a hard sandbox and an audit trail. No first-machine setup required — kaos-control configures the hooks per run.

### One-time setup for `claude-code-cli` only

> Skip this section if every agent in your `lifecycle/config.yaml` uses `claude-mediated`.

kaos-control runs `claude` as a headless subprocess — there is no human at the terminal to approve individual tool calls. Without bypass-permissions mode, every agent run will stall with a message like *"I need write permission to create the file"* and produce no work.

**Before your first agent run, on every machine that runs kaos-control:**

1. **Run `claude` interactively at least once and accept the bypass-permissions warning.** This is a one-time step that Anthropic requires per machine and per user — until you do it, the `--dangerously-skip-permissions` flag kaos-control passes is silently ignored.

   ```
   claude
   ```

   At the first prompt, type any short instruction (e.g. `hello`). Claude will show a one-time warning about bypass mode and ask you to accept. Accept it. You can quit straight after.

2. **Check no settings file is overriding it.** If you have a `~/.claude/settings.json` (or a project-local `.claude/settings.json`) with a `permissions.defaultMode` other than `bypassPermissions`, that overrides the CLI flag. Either remove the key or set it to `bypassPermissions`.

3. **Smoke test.** From any directory, run:

   ```
   claude --dangerously-skip-permissions -p "list the files here" --output-format stream-json | head -20
   ```

   If you see a `Bash` tool call complete successfully (not appear in a `permission_denials` block), you're set.

### Detection at run-start

kaos-control detects an unconfigured bypass mode within the `init_event_timeout_seconds` window (default 10s) and fails the run with a clear `precheck_failure` reason plus a remediation list — no more silent stalls. Tracked under the `agent-permission-precheck` lineage in this project's lifecycle (shipped in v0.1.2).

---

## Install from a released binary

Pre-built single-binary archives are published on GitHub Releases for macOS (Intel + Apple Silicon), Linux (x86-64 + arm64), and Windows (x86-64). Each archive ships the `kaos-control` binary alongside this README, LICENSE, and CONTRIBUTING.

This is the recommended path for most users. If you want to build from source instead, skip ahead to [Install from source](#getting-started-building-from-source).

### 1. Download the archive for your platform

Pick the build for your OS and CPU architecture from the [Releases page](https://github.com/kcsinclair/kaos-control/releases), or from a terminal — replace `0.1.2` with the version you want:

```bash
VERSION=0.1.2
BASE=https://github.com/kcsinclair/kaos-control/releases/download/v${VERSION}

# Pick ONE of the following:
curl -L -o kaos-control.zip "$BASE/kaos-control-${VERSION}-darwin-arm64.zip"    # macOS, Apple Silicon
curl -L -o kaos-control.zip "$BASE/kaos-control-${VERSION}-darwin-amd64.zip"    # macOS, Intel
curl -L -o kaos-control.zip "$BASE/kaos-control-${VERSION}-linux-amd64.zip"     # Linux, x86-64
curl -L -o kaos-control.zip "$BASE/kaos-control-${VERSION}-linux-arm64.zip"     # Linux, arm64
curl -L -o kaos-control.zip "$BASE/kaos-control-${VERSION}-windows-amd64.zip"   # Windows, x86-64
```

### 2. Verify the checksum (recommended)

Each release publishes a `SHA256SUMS` file alongside the archives.

```bash
curl -L -o SHA256SUMS "$BASE/SHA256SUMS"

# macOS
shasum -a 256 -c SHA256SUMS --ignore-missing

# Linux
sha256sum -c SHA256SUMS --ignore-missing
```

Each line that matches a file you downloaded should print `OK`. If any line says `FAILED`, do not run the binary — re-download the archive.

### 3. Unzip and place the binary

The archive extracts to a versioned `kaos-control-<VERSION>/` directory containing the binary plus the docs — e.g. `kaos-control-0.1.2/`:

```bash
unzip kaos-control.zip
cd kaos-control-${VERSION}
```

Two releases unzipped side-by-side won't collide because each gets its own versioned directory.

To quickly get started, just run it from the unzipped directory:

```bash
./kaos-control -d
```

To make `kaos-control` runnable from anywhere, move it onto your PATH:

```bash
# macOS / Linux
sudo mv ./kaos-control /usr/local/bin/
```

On Windows, copy `kaos-control.exe` into a directory that's on your `%PATH%`, or invoke it by its full path.

### 4. macOS first-run only — clear the quarantine attribute

The macOS builds are not yet code-signed, so the first time you run the binary macOS will refuse with a *"cannot be opened because Apple cannot check it for malicious software"* message. Strip the quarantine attribute once and you won't see the dialog again:

```bash
xattr -d com.apple.quarantine /usr/local/bin/kaos-control
```

Linux and Windows have no equivalent step.

### 5. Run it

From here on the bootstrap path is the same whether you built from source or downloaded a release:

- [First run](#2-first-run) — run `kaos-control -d`; it writes `~/.kaos-control/config.yaml`, starts on `:8042`, and waits for you to create the first user.
- [Bootstrap a project](#3-bootstrap-a-project) — register a directory on disk as a project so it appears in the picker.
- [Use it](#4-use-it) — what the SPA looks like once you're in.

If you plan to run agents, the [Claude Code permissions](#claude-code-permissions) section above is required reading — especially if any of your agents use the `claude-code-cli` driver. The mediated driver is plug-and-play; the CLI driver needs a one-time bypass-mode setup per machine.

---

## Run as a service (Linux systemd / macOS launchd)

`kaos-control -d` runs in the foreground and does not fork or write a PID file — backgrounding is the caller's job. On Linux, [`packaging/systemd/kaos-control.service`](packaging/systemd/kaos-control.service) does that as a **systemd user service**.

A *user* unit (not a system unit) is deliberate: the server runs as your login user, so agent runs inherit your `HOME`, your `claude` credentials in `~/.claude`, your git identity, and your `~/.kaos-control` config. **The whole path is rootless** — binary, unit, and boot persistence all live under your home directory, so you can run kaos-control as a service on a machine where you have no sudo.

### 1. Install the binary and the unit

The unit expects the binary at `~/.local/bin/kaos-control`:

```bash
install -Dm755 ./dist/kaos-control ~/.local/bin/kaos-control    # or ./kaos-control from a release archive
mkdir -p ~/.config/systemd/user
cp packaging/systemd/kaos-control.service ~/.config/systemd/user/
systemctl --user daemon-reload
```

Make sure `~/.local/bin` is on your `PATH` for interactive use — most distributions add it from `~/.profile` when the directory exists; log out and back in if you just created it. The unit itself uses an absolute path, so it works either way.

If you'd rather install once for everyone on a shared machine, `sudo install -Dm755 ./dist/kaos-control /usr/local/bin/kaos-control` and change `ExecStart=` to match (`systemctl --user edit --full kaos-control` after installing also works). Each user still gets their own unit, config, and credentials.

The unit invokes the `serve` subcommand rather than the `-d` flag. They are equivalent on current builds, but `serve` also works on binaries older than v0.2.0, which reject `-d` with `unknown flag "-d"` and exit 1.

### 2. Keep it running when you're not logged in

By default a user manager starts at login and stops at logout. To have kaos-control start at boot and survive logout, enable lingering once:

```bash
loginctl enable-linger
```

No sudo: systemd's default policy (`org.freedesktop.login1.set-self-linger`) lets you enable lingering **for yourself**. Only enabling it for *another* user needs root (`sudo loginctl enable-linger someone-else`), and a hardened polkit configuration can withdraw the self-permission — if the command is refused, ask an admin.

Skip this if you only want the service while you're logged in.

### 3. Start it

```bash
systemctl --user enable --now kaos-control     # start now + start at login/boot
systemctl --user status kaos-control
curl -s http://localhost:8042/api/health       # {"ok":true,"version":"..."}
```

Then open <http://localhost:8042>. If this is a fresh install, create the first user as described in [First run](#2-first-run).

### Everyday commands

| Task              | Command                                          |
| ----------------- | ------------------------------------------------ |
| Start             | `systemctl --user start kaos-control`            |
| Stop              | `systemctl --user stop kaos-control`             |
| Restart           | `systemctl --user restart kaos-control`          |
| Status            | `systemctl --user status kaos-control`           |
| Enable at boot    | `systemctl --user enable kaos-control`           |
| Disable at boot   | `systemctl --user disable kaos-control`          |
| Follow logs       | `journalctl --user -u kaos-control -f`           |
| Logs since boot   | `journalctl --user -u kaos-control -b`           |
| Clear start-limit | `systemctl --user reset-failed kaos-control`     |

The server logs structured JSON to stdout, which the unit routes to the journal under the `kaos-control` identifier. Per-run agent logs still land in `~/.kaos-control/data/<project>/runs/`.

### Configuration overrides

The unit pins `-config ~/.kaos-control/config.yaml` explicitly, because the systemd user manager does not necessarily inherit `XDG_CONFIG_HOME` from your login shell. If you keep your config elsewhere, change the `-config` path in the unit.

Environment overrides go in an optional file — no need to edit the unit:

```bash
mkdir -p ~/.config/kaos-control
cat > ~/.config/kaos-control/service.env <<'EOF'
LOG_LEVEL=debug
KAOS_PUBLIC_HOST=kaos.example.com
# GEMINI_API_KEY=...
EOF
systemctl --user restart kaos-control
```

**PATH matters for agent runs.** Agents shell out to `claude`, `git`, `go`, `node` and your DevOps pipeline steps, and the user manager's default `PATH` is minimal. The unit sets a sensible default (`~/.local/bin`, `~/bin`, `~/go/bin`, `/usr/local/go/bin`, then the system paths). If your toolchains live elsewhere — nvm-managed Node is the common case — override it wholesale:

```bash
echo "PATH=$HOME/.nvm/versions/node/v22.22.1/bin:$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin" \
  >> ~/.config/kaos-control/service.env
systemctl --user restart kaos-control
```

Write absolute paths in `service.env` — systemd reads it literally, so `$HOME` and `~` inside that file are **not** expanded (the command above expands `$HOME` in your shell before writing).

A quick way to confirm what the service can actually see: `systemctl --user show-environment` and `journalctl --user -u kaos-control` after a failed agent run.

### Stop and shutdown behaviour

`systemctl --user stop` sends `SIGTERM`; the server stops accepting connections and drains in-flight requests with a 10-second deadline. `TimeoutStopSec=30` gives it room, after which systemd kills the whole control group — including any agent subprocesses (`claude`, `git`, pipeline steps) still running. Stopping mid-run therefore aborts that run; check the Runs view when you start back up.

`Restart=on-failure` restarts a crashed server after 5s. Five failed starts in 60s (typically a bad config or a port already in use) stops the unit for good until you fix it and run `systemctl --user reset-failed kaos-control`.

### Upgrading

```bash
systemctl --user stop kaos-control
install -Dm755 ./dist/kaos-control ~/.local/bin/kaos-control   # or wherever ExecStart points
systemctl --user start kaos-control
curl -s http://localhost:8042/api/health                       # confirm the new version
```

`install` replaces the file by rename, so it also works while a copy is running — but stop first anyway, so the restart is a clean shutdown rather than a swap under a live process.

### Uninstall

```bash
systemctl --user disable --now kaos-control
rm ~/.config/systemd/user/kaos-control.service
systemctl --user daemon-reload
sudo loginctl disable-linger "$USER"       # only if you enabled it
```

### System-wide instead of per-user

Running kaos-control as a system service (`/etc/systemd/system/`) works, but the unit then needs an explicit `User=`/`Group=`, a `HOME=` that matches that account, and that account must have its own `claude` login, git identity, and `~/.kaos-control`. Agents run as that user with that user's credentials. The per-user unit above avoids all of it; prefer it unless you specifically need a shared, headless, multi-admin install.

### macOS (launchd)

On macOS, [`packaging/launchd/io.kaos-control.server.plist`](packaging/launchd/io.kaos-control.server.plist) runs the server as a per-user **LaunchAgent**. Unlike the systemd unit it **rebuilds first**: it launches [`kaos-control-serve.sh`](packaging/launchd/kaos-control-serve.sh) through a login shell, which runs `make build-web && make build` and then `exec`s the fresh binary — so every (re)start serves current code. All output (build logs, server logs, panic traces) is appended to `~/.kaos-control/logs/kaos-control.log`.

```bash
mkdir -p ~/.kaos-control/logs
cp packaging/launchd/io.kaos-control.server.plist ~/Library/LaunchAgents/
# Edit the /Users/keith paths in the copy if your home/repo differ
# (launchd does not expand $HOME in a plist).
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/io.kaos-control.server.plist
launchctl enable   gui/$(id -u)/io.kaos-control.server
```

Rebuild + restart on demand (e.g. after a `git pull`): `launchctl kickstart -k gui/$(id -u)/io.kaos-control.server`. Stop/uninstall: `launchctl bootout gui/$(id -u)/io.kaos-control.server`. Tail the log: `tail -f ~/.kaos-control/logs/kaos-control.log`. Set `KC_BUILD_WEB=0` in the plist's `EnvironmentVariables` to skip the slow SPA build on each start.

---

## Getting started building from source

### Prerequisites

|                                  | Version | Why                                                                                                           |
| -------------------------------- | ------- | ------------------------------------------------------------------------------------------------------------- |
| **Go**                           | 1.25+   | Builds the server binary.                                                                                     |
| **Node.js**                      | 20 LTS+ | Builds the embedded SPA. See [install steps](#nodejs-20-lts-or-newer) below.                                  |
| **pnpm**                         | 9+      | The frontend's package manager. See [install steps](#pnpm-9) below.                                           |
| **Git**                          | 2.30+   | The server commits artifact changes to your project's git repo.                                               |
| **Claude Code CLI** *(optional)* | —       | Required only if you want to run agents. `npm install -g @anthropic-ai/claude-code` then `claude auth login`. |

#### Node.js (20 LTS or newer)

Pick one:

- **macOS (Homebrew)** — `brew install node`
- **macOS / Linux (nvm)** — install [nvm](https://github.com/nvm-sh/nvm), then:

  ```bash
  nvm install --lts
  nvm use --lts
  ```

- **Linux (apt)** — Debian/Ubuntu ships an old version; use NodeSource:

  ```bash
  curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
  sudo apt-get install -y nodejs
  ```

- **Windows** — download the LTS installer from [nodejs.org](https://nodejs.org/), or `winget install OpenJS.NodeJS.LTS`.

Verify: `node --version` should print `v20.x` or higher.

#### pnpm (9+)

The recommended way is `corepack`, which ships with Node ≥16.10:

```bash
corepack enable
corepack prepare pnpm@latest --activate
```

That installs pnpm globally and pins it for this project.

Alternatives:

- **macOS (Homebrew)** — `brew install pnpm`
- **standalone script** — `curl -fsSL https://get.pnpm.io/install.sh | sh -`
- **npm** — `npm install -g pnpm`

Verify: `pnpm --version` should print `9.x` or higher.

### Windows / WSL development

On Windows, develop from WSL and clone this repository into the Linux filesystem, not a Windows-mounted drive. For example, use a path like `~/src/kaos-control` or `/workspaces/kaos-control`, and avoid `/mnt/c/...`.

Keeping the repo off the Windows drive prevents common file-watching, permissions, symlink, and dependency install issues with Go, Node.js, Vite, and pnpm.

### Dev container

VS Code dev container setup is documented in [.devcontainer/README.md](.devcontainer/README.md). The container post-create hook installs project tooling and bootstraps a minimal `~/.kaos-control` config that registers this workspace as a project.

### 1. Build the binary

```bash
git clone https://github.com/kcsinclair/kaos-control.git
cd kaos-control
make all          # builds web/dist + ./dist/kaos-control
```

The Go binary embeds the SPA via `embed.FS`, so `./dist/kaos-control` is a single self-contained executable.

### 2. First run

```bash
./dist/kaos-control -d
```

On first launch, kaos-control writes a default `~/.kaos-control/config.yaml` and starts listening on `:8042`. Open <http://localhost:8042>.

The contents of the config file are:

```yaml
server:
    listen: :8042
    tls:
        enabled: false
        cert_file: ""
        key_file: ""
auth:
    method: local
    session_ttl: 24h0m0s
projects_dir: /Users/you/.kaos-control/projects
limits:
    max_concurrent_agents: 4
    max_concurrent_scheduler_jobs: 2
    scheduler_run_retention_days: 90
data_dir: /Users/you/.kaos-control/data
agent:
    init_event_timeout_seconds: 10
    require_bypass_permissions: true
```

The `agent:` block controls the `agent-permission-precheck` behaviour described earlier — leave the defaults unless you need to extend the init-event grace window or you're using only the `claude-mediated` driver and want to disable the bypass-mode requirement.

The first user can be created without authentication (bootstrap):

```bash
echo STRONGPASSWORD | ./dist/kaos-control auth create-user -admin -email YOUR_EMAIL -name "YOUR NAME" -password-stdin
```

### 3. Bootstrap a project

The fastest path is the CLI scaffolder:

```bash
cd /path/to/your/project        # any directory; an existing git repo or a fresh one
/path/to/dist/kaos-control init -owner-email YOUR_EMAIL  # creates lifecycle/, lifecycle/config.yaml, CLAUDE.md
```

`kaos-control init` creates the standard `lifecycle/` directory tree (`ideas/`, `requirements/`, `backend-plans/`, `frontend-plans/`, `test-plans/`, `tests/`, `defects/`, `releases/`, `sprints/`, `prototypes/`), a skeleton `lifecycle/config.yaml` with the standard role and agent definitions, and a `CLAUDE.md` to guide agent runs in this project.

Then register the project with kaos-control:

```bash
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

---

## Where things live

| Path                                               | Purpose                                                         |
| -------------------------------------------------- | --------------------------------------------------------------- |
| `~/.kaos-control/config.yaml`                      | App-level config (server, auth, agents, ollama)                 |
| `~/.config/systemd/user/kaos-control.service`      | systemd user unit (Linux), if installed                         |
| `~/.config/kaos-control/service.env`               | Optional env overrides read by the systemd unit                 |
| `~/.kaos-control/projects/*.yaml`                  | One file per registered project                                 |
| `~/.kaos-control/data/<project>/index.db`          | Per-project SQLite cache (rebuilt from disk on startup)         |
| `~/.kaos-control/data/<project>/runs/<run_id>.log` | Per-agent-run log (header, streamed events, summary footer)     |
| `~/.kaos-control/devops/<project>/<run_id>.log`    | DevOps pipeline run logs                                        |
| `<project>/lifecycle/config.yaml`                  | Per-project: roles, agents, plan gates, dashboard tracked types |
| `<project>/lifecycle/devops/*.yaml`                | DevOps pipeline definitions for this project                    |
| `<project>/lifecycle/{ideas,requirements,…}/`      | Artifacts (markdown + YAML frontmatter)                         |

## Repository layout

```
cmd/kaos-control/   Go binary entry point
internal/           Backend packages (agent, artifact, http, index, …)
web/                Vue 3 SPA (built into web/dist/, embedded by Go)
lifecycle/          This project's own artifacts (the meta-bootstrap)
tests/              Integration test code
packaging/          systemd user unit for running the server as a service
plans/              Project plan + per-change implementation plans
```

## Documentation

- [kaos-control.io](https://kaos-control.io) — project site with screenshots, architecture diagrams, and the full pitch.
- [CLAUDE.md](CLAUDE.md) — guidance for Claude Code agents working in this repo (commit conventions, lineage rules, build commands).
- [lifecycle/requirements/Innovation Maker - Making Releases from Ideas-1.md](lifecycle/requirements/Innovation%20Maker%20-%20Making%20Releases%20from%20Ideas-1.md) — authoritative product spec.
- [plans/PROJECT_PLAN.md](plans/PROJECT_PLAN.md) — living state-of-the-project document.

## Licence

**[GNU AGPLv3](LICENSE)** — copyleft with a network-use clause. If you run a modified version of kaos-control as a network service, you must publish your modifications under the same licence.

Commercial licences for organisations that cannot accept AGPL terms may be available on request — open an issue to start the conversation.

## Contributing

Contributions are welcome. The project uses the [Developer Certificate of Origin](https://developercertificate.org/) — sign off on every commit with `git commit -s`. See [CONTRIBUTING.md](CONTRIBUTING.md) for the full workflow.
