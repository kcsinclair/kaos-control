// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/initcmd"
)

// Test plan: lifecycle/test-plans/defect-generate-missing-template-4-test.md
// §Milestone 4 — Backend integration: fresh project generates a defect.
//
// Reproduces the exact scenario from GitHub issue #16: "New Defect →
// Generate" must never hard-error with
// `idea-capture agent has no template "defect-generate"`, on a fresh
// project (no agent configured), on a project whose idea-capture agent is
// present but has had the defect-generate key stripped, and in general.

const rawTemplateErrorString = `has no template`

// defectGenerateInput is a >=5-word bug description satisfying the
// generate endpoint's minimum-input-length validation.
const defectGenerateInput = "When I click save the page refreshes and unsaved changes are lost unexpectedly"

// assertDefectProposalShape asserts the standard shape of a successful
// defect-generate proposal response: the three required body sections and
// the mandatory "defect" label.
func assertDefectProposalShape(t *testing.T, data map[string]any) {
	t.Helper()

	fm, _ := data["frontmatter"].(map[string]any)
	if fm == nil {
		t.Fatal("frontmatter is nil")
	}
	if typ, _ := fm["type"].(string); typ != "defect" {
		t.Errorf("frontmatter.type: want 'defect', got %q", typ)
	}

	targetDir, _ := data["target_dir"].(string)
	if targetDir != "lifecycle/defects" {
		t.Errorf("target_dir: want 'lifecycle/defects', got %q", targetDir)
	}

	body, _ := data["body"].(string)
	for _, section := range []string{"## Reproduction Steps", "## Expected Behaviour", "## Actual Behaviour"} {
		if !strings.Contains(body, section) {
			t.Errorf("body missing required section %q; body: %s", section, body)
		}
	}

	labels, _ := data["labels"].([]any)
	var hasDefectLabel bool
	for _, l := range labels {
		if s, _ := l.(string); s == "defect" {
			hasDefectLabel = true
		}
	}
	if !hasDefectLabel {
		t.Errorf("expected 'defect' label in response, got %v", labels)
	}
}

// TestDefectGenerate_FreshProjectFromInitTemplate reproduces GitHub issue #16
// end to end: a project scaffolded exactly as `kaos-control init` would
// produce it must support "New Defect → Generate" out of the box.
func TestDefectGenerate_FreshProjectFromInitTemplate(t *testing.T) {
	skipIfNoAPIKey(t)

	scaffoldDir := t.TempDir()
	if err := initcmd.Run([]string{scaffoldDir}); err != nil {
		t.Fatalf("initcmd.Run failed: %v", err)
	}
	// Sanity-check the scaffolded config loads and repairs cleanly before
	// reusing it to drive a live server — a failure here means the fixture
	// itself is broken, not the generate endpoint.
	if _, err := config.LoadProject(scaffoldDir); err != nil {
		t.Fatalf("scaffolded config.yaml failed to load: %v", err)
	}
	cfgYAML, err := os.ReadFile(filepath.Join(scaffoldDir, "lifecycle", "config.yaml"))
	if err != nil {
		t.Fatalf("reading scaffolded config.yaml: %v", err)
	}

	env := newTestEnvWithCfgYAML(t, nil, string(cfgYAML))
	env.login("admin@test.local", "admin-pass-123")

	resp := generateAPI(env, defectGenerateInput, "defect")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	assertDefectProposalShape(t, data)
}

// TestDefectGenerate_MissingDefectGenerateKeyFallsBack verifies that a
// project whose idea-capture agent is present but has had the
// defect-generate prompt_templates key stripped still returns 200 (the
// resolver falls back to the built-in default template) rather than 500,
// and that the raw internal error string never reaches the client.
func TestDefectGenerate_MissingDefectGenerateKeyFallsBack(t *testing.T) {
	skipIfNoAPIKey(t)

	cfgYAML := defaultCfgYAML + `
agents:
  - name: idea-capture
    role: [product-owner]
    driver: inline
    model: claude-sonnet-4-6
    allowed_write_paths: [lifecycle/ideas]
    prompt_templates:
      idea-capture: "You are an idea-capture assistant."
`
	env := newTestEnvWithCfgYAML(t, nil, cfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	resp := generateAPI(env, defectGenerateInput, "defect")
	raw, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatalf("dumping response: %v", err)
	}
	if strings.Contains(string(raw), rawTemplateErrorString) {
		t.Fatalf("response leaked the raw internal error string %q:\n%s", rawTemplateErrorString, raw)
	}
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	assertDefectProposalShape(t, data)
}

// TestDefectGenerate_NoAgentConfiguredFallsBack verifies that a project with
// no idea-capture agent configured at all (the newTestEnv default config)
// still returns 200 for defect generation via the built-in default
// template, and never leaks the raw resolver error string.
func TestDefectGenerate_NoAgentConfiguredFallsBack(t *testing.T) {
	skipIfNoAPIKey(t)

	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := generateAPI(env, defectGenerateInput, "defect")
	raw, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Fatalf("dumping response: %v", err)
	}
	if strings.Contains(string(raw), rawTemplateErrorString) {
		t.Fatalf("response leaked the raw internal error string %q:\n%s", rawTemplateErrorString, raw)
	}
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)
	assertDefectProposalShape(t, data)
}
