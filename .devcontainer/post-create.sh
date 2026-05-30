#!/usr/bin/env bash
set -euo pipefail

export PNPM_HOME="${PNPM_HOME:-$HOME/.local/share/pnpm}"
export PATH="$PNPM_HOME:$PATH"
PNPM_VERSION="${PNPM_VERSION:-10.21.0}"
mkdir -p "$PNPM_HOME"
sudo mkdir -p "$HOME/.claude" "$HOME/.codex"
sudo chown -R "$(id -u):$(id -g)" "$HOME/.claude" "$HOME/.codex"

bash .devcontainer/bootstrap-kaos-control.sh

corepack enable --install-directory "$PNPM_HOME"
corepack prepare "pnpm@$PNPM_VERSION" --activate
pnpm config set global-bin-dir "$PNPM_HOME" --global
pnpm config set ignore-scripts false --global

pnpm add --global \
  --allow-build=@anthropic-ai/claude-code \
  --allow-build=@openai/codex \
  @anthropic-ai/claude-code \
  @openai/codex

go install github.com/go-delve/delve/cmd/dlv@latest

CLAUDE_INSTALLER="$(pnpm root -g)/@anthropic-ai/claude-code/install.cjs"
if [[ -f "$CLAUDE_INSTALLER" ]]; then
  node "$CLAUDE_INSTALLER"
fi

go version
node --version
pnpm --version
git --version
claude --version
codex --version
