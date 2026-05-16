// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 6 — DevOps Pipeline Integration Test: test-all.yaml
//
// Integration tests for the lifecycle/devops/all-tests.yaml pipeline.
// Verifies that the pipeline is discoverable via the DevOps API, has the
// correct metadata, can be triggered, and appears in run history.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

const (
	devopsTestAllListPath    = "/api/p/testproject/devops/pipelines"
	devopsTestAllRunPath     = "/api/p/testproject/devops/pipelines/all-tests/run"
	devopsTestAllCancelPath  = "/api/p/testproject/devops/pipelines/all-tests/cancel"
)

// allTestsPipelineYAML mirrors lifecycle/devops/all-tests.yaml.
const allTestsPipelineYAML = `name: Run ALL Tests
type: test

steps:
  - name: Lint
    description: go vet + staticcheck. Fails on any finding (pre-commit safety net).
    command: make lint
    timeout: 2m

  - name: Go unit tests
    description: Pure Go unit tests with -short flag (skips long-running cases).
    command: make test-unit
    timeout: 5m

  - name: Frontend tests
    description: Vitest component tests in tests/web/ (Vue 3 + happy-dom).
    command: cd tests/web && pnpm test
    timeout: 5m

  - name: Go integration tests
    description: Integration tests behind the integration build tag (slowest step).
    command: make test-integration
    timeout: 15m

  - name: E2E smoke tests
    description: Playwright flows against a fresh ./dist/kaos-control binary
    command: make test-e2e
    timeout: 5m
`

// TestDevopsTestAll_Discoverable verifies that the test-all pipeline is
// returned by the pipeline listing endpoint with the correct metadata.
func TestDevopsTestAll_Discoverable(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"all-tests.yaml": allTestsPipelineYAML,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodGet, devopsTestAllListPath, nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	pipelines, _ := data["pipelines"].([]any)
	if len(pipelines) == 0 {
		t.Fatal("expected at least one pipeline in listing")
	}

	var found map[string]any
	for _, p := range pipelines {
		entry, _ := p.(map[string]any)
		if slug, _ := entry["slug"].(string); slug == "all-tests" {
			found = entry
			break
		}
	}
	if found == nil {
		t.Fatalf("pipeline 'all-tests' not found in listing: %v", pipelines)
	}
}

// TestDevopsTestAll_Metadata verifies the pipeline has type "test" and the
// expected number of steps.
func TestDevopsTestAll_Metadata(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"all-tests.yaml": allTestsPipelineYAML,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodGet, devopsTestAllListPath, nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	pipelines, _ := data["pipelines"].([]any)
	var found map[string]any
	for _, p := range pipelines {
		entry, _ := p.(map[string]any)
		if slug, _ := entry["slug"].(string); slug == "all-tests" {
			found = entry
			break
		}
	}
	if found == nil {
		t.Fatal("pipeline 'all-tests' not found")
	}

	if typ, _ := found["type"].(string); typ != "test" {
		t.Errorf("pipeline type = %q, want \"test\"", typ)
	}

	steps, _ := found["steps"].([]any)
	if len(steps) != 5 {
		t.Errorf("pipeline has %d steps, want 5", len(steps))
	}

	if name, _ := found["name"].(string); name == "" {
		t.Error("pipeline name should not be empty")
	}
}

// TestDevopsTestAll_Triggerable verifies that POST .../run returns a run_id
// and a 202 Accepted response.
func TestDevopsTestAll_Triggerable(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"all-tests.yaml": allTestsPipelineYAML,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPost, devopsTestAllRunPath, nil)
	requireStatus(t, resp, http.StatusAccepted)
	data := readJSON(t, resp)

	runID, _ := data["run_id"].(string)
	if runID == "" {
		t.Error("expected non-empty run_id in trigger response")
	}
	if len(runID) != 16 {
		t.Errorf("run_id %q has unexpected length %d, want 16", runID, len(runID))
	}

	// Cancel to avoid lingering pipeline steps (make lint etc. will fail in test env).
	cancelResp := env.doRequest(http.MethodPost, devopsTestAllCancelPath, nil)
	cancelResp.Body.Close()
	waitForRunComplete(t, env, "all-tests", 10*time.Second)
}

// TestDevopsTestAll_RunHistory verifies that after triggering the pipeline a
// completed run appears in the NDJSON run log endpoint.
func TestDevopsTestAll_RunHistory(t *testing.T) {
	// Use a pipeline that completes quickly so we can verify history without
	// waiting for make targets that don't exist in the test environment.
	const quickTestPipeline = `name: Run ALL Tests
type: test
steps:
  - name: Echo
    command: echo ok
`
	env := newDevopsTestEnv(t, map[string]string{
		"all-tests.yaml": quickTestPipeline,
	})
	// env is auto-logged in as admin.

	resp := env.doRequest(http.MethodPost, devopsTestAllRunPath, nil)
	requireStatus(t, resp, http.StatusAccepted)
	data := readJSON(t, resp)
	runID, _ := data["run_id"].(string)
	if runID == "" {
		t.Fatal("trigger did not return run_id")
	}

	waitForRunComplete(t, env, "all-tests", 15*time.Second)

	// The run log endpoint returns NDJSON — parse line by line.
	logResp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/runs/"+runID, nil)
	requireStatus(t, logResp, http.StatusOK)

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(logResp.Body)
	logResp.Body.Close()

	var events []map[string]any
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(line, &obj); err != nil {
			t.Errorf("invalid NDJSON line: %v", err)
			continue
		}
		events = append(events, obj)
	}

	if len(events) == 0 {
		t.Fatal("run log returned no events")
	}

	// Every event should carry the run_id.
	for _, ev := range events {
		if id, _ := ev["run_id"].(string); id != runID {
			t.Errorf("event run_id = %q, want %q in event: %v", id, runID, ev)
		}
	}
}

// TestDevopsTestAll_RequiresAuth verifies that anonymous (no session cookie)
// requests to the all-tests pipeline list endpoint are rejected.
func TestDevopsTestAll_RequiresAuth(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"all-tests.yaml": allTestsPipelineYAML,
	})
	// Clear the auto-login cookies to simulate an anonymous request.
	env.cookies = nil

	resp := env.doRequest(http.MethodGet, devopsTestAllListPath, nil)
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Error("anonymous GET /pipelines should be rejected, got 200")
	}
}
