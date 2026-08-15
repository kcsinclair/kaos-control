// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/config"
)

const configFixture = `git:
  default_branch: main
  branch_template: requirement/{slug}
roles:
  - product-owner
  - analyst
  - backend-developer
  - frontend-developer
  - test-developer
  - qa
stages:
  - name: ideas
    dir: ideas
  - name: requirements
    dir: requirements
users:
  - email: keith@sinclair.org.au
    linux_user: keith
    roles:
      - product-owner
kanban:
  columns:
    - name: Backlog
      statuses:
        - draft
  uncategorised: true
  card_fields:
    - title
    - type
agents:
  - name: requirements-analyst
    role:
      - analyst
    driver: claude-mediated
    model: opus
    allowed_write_paths:
      - lifecycle/requirements
      - lifecycle/ideas
    prompt_templates:
      analyst: |
        You are an analyst. Read the idea artifact at {target_path}.
        Produce a detailed requirement artifact in lifecycle/requirements/.
  - name: planning-analyst
    role:
      - analyst
    driver: claude-code-cli
    model: opus
    allowed_write_paths:
      - lifecycle/backend-plans
      - lifecycle/frontend-plans
      - lifecycle/test-plans
    prompt_templates:
      analyst: |
        You are an analyst. Read the requirement at {target_path}.
        Produce THREE plan artifacts.
  - name: backend-developer
    role:
      - backend-developer
    driver: claude-code-cli
    model: sonnet
    allowed_write_paths:
      - internal
      - cmd
    prompt_templates:
      backend-developer: |
        You are a backend developer. Read the backend plan at {target_path}
        and implement it milestone by milestone in Go.
  - name: frontend-developer
    role:
      - frontend-developer
    driver: claude-code-cli
    model: sonnet
    allowed_write_paths:
      - web/src
    prompt_templates:
      frontend-developer: |
        You are a frontend developer. Read the frontend plan at {target_path}.
  - name: test-developer
    role:
      - test-developer
    driver: claude-code-cli
    model: sonnet
    allowed_write_paths:
      - tests
    prompt_templates:
      test-developer: |
        You are a test developer. Read the test plan at {target_path}.
  - name: qa
    role:
      - qa
    driver: gemini-cli
    model: gemini-2.5-flash
    allowed_write_paths:
      - lifecycle/defects
    prompt_templates:
      qa: |
        You are a QA agent. Given the artifact at {target_path}, run tests.
  - name: my-custom-agent
    role:
      - product-owner
    driver: inline
    model: claude-sonnet-4-6
    allowed_write_paths:
      - lifecycle/ideas
    prompt_templates:
      idea-capture: |
        A hand-written custom agent prompt that must never be touched.
`

func writeConfigFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "lifecycle", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(configFixture), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPatchAgentConfig_GoVue_SetsWritePathsAndCommands(t *testing.T) {
	root := writeConfigFixture(t)

	res, err := PatchAgentConfig(root, goVueModel())
	if err != nil {
		t.Fatalf("PatchAgentConfig: %v", err)
	}
	if !res.Changed {
		t.Fatal("expected Changed=true on first patch")
	}
	if len(res.Disabled) != 0 {
		t.Errorf("expected no disabled agents for go-vue, got %v", res.Disabled)
	}

	raw, err := os.ReadFile(filepath.Join(root, "lifecycle", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)

	for _, want := range []string{"internal", "cmd", "lifecycle/backend-plans", "lifecycle/architecture/decisions"} {
		if !strings.Contains(content, want) {
			t.Errorf("patched config missing %q:\n%s", want, content)
		}
	}
	if !strings.Contains(content, "go build ./...") {
		t.Errorf("patched config missing backend build token:\n%s", content)
	}

	// Re-loads cleanly (FR-9).
	if _, err := config.LoadProject(root); err != nil {
		t.Fatalf("patched config does not reload: %v", err)
	}
}

func TestPatchAgentConfig_Idempotent(t *testing.T) {
	root := writeConfigFixture(t)

	if _, err := PatchAgentConfig(root, goVueModel()); err != nil {
		t.Fatalf("first PatchAgentConfig: %v", err)
	}
	res2, err := PatchAgentConfig(root, goVueModel())
	if err != nil {
		t.Fatalf("second PatchAgentConfig: %v", err)
	}
	if res2.Changed {
		t.Error("expected second patch with the same model to be a no-op")
	}
}

func TestPatchAgentConfig_UnrelatedBlocksUntouched(t *testing.T) {
	root := writeConfigFixture(t)

	if _, err := PatchAgentConfig(root, goVueModel()); err != nil {
		t.Fatalf("PatchAgentConfig: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(root, "lifecycle", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)

	if !strings.Contains(content, "A hand-written custom agent prompt that must never be touched.") {
		t.Error("custom agent prompt_templates content was modified")
	}
	if !strings.Contains(content, "email: keith@sinclair.org.au") || !strings.Contains(content, "linux_user: keith") {
		t.Error("users: block was modified")
	}
	if !strings.Contains(content, "uncategorised: true") || !strings.Contains(content, "name: Backlog") {
		t.Error("kanban: block was modified")
	}

	// The custom agent's own allowed_write_paths must be untouched — it is
	// not one of the six standard agents.
	idx := strings.Index(content, "my-custom-agent")
	if idx < 0 {
		t.Fatal("custom agent missing entirely")
	}
	customBlock := content[idx:]
	if !strings.Contains(customBlock, "lifecycle/ideas") {
		t.Error("custom agent allowed_write_paths was modified")
	}
}

func staticHTMLJSModel() DirectiveModel {
	return DirectiveModel{
		ProjectName: "marketing-site",
		Stack: architecture.StackProfile{
			Run: "npx serve htdocs",
			Roles: map[string]architecture.RoleProfile{
				"backend-developer": {Required: boolPtr(false)},
				"frontend-developer": {
					WritePaths: []string{"htdocs", "website-assets"},
					Lint:       "npx html-validate htdocs",
					Test:       "pnpm test",
				},
				"test-developer": {
					WritePaths: []string{"tests"},
					Test:       "pnpm test",
				},
			},
		},
	}
}

func boolPtr(b bool) *bool { return &b }

func TestPatchAgentConfig_StaticSite_DisablesBackendDeveloper(t *testing.T) {
	root := writeConfigFixture(t)

	res, err := PatchAgentConfig(root, staticHTMLJSModel())
	if err != nil {
		t.Fatalf("PatchAgentConfig: %v", err)
	}
	if len(res.Disabled) != 1 || res.Disabled[0] != "backend-developer" {
		t.Fatalf("expected backend-developer disabled, got %v", res.Disabled)
	}

	raw, err := os.ReadFile(filepath.Join(root, "lifecycle", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	idx := strings.Index(content, "name: backend-developer")
	if idx < 0 {
		t.Fatal("backend-developer agent missing")
	}
	// enabled: false should appear before the next agent entry starts.
	next := strings.Index(content[idx:], "\n  - name:")
	block := content[idx:]
	if next > 0 {
		block = content[idx : idx+next]
	}
	if !strings.Contains(block, "enabled: false") {
		t.Errorf("expected enabled: false in backend-developer block:\n%s", block)
	}

	if _, err := config.LoadProject(root); err != nil {
		t.Fatalf("patched config does not reload: %v", err)
	}
}
