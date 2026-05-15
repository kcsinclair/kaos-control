// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 1 — GET Single-Pipeline endpoint tests
//
// Tests for GET /api/p/{project}/devops/pipelines/{slug} covering:
//   - 200 OK; body is the verbatim YAML file content; Content-Type is text/yaml
//   - 404 Not Found for a non-existent slug
//   - 400 Bad Request for a slug containing path-traversal-like characters
//   - 401 Unauthorized when not authenticated
//   - 403 Forbidden for users without the product-owner or devops role

import (
	"io"
	"net/http"
	"testing"
)

// devopsPipelineURL returns the URL for a single pipeline identified by slug.
// It is visible to all integration tests in the package.
func devopsPipelineURL(env *testEnv, slug string) string {
	return env.baseURL + "/api/p/testproject/devops/pipelines/" + slug
}

// TestGetPipeline_Success verifies that GET /devops/pipelines/{slug} for an
// existing pipeline returns 200 OK with the verbatim YAML content and the
// correct Content-Type header.
func TestGetPipeline_Success(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines/quick-pass", nil)
	requireStatus(t, resp, http.StatusOK)

	ct := resp.Header.Get("Content-Type")
	if ct != "text/yaml" {
		t.Errorf("Content-Type = %q, want %q", ct, "text/yaml")
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}
	if string(body) != pipelineQuickPass {
		t.Errorf("response body mismatch:\ngot:  %q\nwant: %q", string(body), pipelineQuickPass)
	}
}

// TestGetPipeline_NotFound verifies that requesting a non-existent slug returns
// 404 Not Found with an appropriate error code.
func TestGetPipeline_NotFound(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines/nonexistent", nil)
	requireStatus(t, resp, http.StatusNotFound)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "not_found" {
		t.Errorf("expected error code 'not_found', got %q", code)
	}
}

// TestGetPipeline_InvalidSlug verifies that a slug that does not match the
// allowed pattern (lowercase alphanumeric with interior hyphens) is rejected
// with 400 Bad Request. The slug "..etc..passwd" mimics a path traversal
// attempt: it starts with '.' which fails the pipelineSlugRe validation regex.
func TestGetPipeline_InvalidSlug(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines/..etc..passwd", nil)
	requireStatus(t, resp, http.StatusBadRequest)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "bad_request" {
		t.Errorf("expected error code 'bad_request', got %q", code)
	}
}

// TestGetPipeline_Unauthorized verifies that an unauthenticated GET request
// (no session cookies) returns 401 Unauthorized.
func TestGetPipeline_Unauthorized(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	// No login — use http.Get directly without any session cookies.
	resp, err := http.Get(devopsPipelineURL(env, "quick-pass"))
	if err != nil {
		t.Fatalf("http.Get failed: %v", err)
	}
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusUnauthorized)
}

// TestGetPipeline_Forbidden verifies that a user without the product-owner or
// devops role (qa@test.local has only the 'qa' role) receives 403 Forbidden.
func TestGetPipeline_Forbidden(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("qa@test.local", "qa-pass-123")

	resp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines/quick-pass", nil)
	requireStatus(t, resp, http.StatusForbidden)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "forbidden" {
		t.Errorf("expected error code 'forbidden', got %q", code)
	}
}
