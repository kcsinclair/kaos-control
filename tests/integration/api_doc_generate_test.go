// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Integration tests for the POST /ideas/generate endpoint with type="doc".
//
// Test plan: lifecycle/test-plans/tech-writer-agent-5-test.md §Milestone 3

import (
	"testing"
)

// docGenerateCfgYAML extends defaultCfgYAML with the `docs` stage and
// tech-writer role so the generate tests have a realistic project config.
const docGenerateCfgYAML = `git:
  default_branch: main
  branch_template: "ticket/{slug}"

roles:
  - product-owner
  - analyst
  - backend-developer
  - frontend-developer
  - test-developer
  - qa
  - reviewer
  - approver
  - tech-writer

stages:
  - {name: ideas, dir: ideas}
  - {name: requirements, dir: requirements}
  - {name: backend-plans, dir: backend-plans}
  - {name: frontend-plans, dir: frontend-plans}
  - {name: test-plans, dir: test-plans}
  - {name: tests, dir: tests}
  - {name: prototypes, dir: prototypes}
  - {name: releases, dir: releases}
  - {name: sprints, dir: sprints}
  - {name: defects, dir: defects}
  - {name: docs, dir: docs}

users:
  - email: admin@test.local
    roles: [product-owner, analyst, reviewer, approver]
  - email: dev@test.local
    roles: [backend-developer, frontend-developer, test-developer, tech-writer]
  - email: qa@test.local
    roles: [qa]

required_plans:
  ticket: [plan-backend, plan-frontend, plan-test]
  epic: []
`

// ── Milestone 3, TC1: standalone doc generation ───────────────────────────────

// TestDocGenerate_StandaloneDoc verifies that POSTing with type="doc" returns a
// well-formed proposal with target_dir "lifecycle/docs" and doc-shaped frontmatter.
// Requires ANTHROPIC_API_KEY — skipped in CI without a key.
func TestDocGenerate_StandaloneDoc(t *testing.T) {
	skipIfNoAPIKey(t)

	env := newTestEnvWithCfgYAML(t, nil, docGenerateCfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	input := "Document the installation process for new users who need to set up the application from scratch on a fresh Linux server including all prerequisite steps"
	resp := generateAPI(env, input, "doc")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	// target_dir must be lifecycle/docs.
	if targetDir, _ := data["target_dir"].(string); targetDir != "lifecycle/docs" {
		t.Errorf("expected target_dir 'lifecycle/docs', got %q", targetDir)
	}

	// slug must be non-empty and valid.
	slug, _ := data["slug"].(string)
	if slug == "" {
		t.Error("response missing non-empty 'slug'")
	}
	if slug != "" && !slugPattern.MatchString(slug) {
		t.Errorf("slug %q does not match valid slug pattern", slug)
	}

	// frontmatter must have type=doc and status=draft.
	fm, _ := data["frontmatter"].(map[string]any)
	if fm == nil {
		t.Fatal("frontmatter is nil")
	}
	if typ, _ := fm["type"].(string); typ != "doc" {
		t.Errorf("frontmatter.type: want 'doc', got %q", typ)
	}
	if status, _ := fm["status"].(string); status != "draft" {
		t.Errorf("frontmatter.status: want 'draft', got %q", status)
	}
}

// ── Milestone 3, TC2: source-linked doc generation ───────────────────────────

// TestDocGenerate_SourceLinkedDoc verifies that providing source_lineage and
// source_path causes the response frontmatter to carry lineage and parent fields
// matching those values.
// Requires ANTHROPIC_API_KEY — skipped in CI without a key.
func TestDocGenerate_SourceLinkedDoc(t *testing.T) {
	skipIfNoAPIKey(t)

	seeds := []seedArtifact{
		{
			relPath: "lifecycle/requirements/login-2.md",
			content: makeArtifact("Login Feature", "requirement", "done", "login",
				"", "Requirement body for the login feature."),
		},
	}
	env := newTestEnvWithCfgYAML(t, seeds, docGenerateCfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	input := "Document the login feature end-to-end covering authentication flow error messages and recovery steps for users who forget their credentials"

	resp := env.doRequest("POST", "/api/p/testproject/ideas/generate", map[string]any{
		"input":          input,
		"type":           "doc",
		"source_lineage": "login",
		"source_path":    "lifecycle/requirements/login-2.md",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	fm, _ := data["frontmatter"].(map[string]any)
	if fm == nil {
		t.Fatal("frontmatter is nil")
	}

	// lineage must match the source lineage.
	if lineage, _ := fm["lineage"].(string); lineage != "login" {
		t.Errorf("frontmatter.lineage: want 'login', got %q", lineage)
	}

	// parent must match the source path.
	if parent, _ := fm["parent"].(string); parent != "lifecycle/requirements/login-2.md" {
		t.Errorf("frontmatter.parent: want 'lifecycle/requirements/login-2.md', got %q", parent)
	}
}

// ── Milestone 3, TC3: input too short ─────────────────────────────────────────

// TestDocGenerate_InputTooShort asserts that a very short input returns 400 with
// an error field. Does not require an API key.
func TestDocGenerate_InputTooShort(t *testing.T) {
	env := newTestEnvWithCfgYAML(t, nil, docGenerateCfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/ideas/generate", map[string]any{
		"input": "docs",
		"type":  "doc",
	})
	requireStatus(t, resp, 400)
	data := readJSON(t, resp)

	if _, ok := data["error"]; !ok {
		t.Error("400 response should contain an 'error' field")
	}
}

// ── Milestone 3, TC4: existing types unaffected ───────────────────────────────

// TestDocGenerate_IdeaTypeUnaffected verifies that type="idea" still produces
// target_dir="lifecycle/ideas" after the doc route was added.
// Requires ANTHROPIC_API_KEY — skipped in CI without a key.
func TestDocGenerate_IdeaTypeUnaffected(t *testing.T) {
	skipIfNoAPIKey(t)

	env := newTestEnvWithCfgYAML(t, nil, docGenerateCfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	input := "Add a dark mode toggle to the settings page so users can switch between light and dark themes based on their viewing environment"
	resp := generateAPI(env, input, "idea")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	if targetDir, _ := data["target_dir"].(string); targetDir != "lifecycle/ideas" {
		t.Errorf("type=idea: expected target_dir 'lifecycle/ideas', got %q", targetDir)
	}

	fm, _ := data["frontmatter"].(map[string]any)
	if fm == nil {
		t.Fatal("frontmatter is nil")
	}
	if typ, _ := fm["type"].(string); typ != "idea" {
		t.Errorf("type=idea: frontmatter.type: want 'idea', got %q", typ)
	}
}

// TestDocGenerate_DefectTypeUnaffected verifies that type="defect" still
// produces target_dir="lifecycle/defects" after the doc route was added.
// Requires ANTHROPIC_API_KEY — skipped in CI without a key.
func TestDocGenerate_DefectTypeUnaffected(t *testing.T) {
	skipIfNoAPIKey(t)

	env := newTestEnvWithCfgYAML(t, nil, docGenerateCfgYAML)
	env.login("admin@test.local", "admin-pass-123")

	input := "When I click the save button on the artifact editor the page refreshes and all unsaved changes are lost — expected behavior is to save without page refresh"
	resp := generateAPI(env, input, "defect")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	if targetDir, _ := data["target_dir"].(string); targetDir != "lifecycle/defects" {
		t.Errorf("type=defect: expected target_dir 'lifecycle/defects', got %q", targetDir)
	}

	fm, _ := data["frontmatter"].(map[string]any)
	if fm == nil {
		t.Fatal("frontmatter is nil")
	}
	if typ, _ := fm["type"].(string); typ != "defect" {
		t.Errorf("type=defect: frontmatter.type: want 'defect', got %q", typ)
	}
}
