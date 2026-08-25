// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 4 — Prompt Template Rendering & Scaffolding Tests
// (local-model-operability: lifecycle/test-plans/local-model-operability-5-test.md)
//
// FR-1: an openai-compatible agent whose config omits a prompt_templates
// entry for its active role must not hard-fail — it falls back to the
// concise, local-model-tuned preset in internal/agent/prompt_defaults.go
// (LocalModelPromptDefaults, keyed by agent name).
//
// internal/agent/prompt_defaults_test.go already covers the fallback and its
// <1200 token budget at the Manager level. This test drives the same
// scenario through the full HTTP API + a real HTTP round trip to the mock
// inference server, confirming the fallback prompt is what's actually sent
// on the wire — not just what Manager.StartRun selects internally.
//
// The "backend-developer" agent used here is defined in the shared
// openAIAgentCfgTemplate (openai_agent_run_test.go) specifically without a
// prompt_templates entry, so its name is used to key into
// LocalModelPromptDefaults.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/agent"
)

// TestOpenAIAgentRun_LocalModelPromptFallback_ReachesProvider verifies that
// when an openai-compatible agent has no prompt_templates entry for its
// role, the fallback prompt defined in LocalModelPromptDefaults is what
// actually reaches the provider's /v1/chat/completions endpoint.
func TestOpenAIAgentRun_LocalModelPromptFallback_ReachesProvider(t *testing.T) {
	env := newOpenAIAgentTestEnv(t, nil, 4)
	env.login("admin@test.local", "admin-pass-123")

	runID := startAgentRun(t, env.testEnv, "backend-developer", "lifecycle/ideas/test-idea.md")
	waitForRunCompletion(t, env.testEnv, runID)

	wantMarker := firstLine(agent.PromptDefaultBackendDeveloper)

	requests := env.mock.RequestsForPath("/v1/chat/completions")
	if len(requests) == 0 {
		t.Fatal("expected at least one /v1/chat/completions request")
	}

	found := false
	for _, r := range requests {
		var body struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(r.Body, &body); err != nil {
			continue
		}
		for _, m := range body.Messages {
			if strings.Contains(m.Content, wantMarker) {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("expected a chat completion request to contain the local-model fallback prompt (marker %q), none of %d requests did", wantMarker, len(requests))
	}
}

func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}
