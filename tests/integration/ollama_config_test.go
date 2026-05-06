//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/config"
)

// ── Milestone 2 — App Config Tests (Instance CRUD) ────────────────────────────

// validAppCfgWithInstances is a minimal valid app config YAML with ollama_instances.
const validAppCfgWithInstances = `
server:
  listen: ":8080"
auth:
  method: local
  session_ttl: 24h
ollama_instances:
  - name: local
    base_url: http://localhost:11434
  - name: remote
    base_url: https://ollama.example.com
    api_key: "secret-key"
`

// TestOllamaConfig_LoadWithInstances verifies that a YAML config with
// ollama_instances parses correctly and all fields are populated.
func TestOllamaConfig_LoadWithInstances(t *testing.T) {
	path := writeAppCfgFile(t, validAppCfgWithInstances)

	cfg, err := config.LoadApp(path)
	if err != nil {
		t.Fatalf("LoadApp: unexpected error: %v", err)
	}

	if len(cfg.OllamaInstances) != 2 {
		t.Fatalf("expected 2 ollama instances, got %d", len(cfg.OllamaInstances))
	}

	local := cfg.OllamaInstances[0]
	if local.Name != "local" {
		t.Errorf("instances[0].Name: got %q, want %q", local.Name, "local")
	}
	if local.BaseURL != "http://localhost:11434" {
		t.Errorf("instances[0].BaseURL: got %q, want %q", local.BaseURL, "http://localhost:11434")
	}
	if local.APIKey != "" {
		t.Errorf("instances[0].APIKey: expected empty, got %q", local.APIKey)
	}

	remote := cfg.OllamaInstances[1]
	if remote.Name != "remote" {
		t.Errorf("instances[1].Name: got %q, want %q", remote.Name, "remote")
	}
	if remote.APIKey != "secret-key" {
		t.Errorf("instances[1].APIKey: got %q, want %q", remote.APIKey, "secret-key")
	}
}

// TestOllamaConfig_RoundTrip verifies that Load → Save → Load produces identical config.
func TestOllamaConfig_RoundTrip(t *testing.T) {
	path := writeAppCfgFile(t, validAppCfgWithInstances)

	cfg1, err := config.LoadApp(path)
	if err != nil {
		t.Fatalf("first LoadApp: %v", err)
	}

	// Save back to the same path.
	if err := config.SaveApp(path, *cfg1); err != nil {
		t.Fatalf("SaveApp: %v", err)
	}

	// Load again and compare instances.
	cfg2, err := config.LoadApp(path)
	if err != nil {
		t.Fatalf("second LoadApp after save: %v", err)
	}

	if len(cfg2.OllamaInstances) != len(cfg1.OllamaInstances) {
		t.Fatalf("round-trip: instance count changed: %d → %d",
			len(cfg1.OllamaInstances), len(cfg2.OllamaInstances))
	}
	for i, inst := range cfg1.OllamaInstances {
		got := cfg2.OllamaInstances[i]
		if inst.Name != got.Name || inst.BaseURL != got.BaseURL || inst.APIKey != got.APIKey {
			t.Errorf("round-trip mismatch at index %d: want %+v, got %+v", i, inst, got)
		}
	}
}

// TestOllamaConfig_DuplicateNameRejected verifies that a config with two instances
// sharing a name returns a validation error.
func TestOllamaConfig_DuplicateNameRejected(t *testing.T) {
	yaml := `
server:
  listen: ":8080"
auth:
  method: local
  session_ttl: 24h
ollama_instances:
  - name: dup
    base_url: http://localhost:11434
  - name: dup
    base_url: http://localhost:11435
`
	path := writeAppCfgFile(t, yaml)

	_, err := config.LoadApp(path)
	if err == nil {
		t.Fatal("expected validation error for duplicate instance name, got nil")
	}
	if !strings.Contains(err.Error(), "dup") {
		t.Errorf("error should mention duplicate name 'dup': %v", err)
	}
}

// TestOllamaConfig_EmptyBaseURLRejected verifies that an instance with no base_url
// returns a validation error mentioning the field and constraint.
func TestOllamaConfig_EmptyBaseURLRejected(t *testing.T) {
	yaml := `
server:
  listen: ":8080"
auth:
  method: local
  session_ttl: 24h
ollama_instances:
  - name: nourl
    base_url: ""
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

// TestOllamaConfig_InvalidURLRejected verifies that a non-http/https base_url is rejected.
func TestOllamaConfig_InvalidURLRejected(t *testing.T) {
	yaml := `
server:
  listen: ":8080"
auth:
  method: local
  session_ttl: 24h
ollama_instances:
  - name: badurl
    base_url: "ftp://not-valid"
`
	path := writeAppCfgFile(t, yaml)

	_, err := config.LoadApp(path)
	if err == nil {
		t.Fatal("expected validation error for ftp URL, got nil")
	}
}

// TestOllamaConfig_NoInstancesKey verifies that an app config without ollama_instances
// loads successfully (empty list, not an error).
func TestOllamaConfig_NoInstancesKey(t *testing.T) {
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
		t.Fatalf("LoadApp without ollama_instances: %v", err)
	}
	if len(cfg.OllamaInstances) != 0 {
		t.Errorf("expected empty OllamaInstances, got %d", len(cfg.OllamaInstances))
	}
}

// TestOllamaConfig_AgentWithOllamaDriver verifies that an AgentConfig with driver=ollama,
// ollama_instance, and model parses and validates correctly.
func TestOllamaConfig_AgentWithOllamaDriver(t *testing.T) {
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
  - name: ollama-agent
    role: [analyst]
    driver: ollama
    model: gemma2:2b
    ollama_instance: local-ollama
    ollama_endpoint: chat
    allowed_write_paths: [lifecycle/requirements]
    git_identity:
      name: Ollama Agent
      email: ollama@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"
`)

	cfg, err := config.LoadProject(root)
	if err != nil {
		t.Fatalf("LoadProject: unexpected error: %v", err)
	}

	if len(cfg.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(cfg.Agents))
	}
	ag := cfg.Agents[0]
	if ag.Driver != "ollama" {
		t.Errorf("Driver: got %q, want %q", ag.Driver, "ollama")
	}
	if ag.Model != "gemma2:2b" {
		t.Errorf("Model: got %q, want %q", ag.Model, "gemma2:2b")
	}
	if ag.OllamaInstanceName != "local-ollama" {
		t.Errorf("OllamaInstanceName: got %q, want %q", ag.OllamaInstanceName, "local-ollama")
	}
	if ag.OllamaEndpoint != "chat" {
		t.Errorf("OllamaEndpoint: got %q, want %q", ag.OllamaEndpoint, "chat")
	}
}

// TestOllamaConfig_AgentValidation_MissingInstance verifies that an agent with
// driver=ollama but no ollama_instance fails validation with a specific error.
func TestOllamaConfig_AgentValidation_MissingInstance(t *testing.T) {
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
  - name: bad-ollama-agent
    role: [analyst]
    driver: ollama
    model: gemma2:2b
    git_identity:
      name: Bad Ollama Agent
      email: bad@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"
`)

	_, err := config.LoadProject(root)
	if err == nil {
		t.Fatal("expected validation error for driver=ollama without ollama_instance, got nil")
	}
	if !strings.Contains(err.Error(), "ollama_instance") {
		t.Errorf("error should mention 'ollama_instance': %v", err)
	}
}

// TestOllamaConfig_AgentValidation_MissingModel verifies that an agent with
// driver=ollama but no model fails validation.
func TestOllamaConfig_AgentValidation_MissingModel(t *testing.T) {
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
    driver: ollama
    ollama_instance: local-ollama
    git_identity:
      name: No Model Agent
      email: nomodel@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"
`)

	_, err := config.LoadProject(root)
	if err == nil {
		t.Fatal("expected validation error for driver=ollama without model, got nil")
	}
	if !strings.Contains(err.Error(), "model") {
		t.Errorf("error should mention 'model': %v", err)
	}
}

// TestOllamaConfig_OllamaEndpointDefaultsToChat verifies that when ollama_endpoint
// is omitted, it defaults to "chat" after validation.
func TestOllamaConfig_OllamaEndpointDefaultsToChat(t *testing.T) {
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
  - name: ollama-no-endpoint
    role: [analyst]
    driver: ollama
    model: gemma2:2b
    ollama_instance: local-ollama
    git_identity:
      name: No Endpoint Agent
      email: noendpoint@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"
`)

	cfg, err := config.LoadProject(root)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if cfg.Agents[0].OllamaEndpoint != "chat" {
		t.Errorf("OllamaEndpoint default: got %q, want %q", cfg.Agents[0].OllamaEndpoint, "chat")
	}
}

// TestOllamaConfig_InvalidOllamaEndpointRejected verifies that an unsupported
// ollama_endpoint value is rejected.
func TestOllamaConfig_InvalidOllamaEndpointRejected(t *testing.T) {
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
  - name: bad-endpoint-agent
    role: [analyst]
    driver: ollama
    model: gemma2:2b
    ollama_instance: local-ollama
    ollama_endpoint: stream
    git_identity:
      name: Bad Endpoint Agent
      email: bad@test.local
    prompt_templates:
      analyst: "Analyse {target_path}"
`)

	_, err := config.LoadProject(root)
	if err == nil {
		t.Fatal("expected validation error for invalid ollama_endpoint, got nil")
	}
	if !strings.Contains(err.Error(), "ollama_endpoint") {
		t.Errorf("error should mention 'ollama_endpoint': %v", err)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// writeAppCfgFile writes a YAML string to a temp file and returns its path.
func writeAppCfgFile(t *testing.T, yaml string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// makeProjectRoot creates a minimal project directory structure with the given
// lifecycle/config.yaml content and returns the project root path.
func makeProjectRoot(t *testing.T, cfgYAML string) string {
	t.Helper()
	root := t.TempDir()
	lcDir := filepath.Join(root, "lifecycle")
	if err := os.MkdirAll(lcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lcDir, "config.yaml"), []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	// Ensure the default timeout default applies (referenced by validateProject internally).
	_ = time.Second // import kept alive
	return root
}
