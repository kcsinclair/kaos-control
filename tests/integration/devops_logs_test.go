//go:build integration

package integration

// Milestone 6 — Run Log Persistence Tests
//
// Tests verifying that run logs are persisted to disk and readable via the API:
//   - After a run, a log file exists at <dataDir>/devops/testproject/<run_id>.log
//   - Log file contains valid JSON-lines with all events
//   - GET /api/p/{project}/devops/runs/{run_id} returns the log content
//   - In-progress runs return partial log content via the API
//   - The devops log directory is auto-created on first run
//   - GET .../runs/{run_id} for a non-existent run returns 404

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestDevopsLogs_FileExistsAfterRun verifies that after a pipeline run
// completes, a log file exists at the expected path.
func TestDevopsLogs_FileExistsAfterRun(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	data := readJSON(t, resp)
	runID, _ := data["run_id"].(string)
	if runID == "" {
		t.Fatal("expected non-empty run_id")
	}

	waitForRunComplete(t, env, "quick-pass", 10*time.Second)

	// Check log file exists on disk.
	logPath := filepath.Join(env.dataDir, "devops", "testproject", runID+".log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("log file not found at %s", logPath)
	}
}

// TestDevopsLogs_ValidJSONLines verifies that the log file contains valid
// JSON-lines entries with the expected event fields.
func TestDevopsLogs_ValidJSONLines(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	data := readJSON(t, resp)
	runID, _ := data["run_id"].(string)

	waitForRunComplete(t, env, "quick-pass", 10*time.Second)

	logPath := filepath.Join(env.dataDir, "devops", "testproject", runID+".log")
	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("opening log file: %v", err)
	}
	defer f.Close()

	var eventTypes []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("invalid JSON line: %q: %v", line, err)
			continue
		}
		// Each entry must have time, event_type, and payload.
		if _, ok := entry["time"]; !ok {
			t.Errorf("log entry missing 'time': %s", line)
		}
		if _, ok := entry["event_type"]; !ok {
			t.Errorf("log entry missing 'event_type': %s", line)
		}
		if _, ok := entry["payload"]; !ok {
			t.Errorf("log entry missing 'payload': %s", line)
		}
		if evtType, _ := entry["event_type"].(string); evtType != "" {
			eventTypes = append(eventTypes, evtType)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanning log file: %v", err)
	}

	// Must have at least run.started and run.completed entries.
	hasRunStarted := false
	hasRunCompleted := false
	for _, et := range eventTypes {
		if et == "pipeline.run.started" {
			hasRunStarted = true
		}
		if et == "pipeline.run.completed" {
			hasRunCompleted = true
		}
	}
	if !hasRunStarted {
		t.Error("log file missing pipeline.run.started entry")
	}
	if !hasRunCompleted {
		t.Error("log file missing pipeline.run.completed entry")
	}
}

// TestDevopsLogs_APIReturnsContent verifies that GET /devops/runs/{run_id}
// returns the log content for a completed run.
func TestDevopsLogs_APIReturnsContent(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	data := readJSON(t, resp)
	runID, _ := data["run_id"].(string)

	waitForRunComplete(t, env, "quick-pass", 10*time.Second)

	logResp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/runs/"+runID, nil)
	requireStatus(t, logResp, http.StatusOK)

	defer logResp.Body.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(logResp.Body); err != nil {
		t.Fatal(err)
	}
	body := buf.String()

	if body == "" {
		t.Error("log API returned empty body for completed run")
	}
	// Should contain at least one JSON-lines entry.
	if !strings.Contains(body, "pipeline.run.started") {
		t.Error("log response missing pipeline.run.started entry")
	}
	if !strings.Contains(body, "pipeline.run.completed") {
		t.Error("log response missing pipeline.run.completed entry")
	}
}

// TestDevopsLogs_InProgressReturnsPartialContent verifies that calling the log
// API while a run is still in progress returns partial content (at least the
// run.started entry, which is written before any steps execute).
func TestDevopsLogs_InProgressReturnsPartialContent(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"slow-step.yaml": pipelineSlowStep,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	data := readJSON(t, resp)
	runID, _ := data["run_id"].(string)

	// Give the run a moment to write at least the run.started event to disk.
	time.Sleep(200 * time.Millisecond)

	// Read the log while still running.
	logResp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/runs/"+runID, nil)
	requireStatus(t, logResp, http.StatusOK)

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(logResp.Body)
	logResp.Body.Close()

	body := buf.String()
	if !strings.Contains(body, "pipeline.run.started") {
		t.Error("in-progress log does not contain pipeline.run.started")
	}

	// Cancel the run to clean up.
	cancelResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/cancel", nil)
	cancelResp.Body.Close()
	waitForRunComplete(t, env, "slow-step", 10*time.Second)
}

// TestDevopsLogs_DirectoryAutoCreated verifies that the devops log directory is
// created automatically on the first pipeline run.
func TestDevopsLogs_DirectoryAutoCreated(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Verify the log directory does not yet exist.
	logDir := filepath.Join(env.dataDir, "devops", "testproject")
	if _, err := os.Stat(logDir); !os.IsNotExist(err) {
		// Directory already exists — remove it so we can test creation.
		os.RemoveAll(logDir)
	}

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	resp.Body.Close()
	waitForRunComplete(t, env, "quick-pass", 10*time.Second)

	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		t.Errorf("devops log directory was not auto-created at %s", logDir)
	}
}

// TestDevopsLogs_NotFoundForUnknownRunID verifies that requesting the log for
// a non-existent run_id returns 404.
func TestDevopsLogs_NotFoundForUnknownRunID(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	logResp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/runs/nonexistent-run-id-000", nil)
	requireStatus(t, logResp, http.StatusNotFound)
	data := readJSON(t, logResp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "not_found" {
		t.Errorf("expected error code 'not_found', got %q", code)
	}
}

// TestDevopsLogs_ContentTypeIsNDJSON verifies that the log API returns the
// application/x-ndjson content type for completed runs.
func TestDevopsLogs_ContentTypeIsNDJSON(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	data := readJSON(t, resp)
	runID, _ := data["run_id"].(string)

	waitForRunComplete(t, env, "quick-pass", 10*time.Second)

	logResp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/runs/"+runID, nil)
	requireStatus(t, logResp, http.StatusOK)
	logResp.Body.Close()

	ct := logResp.Header.Get("Content-Type")
	if ct != "application/x-ndjson" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/x-ndjson")
	}
}
