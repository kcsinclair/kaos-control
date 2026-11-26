// SPDX-License-Identifier: AGPL-3.0-or-later

package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
)

// TestLocalModelPromptDefaults_TokenBudget approximates the plan's <1200
// token requirement using a word-count proxy (~1.3 tokens/word for English
// prose), and checks the mandatory single-step ordering and frontmatter
// markers are present.
func TestLocalModelPromptDefaults_TokenBudget(t *testing.T) {
	const maxWords = 900 // ~1170 tokens at 1.3 tokens/word

	for name, prompt := range LocalModelPromptDefaults {
		words := len(strings.Fields(prompt))
		if words > maxWords {
			t.Errorf("prompt %q: %d words exceeds budget of %d (~1200 tokens)", name, words, maxWords)
		}
		if !strings.Contains(prompt, "1.") {
			t.Errorf("prompt %q: missing explicit step ordering (expected a %q marker)", name, "1.")
		}
		// Only artifact-authoring roles need an embedded frontmatter few-shot;
		// backend/frontend-developer produce code commits, not markdown artifacts.
		if name != "backend-developer" && name != "frontend-developer" && !strings.Contains(prompt, "```yaml") {
			t.Errorf("prompt %q: missing a concrete frontmatter block", name)
		}
	}
}

// TestStartRun_LocalModelPromptFallback verifies that an agent whose config
// omits a prompt_templates entry for the active role falls back to
// LocalModelPromptDefaults by agent name instead of failing.
func TestStartRun_LocalModelPromptFallback(t *testing.T) {
	agents := []config.AgentConfig{
		{
			Name:         "backend-developer",
			Roles:        []string{"backend-developer"},
			Driver:       "shell-stub",
			Model:        "n/a",
			ActiveStatus: "",
			// No PromptTemplates set: must fall back to LocalModelPromptDefaults.
		},
	}
	mgr, cleanup := newMinimalManager(t, agents, 4)
	defer cleanup()

	_, err := mgr.StartRun(context.Background(), "backend-developer", "lifecycle/backend-plans/test-3-be.md", "backend-developer", nil)
	if err != nil && strings.Contains(err.Error(), "has no prompt template") {
		t.Fatalf("expected local-model fallback prompt to be used, got: %v", err)
	}
}
