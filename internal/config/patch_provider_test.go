// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeProviderPatchTestConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "lifecycle"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lifecycle", "config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const providerPatchBaseConfig = `# top-level comment preserved
stages:
  - {name: ideas, dir: ideas}
git:
  default_branch: main
roles:
  - analyst
agents:
  - name: analyst-agent # inline comment preserved
    role: [analyst]
    driver: openai-compatible
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    prompt_templates:
      analyst: "x"
  - name: other-agent
    role: [qa]
    driver: openai-compatible
    provider: anthropic-cloud
    model: claude-3-7-sonnet
    prompt_templates:
      qa: "y"
`

func TestPatchAgentProviders_PreservesCommentsAndFormatting(t *testing.T) {
	dir := writeProviderPatchTestConfig(t, providerPatchBaseConfig)

	err := PatchAgentProviders(dir, []AgentProviderPatch{{
		AgentName: "analyst-agent",
		Provider:  "gemini-cloud",
		Model:     "gemini-2.5-flash",
	}})
	if err != nil {
		t.Fatalf("PatchAgentProviders: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "lifecycle", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, "# top-level comment preserved") {
		t.Error("expected top-level comment to survive the patch")
	}
	if !strings.Contains(content, "# inline comment preserved") {
		t.Error("expected inline comment to survive the patch")
	}
	if !strings.Contains(content, "provider: gemini-cloud") {
		t.Error("expected patched provider in output")
	}
	if !strings.Contains(content, "model: gemini-2.5-flash") {
		t.Error("expected patched model in output")
	}

	cfg, err := LoadProject(dir)
	if err != nil {
		t.Fatalf("LoadProject after patch: %v", err)
	}
	var analyst, other *AgentConfig
	for i := range cfg.Agents {
		switch cfg.Agents[i].Name {
		case "analyst-agent":
			analyst = &cfg.Agents[i]
		case "other-agent":
			other = &cfg.Agents[i]
		}
	}
	if analyst == nil || analyst.Provider != "gemini-cloud" || analyst.Model != "gemini-2.5-flash" {
		t.Fatalf("analyst-agent not patched correctly: %+v", analyst)
	}
	if other == nil || other.Provider != "anthropic-cloud" {
		t.Fatalf("other-agent should be untouched: %+v", other)
	}
}

func TestPatchAgentProviders_PrimaryProviderAddAndRemove(t *testing.T) {
	dir := writeProviderPatchTestConfig(t, providerPatchBaseConfig)

	primaryProvider := "anthropic-cloud"
	primaryModel := "claude-3-7-sonnet"
	err := PatchAgentProviders(dir, []AgentProviderPatch{{
		AgentName:       "analyst-agent",
		Provider:        "gemini-cloud",
		Model:           "gemini-2.5-flash",
		PrimaryProvider: &primaryProvider,
		PrimaryModel:    &primaryModel,
	}})
	if err != nil {
		t.Fatalf("PatchAgentProviders (add primary): %v", err)
	}

	cfg, err := LoadProject(dir)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	ag := cfg.Agents[0]
	if ag.PrimaryProvider != "anthropic-cloud" || ag.PrimaryModel != "claude-3-7-sonnet" {
		t.Fatalf("expected primary fields set, got %+v", ag)
	}

	// Now remove them (restore path): empty string pointer means "delete key".
	empty := ""
	err = PatchAgentProviders(dir, []AgentProviderPatch{{
		AgentName:       "analyst-agent",
		Provider:        "anthropic-cloud",
		Model:           "claude-3-7-sonnet",
		PrimaryProvider: &empty,
		PrimaryModel:    &empty,
	}})
	if err != nil {
		t.Fatalf("PatchAgentProviders (remove primary): %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "lifecycle", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "primary_provider") {
		t.Error("expected primary_provider key to be removed from disk")
	}

	cfg, err = LoadProject(dir)
	if err != nil {
		t.Fatalf("LoadProject after removal: %v", err)
	}
	ag = cfg.Agents[0]
	if ag.PrimaryProvider != "" || ag.PrimaryModel != "" {
		t.Fatalf("expected primary fields cleared, got %+v", ag)
	}
	if ag.Provider != "anthropic-cloud" || ag.Model != "claude-3-7-sonnet" {
		t.Fatalf("expected restored provider/model, got %+v", ag)
	}
}

func TestPatchAgentProviders_MultiAgentBatchAtomic(t *testing.T) {
	dir := writeProviderPatchTestConfig(t, providerPatchBaseConfig)

	err := PatchAgentProviders(dir, []AgentProviderPatch{
		{AgentName: "analyst-agent", Provider: "gemini-cloud", Model: "gemini-2.5-flash"},
		{AgentName: "other-agent", Provider: "gemini-cloud", Model: "gemini-2.5-flash"},
	})
	if err != nil {
		t.Fatalf("PatchAgentProviders: %v", err)
	}

	cfg, err := LoadProject(dir)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	for _, ag := range cfg.Agents {
		if ag.Name != "analyst-agent" && ag.Name != "other-agent" {
			continue
		}
		if ag.Provider != "gemini-cloud" || ag.Model != "gemini-2.5-flash" {
			t.Errorf("agent %q not patched: %+v", ag.Name, ag)
		}
	}
}

func TestPatchAgentProviders_UnknownAgentErrors(t *testing.T) {
	dir := writeProviderPatchTestConfig(t, providerPatchBaseConfig)

	err := PatchAgentProviders(dir, []AgentProviderPatch{{
		AgentName: "does-not-exist",
		Provider:  "gemini-cloud",
		Model:     "gemini-2.5-flash",
	}})
	if err == nil || !strings.Contains(err.Error(), "does-not-exist") {
		t.Fatalf("expected unknown-agent error, got: %v", err)
	}

	// The file must be untouched on error.
	raw, readErr := os.ReadFile(filepath.Join(dir, "lifecycle", "config.yaml"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(raw) != providerPatchBaseConfig {
		t.Error("expected config.yaml to be untouched after a failed patch")
	}
}

func TestPatchAgentProviders_InvalidPatchRejectedBeforeWrite(t *testing.T) {
	dir := writeProviderPatchTestConfig(t, providerPatchBaseConfig)

	// Empty model with a non-empty provider fails validateProject
	// (openai-compatible requires model), so the write must be rejected.
	err := PatchAgentProviders(dir, []AgentProviderPatch{{
		AgentName: "analyst-agent",
		Provider:  "gemini-cloud",
		Model:     "",
	}})
	if err == nil {
		t.Fatal("expected validation error for missing model")
	}

	raw, readErr := os.ReadFile(filepath.Join(dir, "lifecycle", "config.yaml"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(raw) != providerPatchBaseConfig {
		t.Error("expected config.yaml to be untouched after a rejected patch")
	}
}
