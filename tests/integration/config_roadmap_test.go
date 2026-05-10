// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// TestConfigRoadmap_* — Milestone 1: Backend config tests
//
// Verifies that the project configuration correctly parses, validates, and serves
// the roadmap.default_period_mode setting via GET /api/p/{project}/config/roadmap.

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
)

// ── Milestone 1.1: endpoint returns the configured default_period_mode ────────

// TestConfigRoadmap_ReturnsPeriodModeFromConfig verifies that
// GET /api/p/{project}/config/roadmap returns the roadmap.default_period_mode
// that matches the value written in lifecycle/config.yaml.
func TestConfigRoadmap_ReturnsPeriodModeFromConfig(t *testing.T) {
	cfgYAML := defaultCfgYAML + `
roadmap:
  default_period_mode: quarter
`
	env := newTestEnvWithCfgYAML(t, nil, cfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/config/roadmap", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	roadmap, ok := body["roadmap"].(map[string]any)
	if !ok {
		t.Fatalf("roadmap key missing or wrong type in response: %v", body)
	}
	got, _ := roadmap["default_period_mode"].(string)
	if got != "quarter" {
		t.Errorf("default_period_mode: want %q, got %q", "quarter", got)
	}
}

// TestConfigRoadmap_AllValidModesRoundtrip verifies that each accepted
// default_period_mode value is returned verbatim by the endpoint.
func TestConfigRoadmap_AllValidModesRoundtrip(t *testing.T) {
	validModes := []string{"autoscale", "month", "quarter", "half-year", "year"}

	for _, mode := range validModes {
		mode := mode
		t.Run(mode, func(t *testing.T) {
			cfgYAML := defaultCfgYAML + `
roadmap:
  default_period_mode: ` + mode + `
`
			env := newTestEnvWithCfgYAML(t, nil, cfgYAML)
			env.login("admin@test.local", "admin-pass-123")

			resp := env.doRequest("GET", "/api/p/testproject/config/roadmap", nil)
			requireStatus(t, resp, http.StatusOK)
			body := readJSON(t, resp)

			roadmap, ok := body["roadmap"].(map[string]any)
			if !ok {
				t.Fatalf("roadmap key missing or wrong type: %v", body)
			}
			got, _ := roadmap["default_period_mode"].(string)
			if got != mode {
				t.Errorf("mode %q: want %q, got %q", mode, mode, got)
			}
		})
	}
}

// ── Milestone 1.2: default when no roadmap section ────────────────────────────

// TestConfigRoadmap_DefaultsToAutoscaleWhenNoRoadmapSection verifies that
// GET /api/p/{project}/config/roadmap returns "autoscale" when lifecycle/config.yaml
// has no roadmap section at all.
func TestConfigRoadmap_DefaultsToAutoscaleWhenNoRoadmapSection(t *testing.T) {
	// defaultCfgYAML has no roadmap section.
	env := newTestEnvWithCfgYAML(t, nil, defaultCfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/config/roadmap", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	roadmap, ok := body["roadmap"].(map[string]any)
	if !ok {
		t.Fatalf("roadmap key missing or wrong type: %v", body)
	}
	got, _ := roadmap["default_period_mode"].(string)
	if got != "autoscale" {
		t.Errorf("default_period_mode: want %q, got %q", "autoscale", got)
	}
}

// TestConfigRoadmap_DefaultsToAutoscaleWhenRoadmapSectionEmpty verifies that
// when the roadmap section exists but default_period_mode is absent, "autoscale"
// is still returned.
func TestConfigRoadmap_DefaultsToAutoscaleWhenRoadmapSectionEmpty(t *testing.T) {
	cfgYAML := defaultCfgYAML + `
roadmap: {}
`
	env := newTestEnvWithCfgYAML(t, nil, cfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", "/api/p/testproject/config/roadmap", nil)
	requireStatus(t, resp, http.StatusOK)
	body := readJSON(t, resp)

	roadmap, ok := body["roadmap"].(map[string]any)
	if !ok {
		t.Fatalf("roadmap key missing or wrong type: %v", body)
	}
	got, _ := roadmap["default_period_mode"].(string)
	if got != "autoscale" {
		t.Errorf("default_period_mode: want %q, got %q", "autoscale", got)
	}
}

// ── Milestone 1.3: invalid value causes a config load error ──────────────────

// TestConfigRoadmap_InvalidModeRejectedByLoadProject is a unit test verifying
// that config.LoadProject returns an error when lifecycle/config.yaml contains
// an invalid roadmap.default_period_mode value (e.g., "weekly").
func TestConfigRoadmap_InvalidModeRejectedByLoadProject(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "lifecycle"), 0o755); err != nil {
		t.Fatal(err)
	}

	invalidCfg := defaultCfgYAML + `
roadmap:
  default_period_mode: weekly
`
	cfgPath := filepath.Join(root, "lifecycle", "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(invalidCfg), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := config.LoadProject(root)
	if err == nil {
		t.Fatal("expected config.LoadProject to return an error for invalid default_period_mode, got nil")
	}
}

// TestConfigRoadmap_OtherInvalidModesRejected verifies several additional
// invalid values are each rejected by config.LoadProject.
func TestConfigRoadmap_OtherInvalidModesRejected(t *testing.T) {
	invalidModes := []string{"weekly", "daily", "biannual", "auto", "AUTOSCALE", ""}

	for _, mode := range invalidModes {
		if mode == "" {
			// Empty string is treated as "autoscale" by validateProject; skip.
			continue
		}
		mode := mode
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "lifecycle"), 0o755); err != nil {
				t.Fatal(err)
			}

			cfgContent := defaultCfgYAML + `
roadmap:
  default_period_mode: ` + mode + `
`
			cfgPath := filepath.Join(root, "lifecycle", "config.yaml")
			if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
				t.Fatal(err)
			}

			_, err := config.LoadProject(root)
			if err == nil {
				t.Errorf("mode %q: expected config load error, got nil", mode)
			}
		})
	}
}

// TestConfigRoadmap_RequiresAuth verifies that the roadmap config endpoint
// returns 401 for unauthenticated requests.
func TestConfigRoadmap_RequiresAuth(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, nil, defaultCfgYAML)
	// Do NOT call env.login — no session cookies.

	resp := env.doRequest("GET", "/api/p/testproject/config/roadmap", nil)
	requireStatus(t, resp, http.StatusUnauthorized)
}
