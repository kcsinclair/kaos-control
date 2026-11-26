// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/project"
)

// TestResolveIdeaCaptureConfig_NoProviderRoutesToCLIDefault verifies an
// inline agent with no provider set yields ModelConfig.Provider == nil
// (FR-8: byte-identical behaviour to before this feature).
func TestResolveIdeaCaptureConfig_NoProviderRoutesToCLIDefault(t *testing.T) {
	p := &project.Project{
		Cfg: &config.Project{
			Agents: []config.AgentConfig{{
				Name:            "idea-capture",
				Driver:          "inline",
				Model:           "claude-sonnet-4-6",
				PromptTemplates: map[string]string{"idea-capture": "system prompt"},
			}},
		},
	}

	cfg, err := resolveIdeaCaptureConfig(p, "idea-capture")
	if err != nil {
		t.Fatalf("resolveIdeaCaptureConfig: %v", err)
	}
	if cfg.Provider != nil {
		t.Errorf("Provider = %+v, want nil", cfg.Provider)
	}
	if cfg.Model != "claude-sonnet-4-6" {
		t.Errorf("Model = %q, want claude-sonnet-4-6", cfg.Model)
	}
}

// TestResolveIdeaCaptureConfig_ProviderResolvesFromAppConfig verifies an
// inline agent with provider set resolves the named provider from the
// project's app-level provider snapshot onto ModelConfig.Provider (FR-4).
func TestResolveIdeaCaptureConfig_ProviderResolvesFromAppConfig(t *testing.T) {
	p := &project.Project{
		Cfg: &config.Project{
			Agents: []config.AgentConfig{{
				Name:            "idea-capture",
				Driver:          "inline",
				Provider:        "local-llama",
				Model:           "gemma",
				PromptTemplates: map[string]string{"idea-capture": "system prompt"},
			}},
		},
		Providers: []config.Provider{
			{Name: "local-llama", BaseURL: "http://localhost:8080", Driver: "openai-compatible", APIKey: "sekret"},
			{Name: "other", BaseURL: "http://example.com"},
		},
	}

	cfg, err := resolveIdeaCaptureConfig(p, "idea-capture")
	if err != nil {
		t.Fatalf("resolveIdeaCaptureConfig: %v", err)
	}
	if cfg.Provider == nil {
		t.Fatal("Provider = nil, want resolved provider")
	}
	if cfg.Provider.Name != "local-llama" || cfg.Provider.BaseURL != "http://localhost:8080" || cfg.Provider.APIKey != "sekret" {
		t.Errorf("Provider = %+v, want local-llama with matching fields", cfg.Provider)
	}
}

// TestResolveIdeaCaptureConfig_AllFourTemplateKeysResolveProvider verifies
// the single resolveIdeaCaptureConfig function wires provider identity for
// all four inline template keys (idea-capture, idea-generate,
// defect-generate via idea-capture agent; doc-generate via docs-capture).
func TestResolveIdeaCaptureConfig_AllFourTemplateKeysResolveProvider(t *testing.T) {
	p := &project.Project{
		Cfg: &config.Project{
			Agents: []config.AgentConfig{
				{
					Name:     "idea-capture",
					Driver:   "inline",
					Provider: "local-llama",
					Model:    "gemma",
					PromptTemplates: map[string]string{
						"idea-capture":    "p1",
						"idea-generate":   "p2",
						"defect-generate": "p3",
					},
				},
				{
					Name:            "docs-capture",
					Driver:          "inline",
					Provider:        "local-llama",
					Model:           "gemma",
					PromptTemplates: map[string]string{"doc-generate": "p4"},
				},
			},
		},
		Providers: []config.Provider{
			{Name: "local-llama", BaseURL: "http://localhost:8080", Driver: "openai-compatible"},
		},
	}

	for _, key := range []string{"idea-capture", "idea-generate", "defect-generate", "doc-generate"} {
		cfg, err := resolveIdeaCaptureConfig(p, key)
		if err != nil {
			t.Fatalf("resolveIdeaCaptureConfig(%q): %v", key, err)
		}
		if cfg.Provider == nil || cfg.Provider.Name != "local-llama" {
			t.Errorf("resolveIdeaCaptureConfig(%q).Provider = %+v, want local-llama", key, cfg.Provider)
		}
	}
}
