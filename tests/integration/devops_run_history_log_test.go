// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 3 — Integration: single-run log retrieval (F3; NF1, NF4)
//
// Tests for GET /api/p/{project}/devops/pipelines/{slug}/runs/{run_id}/log.
// Verifies NDJSON content, pipeline-slug scoping, path-traversal rejection,
// and role gating.
//
// Run with: go test ./tests/... -tags integration -run TestRunHistoryLog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestRunHistoryLog_ReturnsNDJSON runs a pipeline, then fetches the scoped log
// via the new endpoint. Verifies Content-Type and that every non-empty line
// parses as JSON.
func TestRunHistoryLog_ReturnsNDJSON(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Trigger a run via the run endpoint and wait for completion.
	startResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, startResp, http.StatusAccepted)
	runID, _ := readJSON(t, startResp)["run_id"].(string)
	if runID == "" {
		t.Fatal("expected non-empty run_id")
	}
	waitForRunComplete(t, env, "quick-pass", 15*time.Second)

	// Fetch via the scoped log endpoint.
	logResp := env.doRequest(http.MethodGet, devopsPipelineRunLogPath("quick-pass", runID), nil)
	requireStatus(t, logResp, http.StatusOK)

	ct := logResp.Header.Get("Content-Type")
	if ct != "application/x-ndjson" {
		t.Errorf("Content-Type = %q, want application/x-ndjson", ct)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(logResp.Body)
	logResp.Body.Close()

	rawBytes := buf.Bytes()
	if len(rawBytes) == 0 {
		t.Fatal("log response was empty")
	}

	// First pass: verify every non-empty line is valid JSON.
	lineCount := 0
	scanner := bufio.NewScanner(bytes.NewReader(rawBytes))
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		lineCount++
		var obj map[string]any
		if err := json.Unmarshal(line, &obj); err != nil {
			t.Errorf("NDJSON line %d is not valid JSON: %q: %v", lineCount, line, err)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanning NDJSON: %v", err)
	}
	if lineCount == 0 {
		t.Error("NDJSON response had no parseable lines")
	}

	// Second pass: the run.started event must be present.
	found := false
	scanner2 := bufio.NewScanner(bytes.NewReader(rawBytes))
	for scanner2.Scan() {
		line := bytes.TrimSpace(scanner2.Bytes())
		if len(line) == 0 {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(line, &obj); err != nil {
			continue
		}
		if obj["type"] == "pipeline.run.started" {
			found = true
		}
	}
	if !found {
		t.Error("NDJSON log missing pipeline.run.started event")
	}
}

// TestRunHistoryLog_UnknownRunID404 verifies that a valid-format run ID that
// does not exist returns 404.
func TestRunHistoryLog_UnknownRunID404(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// "0000000000000000" has the right format but no corresponding log.
	resp := env.doRequest(http.MethodGet, devopsPipelineRunLogPath("quick-pass", "0000000000000000"), nil)
	requireStatus(t, resp, http.StatusNotFound)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "not_found" {
		t.Errorf("error.code = %q, want not_found", code)
	}
}

// TestRunHistoryLog_RunIDFromOtherPipeline404 verifies that requesting a real
// run_id under the wrong slug returns 404 (pipeline-scoping check, backend M4).
func TestRunHistoryLog_RunIDFromOtherPipeline404(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
		"quick-fail.yaml": pipelineQuickFail,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Run quick-pass and capture its run ID.
	startResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, startResp, http.StatusAccepted)
	runID, _ := readJSON(t, startResp)["run_id"].(string)
	waitForRunComplete(t, env, "quick-pass", 15*time.Second)

	// Request that run ID under the quick-fail slug → must 404.
	resp := env.doRequest(http.MethodGet, devopsPipelineRunLogPath("quick-fail", runID), nil)
	requireStatus(t, resp, http.StatusNotFound)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "not_found" {
		t.Errorf("cross-pipeline 404: error.code = %q, want not_found", code)
	}
}

// TestRunHistoryLog_ForbiddenRole verifies that a non-devops/non-owner role
// receives 403 on the scoped log endpoint.
func TestRunHistoryLog_ForbiddenRole(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Run pipeline as admin to get a real run ID.
	startResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, startResp, http.StatusAccepted)
	runID, _ := readJSON(t, startResp)["run_id"].(string)
	waitForRunComplete(t, env, "quick-pass", 15*time.Second)

	// Switch to qa role.
	env.login("qa@test.local", "qa-pass-123")
	resp := env.doRequest(http.MethodGet, devopsPipelineRunLogPath("quick-pass", runID), nil)
	requireStatus(t, resp, http.StatusForbidden)
	resp.Body.Close()
}

// TestRunHistoryLog_PathTraversalRejected verifies that slugs or run IDs
// containing path-traversal sequences are rejected with 400 before any
// file read is attempted (NF4).
func TestRunHistoryLog_PathTraversalRejected(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	cases := []struct {
		name    string
		slug    string
		runID   string
		wantGTE int // minimum expected status code (both 400 and 404 are acceptable)
	}{
		{
			name:    "slug with leading dot",
			slug:    "..some-pipe",
			runID:   "abcdef0123456789",
			wantGTE: http.StatusBadRequest,
		},
		{
			name:    "slug with internal slash encoded",
			slug:    "pipe%2fpipe",
			runID:   "abcdef0123456789",
			wantGTE: http.StatusBadRequest,
		},
		{
			name:    "runID too short",
			slug:    "quick-pass",
			runID:   "../etc/passwd",
			wantGTE: http.StatusBadRequest,
		},
		{
			name:    "runID with URL-encoded slash",
			slug:    "quick-pass",
			runID:   "..%2f..%2fetc%2fpasswd",
			wantGTE: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := "/api/p/testproject/devops/pipelines/" + tc.slug + "/runs/" + tc.runID + "/log"
			resp := env.doRequest(http.MethodGet, path, nil)
			if resp.StatusCode < tc.wantGTE {
				var buf bytes.Buffer
				_, _ = buf.ReadFrom(resp.Body)
				resp.Body.Close()
				t.Errorf("expected status ≥%d for %s, got %d: %s",
					tc.wantGTE, tc.name, resp.StatusCode, buf.String())
			} else {
				resp.Body.Close()
			}
		})
	}
}
