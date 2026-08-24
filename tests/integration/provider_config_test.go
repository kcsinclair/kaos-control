// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
)

// ── Provider & Agent Configuration Integration Tests ─────────────────────────

const validAppCfgWithProviders = `
server:
  listen: ":8080"
auth:
  method: local
  session_ttl: 24h
providers:
  - name: local-llama
    base_url: http://localhost:7442
    driver: openai-compatible
  - name: openrouter
    base_url: https://openrouter.ai/api
    driver: openai-compatible
    api_key: "sk-or-v1-secret"
    extra_headers:
      HTTP-Referer: "https://kaos-control.local"
      X-Title: "Kaos Control"
`

// TestProviderConfig_LoadWithProviders verifies that an app config YAML with
// providers parses correctly and populates all fields.
func TestProviderConfig_LoadWithProviders(t *testing.T) {
	path := writeAppCfgFile(t, validAppCfgWithProviders)

	cfg, err := config.LoadApp(path)
	if err != nil {
		t.Fatalf("LoadApp: unexpected error: %v", err)
	}

	if len(cfg.Providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(cfg.Providers))
	}

	local := cfg.Providers[0]
	if local.Name != "local-llama" {
		t.Errorf("providers[0].Name: got %q, want %q", local.Name, "local-llama")
	}
	if local.BaseURL != "http://localhost:7442" {
		t.Errorf("providers[0].BaseURL: got %q, want %q", local.BaseURL, "http://localhost:7442")
	}
	if local.Driver != "openai-compatible" {
		t.Errorf("providers[0].Driver: got %q, want %q", local.Driver, "openai-compatible")
	}
	if local.APIKey != "" {
		t.Errorf("providers[0].APIKey: expected empty, got %q", local.APIKey)
	}

	remote := cfg.Providers[1]
	if remote.Name != "openrouter" {
		t.Errorf("providers[1].Name: got %q, want %q", remote.Name, "openrouter")
	}
	if remote.APIKey != "sk-or-v1-secret" {
		t.Errorf("providers[1].APIKey: got %q, want %q", remote.APIKey, "sk-or-v1-secret")
	}
	if remote.ExtraHeaders["HTTP-Referer"] != "https://kaos-control.local" {
		t.Errorf("providers[1].ExtraHeaders[HTTP-Referer]: got %q, want %q",
			remote.ExtraHeaders["HTTP-Referer"], "https://kaos-control.local")
	}
	if remote.ExtraHeaders["X-Title"] != "Kaos Control" {
		t.Errorf("providers[1].ExtraHeaders[X-Title]: got %q, want %q",
			remote.ExtraHeaders["X-Title"], "Kaos Control")
	}
}

// TestProviderConfig_RoundTrip verifies that Load -> Save -> Load preserves providers accurately.
func TestProviderConfig_RoundTrip(t *testing.T) {
	path := writeAppCfgFile(t, validAppCfgWithProviders)

	cfg1, err := config.LoadApp(path)
	if err != nil {
		t.Fatalf("first LoadApp: %v", err)
	}

	if err := config.SaveApp(path, *cfg1); err != nil {
		t.Fatalf("SaveApp: %v", err)
	}

	cfg2, err := config.LoadApp(path)
	if err != nil {
		t.Fatalf("second LoadApp after save: %v", err)
	}

	if len(cfg2.Providers) != len(cfg1.Providers) {
		t.Fatalf("round-trip: provider count changed: %d -> %d",
			len(cfg1.Providers), len(cfg2.Providers))
	}
	for i, p := range cfg1.Providers {
		got := cfg2.Providers[i]
		if p.Name != got.Name || p.BaseURL != got.BaseURL || p.Driver != got.Driver || p.APIKey != got.APIKey {
			t.Errorf("round-trip mismatch at index %d: want %+v, got %+v", i, p, got)
		}
	}
}

// TestProviderConfig_DuplicateNameRejected verifies that duplicate provider names fail validation.
func TestProviderConfig_DuplicateNameRejected(t *testing.T) {
	yaml := `
server:
  listen: ":8080"
auth:
  method: local
  session_ttl: 24h
providers:
  - name: my-provider
    base_url: http://localhost:7442
    driver: openai-compatible
  - name: my-provider
    base_url: http://localhost:7443
    driver: openai-compatible
`
	path := writeAppCfgFile(t, yaml)

	_, err := config.LoadApp(path)
	if err == nil {
		t.Fatal("expected validation error for duplicate provider name, got nil")
	}
	if !strings.Contains(err.Error(), "my-provider") {
		t.Errorf("error should mention duplicate name 'my-provider': %v", err)
	}
}

// TestProviderConfig_EmptyBaseURLRejected verifies that missing base_url is rejected.
func TestProviderConfig_EmptyBaseURLRejected(t *testing.T) {
	yaml := `
server:
  listen: ":8080"
auth:
  method: local
  session_ttl: 24h
providers:
  - name: nourl
    base_url: ""
    driver: openai-compatible
`
	path := writeAppCfgFile(t, yaml)

	_, err := config.LoadApp(path)
	if err == nil {
		t.Fatal("expected validation error for empty base_url, got nil")
	}
	if !strings.Contains(err.Error(), "base_url") {
		t.Errorf("error should mention 'base_url': %v", err)
	}
}

// TestProviderConfig_InvalidURLRejected verifies that an invalid URL scheme is rejected.
func TestProviderConfig_InvalidURLRejected(t *testing.T) {
	yaml := `
server:
  listen: ":8080"
auth:
  method: local
  session_ttl: 24h
providers:
  - name: badurl
    base_url: "ftp://not-supported"
    driver: openai-compatible
`
	path := writeAppCfgFile(t, yaml)

	_, err := config.LoadApp(path)
	if err == nil {
		t.Fatal("expected validation error for invalid URL scheme, got nil")
	}
}

// TestProviderConfig_EmptyDriverRejected verifies that missing driver is rejected.
func TestProviderConfig_EmptyDriverRejected(t *testing.T) {
	yaml := `
server:
  listen: ":8080"
auth:
  method: local
  session_ttl: 24h
providers:
  - name: nodriver
    base_url: "http://localhost:7442"
    driver: ""
`
	path := writeAppCfgFile(t, yaml)

	_, err := config.LoadApp(path)
	if err == nil {
		t.Fatal("expected validation error for empty driver, got nil")
	}
	if !strings.Contains(err.Error(), "driver") {
		t.Errorf("error should mention 'driver': %v", err)
	}
}

// TestProviderConfig_NoProvidersKey verifies that an app config without providers key loads with empty slice.
func TestProviderConfig_NoProvidersKey(t *testing.T) {
	yaml := `
server:
  listen: ":8080"
auth:
  method: local
  session_ttl: 24h
`
	path := writeAppCfgFile(t, yaml)

	cfg, err := config.LoadApp(path)
	if err != nil {
		t.Fatalf("LoadApp without providers: %v", err)
	}
	if len(cfg.Providers) != 0 {
		t.Errorf("expected empty Providers, got %d", len(cfg.Providers))
	}
}

// TestProviderConfig_LegacyOllamaInstancesMigration verifies that legacy ollama_instances
// automatically map to Provider records during unmarshal.
func TestProviderConfig_LegacyOllamaInstancesMigration(t *testing.T) {
	yaml := `
server:
  listen: ":8080"
auth:
  method: local
  session_ttl: 24h
ollama_instances:
  - name: legacy-ollama
    base_url: http://localhost:11434
    api_key: "legacy-token"
`
	path := writeAppCfgFile(t, yaml)

	cfg, err := config.LoadApp(path)
	if err != nil {
		t.Fatalf("LoadApp with legacy ollama_instances: %v", err)
	}

	if len(cfg.Providers) != 1 {
		t.Fatalf("expected 1 migrated Provider, got %d", len(cfg.Providers))
	}
	p := cfg.Providers[0]
	if p.Name != "legacy-ollama" {
		t.Errorf("migrated provider Name: got %q, want %q", p.Name, "legacy-ollama")
	}
	if p.BaseURL != "http://localhost:11434" {
		t.Errorf("migrated provider BaseURL: got %q, want %q", p.BaseURL, "http://localhost:11434")
	}
	if p.Driver != "openai-compatible" {
		t.Errorf("migrated provider Driver: got %q, want %q", p.Driver, "openai-compatible")
	}
	if p.APIKey != "legacy-token" {
		t.Errorf("migrated provider APIKey: got %q, want %q", p.APIKey, "legacy-token")
	}
}

// TestProviderConfig_AgentWithProvider verifies that an agent with driver=openai-compatible,
// provider, and model parses and validates correctly.
func TestProviderConfig_AgentWithProvider(t *testing.T) {
	root := makeProjectRoot(t, `
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
  - name: llama-analyst
    role: [analyst]
    driver: openai-compatible
    provider: local-llama
    model: gemma-4-26B-A4B-it-UD-Q8_K_XL
    max_tool_iterations: 15
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Llama Analyst
      email: llama@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"
`)

	cfg, err := config.LoadProject(root)
	if err != nil {
		t.Fatalf("LoadProject: unexpected error: %v", err)
	}

	agp := findAgentConfig(cfg, "llama-analyst")
	if agp == nil {
		t.Fatal("llama-analyst not found in loaded config")
	}
	ag := *agp
	if ag.Driver != "openai-compatible" {
		t.Errorf("Driver: got %q, want %q", ag.Driver, "openai-compatible")
	}
	if ag.Provider != "local-llama" {
		t.Errorf("Provider: got %q, want %q", ag.Provider, "local-llama")
	}
	if ag.Model != "gemma-4-26B-A4B-it-UD-Q8_K_XL" {
		t.Errorf("Model: got %q, want %q", ag.Model, "gemma-4-26B-A4B-it-UD-Q8_K_XL")
	}
	if ag.MaxToolIterations != 15 {
		t.Errorf("MaxToolIterations: got %d, want 15", ag.MaxToolIterations)
	}
}

// TestProviderConfig_AgentMissingProviderRejected verifies that openai-compatible agent
// missing provider fails validation.
func TestProviderConfig_AgentMissingProviderRejected(t *testing.T) {
	root := makeProjectRoot(t, `
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
  - name: no-provider-agent
    role: [analyst]
    driver: openai-compatible
    model: gemma-4-26B
    git_identity:
      name: Bad Agent
      email: bad@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"
`)

	_, err := config.LoadProject(root)
	if err == nil {
		t.Fatal("expected validation error for openai-compatible agent without provider, got nil")
	}
	if !strings.Contains(err.Error(), "provider") {
		t.Errorf("error should mention 'provider': %v", err)
	}
}

// TestProviderConfig_AgentMissingModelRejected verifies that openai-compatible agent
// missing model fails validation.
func TestProviderConfig_AgentMissingModelRejected(t *testing.T) {
	root := makeProjectRoot(t, `
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
  - name: no-model-agent
    role: [analyst]
    driver: openai-compatible
    provider: local-llama
    git_identity:
      name: Bad Agent
      email: bad@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"
`)

	_, err := config.LoadProject(root)
	if err == nil {
		t.Fatal("expected validation error for openai-compatible agent without model, got nil")
	}
	if !strings.Contains(err.Error(), "model") {
		t.Errorf("error should mention 'model': %v", err)
	}
}
