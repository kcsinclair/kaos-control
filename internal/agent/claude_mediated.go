// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

// ClaudeHooksDriver spawns `claude --settings <path> -p ...` without any
// bypass-permissions flags. Instead it relies on the PreToolUse hook
// (wired via the generated settings.json) to mediate every tool call (FR1–FR3).
//
// The driver generates a per-run secret, writes a temporary settings.json, and
// injects KC_HOOK_SECRET into the subprocess environment (FR5/FR6).
type ClaudeHooksDriver struct {
	// ServerAddr is the address the hook-helper should POST permission requests
	// to, e.g. "127.0.0.1:9600".
	ServerAddr string
	// BinaryPath is the absolute path to the running kaos-control binary (used
	// as the hook-helper command). Defaults to os.Executable() if empty.
	BinaryPath string
	// StoreSecret is a callback that saves the per-run secret in the Manager so
	// the permission HTTP endpoint can validate incoming requests.
	StoreSecret func(runID, secret string)
}

// Start generates a per-run secret, writes a temporary settings.json, and
// spawns `claude` with the settings file — without any bypass-permissions flags.
// The settings.json is deleted when the process exits (NFR4).
func (d *ClaudeHooksDriver) Start(ctx context.Context, run Run) (Process, error) {
	binary := d.BinaryPath
	if binary == "" {
		var err error
		binary, err = os.Executable()
		if err != nil {
			return nil, fmt.Errorf("resolving binary path: %w", err)
		}
	}

	// 1. Generate per-run secret.
	secret, err := GenerateRunSecret()
	if err != nil {
		return nil, fmt.Errorf("generating run secret: %w", err)
	}

	// 2. Store secret in Manager via callback.
	if d.StoreSecret != nil {
		d.StoreSecret(run.RunID, secret)
	}

	// 3. Write temp settings.json.
	settingsDir := os.TempDir()
	settingsPath, cleanup, err := WriteHookSettings(settingsDir, binary, d.ServerAddr, run.RunID)
	if err != nil {
		return nil, fmt.Errorf("writing hook settings: %w", err)
	}

	// 4. Build args — no bypass flags (FR2).
	args := d.buildArgs(run, settingsPath)

	// 5. Set up subprocess.
	cmd := exec.CommandContext(ctx, "claude", args...)
	cmd.Dir = run.ProjectRoot

	// Inherit parent environment, then inject the secret (FR5).
	cmd.Env = append(os.Environ(), "KC_HOOK_SECRET="+secret)

	slog.Debug("agent: starting claude-mediated", "run_id", run.RunID, "settings", settingsPath, "server", d.ServerAddr)

	// 6. Start the process, reusing shared stream-JSON logic (FR3).
	proc, err := startCommandProcess(ctx, cmd, run, args, "claude")
	if err != nil {
		cleanup()
		return nil, err
	}

	// 7. Wrap the process so the settings file is removed on exit (NFR4).
	return &mediatedProcess{Process: proc, cleanup: cleanup}, nil
}

// buildArgs constructs the claude CLI arguments for the mediated driver.
// It does NOT include --permission-mode or --dangerously-skip-permissions.
func (d *ClaudeHooksDriver) buildArgs(run Run, settingsPath string) []string {
	args := []string{
		"--settings", settingsPath,
		"-p", run.PromptText,
		"--output-format", "stream-json",
		"--verbose",
	}
	if run.Model != "" {
		args = append(args, "--model", run.Model)
	}
	return args
}

// mediatedProcess wraps a Process and invokes a cleanup function when the
// process exits via Wait().
type mediatedProcess struct {
	Process
	cleanup   func()
	cleanOnce bool
}

func (p *mediatedProcess) Wait() error {
	err := p.Process.Wait()
	if !p.cleanOnce {
		p.cleanOnce = true
		p.cleanup()
	}
	return err
}
