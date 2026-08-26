// SPDX-License-Identifier: AGPL-3.0-or-later

package ideachat

import "context"

// Completer abstracts an LLM backend invoked by the inline conversational
// and single-shot generation flows. Implementations turn a system prompt +
// message history into a single completion string.
type Completer interface {
	Complete(ctx context.Context, cfg ModelConfig, messages []LLMMessage) (string, error)
}

// CallLLM is the package-level function used to invoke the LLM. Tests can
// replace it with a deterministic fake; production code should never reassign it.
var CallLLM = dispatchComplete

// dispatchComplete selects a Completer and delegates to it. With no provider
// information on cfg, it always selects the Claude CLI completer.
func dispatchComplete(ctx context.Context, cfg ModelConfig, messages []LLMMessage) (string, error) {
	return (claudeCLICompleter{}).Complete(ctx, cfg, messages)
}
