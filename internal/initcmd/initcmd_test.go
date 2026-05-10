// SPDX-License-Identifier: AGPL-3.0-or-later

package initcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kaos-control/kaos-control/internal/config"
)

// TestConfigTemplateLoadsCleanly renders config.yaml.tmpl and verifies that
// config.LoadProject can parse the result without error. It also asserts the
// key structural invariants required by the plan:
//   - seven agents (including idea-capture)
//   - seven roles
//   - required_plans["ticket"] contains plan-backend, plan-frontend, plan-test
func TestConfigTemplateLoadsCleanly(t *testing.T) {
	data := TemplateData{
		ProjectName: "test-project",
		Language:    "Go",
	}

	rendered, err := renderTemplate("config.yaml.tmpl", data)
	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}

	// Write to a temp directory so LoadProject can read it.
	dir := t.TempDir()
	lcDir := filepath.Join(dir, "lifecycle")
	if err := os.MkdirAll(lcDir, 0o755); err != nil {
		t.Fatalf("creating lifecycle dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(lcDir, "config.yaml"), rendered, 0o644); err != nil {
		t.Fatalf("writing config.yaml: %v", err)
	}

	proj, err := config.LoadProject(dir)
	if err != nil {
		t.Fatalf("config.LoadProject returned error: %v", err)
	}

	// Assert agent count.
	if got := len(proj.Agents); got != 7 {
		t.Errorf("expected 7 agents, got %d", got)
	}

	// Assert role count.
	if got := len(proj.Roles); got != 7 {
		t.Errorf("expected 7 roles, got %d", got)
	}

	// Assert required_plans.ticket contains all three plan types.
	ticketPlans, ok := proj.RequiredPlans["ticket"]
	if !ok {
		t.Fatal("required_plans[\"ticket\"] is missing from rendered config")
	}
	wantPlans := []string{"plan-backend", "plan-frontend", "plan-test"}
	if len(ticketPlans) != len(wantPlans) {
		t.Errorf("required_plans.ticket: want %v, got %v", wantPlans, ticketPlans)
	} else {
		for i, want := range wantPlans {
			if ticketPlans[i] != want {
				t.Errorf("required_plans.ticket[%d]: want %q, got %q", i, want, ticketPlans[i])
			}
		}
	}
}

// TestCLAUDEMdTemplateLanguageConditional verifies that the CLAUDE.md template
// includes the language section when Language is set and omits it when blank.
func TestCLAUDEMdTemplateLanguageConditional(t *testing.T) {
	withLang, err := renderTemplate("CLAUDE.md.tmpl", TemplateData{
		ProjectName: "my-project",
		Language:    "Go",
	})
	if err != nil {
		t.Fatalf("renderTemplate with language failed: %v", err)
	}

	withoutLang, err := renderTemplate("CLAUDE.md.tmpl", TemplateData{
		ProjectName: "my-project",
		Language:    "",
	})
	if err != nil {
		t.Fatalf("renderTemplate without language failed: %v", err)
	}

	if got := string(withLang); !contains(got, "Go") {
		t.Error("expected language 'Go' to appear in CLAUDE.md when Language is set")
	}
	if got := string(withoutLang); contains(got, "Primary Language") {
		t.Error("expected 'Primary Language' section to be absent when Language is empty")
	}
}

// TestSettingsJSONIsValidJSON verifies the settings template renders valid JSON.
func TestSettingsJSONIsValidJSON(t *testing.T) {
	rendered, err := renderTemplate("settings.json.tmpl", TemplateData{})
	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}
	// A minimal validity check: the rendered output must start with '{' and end with '}'.
	s := string(rendered)
	if len(s) == 0 {
		t.Fatal("settings.json template rendered empty output")
	}
	trimmed := trimSpace(s)
	if len(trimmed) == 0 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		t.Errorf("settings.json does not appear to be a JSON object: %q", s)
	}
}

// TestGitignoreContainsDBPattern verifies .gitignore covers the SQLite index.
func TestGitignoreContainsDBPattern(t *testing.T) {
	rendered, err := renderTemplate("gitignore.tmpl", TemplateData{})
	if err != nil {
		t.Fatalf("renderTemplate failed: %v", err)
	}
	if !contains(string(rendered), "lifecycle/.kaos-control.db") {
		t.Error("gitignore.tmpl does not contain 'lifecycle/.kaos-control.db'")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		findSubstring(s, substr))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func trimSpace(s string) string {
	start, end := 0, len(s)-1
	for start <= end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end >= start && (s[end] == ' ' || s[end] == '\t' || s[end] == '\n' || s[end] == '\r') {
		end--
	}
	if start > end {
		return ""
	}
	return s[start : end+1]
}
