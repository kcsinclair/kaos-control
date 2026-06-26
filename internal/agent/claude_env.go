// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"os"
	"os/exec"
)

// ClaudeEnvDriver runs the standard `claude` CLI agentic loop (bypass
// permissions, stream-json) but retargets it to an Anthropic-compatible
// endpoint by injecting ANTHROPIC_BASE_URL and ANTHROPIC_AUTH_TOKEN into the
// subprocess environment, plus --model. It reuses ClaudeCodeDriver.buildArgs
// so the argument vector is identical to claude-code-cli, and delegates to
// startCommandProcess for progress streaming, TTFT, log files, kill, and Wait.
type ClaudeEnvDriver struct{}

// Start spawns `claude` with bypass-permissions args and the configured
// endpoint/token injected via environment variables.
func (d *ClaudeEnvDriver) Start(ctx context.Context, run Run) (Process, error) {
	args := (&ClaudeCodeDriver{}).buildArgs(run)

	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = run.ProjectRoot

	// Append after os.Environ() so the injected values take precedence over
	// any inherited ANTHROPIC_BASE_URL / ANTHROPIC_AUTH_TOKEN (Go exec uses
	// the last occurrence of a duplicated key).
	cmd.Env = append(os.Environ(),
		"ANTHROPIC_BASE_URL="+run.BaseURL,
		"ANTHROPIC_AUTH_TOKEN="+run.AuthToken,
	)

	return startCommandProcess(ctx, cmd, run, args, "claude")
}
