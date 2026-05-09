// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 2 — Pipeline Listing API Tests
//
// Tests for GET /api/p/testproject/devops/pipelines covering:
//   - Role-based access control (product-owner, devops, unauthenticated, wrong role)
//   - Response schema (slug, name, type, steps with name/description but not command)
//   - Malformed YAML files are excluded from the response
//   - Performance: response time < 200ms with 50 pipeline fixture files

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestDevopsListPipelines_ProductOwnerAccess verifies that a user with the
// product-owner role receives a 200 response containing the pipelines list.
func TestDevopsListPipelines_ProductOwnerAccess(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"build.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	pipelines, ok := data["pipelines"].([]any)
	if !ok {
		t.Fatalf("expected 'pipelines' array in response, got: %v", data)
	}
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}
}

// TestDevopsListPipelines_DevopsRoleAccess verifies that a user with the devops
// role receives a 200 response (dev@test.local has devops role in devopsCfgYAML).
func TestDevopsListPipelines_DevopsRoleAccess(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"build.yaml": pipelineQuickPass,
	})
	env.login("dev@test.local", "dev-pass-123")

	resp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines", nil)
	requireStatus(t, resp, http.StatusOK)

	data := readJSON(t, resp)
	if _, ok := data["pipelines"].([]any); !ok {
		t.Fatalf("expected 'pipelines' array in response, got %v", data)
	}
}

// TestDevopsListPipelines_Unauthenticated verifies that unauthenticated requests
// receive a 401 response.
func TestDevopsListPipelines_Unauthenticated(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"build.yaml": pipelineQuickPass,
	})
	// No login — do not set cookies

	resp, err := http.Get(devopsListURL(env))
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}

// TestDevopsListPipelines_ForbiddenRole verifies that a user without the
// product-owner or devops role receives a 403 response.
func TestDevopsListPipelines_ForbiddenRole(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"build.yaml": pipelineQuickPass,
	})
	env.login("qa@test.local", "qa-pass-123")

	resp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines", nil)
	requireStatus(t, resp, http.StatusForbidden)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "forbidden" {
		t.Errorf("expected error code 'forbidden', got %q", code)
	}
}

// TestDevopsListPipelines_ResponseSchema validates the response shape:
// each pipeline must have slug, name, type, and steps (with name+description but no command).
func TestDevopsListPipelines_ResponseSchema(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"build.yaml": `name: Schema Test
type: build
steps:
  - name: Compile
    description: Build step
    command: make build
  - name: Test
    command: make test
`,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	pipelines, _ := data["pipelines"].([]any)
	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}

	pl, _ := pipelines[0].(map[string]any)

	// Required top-level fields
	if slug, _ := pl["slug"].(string); slug != "build" {
		t.Errorf("slug = %q, want %q", slug, "build")
	}
	if name, _ := pl["name"].(string); name != "Schema Test" {
		t.Errorf("name = %q, want %q", name, "Schema Test")
	}
	if typ, _ := pl["type"].(string); typ != "build" {
		t.Errorf("type = %q, want %q", typ, "build")
	}

	steps, _ := pl["steps"].([]any)
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(steps))
	}

	step0, _ := steps[0].(map[string]any)

	// Steps must have name and description
	if n, _ := step0["name"].(string); n != "Compile" {
		t.Errorf("step[0].name = %q, want %q", n, "Compile")
	}
	if d, _ := step0["description"].(string); d != "Build step" {
		t.Errorf("step[0].description = %q, want %q", d, "Build step")
	}

	// Steps must NOT expose the command
	if _, hasCmd := step0["command"]; hasCmd {
		t.Error("step response must not include 'command' field")
	}
}

// TestDevopsListPipelines_MalformedFilesExcluded verifies that malformed YAML
// files are excluded from the response while valid ones are included.
func TestDevopsListPipelines_MalformedFilesExcluded(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"valid.yaml":   pipelineQuickPass,
		"invalid.yaml": `{not: valid: yaml: [`,
		"missing.yaml": `type: build
steps:
  - name: Step
    command: echo hi
`,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	pipelines, _ := data["pipelines"].([]any)
	if len(pipelines) != 1 {
		t.Errorf("expected 1 valid pipeline, got %d (malformed files should be excluded)", len(pipelines))
	}
}

// TestDevopsListPipelines_EmptyDirectory verifies that an empty or missing
// lifecycle/devops/ directory returns an empty pipelines list.
func TestDevopsListPipelines_EmptyDirectory(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	pipelines, _ := data["pipelines"].([]any)
	if len(pipelines) != 0 {
		t.Errorf("expected 0 pipelines for empty devops dir, got %d", len(pipelines))
	}
}

// TestDevopsListPipelines_Performance validates that the listing endpoint
// responds in under 200ms with 50 pipeline fixture files (NF1).
func TestDevopsListPipelines_Performance(t *testing.T) {
	// Generate 50 valid pipeline files.
	pipelines := make(map[string]string, 50)
	for i := 1; i <= 50; i++ {
		name := fmt.Sprintf("pipeline-%02d.yaml", i)
		content := fmt.Sprintf(`name: Pipeline %d
type: build
steps:
  - name: Step One
    command: echo %d
`, i, i)
		pipelines[name] = content
	}

	env := newDevopsTestEnv(t, pipelines)
	env.login("admin@test.local", "admin-pass-123")

	start := time.Now()
	resp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines", nil)
	elapsed := time.Since(start)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	pl, _ := data["pipelines"].([]any)
	if len(pl) != 50 {
		t.Errorf("expected 50 pipelines, got %d", len(pl))
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("listing 50 pipelines took %v, want < 200ms", elapsed)
	}
}
