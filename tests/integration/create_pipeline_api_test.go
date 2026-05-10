// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 3 — Create Pipeline API endpoint
//
// Tests for POST /api/p/{project}/devops/pipelines covering:
//   - 201 Created for valid input; response slug matches; file created on disk
//   - 409 Conflict when slug already exists
//   - 400 Bad Request for invalid YAML
//   - 400 Bad Request for valid YAML missing required fields (name / steps)
//   - 400 Bad Request for an invalid slug (uppercase, spaces)
//   - 401 Unauthorized when not authenticated
//   - 403 Forbidden for users without product-owner or devops role
//   - Created pipeline appears in GET /devops/pipelines listing

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

const (
	// validPipelineDef is a minimal well-formed pipeline definition.
	validPipelineDef = "name: CI\ntype: build\nsteps:\n  - name: test\n    command: go test ./...\n"
)

// TestCreatePipeline_Success verifies that POST /devops/pipelines with valid
// input returns 201 Created, the response body contains the correct slug, and
// the pipeline file is written to devops/{slug}.yaml on disk.
func TestCreatePipeline_Success(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	body := map[string]any{
		"slug":       "ci",
		"definition": validPipelineDef,
	}
	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines", body)
	requireStatus(t, resp, http.StatusCreated)
	data := readJSON(t, resp)

	if slug, _ := data["slug"].(string); slug != "ci" {
		t.Errorf("response slug = %q, want %q", slug, "ci")
	}

	// The handler writes to devops/{slug}.yaml (at the project root level).
	destPath := filepath.Join(env.projectRoot, "devops", "ci.yaml")
	if _, err := os.Stat(destPath); err != nil {
		t.Fatalf("expected pipeline file at %s, got error: %v", destPath, err)
	}
}

// TestCreatePipeline_DuplicateSlug verifies that creating a pipeline with a
// slug that already exists returns 409 Conflict.
func TestCreatePipeline_DuplicateSlug(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	body := map[string]any{
		"slug":       "my-pipe",
		"definition": "name: My Pipe\ntype: build\nsteps:\n  - name: run\n    command: echo ok\n",
	}

	// First create — must succeed.
	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines", body)
	requireStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	// Second create with same slug — must conflict.
	resp2 := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines", body)
	requireStatus(t, resp2, http.StatusConflict)
	data := readJSON(t, resp2)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "conflict" {
		t.Errorf("expected error code 'conflict', got %q", code)
	}
}

// TestCreatePipeline_InvalidYAML verifies that posting malformed YAML returns
// 400 Bad Request with a descriptive error body.
func TestCreatePipeline_InvalidYAML(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	body := map[string]any{
		"slug":       "bad",
		"definition": "not: valid: yaml: [",
	}
	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines", body)
	requireStatus(t, resp, http.StatusBadRequest)
	data := readJSON(t, resp)

	if _, hasErr := data["error"]; !hasErr {
		t.Error("expected 'error' field in 400 response body")
	}
}

// TestCreatePipeline_MissingRequiredFields verifies that posting valid YAML
// that lacks the required 'name' or 'steps' fields returns 400 Bad Request.
func TestCreatePipeline_MissingRequiredFields(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	cases := []struct {
		name string
		body map[string]any
	}{
		{
			name: "missing name",
			body: map[string]any{
				"slug":       "no-name",
				"definition": "type: build\nsteps:\n  - name: step\n    command: echo ok\n",
			},
		},
		{
			name: "missing steps",
			body: map[string]any{
				"slug":       "no-steps",
				"definition": "name: No Steps\ntype: build\n",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines", tc.body)
			requireStatus(t, resp, http.StatusBadRequest)
			resp.Body.Close()
		})
	}
}

// TestCreatePipeline_InvalidSlug verifies that posting a slug containing
// uppercase letters or spaces returns 400 Bad Request.
func TestCreatePipeline_InvalidSlug(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	cases := []struct {
		name string
		slug string
	}{
		{"uppercase letters", "My-Pipeline"},
		{"spaces", "my pipeline"},
		{"leading hyphen", "-my-pipe"},
		{"trailing hyphen", "my-pipe-"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]any{
				"slug":       tc.slug,
				"definition": validPipelineDef,
			}
			resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines", body)
			requireStatus(t, resp, http.StatusBadRequest)
			resp.Body.Close()
		})
	}
}

// TestCreatePipeline_Unauthorized verifies that an unauthenticated POST to the
// create endpoint returns 401 Unauthorized.
func TestCreatePipeline_Unauthorized(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	// No login — send a raw POST without session cookies.

	body := `{"slug":"unauth","definition":"` + validPipelineDef + `"}`
	resp, err := http.Post(devopsListURL(env), "application/json", stringReader(body))
	if err != nil {
		t.Fatalf("http.Post failed: %v", err)
	}
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusUnauthorized)
}

// TestCreatePipeline_Forbidden verifies that a user without the product-owner
// or devops role (qa@test.local has only the 'qa' role) receives 403 Forbidden.
func TestCreatePipeline_Forbidden(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("qa@test.local", "qa-pass-123") // qa role only

	body := map[string]any{
		"slug":       "forbidden-pipe",
		"definition": validPipelineDef,
	}
	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines", body)
	requireStatus(t, resp, http.StatusForbidden)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "forbidden" {
		t.Errorf("expected error code 'forbidden', got %q", code)
	}
}

// TestCreatePipeline_AppearsInList verifies that a newly created pipeline
// appears in the response of GET /devops/pipelines.
func TestCreatePipeline_AppearsInList(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	body := map[string]any{
		"slug":       "listed-pipe",
		"definition": "name: Listed Pipe\ntype: build\nsteps:\n  - name: run\n    command: echo ok\n",
	}
	createResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines", body)
	requireStatus(t, createResp, http.StatusCreated)
	createResp.Body.Close()

	listResp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines", nil)
	requireStatus(t, listResp, http.StatusOK)
	listData := readJSON(t, listResp)

	pipelines, _ := listData["pipelines"].([]any)
	for _, pl := range pipelines {
		p, _ := pl.(map[string]any)
		if slug, _ := p["slug"].(string); slug == "listed-pipe" {
			return // found — test passes
		}
	}
	t.Errorf("created pipeline 'listed-pipe' not found in GET /devops/pipelines response; got %d pipeline(s)", len(pipelines))
}
