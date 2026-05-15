#!/bin/sh
# mock_claude_mediated.sh — Stand-in for `claude` in mediated-driver integration tests.
#
# Emits stream-json events to stdout, simulating a Claude Code process that has
# been started with a --settings file (i.e. hooks are wired but no
# --dangerously-skip-permissions flag is present).
#
# Usage (tests prepend the script's directory to PATH as "claude"):
#   MOCK_CLAUDE_PERMISSION_MODE=default ./mock_claude_mediated.sh [ignored args]
#
# Environment variables:
#   MOCK_CLAUDE_PERMISSION_MODE   permissionMode to emit in system/init event.
#                                 Use "bypassPermissions" to trigger the mediated
#                                 precheck failure path (AC13).
#                                 Default: "default"
#   MOCK_CLAUDE_HOLD_AFTER_INIT   If "true", block indefinitely after init
#                                 (lets the test supervisor kill the process).
#                                 Default: "false"
#   MOCK_CLAUDE_EMIT_WRITE_TOOL   If "true", emit a synthetic Write tool-use
#                                 assistant message after the init event.
#                                 Default: "false"
#   FAKE_CLAUDE_MODE              Alias for MOCK_CLAUDE_PERMISSION_MODE (for
#                                 compatibility with fake_precheck_claude env var).
#   FAKE_CLAUDE_HOLD_AFTER_INIT   Alias for MOCK_CLAUDE_HOLD_AFTER_INIT.
#   FAKE_CLAUDE_ARGS_FILE         If set, write os.Args as JSON to this file
#                                 (matches fake_precheck_claude behaviour).

# Aliases: honour fake_precheck_claude env vars when the dedicated vars are unset.
PERMISSION_MODE="${MOCK_CLAUDE_PERMISSION_MODE:-${FAKE_CLAUDE_MODE:-default}}"
HOLD="${MOCK_CLAUDE_HOLD_AFTER_INIT:-${FAKE_CLAUDE_HOLD_AFTER_INIT:-false}}"
EMIT_WRITE="${MOCK_CLAUDE_EMIT_WRITE_TOOL:-false}"

# Optionally record args (for arg-inspection tests).
if [ -n "$FAKE_CLAUDE_ARGS_FILE" ]; then
    printf '["mock_claude_mediated"' > "$FAKE_CLAUDE_ARGS_FILE"
    for arg in "$@"; do
        printf ',"%s"' "$arg" >> "$FAKE_CLAUDE_ARGS_FILE"
    done
    printf ']\n' >> "$FAKE_CLAUDE_ARGS_FILE"
fi

# 1. Emit system/init event.
printf '{"type":"system","subtype":"init","permissionMode":"%s"}\n' "$PERMISSION_MODE"

# 2. Optionally hold (to let the supervisor time out or kill us).
if [ "$HOLD" = "true" ]; then
    sleep 3600
    exit 0
fi

# 3. Optionally emit a synthetic Write tool-use message.
if [ "$EMIT_WRITE" = "true" ]; then
    printf '{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tool1","name":"Write","input":{"file_path":"lifecycle/requirements/mock-output.md","content":"# Mock Output\n"}}]}}\n'
    sleep 0.05
fi

# 4. Emit a successful result event and exit cleanly.
printf '{"type":"result","subtype":"success","result":"Task completed by mock"}\n'
