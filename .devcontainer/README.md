# Dev Container

This repository includes a VS Code dev container for local kaos-control development.

On container creation, `.devcontainer/post-create.sh` installs the Go, Node, pnpm, Claude Code, Codex, and Delve tooling used by the project. It also runs `.devcontainer/bootstrap-kaos-control.sh`, which creates a minimal local kaos-control app config:

```text
~/.kaos-control/config.yaml
~/.kaos-control/projects/kaos-control.yaml
~/.kaos-control/data/
```

The project entry points at the checked-out workspace, so the debug server can list this repo in the project picker.

## Useful Commands

```sh
# Recreate the local kaos-control config/project entry.
.devcontainer/bootstrap-kaos-control.sh

# Start the backend with the devcontainer config.
go run ./cmd/kaos-control serve -config ~/.kaos-control/config.yaml
```

The dev container forwards:

| Port | Purpose |
|---|---|
| `5173` | Vite dev server |
| `8080` | kaos-control server |

## Bootstrap Overrides

`bootstrap-kaos-control.sh` supports these environment variables:

| Variable | Default |
|---|---|
| `KAOS_CONTROL_HOME` | `$HOME/.kaos-control` |
| `KAOS_CONTROL_PROJECTS_DIR` | `$KAOS_CONTROL_HOME/projects` |
| `KAOS_CONTROL_DATA_DIR` | `$KAOS_CONTROL_HOME/data` |
| `KAOS_CONTROL_PROJECT_NAME` | repository directory name |
| `KAOS_CONTROL_PROJECT_OWNER` | `git config user.email`, then `vscode@localhost` |
