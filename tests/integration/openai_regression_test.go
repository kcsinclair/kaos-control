// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
)

// ── OpenAI & Provider Integration Regression Tests ───────────────────────────

// TestOpenAIRegression_ExistingDriversWork verifies that standard driver configurations
// continue to load and validate without regression (NFR-2).
func TestOpenAIRegression_ExistingDriversWork(t *testing.T) {
	drivers := []string{
		"claude-code-cli",
		"claude-mediated",
		"codex-cli",
		"gemini",
		"gemini-cli",
		"shell-stub",
		"ollama",
	}

	for _, drv := range drivers {
		t.Run(drv, func(t *testing.T) {
			extraProps := ""
			if drv == "gemini" || drv == "gemini-cli" {
				extraProps = "    model: gemini-2.0-flash\n"
			}
			if drv == "ollama" {
				extraProps = "    model: test-model\n    ollama_instance: test-instance\n"
			}
			cfgYAML := `
git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles: [product-owner, analyst]

stages:
  - {name: ideas, dir: ideas}

users:
  - email: admin@test.local
    roles: [product-owner]

agents:
  - name: test-agent-` + drv + `
    role: [analyst]
    driver: ` + drv + `
` + extraProps + `    git_identity:
      name: Test Agent
      email: test@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"
`
			root := makeProjectRoot(t, cfgYAML)
			cfg, err := config.LoadProject(root)
			if err != nil {
				t.Fatalf("LoadProject for driver %q failed: %v", drv, err)
			}
			ag := findAgentConfig(cfg, "test-agent-"+drv)
			if ag == nil {
				t.Fatalf("agent with driver %q not found in config", drv)
			}
			if ag.Driver != drv {
				t.Errorf("got driver %q, want %q", ag.Driver, drv)
			}
		})
	}
}

// TestOpenAIRegression_MixedAgentsConfig verifies that mixed agent rosters validate seamlessly.
func TestOpenAIRegression_MixedAgentsConfig(t *testing.T) {
	cfgYAML := `
git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles: [product-owner, analyst, backend-developer]

stages:
  - {name: ideas, dir: ideas}

users:
  - email: admin@test.local
    roles: [product-owner, analyst, backend-developer]

agents:
  - name: claude-agent
    role: [analyst]
    driver: claude-code-cli
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Claude Agent
      email: claude@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"

  - name: openai-agent
    role: [backend-developer]
    driver: openai-compatible
    provider: my-provider
    model: gemma-4-26B-A4B-it-UD-Q8_K_XL
    allowed_write_paths: [lifecycle/backend-plans]
    git_identity:
      name: OpenAI Agent
      email: openai@test.local
    prompt_templates:
      backend-developer: "Plan {target_path}"

  - name: stub-agent
    role: [analyst]
    driver: shell-stub
    git_identity:
      name: Stub Agent
      email: stub@test.local
    prompt_templates:
      analyst: "Stub {target_path}"
`
	root := makeProjectRoot(t, cfgYAML)
	cfg, err := config.LoadProject(root)
	if err != nil {
		t.Fatalf("LoadProject: unexpected error: %v", err)
	}

	claudeAg := findAgentConfig(cfg, "claude-agent")
	if claudeAg == nil || claudeAg.Driver != "claude-code-cli" {
		t.Errorf("claude-agent not loaded correctly")
	}

	openaiAg := findAgentConfig(cfg, "openai-agent")
	if openaiAg == nil || openaiAg.Driver != "openai-compatible" || openaiAg.Provider != "my-provider" {
		t.Errorf("openai-agent not loaded correctly")
	}
}

// TestOpenAIRegression_SecretMasking verifies API key masking across provider and agent APIs (NFR-1).
func TestOpenAIRegression_SecretMasking(t *testing.T) {
	env := newProviderAPITestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// 1. Check /api/providers masking
	provResp := env.doRequest(http.MethodGet, "/api/providers", nil)
	if provResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/providers failed: %d", provResp.StatusCode)
	}
	provData := readJSON(t, provResp)
	provList := provData["providers"].([]any)
	for _, p := range provList {
		pm := p.(map[string]any)
		if pm["name"] == "mock-provider" {
			if pm["api_key"] != "***" {
				t.Errorf("provider api_key not masked: %v", pm["api_key"])
			}
		}
	}

	// 2. Check /api/p/{project}/agents does not expose raw secret tokens
	agentsResp := env.doRequest(http.MethodGet, "/api/p/testproject/agents", nil)
	if agentsResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/p/testproject/agents failed: %d", agentsResp.StatusCode)
	}
	b, err := io.ReadAll(agentsResp.Body)
	agentsResp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	bodyStr := string(b)
	if strings.Contains(bodyStr, "secret-key-123") {
		t.Errorf("raw secret key leaked in agents API response: %s", bodyStr)
	}
}
