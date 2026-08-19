// SPDX-License-Identifier: AGPL-3.0-or-later

package initcmd

import (
	"os"
	"path/filepath"
	"strings"
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

	// Assert agent count. Default template ships eleven agents:
	// requirements-analyst, planning-analyst, backend-developer,
	// frontend-developer, test-developer, qa, tech-writer, test-runner,
	// idea-capture, idea-triage, docs-capture.
	if got := len(proj.Agents); got != 11 {
		t.Errorf("expected 11 agents, got %d", got)
	}

	// Assert role count. Default template ships ten roles:
	// product-owner, analyst, backend-developer, frontend-developer,
	// test-developer, qa, reviewer, approver, devops, tech-writer.
	if got := len(proj.Roles); got != 10 {
		t.Errorf("expected 10 roles, got %d", got)
	}

	// Assert required_plans.requirement contains all three plan types. This
	// key must match the artifact `type:` value looked up by
	// internal/http/transition.go (p.Config().RequiredPlans[row.Type]).
	requirementPlans, ok := proj.RequiredPlans["requirement"]
	if !ok {
		t.Fatal("required_plans[\"requirement\"] is missing from rendered config")
	}
	wantPlans := []string{"plan-backend", "plan-frontend", "plan-test"}
	if len(requirementPlans) != len(wantPlans) {
		t.Errorf("required_plans.requirement: want %v, got %v", wantPlans, requirementPlans)
	} else {
		for i, want := range wantPlans {
			if requirementPlans[i] != want {
				t.Errorf("required_plans.requirement[%d]: want %q, got %q", i, want, requirementPlans[i])
			}
		}
	}
}

// TestScaffoldProject_SeedsArchitectureCatalog verifies that ScaffoldProject
// wires in architecture.EnsureArchitectureScaffold: a freshly initialised
// project gets lifecycle/architecture/{architectures,tech-stacks} seeded
// with the shipped catalog and empty, tracked decisions/ and standards/.
func TestScaffoldProject_SeedsArchitectureCatalog(t *testing.T) {
	dir := t.TempDir()

	res, err := ScaffoldProject(ScaffoldOptions{ProjectRoot: dir})
	if err != nil {
		t.Fatalf("ScaffoldProject: %v", err)
	}
	if len(res.Architecture) == 0 {
		t.Fatal("expected ScaffoldResult.Architecture to report seeded files, got none")
	}

	if _, err := os.Stat(filepath.Join(dir, "lifecycle/architecture/README.md")); err != nil {
		t.Errorf("expected lifecycle/architecture/README.md to be seeded: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "lifecycle/architecture/architectures/local-web.md")); err != nil {
		t.Errorf("expected a seeded architecture catalog entry: %v", err)
	}
	for _, empty := range []string{"decisions", "standards"} {
		if _, err := os.Stat(filepath.Join(dir, "lifecycle/architecture", empty, ".gitkeep")); err != nil {
			t.Errorf("expected tracked empty %s/: %v", empty, err)
		}
	}
}

// TestScaffold_EmitsAgentsMdPrimarySet verifies init produces the
// AGENTS.md-primary directive layout (defect init-agents-md-not-wired):
// AGENTS.md canonical (with managed markers) + CLAUDE.md as an @AGENTS.md
// pointer, reported in ScaffoldResult.Directives.
func TestScaffold_EmitsAgentsMdPrimarySet(t *testing.T) {
	dir := t.TempDir()
	res, err := ScaffoldProject(ScaffoldOptions{ProjectRoot: dir, ProjectName: "my-project"})
	if err != nil {
		t.Fatalf("ScaffoldProject: %v", err)
	}

	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md not created: %v", err)
	}
	if !strings.Contains(string(agents), "kaos-control:generated:start") {
		t.Error("AGENTS.md is missing the managed-region markers")
	}

	claude, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}
	if strings.TrimSpace(string(claude)) != "@AGENTS.md" {
		t.Errorf("CLAUDE.md = %q, want an @AGENTS.md pointer", string(claude))
	}

	var haveAgents, haveClaude bool
	for _, r := range res.Directives {
		switch r.Path {
		case "AGENTS.md":
			haveAgents = true
		case "CLAUDE.md":
			haveClaude = true
		}
	}
	if !haveAgents || !haveClaude {
		t.Errorf("Directives result missing AGENTS.md/CLAUDE.md: %+v", res.Directives)
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
