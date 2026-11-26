// SPDX-License-Identifier: AGPL-3.0-or-later

package ideachat

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// claudeCLICompleter is the default Completer: it shells out to the local
// `claude` binary. This is the pre-existing inline behaviour, preserved
// byte-identical.
type claudeCLICompleter struct{}

func (claudeCLICompleter) Complete(ctx context.Context, cfg ModelConfig, messages []LLMMessage) (string, error) {
	prompt := buildPrompt(cfg.SystemPrompt, messages)

	args := []string{"--dangerously-skip-permissions", "-p", prompt}
	if cfg.Model != "" {
		args = append(args, "--model", cfg.Model)
	}

	out, err := exec.CommandContext(ctx, "claude", args...).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude exited %d: %s", ee.ExitCode(), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("calling claude: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
