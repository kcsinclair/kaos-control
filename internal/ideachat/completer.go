// SPDX-License-Identifier: AGPL-3.0-or-later

package ideachat

import (
	"context"
	"fmt"
)

// Completer abstracts an LLM backend invoked by the inline conversational
// and single-shot generation flows. Implementations turn a system prompt +
// message history into a single completion string.
type Completer interface {
	Complete(ctx context.Context, cfg ModelConfig, messages []LLMMessage) (string, error)
}

// CallLLM is the package-level function used to invoke the LLM. Tests can
// replace it with a deterministic fake; production code should never reassign it.
var CallLLM = dispatchComplete

// dispatchComplete selects a Completer for cfg and delegates to it.
func dispatchComplete(ctx context.Context, cfg ModelConfig, messages []LLMMessage) (string, error) {
	completer, err := selectCompleter(cfg)
	if err != nil {
		return "", err
	}
	return completer.Complete(ctx, cfg, messages)
}

// selectCompleter picks a Completer based on cfg.Provider. A nil Provider
// selects the Claude CLI default. An unrecognized provider driver is
// defensive: config validation (config.ValidateAgentProviders) is expected
// to prevent it reaching here.
func selectCompleter(cfg ModelConfig) (Completer, error) {
	if cfg.Provider == nil {
		return claudeCLICompleter{}, nil
	}
	switch cfg.Provider.Driver {
	case "openai-compatible":
		return &openAICompleter{provider: *cfg.Provider}, nil
	default:
		return nil, fmt.Errorf("ideachat: provider %q has unsupported driver %q", cfg.Provider.Name, cfg.Provider.Driver)
	}
}
