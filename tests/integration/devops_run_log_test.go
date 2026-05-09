// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 2 — REST completed-run log endpoint tests (devops-pipeline-log-streaming)
//
// Tests for GET /api/p/{project}/devops/runs/{run_id} that validate the NDJSON
// response format produced by LogStore.ReadLogNDJSON. Each line is a flat JSON
// object whose fields come from the original event payload merged with a "type"
// key — internal log-store fields (time, event_type) are not forwarded.
//
// Scenarios covered:
//   - Every NDJSON line has a "type" field.
//   - The sequence includes all five event types in the correct order.
//   - Step boundary lines (step.started, step.completed) include step,
//     step_index, and timestamp (where the struct carries one).
//   - The final pipeline.run.completed line includes status and duration_seconds.
//   - Content-Type header is application/x-ndjson.
//   - pipeline_slug (not legacy "pipeline") is present in all event lines.
//
// Run with: go test ./tests/... -tags integration -run TestDevOpsRunLog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// pipelineThreeSteps is a three-step pipeline giving a rich event sequence.
const pipelineThreeSteps = `name: Three Steps
type: build
steps:
  - name: Step One
    command: echo one
  - name: Step Two
    command: echo two
  - name: Step Three
    command: echo three
`

// runAndFetchNDJSON triggers the given pipeline, waits for completion, fetches
// the run log via REST, and parses it into a slice of flat JSON objects.
func runAndFetchNDJSON(t *testing.T, env *testEnv, slug string) ([]map[string]any, *http.Response) {
	t.Helper()

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/"+slug+"/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	apiData := readJSON(t, resp)
	runID, _ := apiData["run_id"].(string)
	if runID == "" {
		t.Fatal("trigger API did not return run_id")
	}

	waitForRunComplete(t, env, slug, 15*time.Second)

	logResp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/runs/"+runID, nil)
	requireStatus(t, logResp, http.StatusOK)

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(logResp.Body)
	logResp.Body.Close()

	var lines []map[string]any
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal(line, &obj); err != nil {
			t.Errorf("invalid NDJSON line %q: %v", line, err)
			continue
		}
		lines = append(lines, obj)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanning NDJSON response: %v", err)
	}

	return lines, logResp
}

// TestDevOpsRunLog_ContentTypeIsNDJSON verifies the Content-Type header.
func TestDevOpsRunLog_ContentTypeIsNDJSON(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"three-steps.yaml": pipelineThreeSteps,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/three-steps/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	apiData := readJSON(t, resp)
	runID, _ := apiData["run_id"].(string)

	waitForRunComplete(t, env, "three-steps", 15*time.Second)

	logResp := env.doRequest(http.MethodGet, "/api/p/testproject/devops/runs/"+runID, nil)
	requireStatus(t, logResp, http.StatusOK)
	logResp.Body.Close()

	ct := logResp.Header.Get("Content-Type")
	if ct != "application/x-ndjson" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/x-ndjson")
	}
}

// TestDevOpsRunLog_EveryLineHasTypeField verifies that every NDJSON line in the
// run log response has a non-empty "type" field.
func TestDevOpsRunLog_EveryLineHasTypeField(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"three-steps.yaml": pipelineThreeSteps,
	})
	env.login("admin@test.local", "admin-pass-123")

	lines, _ := runAndFetchNDJSON(t, env, "three-steps")

	if len(lines) == 0 {
		t.Fatal("NDJSON response contained no lines")
	}
	for i, obj := range lines {
		typ, _ := obj["type"].(string)
		if typ == "" {
			t.Errorf("line %d: missing or empty 'type' field: %v", i, obj)
		}
	}
}

// TestDevOpsRunLog_AllEventTypesPresent verifies that the log contains all five
// expected event types.
func TestDevOpsRunLog_AllEventTypesPresent(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"three-steps.yaml": pipelineThreeSteps,
	})
	env.login("admin@test.local", "admin-pass-123")

	lines, _ := runAndFetchNDJSON(t, env, "three-steps")

	typesSeen := make(map[string]int)
	for _, obj := range lines {
		typ, _ := obj["type"].(string)
		typesSeen[typ]++
	}

	required := []string{
		"pipeline.run.started",
		"pipeline.step.started",
		"pipeline.step.output",
		"pipeline.step.completed",
		"pipeline.run.completed",
	}
	for _, typ := range required {
		if typesSeen[typ] == 0 {
			t.Errorf("expected event type %q not found in NDJSON log", typ)
		}
	}
}

// TestDevOpsRunLog_StepBoundaryFields verifies that step.started lines include
// step, step_index, and timestamp, and step.completed lines include step,
// step_index, status, exit_code, and duration_seconds.
func TestDevOpsRunLog_StepBoundaryFields(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"three-steps.yaml": pipelineThreeSteps,
	})
	env.login("admin@test.local", "admin-pass-123")

	lines, _ := runAndFetchNDJSON(t, env, "three-steps")

	var gotStepStarted, gotStepCompleted bool
	for _, obj := range lines {
		typ, _ := obj["type"].(string)

		switch typ {
		case "pipeline.step.started":
			gotStepStarted = true
			if step, _ := obj["step"].(string); step == "" {
				t.Error("step.started NDJSON line: missing 'step' field")
			}
			if _, ok := obj["step_index"]; !ok {
				t.Error("step.started NDJSON line: missing 'step_index' field")
			}
			ts, _ := obj["timestamp"].(string)
			if !isRFC3339(ts) {
				t.Errorf("step.started NDJSON line: timestamp %q is not valid RFC 3339", ts)
			}

		case "pipeline.step.completed":
			gotStepCompleted = true
			if step, _ := obj["step"].(string); step == "" {
				t.Error("step.completed NDJSON line: missing 'step' field")
			}
			if _, ok := obj["step_index"]; !ok {
				t.Error("step.completed NDJSON line: missing 'step_index' field")
			}
			if status, _ := obj["status"].(string); status == "" {
				t.Error("step.completed NDJSON line: missing 'status' field")
			}
			if _, ok := obj["exit_code"]; !ok {
				t.Error("step.completed NDJSON line: missing 'exit_code' field")
			}
			if _, ok := obj["duration_seconds"]; !ok {
				t.Error("step.completed NDJSON line: missing 'duration_seconds' field")
			}
		}
	}

	if !gotStepStarted {
		t.Error("no pipeline.step.started lines found in NDJSON log")
	}
	if !gotStepCompleted {
		t.Error("no pipeline.step.completed lines found in NDJSON log")
	}
}

// TestDevOpsRunLog_FinalEventHasStatusAndDuration verifies that the final
// pipeline.run.completed line includes status and duration_seconds.
func TestDevOpsRunLog_FinalEventHasStatusAndDuration(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"three-steps.yaml": pipelineThreeSteps,
	})
	env.login("admin@test.local", "admin-pass-123")

	lines, _ := runAndFetchNDJSON(t, env, "three-steps")

	if len(lines) == 0 {
		t.Fatal("NDJSON response was empty")
	}

	last := lines[len(lines)-1]
	typ, _ := last["type"].(string)
	if typ != "pipeline.run.completed" {
		t.Errorf("last NDJSON line type = %q, want %q", typ, "pipeline.run.completed")
	}
	if status, _ := last["status"].(string); status == "" {
		t.Error("run.completed NDJSON line: missing 'status' field")
	}
	if _, ok := last["duration_seconds"]; !ok {
		t.Error("run.completed NDJSON line: missing 'duration_seconds' field")
	}
}

// TestDevOpsRunLog_FirstEventIsRunStarted verifies that the first NDJSON line
// is pipeline.run.started.
func TestDevOpsRunLog_FirstEventIsRunStarted(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"three-steps.yaml": pipelineThreeSteps,
	})
	env.login("admin@test.local", "admin-pass-123")

	lines, _ := runAndFetchNDJSON(t, env, "three-steps")

	if len(lines) == 0 {
		t.Fatal("NDJSON response was empty")
	}
	first := lines[0]
	typ, _ := first["type"].(string)
	if typ != "pipeline.run.started" {
		t.Errorf("first NDJSON line type = %q, want %q", typ, "pipeline.run.started")
	}
}

// TestDevOpsRunLog_PipelineSlugField verifies that pipeline_slug (not the
// legacy "pipeline" key) is present in run.started and run.completed lines.
func TestDevOpsRunLog_PipelineSlugField(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"three-steps.yaml": pipelineThreeSteps,
	})
	env.login("admin@test.local", "admin-pass-123")

	lines, _ := runAndFetchNDJSON(t, env, "three-steps")

	for _, obj := range lines {
		typ, _ := obj["type"].(string)
		if typ != "pipeline.run.started" && typ != "pipeline.run.completed" {
			continue
		}
		slug, _ := obj["pipeline_slug"].(string)
		if slug == "" {
			t.Errorf("NDJSON line type=%q: missing 'pipeline_slug' field", typ)
		}
		if _, hasLegacy := obj["pipeline"]; hasLegacy {
			t.Errorf("NDJSON line type=%q: has legacy 'pipeline' key; expected 'pipeline_slug'", typ)
		}
	}
}

// TestDevOpsRunLog_MultiStepSequenceOrder verifies the event ordering:
// run.started → (step.started → step.output* → step.completed)+ → run.completed.
func TestDevOpsRunLog_MultiStepSequenceOrder(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"three-steps.yaml": pipelineThreeSteps,
	})
	env.login("admin@test.local", "admin-pass-123")

	lines, _ := runAndFetchNDJSON(t, env, "three-steps")

	if len(lines) == 0 {
		t.Fatal("NDJSON response was empty")
	}

	// First must be run.started.
	firstType, _ := lines[0]["type"].(string)
	if firstType != "pipeline.run.started" {
		t.Errorf("lines[0] type = %q, want pipeline.run.started", firstType)
	}

	// Last must be run.completed.
	lastType, _ := lines[len(lines)-1]["type"].(string)
	if lastType != "pipeline.run.completed" {
		t.Errorf("lines[%d] type = %q, want pipeline.run.completed", len(lines)-1, lastType)
	}

	// For each step: step.started must appear before step.completed.
	lastStartedIdx := map[string]int{}
	for i, obj := range lines {
		typ, _ := obj["type"].(string)
		step, _ := obj["step"].(string)
		switch typ {
		case "pipeline.step.started":
			lastStartedIdx[step] = i
		case "pipeline.step.completed":
			startIdx, ok := lastStartedIdx[step]
			if !ok {
				t.Errorf("step.completed for %q at line %d without prior step.started", step, i)
			} else if i <= startIdx {
				t.Errorf("step.completed[%d] not after step.started[%d] for %q", i, startIdx, step)
			}
		}
	}

	// Three steps should yield three step.started events.
	var stepStarts []string
	for _, obj := range lines {
		typ, _ := obj["type"].(string)
		if typ == "pipeline.step.started" {
			if step, _ := obj["step"].(string); step != "" {
				stepStarts = append(stepStarts, step)
			}
		}
	}
	want := []string{"Step One", "Step Two", "Step Three"}
	if len(stepStarts) != len(want) {
		t.Errorf("got %d step.started lines, want %d: %v", len(stepStarts), len(want), stepStarts)
	} else {
		for i, name := range want {
			if stepStarts[i] != name {
				t.Errorf("step.started[%d] = %q, want %q", i, stepStarts[i], name)
			}
		}
	}
}
