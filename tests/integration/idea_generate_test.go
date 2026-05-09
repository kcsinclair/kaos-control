// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// generateAPI posts to POST /api/p/testproject/ideas/generate.
// Pass artifactType="" to omit the optional type field (defaults to "idea" on the server).
func generateAPI(env *testEnv, input string, artifactType string) *http.Response {
	body := map[string]any{
		"input": input,
	}
	if artifactType != "" {
		body["type"] = artifactType
	}
	return env.doRequest("POST", "/api/p/testproject/ideas/generate", body)
}

// ---
// Milestone 1 – Generate Endpoint: Happy Path (Idea)
// ---

// TestIdeaGenerate_HappyPath verifies a single-request generate call returns
// a well-formed proposal with all required fields and writes nothing to disk.
func TestIdeaGenerate_HappyPath(t *testing.T) {
	skipIfNoAPIKey(t)

	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/existing-feature.md",
			content: makeArtifact("Existing Feature", "idea", "draft", "existing-feature", "",
				"An existing idea.", "ui", "backend", "auth"),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	input := "We should add a dark mode toggle to the settings page so users can switch between light and dark themes based on their preference"
	resp := generateAPI(env, input, "")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	// Assert required top-level fields are non-empty.
	slug, _ := data["slug"].(string)
	if slug == "" {
		t.Error("response missing non-empty 'slug'")
	}
	title, _ := data["title"].(string)
	if title == "" {
		t.Error("response missing non-empty 'title'")
	}
	body, _ := data["body"].(string)
	if body == "" {
		t.Error("response missing non-empty 'body'")
	}
	targetDir, _ := data["target_dir"].(string)
	if targetDir == "" {
		t.Error("response missing non-empty 'target_dir'")
	}
	if _, ok := data["labels"]; !ok {
		t.Error("response missing 'labels' field")
	}
	if _, ok := data["frontmatter"]; !ok {
		t.Error("response missing 'frontmatter' field")
	}

	// Assert slug format.
	if slug != "" && !slugPattern.MatchString(slug) {
		t.Errorf("slug %q does not match ^[a-z0-9][a-z0-9\\-]*[a-z0-9]$|^[a-z0-9]$", slug)
	}

	// Assert target_dir.
	if targetDir != "lifecycle/ideas" {
		t.Errorf("expected target_dir 'lifecycle/ideas', got %q", targetDir)
	}

	// Assert frontmatter fields.
	fm, _ := data["frontmatter"].(map[string]any)
	if fm == nil {
		t.Fatal("frontmatter is nil – cannot check fields")
	}
	if typ, _ := fm["type"].(string); typ != "idea" {
		t.Errorf("frontmatter.type: want 'idea', got %q", typ)
	}
	if fmStatus, _ := fm["status"].(string); fmStatus != "draft" {
		t.Errorf("frontmatter.status: want 'draft', got %q", fmStatus)
	}
	if lineage, _ := fm["lineage"].(string); lineage != slug {
		t.Errorf("frontmatter.lineage %q != slug %q", lineage, slug)
	}

	// Assert body contains a level-1 heading.
	if !strings.Contains(body, "# ") {
		preview := body
		if len(preview) > 120 {
			preview = preview[:120]
		}
		t.Errorf("body does not contain a level-1 heading ('# '), got: %q", preview)
	}

	// Assert no file was written to disk.
	if slug != "" {
		diskPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", slug+".md")
		if _, err := os.Stat(diskPath); !os.IsNotExist(err) {
			t.Errorf("generate endpoint wrote a file to disk at %s (should be preview-only)", diskPath)
		}
	}
}

// ---
// Milestone 2 – Generate Endpoint: Input Validation
// ---

// TestIdeaGenerate_TooShort verifies that a very short input (1 word) returns 400.
func TestIdeaGenerate_TooShort(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := generateAPI(env, "hi", "")
	requireStatus(t, resp, 400)
	data := readJSON(t, resp)

	if _, ok := data["error"]; !ok {
		t.Error("400 response should contain an 'error' field with a user-facing message")
	}
}

// TestIdeaGenerate_EmptyInput verifies that an empty input string returns 400.
func TestIdeaGenerate_EmptyInput(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := generateAPI(env, "", "")
	requireStatus(t, resp, 400)
	data := readJSON(t, resp)

	if _, ok := data["error"]; !ok {
		t.Error("400 response should contain an 'error' field")
	}
}

// TestIdeaGenerate_MissingInput verifies that a request body with no input field returns 400.
func TestIdeaGenerate_MissingInput(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/ideas/generate", map[string]any{})
	requireStatus(t, resp, 400)
	data := readJSON(t, resp)

	if _, ok := data["error"]; !ok {
		t.Error("400 response should contain an 'error' field")
	}
}

// TestIdeaGenerate_FewWords verifies that input below the 5-word minimum returns 400.
func TestIdeaGenerate_FewWords(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := generateAPI(env, "fix bug", "")
	requireStatus(t, resp, 400)
	data := readJSON(t, resp)

	if _, ok := data["error"]; !ok {
		t.Error("400 response should contain an 'error' field")
	}
}

// ---
// Milestone 3 – Generate Endpoint: Defect Mode
// ---

// TestIdeaGenerate_DefectMode verifies that type="defect" produces a defect-shaped
// proposal with correct frontmatter, target directory, and structured body.
func TestIdeaGenerate_DefectMode(t *testing.T) {
	skipIfNoAPIKey(t)

	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	input := "When I click the save button on the artifact editor the page refreshes and all unsaved changes are lost. Expected: changes are saved without page refresh."
	resp := generateAPI(env, input, "defect")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	// Assert frontmatter.type is "defect".
	fm, _ := data["frontmatter"].(map[string]any)
	if fm == nil {
		t.Fatal("frontmatter is nil")
	}
	if typ, _ := fm["type"].(string); typ != "defect" {
		t.Errorf("frontmatter.type: want 'defect', got %q", typ)
	}

	// Assert target directory is lifecycle/defects.
	targetDir, _ := data["target_dir"].(string)
	if targetDir != "lifecycle/defects" {
		t.Errorf("expected target_dir 'lifecycle/defects', got %q", targetDir)
	}

	// Assert body contains at least one standard defect section heading.
	body, _ := data["body"].(string)
	defectSections := []string{
		"Reproduction Steps", "Reproduce",
		"Expected Behaviour", "Expected Behavior", "Expected",
		"Actual Behaviour", "Actual Behavior", "Actual",
	}
	var foundSection bool
	for _, section := range defectSections {
		if strings.Contains(body, section) {
			foundSection = true
			break
		}
	}
	if !foundSection {
		preview := body
		if len(preview) > 200 {
			preview = preview[:200]
		}
		t.Errorf("defect body does not contain expected structured sections; body preview: %q", preview)
	}

	// Assert "defect" label is present.
	labels, _ := data["labels"].([]any)
	var hasDefectLabel bool
	for _, l := range labels {
		if s, _ := l.(string); s == "defect" {
			hasDefectLabel = true
			break
		}
	}
	if !hasDefectLabel {
		t.Errorf("expected 'defect' label in response, got %v", labels)
	}
}

// ---
// Milestone 4 – Slug Collision Detection
// ---

// TestIdeaGenerate_SlugCollision verifies that when a file with the proposed slug
// already exists on disk, the generate endpoint returns a disambiguated slug.
func TestIdeaGenerate_SlugCollision(t *testing.T) {
	skipIfNoAPIKey(t)

	// Pre-seed lifecycle/ideas/dark-mode.md so its slug is already taken.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/dark-mode.md",
			content: makeArtifact("Dark Mode", "idea", "draft", "dark-mode", "", "An existing dark mode idea."),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Input strongly implies "dark-mode" as the slug.
	input := "We need a dark mode feature for the settings page to toggle between light and dark themes so users can reduce eye strain in low light environments"
	resp := generateAPI(env, input, "")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	slug, _ := data["slug"].(string)
	if slug == "" {
		t.Fatal("response missing slug")
	}

	// Slug must differ from the existing "dark-mode".
	if slug == "dark-mode" {
		t.Error("slug collision not resolved: slug is still 'dark-mode', which already exists on disk")
	}

	// The disambiguated slug must still be a valid slug.
	if !slugPattern.MatchString(slug) {
		t.Errorf("disambiguated slug %q does not match the valid slug pattern", slug)
	}

	// The pre-existing file must be intact.
	existingPath := filepath.Join(env.projectRoot, "lifecycle", "ideas", "dark-mode.md")
	if _, err := os.Stat(existingPath); os.IsNotExist(err) {
		t.Error("pre-existing lifecycle/ideas/dark-mode.md was removed during collision resolution")
	}
}

// ---
// Milestone 5 – Label Vocabulary Constraint
// ---

// TestIdeaGenerate_LabelVocabulary verifies that all labels returned by the
// generate endpoint are members of the project's existing label vocabulary.
func TestIdeaGenerate_LabelVocabulary(t *testing.T) {
	skipIfNoAPIKey(t)

	// Seed artifacts with a known label corpus so the vocabulary is non-trivial.
	seeds := []seedArtifact{
		{
			relPath: "lifecycle/ideas/auth-feature.md",
			content: makeArtifact("Auth Feature", "idea", "draft", "auth-feature", "",
				"Body.", "auth", "backend", "api"),
		},
		{
			relPath: "lifecycle/ideas/ui-improvement.md",
			content: makeArtifact("UI Improvement", "idea", "draft", "ui-improvement", "",
				"Body.", "ui", "frontend", "usability"),
		},
	}
	env := newTestEnv(t, seeds)
	env.login("admin@test.local", "admin-pass-123")

	// Fetch the project's label vocabulary from the index.
	labelsResp := env.doRequest("GET", "/api/p/testproject/labels", nil)
	requireStatus(t, labelsResp, 200)
	labelsData := readJSON(t, labelsResp)
	rawLabels, _ := labelsData["labels"].([]any)
	if len(rawLabels) == 0 {
		t.Fatal("project has no labels after seeding – cannot exercise label constraint")
	}
	vocab := make(map[string]bool, len(rawLabels))
	for _, l := range rawLabels {
		if s, ok := l.(string); ok {
			vocab[s] = true
		}
	}

	// Input that might tempt the LLM to invent labels outside the vocabulary.
	input := "I want to add blockchain-based AI verification for all lifecycle artifacts so they can be cryptographically audited by external parties without accessing the live system"
	resp := generateAPI(env, input, "")
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	if _, ok := data["labels"]; !ok {
		t.Error("response missing 'labels' field")
	}

	labels, _ := data["labels"].([]any)
	for _, l := range labels {
		label, _ := l.(string)
		if label == "" {
			continue
		}
		if !vocab[label] {
			t.Errorf("label %q is not in the project vocabulary %v", label, rawLabels)
		}
	}
}

// ---
// Milestone 6 – Accept Flow: End-to-End Write
// ---

// TestIdeaGenerate_AcceptFlow verifies the full generate → write → index cycle.
func TestIdeaGenerate_AcceptFlow(t *testing.T) {
	skipIfNoAPIKey(t)

	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Step 1: generate a proposal.
	input := "Add a keyboard shortcut overlay so users can discover and invoke all application commands without touching the mouse, with shortcuts configurable per user profile and accessible via a help modal triggered by pressing the question mark key"
	genResp := generateAPI(env, input, "")
	requireStatus(t, genResp, 200)
	proposal := readJSON(t, genResp)

	slug, _ := proposal["slug"].(string)
	if slug == "" {
		t.Fatal("generate response missing slug")
	}
	body, _ := proposal["body"].(string)
	fm, _ := proposal["frontmatter"].(map[string]any)

	// Step 2: write the artifact via POST /artifacts.
	createResp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage":       "ideas",
		"slug":        slug,
		"frontmatter": fm,
		"body":        body,
	})
	requireStatus(t, createResp, 201)
	createData := readJSON(t, createResp)

	path, _ := createData["path"].(string)
	if path == "" {
		t.Fatal("create response missing path")
	}
	expectedPath := fmt.Sprintf("lifecycle/ideas/%s.md", slug)
	if path != expectedPath {
		t.Errorf("create response path %q != expected %q", path, expectedPath)
	}

	// Step 3: verify file exists on disk with correct content.
	absPath := filepath.Join(env.projectRoot, path)
	fileContent, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("artifact file not found on disk at %s: %v", absPath, err)
	}
	text := string(fileContent)

	if !strings.Contains(text, "type: idea") {
		t.Error("artifact file missing 'type: idea'")
	}
	if !strings.Contains(text, "status: draft") {
		t.Error("artifact file missing 'status: draft'")
	}
	if !strings.Contains(text, "lineage: "+slug) {
		t.Errorf("artifact file missing 'lineage: %s'", slug)
	}
	if !strings.Contains(text, body) {
		t.Error("artifact file does not contain the generated body")
	}

	// Step 4: verify the index has the artifact. Poll briefly for the watcher.
	var found bool
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		getResp := env.doRequest("GET", "/api/p/testproject/artifacts/"+path, nil)
		if getResp.StatusCode == 200 {
			getResp.Body.Close()
			found = true
			break
		}
		getResp.Body.Close()
		time.Sleep(100 * time.Millisecond)
	}
	if !found {
		t.Errorf("artifact at %q not found in index after creation", path)
	}
}

// ---
// Milestone 7 – Non-Regression: Existing Endpoints
// ---

// TestIdeaConverse_StillWorks verifies the conversational idea-capture endpoint
// is unaffected by the addition of the generate endpoint.
func TestIdeaConverse_StillWorks(t *testing.T) {
	skipIfNoAPIKey(t)

	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/ideas/converse", map[string]any{
		"message": "I have an idea for improving search so users can find artifacts faster using full-text queries across all lifecycle stages",
	})
	requireStatus(t, resp, 200)
	data := readJSON(t, resp)

	if sessionID, _ := data["session_id"].(string); sessionID == "" {
		t.Error("converse response missing 'session_id'")
	}
	if _, ok := data["reply"]; !ok {
		t.Error("converse response missing 'reply' field")
	}
}

// TestCreateArtifact_StillWorks verifies that POST /artifacts creates an artifact
// directly without involving the generate endpoint.
func TestCreateArtifact_StillWorks(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("POST", "/api/p/testproject/artifacts", map[string]any{
		"stage": "ideas",
		"slug":  "manual-test-idea",
		"frontmatter": map[string]any{
			"title":   "Manual Test",
			"type":    "idea",
			"status":  "draft",
			"lineage": "manual-test-idea",
		},
		"body": "# Manual Test\n\nTest body.",
	})
	requireStatus(t, resp, 201)

	data := readJSON(t, resp)
	path, _ := data["path"].(string)
	if path == "" {
		t.Fatal("create response missing path")
	}

	absPath := filepath.Join(env.projectRoot, path)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		t.Errorf("artifact file not found on disk at %s after direct creation", absPath)
	}
}

// ---
// Milestone 8 – Unauthenticated Access Denied
// ---

// TestIdeaGenerate_Unauthenticated verifies that the generate endpoint returns
// 401 or 403 when called without a session cookie.
func TestIdeaGenerate_Unauthenticated(t *testing.T) {
	env := newTestEnv(t, nil)
	// Deliberately NOT calling env.login — no session cookies.

	reqBody := `{"input":"We should add a dark mode toggle to the settings page so users can switch themes"}`
	resp, err := http.Post(
		env.baseURL+"/api/p/testproject/ideas/generate",
		"application/json",
		bytes.NewReader([]byte(reqBody)),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// CSRF middleware runs before auth: without a CSRF token the request gets
	// 403 (csrf_missing). Either 401 or 403 confirms the endpoint is protected.
	if resp.StatusCode != 401 && resp.StatusCode != 403 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401 or 403 for unauthenticated request, got %d: %s", resp.StatusCode, b)
	}
}
