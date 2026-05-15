// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestones 2, 3 & 4 — PUT update-pipeline, atomic write integrity, and regression tests
//
// Tests for PUT /api/p/{project}/devops/pipelines/{slug} covering:
//   - 200 OK with updated summary; file on disk matches submitted YAML
//   - 404 Not Found when the pipeline does not exist
//   - 400 Bad Request for invalid YAML, missing required fields, and invalid timeout
//   - 409 Conflict when any pipeline is currently running (global active-run guard)
//   - 200 OK after a run completes (stale lock is cleared)
//   - 401 Unauthorized and 403 Forbidden access control
//   - Slug is preserved (filename unchanged) after a name-only edit
//   - No .tmp files are left on disk after a successful atomic write
//
// Milestone 3 — atomic write integrity:
//   - GET after PUT returns the new content (not a partial write)
//   - A failed validation leaves the original file unchanged on disk
//
// Milestone 4 — regression:
//   - TestCreateThenEditThenRun: create → edit via PUT → run executes updated definition

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// pipelineUpdatedDef is a valid pipeline that replaces quick-pass in update tests.
// It changes the name and adds a second step to verify both name and step_count.
const pipelineUpdatedDef = `name: Quick Pass Updated
type: build
steps:
  - name: Echo OK
    description: Verify the environment works
    command: echo ok
  - name: Extra Step
    command: echo extra
`

// TestUpdatePipeline_Success verifies that PUT /devops/pipelines/{slug} with a
// valid YAML definition returns 200 OK. The response body contains the updated
// name and step_count, and the file on disk matches the submitted YAML byte-for-byte.
func TestUpdatePipeline_Success(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/quick-pass",
		map[string]any{"definition": pipelineUpdatedDef})
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	if name, _ := data["name"].(string); name != "Quick Pass Updated" {
		t.Errorf("response name = %q, want %q", name, "Quick Pass Updated")
	}
	if stepCount, _ := data["step_count"].(float64); int(stepCount) != 2 {
		t.Errorf("response step_count = %v, want 2", data["step_count"])
	}
	if slug, _ := data["slug"].(string); slug != "quick-pass" {
		t.Errorf("response slug = %q, want %q", slug, "quick-pass")
	}

	// Verify file on disk matches submitted YAML byte-for-byte.
	destPath := filepath.Join(env.projectRoot, "lifecycle", "devops", "quick-pass.yaml")
	diskContent, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("reading pipeline file from disk: %v", err)
	}
	if string(diskContent) != pipelineUpdatedDef {
		t.Errorf("disk content mismatch:\ngot:  %q\nwant: %q", string(diskContent), pipelineUpdatedDef)
	}
}

// TestUpdatePipeline_NotFound verifies that PUT to a non-existent pipeline slug
// returns 404 Not Found.
func TestUpdatePipeline_NotFound(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/nonexistent",
		map[string]any{"definition": pipelineQuickPass})
	requireStatus(t, resp, http.StatusNotFound)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "not_found" {
		t.Errorf("expected error code 'not_found', got %q", code)
	}
}

// TestUpdatePipeline_InvalidYAML verifies that submitting syntactically invalid
// YAML returns 400 Bad Request with a descriptive error message.
func TestUpdatePipeline_InvalidYAML(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/quick-pass",
		map[string]any{"definition": "{{{"})
	requireStatus(t, resp, http.StatusBadRequest)
	data := readJSON(t, resp)

	if _, hasErr := data["error"]; !hasErr {
		t.Error("expected 'error' field in 400 response body")
	}
}

// TestUpdatePipeline_MissingRequiredFields verifies that submitting valid YAML
// that lacks required pipeline fields returns 400 Bad Request.
func TestUpdatePipeline_MissingRequiredFields(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Definition is valid YAML but missing the 'name' field.
	const missingName = "type: build\nsteps:\n  - name: step\n    command: echo ok\n"
	resp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/quick-pass",
		map[string]any{"definition": missingName})
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestUpdatePipeline_MissingStepCommand verifies that a pipeline definition
// containing a step with no command field returns 400 Bad Request.
func TestUpdatePipeline_MissingStepCommand(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	const noCommand = "name: Test\ntype: build\nsteps:\n  - name: step-no-cmd\n"
	resp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/quick-pass",
		map[string]any{"definition": noCommand})
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestUpdatePipeline_InvalidTimeout verifies that a step with an unparseable
// timeout value (not a valid Go duration) returns 400 Bad Request.
func TestUpdatePipeline_InvalidTimeout(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	const badTimeout = "name: Test\ntype: build\nsteps:\n  - name: s\n    command: echo ok\n    timeout: not-a-duration\n"
	resp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/quick-pass",
		map[string]any{"definition": badTimeout})
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestUpdatePipeline_ConflictWhileRunning verifies that PUT to a pipeline that
// is currently running returns 409 Conflict (global active-run guard).
func TestUpdatePipeline_ConflictWhileRunning(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"slow-step.yaml": pipelineSlowStep,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Start the slow-running pipeline.
	runResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/run", nil)
	requireStatus(t, runResp, http.StatusAccepted)
	runResp.Body.Close()

	// Immediately attempt to edit the same pipeline — must conflict.
	putResp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/slow-step",
		map[string]any{"definition": pipelineUpdatedDef})
	requireStatus(t, putResp, http.StatusConflict)
	data := readJSON(t, putResp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "conflict" {
		t.Errorf("expected error code 'conflict', got %q", code)
	}

	// Clean up: cancel the run and wait for it to finish.
	cancelResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/cancel", nil)
	cancelResp.Body.Close()
	waitForRunComplete(t, env, "slow-step", 10*time.Second)
}

// TestUpdatePipeline_ConflictOtherPipelineRunning verifies that when any pipeline
// is running, editing a DIFFERENT pipeline is also blocked with 409 Conflict.
// The global AnyRunning() guard applies to all edits, not just the running pipeline.
func TestUpdatePipeline_ConflictOtherPipelineRunning(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"slow-step.yaml": pipelineSlowStep,
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Start slow-step (pipeline A).
	runResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/run", nil)
	requireStatus(t, runResp, http.StatusAccepted)
	runResp.Body.Close()

	// Attempt to edit quick-pass (pipeline B) while slow-step is running — must conflict.
	putResp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/quick-pass",
		map[string]any{"definition": pipelineUpdatedDef})
	requireStatus(t, putResp, http.StatusConflict)
	data := readJSON(t, putResp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "conflict" {
		t.Errorf("expected error code 'conflict', got %q", code)
	}

	// Clean up: cancel slow-step.
	cancelResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/cancel", nil)
	cancelResp.Body.Close()
	waitForRunComplete(t, env, "slow-step", 10*time.Second)
}

// TestUpdatePipeline_SuccessAfterRunCompletes verifies that once a run finishes,
// editing the pipeline succeeds (no stale lock remains).
func TestUpdatePipeline_SuccessAfterRunCompletes(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Start quick-pass and wait for completion.
	runResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, runResp, http.StatusAccepted)
	runResp.Body.Close()
	waitForRunComplete(t, env, "quick-pass", 10*time.Second)

	// After the run completes, the edit must succeed.
	putResp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/quick-pass",
		map[string]any{"definition": pipelineUpdatedDef})
	requireStatus(t, putResp, http.StatusOK)
	putResp.Body.Close()
}

// TestUpdatePipeline_Unauthorized verifies that a PUT request without a session
// cookie returns 401 Unauthorized.
func TestUpdatePipeline_Unauthorized(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	// No login — construct a raw request without session cookies.
	b, _ := json.Marshal(map[string]string{"definition": pipelineQuickPass})
	req, err := http.NewRequest(http.MethodPut, devopsPipelineURL(env, "quick-pass"), bytes.NewReader(b))
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http request failed: %v", err)
	}
	defer resp.Body.Close()
	requireStatus(t, resp, http.StatusUnauthorized)
}

// TestUpdatePipeline_Forbidden verifies that a user without the product-owner or
// devops role (qa@test.local has only the 'qa' role) receives 403 Forbidden.
func TestUpdatePipeline_Forbidden(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("qa@test.local", "qa-pass-123")

	resp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/quick-pass",
		map[string]any{"definition": pipelineUpdatedDef})
	requireStatus(t, resp, http.StatusForbidden)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "forbidden" {
		t.Errorf("expected error code 'forbidden', got %q", code)
	}
}

// TestUpdatePipeline_PreservesSlug verifies that editing a pipeline's name does
// not rename the file on disk: the slug (and filename) remain unchanged.
func TestUpdatePipeline_PreservesSlug(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"my-pipe.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	const renamedDef = "name: Renamed Pipeline\ntype: build\nsteps:\n  - name: run\n    command: echo ok\n"
	resp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/my-pipe",
		map[string]any{"definition": renamedDef})
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	// Response slug must still be "my-pipe".
	if slug, _ := data["slug"].(string); slug != "my-pipe" {
		t.Errorf("response slug = %q, want %q", slug, "my-pipe")
	}
	// Name in response reflects the new value.
	if name, _ := data["name"].(string); name != "Renamed Pipeline" {
		t.Errorf("response name = %q, want %q", name, "Renamed Pipeline")
	}

	// File on disk is still my-pipe.yaml (not renamed to renamed-pipeline.yaml).
	expectedPath := filepath.Join(env.projectRoot, "lifecycle", "devops", "my-pipe.yaml")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected pipeline file at %s after rename: %v", expectedPath, err)
	}
}

// TestUpdatePipeline_AtomicWrite verifies that after a successful PUT, no
// temporary files (*.tmp*) are left behind in the devops directory.
func TestUpdatePipeline_AtomicWrite(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/quick-pass",
		map[string]any{"definition": pipelineUpdatedDef})
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	devopsDir := filepath.Join(env.projectRoot, "lifecycle", "devops")
	tmpFiles, err := filepath.Glob(filepath.Join(devopsDir, "*.tmp*"))
	if err != nil {
		t.Fatalf("globbing for temp files: %v", err)
	}
	if len(tmpFiles) > 0 {
		t.Errorf("found %d leftover temp file(s) after successful PUT: %v", len(tmpFiles), tmpFiles)
	}
}

// --- Milestone 3: Atomic write integrity ---

// TestUpdatePipeline_FileIntegrity_ConcurrentRead verifies that a GET performed
// immediately after a successful PUT returns the updated content, not a partial
// write. Atomic rename guarantees that readers always see a complete file.
func TestUpdatePipeline_FileIntegrity_ConcurrentRead(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// PUT the new definition.
	putResp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/quick-pass",
		map[string]any{"definition": pipelineUpdatedDef})
	requireStatus(t, putResp, http.StatusOK)
	putResp.Body.Close()

	// Immediately GET — must return the new content, not the original or a partial write.
	getResp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines/quick-pass", nil)
	requireStatus(t, getResp, http.StatusOK)
	body, err := io.ReadAll(getResp.Body)
	getResp.Body.Close()
	if err != nil {
		t.Fatalf("reading GET response: %v", err)
	}
	if string(body) != pipelineUpdatedDef {
		t.Errorf("GET after PUT returned stale/partial content:\ngot:  %q\nwant: %q",
			string(body), pipelineUpdatedDef)
	}
}

// TestUpdatePipeline_OriginalPreservedOnValidationFailure verifies that when a
// PUT is rejected due to invalid YAML, the existing pipeline file on disk is
// unchanged: a subsequent GET still returns the original content.
func TestUpdatePipeline_OriginalPreservedOnValidationFailure(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// PUT with invalid YAML.
	putResp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/quick-pass",
		map[string]any{"definition": "{{{"})
	requireStatus(t, putResp, http.StatusBadRequest)
	putResp.Body.Close()

	// GET must still return the original, unmodified content.
	getResp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/pipelines/quick-pass", nil)
	requireStatus(t, getResp, http.StatusOK)
	body, err := io.ReadAll(getResp.Body)
	getResp.Body.Close()
	if err != nil {
		t.Fatalf("reading GET response after failed PUT: %v", err)
	}
	if string(body) != pipelineQuickPass {
		t.Errorf("file was modified after failed PUT:\ngot:  %q\nwant: %q",
			string(body), pipelineQuickPass)
	}
}

// --- Milestone 4: Regression ---

// TestCreateThenEditThenRun is a regression test verifying the full
// create → edit → run lifecycle. The run after editing must execute the
// updated command, not the original one.
func TestCreateThenEditThenRun(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Step 1: Create a pipeline with "echo original".
	const origDef = "name: Editable\ntype: build\nsteps:\n  - name: Output Step\n    command: echo original\n"
	createResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines",
		map[string]any{"slug": "editable", "definition": origDef})
	requireStatus(t, createResp, http.StatusCreated)
	createResp.Body.Close()

	// Step 2: Edit via PUT — change the command to "echo updated".
	const editedDef = "name: Editable\ntype: build\nsteps:\n  - name: Output Step\n    command: echo updated\n"
	putResp := env.doRequest(http.MethodPut, "/api/p/testproject/devops/pipelines/editable",
		map[string]any{"definition": editedDef})
	requireStatus(t, putResp, http.StatusOK)
	putResp.Body.Close()

	// Step 3: Subscribe to hub events then trigger the run.
	ch := make(chan []byte, 256)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	runResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/editable/run", nil)
	requireStatus(t, runResp, http.StatusAccepted)
	apiData := readJSON(t, runResp)
	runID, _ := apiData["run_id"].(string)
	if runID == "" {
		t.Fatal("trigger API did not return run_id")
	}

	// Wait for the run to complete.
	deadline := time.After(15 * time.Second)
WAIT:
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(raw, &evt) == nil && evt.Type == "pipeline.run.completed" {
				break WAIT
			}
		case <-deadline:
			t.Fatal("timed out waiting for pipeline.run.completed")
		}
	}

	// Step 4: Fetch the run log and verify "updated" appears in output,
	// "original" does not.
	logResp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/runs/"+runID, nil)
	requireStatus(t, logResp, http.StatusOK)
	var logBuf bytes.Buffer
	_, _ = logBuf.ReadFrom(logResp.Body)
	logResp.Body.Close()

	var foundUpdated, foundOriginal bool
	scanner := bufio.NewScanner(&logBuf)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(line, &obj); err != nil {
			continue
		}
		if typ, _ := obj["type"].(string); typ != "pipeline.step.output" {
			continue
		}
		if text, _ := obj["text"].(string); text != "" {
			if strings.Contains(text, "updated") {
				foundUpdated = true
			}
			if strings.Contains(text, "original") {
				foundOriginal = true
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanning NDJSON log: %v", err)
	}

	if !foundUpdated {
		t.Error("run log does not contain 'updated'; pipeline may have executed the old definition")
	}
	if foundOriginal {
		t.Error("run log contains 'original'; pipeline executed the pre-edit definition")
	}
}
