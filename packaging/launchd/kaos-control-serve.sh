#!/bin/zsh
# kaos-control — launchd server wrapper (macOS).
#
# Rebuilds the binary, then exec's the foreground server. Invoked by the
# io.kaos-control.server LaunchAgent (see io.kaos-control.server.plist).
# Everything this script prints — build output, the server's own logs, and any
# panic stack traces — is captured to the log file configured in the plist,
# because launchd redirects our stdout/stderr there.
#
# PATH (go, pnpm, node, make) comes from your login shell: the plist launches
# us via `zsh -lc`, which sources your profile before exec'ing this script.
set -euo pipefail

# --- config (override via the LaunchAgent's EnvironmentVariables) ------------
REPO="${KC_REPO:-/Users/keith/Code/kaos-control}"
CONFIG="${KC_CONFIG:-$HOME/.kaos-control/config.yaml}"
BUILD_WEB="${KC_BUILD_WEB:-1}"          # set 0 to skip the (slow) pnpm SPA build
export LOG_LEVEL="${LOG_LEVEL:-info}"

trap 'echo "=== kaos-control launchd: startup FAILED (see above); launchd will retry ==="' ERR

cd "$REPO"
echo "=== $(date '+%Y-%m-%d %H:%M:%S') kaos-control launchd start — build then serve ==="
echo "repo=$REPO  config=$CONFIG  build_web=$BUILD_WEB  log_level=$LOG_LEVEL"

command -v go >/dev/null || { echo "FATAL: 'go' is not on PATH (check your login shell profile)"; exit 127; }

if [[ "$BUILD_WEB" == "1" ]]; then
  echo "--- make build-web ---"
  make build-web
else
  echo "--- skipping build-web (KC_BUILD_WEB=0); reusing existing web/dist ---"
fi

echo "--- make build ---"
make build

echo "--- exec: ./dist/kaos-control serve ---"
# exec so the server replaces this shell: launchd tracks the server's PID and
# its SIGTERM reaches the server directly (graceful drain, ~10s).
exec ./dist/kaos-control serve -config "$CONFIG"
