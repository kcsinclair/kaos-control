// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"strings"
	"testing"
)

// Milestone 7 – Agent Configuration Tests
//
// These tests verify that the idea-capture agent is correctly configured in
// lifecycle/config.yaml and accessible via the agents API.
// They use newTestEnvCustomConfig (defined in idea_chat_helpers_test.go) so
// that the idea-capture agent is present from project-open time.
// None of these tests require ANTHROPIC_API_KEY.

// TestIdeaChatAgentListedInConfig verifies that GET /api/p/:project/agents
// returns a list that includes an agent named "idea-capture".
func TestIdeaChatAgentListedInConfig(t *testing.T) {
	env := newTestEnvCustomConfig(t, ideaCaptureConfigYAML, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/agents", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	agents, _ := data["agents"].([]any)
	if len(agents) == 0 {
		t.Fatal("GET /agents returned an empty list; expected at least 'idea-capture'")
	}

	var found bool
	for _, a := range agents {
		entry, _ := a.(map[string]any)
		if name, _ := entry["name"].(string); name == "idea-capture" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("agent 'idea-capture' not found in GET /agents response; got %v", agents)
	}
}

// TestIdeaChatAgentHasCorrectFields verifies that the idea-capture agent entry
// returned by GET /api/p/:project/agents has driver "inline" and
// allowed_write_paths containing "lifecycle/ideas".
func TestIdeaChatAgentHasCorrectFields(t *testing.T) {
	env := newTestEnvCustomConfig(t, ideaCaptureConfigYAML, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/agents", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	agents, _ := data["agents"].([]any)
	var agentEntry map[string]any
	for _, a := range agents {
		entry, _ := a.(map[string]any)
		if name, _ := entry["name"].(string); name == "idea-capture" {
			agentEntry = entry
			break
		}
	}
	if agentEntry == nil {
		t.Fatal("idea-capture agent not found; cannot check fields")
	}

	// driver should be "inline"
	driver, _ := agentEntry["driver"].(string)
	if driver != "inline" {
		t.Errorf("idea-capture driver: want 'inline', got %q", driver)
	}

	// allowed_write_paths should contain "lifecycle/ideas"
	paths, _ := agentEntry["allowed_write_paths"].([]any)
	var hasIdeasPath bool
	for _, p := range paths {
		if s, _ := p.(string); s == "lifecycle/ideas" {
			hasIdeasPath = true
			break
		}
	}
	if !hasIdeasPath {
		t.Errorf("idea-capture allowed_write_paths should contain 'lifecycle/ideas', got %v", paths)
	}
}

// TestIdeaChatPromptTemplateExists verifies that the project config YAML served
// by GET /api/p/:project/config includes a prompt_templates entry with an
// "idea-capture" key containing a non-empty string.
func TestIdeaChatPromptTemplateExists(t *testing.T) {
	env := newTestEnvCustomConfig(t, ideaCaptureConfigYAML, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/config", nil)
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	raw, _ := data["raw"].(string)
	if raw == "" {
		t.Fatal("GET /config returned empty raw YAML")
	}

	// The config YAML must reference the idea-capture prompt template.
	// We check for the key name and a non-empty value heuristically.
	if !strings.Contains(raw, "idea-capture:") {
		t.Error("config YAML does not contain 'idea-capture:' key in prompt_templates")
	}
	if !strings.Contains(raw, "prompt_templates:") {
		t.Error("config YAML does not contain 'prompt_templates:' section")
	}

	// The prompt must be non-trivially long (at least 50 characters of content).
	idx := strings.Index(raw, "idea-capture:")
	if idx >= 0 {
		excerpt := raw[idx:]
		// Everything after the key, up to the next top-level key, should be non-empty.
		if len(strings.TrimSpace(excerpt)) < 50 {
			t.Errorf("idea-capture prompt template appears too short (<%d chars after key)", 50)
		}
	}
}
