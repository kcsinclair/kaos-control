//go:build integration

package integration

// Milestone 1 — Backend event payload tests (devops-pipeline-log-streaming)
//
// Tests verifying that WebSocket event payloads emitted by the pipeline runner:
//   - Carry all required snake_case JSON keys for each event type.
//   - Include valid RFC 3339 timestamps in every event that declares a timestamp field.
//   - Strip ANSI escape sequences from step output text.
//   - Return a consistent run_id across all events matching the API response.
//
// Events are captured via the Hub's in-process channel (same mechanism as
// devops_ws_test.go) — no real WebSocket connection is required.
//
// Run with: go test ./tests/... -tags integration -run TestDevOpsEventPayload

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// pipelineANSIOutput emits ANSI colour codes on stdout (VT100 CSI red + reset).
const pipelineANSIOutput = `name: ANSI Output
type: build
steps:
  - name: Coloured Step
    command: printf '\033[31mred text\033[0m\n'
`

// pipelineMultiStepPayload is a two-step pipeline used for per-event key checks.
const pipelineMultiStepPayload = `name: Multi Step Payload
type: build
steps:
  - name: Alpha
    command: echo alpha-output
  - name: Beta
    command: echo beta-output
`

// containsANSI returns true when s includes an ESC character (start of any ANSI sequence).
func containsANSI(s string) bool {
	return strings.ContainsRune(s, '\x1b')
}

// isRFC3339 returns true when s parses successfully as an RFC 3339 timestamp.
func isRFC3339(s string) bool {
	if s == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339, s)
	return err == nil
}

// collectPayloadEvents starts a pipeline run and returns all pipeline.* events
// received until pipeline.run.completed (or timeout).
func collectPayloadEvents(t *testing.T, env *testEnv, slug string, timeout time.Duration) []wsEvent {
	t.Helper()
	ch := make(chan []byte, 256)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/"+slug+"/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	resp.Body.Close()

	var events []wsEvent
	deadline := time.After(timeout)
COLLECT:
	for {
		select {
		case raw := <-ch:
			var evt wsEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if len(evt.Type) < 9 || evt.Type[:9] != "pipeline." {
				continue
			}
			events = append(events, evt)
			if evt.Type == "pipeline.run.completed" {
				break COLLECT
			}
		case <-deadline:
			t.Error("timed out waiting for pipeline.run.completed")
			break COLLECT
		}
	}
	return events
}

// TestDevOpsEventPayload_AllFiveTypes verifies that all five expected pipeline
// event types are received for a multi-step pipeline run.
func TestDevOpsEventPayload_AllFiveTypes(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"multi-step-payload.yaml": pipelineMultiStepPayload,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectPayloadEvents(t, env, "multi-step-payload", 15*time.Second)

	typesSeen := make(map[string]bool)
	for _, e := range events {
		typesSeen[e.Type] = true
	}

	required := []string{
		"pipeline.run.started",
		"pipeline.step.started",
		"pipeline.step.output",
		"pipeline.step.completed",
		"pipeline.run.completed",
	}
	for _, typ := range required {
		if !typesSeen[typ] {
			t.Errorf("expected event type %q not received", typ)
		}
	}
}

// TestDevOpsEventPayload_RunStartedKeys verifies that pipeline.run.started uses
// snake_case keys: run_id, pipeline_slug, project.
func TestDevOpsEventPayload_RunStartedKeys(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"multi-step-payload.yaml": pipelineMultiStepPayload,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectPayloadEvents(t, env, "multi-step-payload", 15*time.Second)

	var found bool
	for _, e := range events {
		if e.Type != "pipeline.run.started" {
			continue
		}
		found = true

		if runID, _ := e.Payload["run_id"].(string); runID == "" {
			t.Error("run.started: missing or empty 'run_id'")
		}
		pipelineSlug, _ := e.Payload["pipeline_slug"].(string)
		if pipelineSlug != "multi-step-payload" {
			t.Errorf("run.started: pipeline_slug = %q, want %q", pipelineSlug, "multi-step-payload")
		}
		if project, _ := e.Payload["project"].(string); project == "" {
			t.Error("run.started: missing or empty 'project'")
		}
		// Legacy key 'pipeline' must not be present; correct key is 'pipeline_slug'.
		if _, hasLegacy := e.Payload["pipeline"]; hasLegacy {
			t.Error("run.started: payload has legacy 'pipeline' key; expected 'pipeline_slug'")
		}
		break
	}
	if !found {
		t.Error("pipeline.run.started event not received")
	}
}

// TestDevOpsEventPayload_StepStartedKeys verifies that pipeline.step.started
// carries: run_id, pipeline_slug, step, step_index, timestamp (RFC 3339).
func TestDevOpsEventPayload_StepStartedKeys(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"multi-step-payload.yaml": pipelineMultiStepPayload,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectPayloadEvents(t, env, "multi-step-payload", 15*time.Second)

	var found bool
	for _, e := range events {
		if e.Type != "pipeline.step.started" {
			continue
		}
		found = true

		if runID, _ := e.Payload["run_id"].(string); runID == "" {
			t.Error("step.started: missing 'run_id'")
		}
		if slug, _ := e.Payload["pipeline_slug"].(string); slug == "" {
			t.Error("step.started: missing 'pipeline_slug'")
		}
		if step, _ := e.Payload["step"].(string); step == "" {
			t.Error("step.started: missing 'step'")
		}
		if _, ok := e.Payload["step_index"]; !ok {
			t.Error("step.started: missing 'step_index'")
		}
		ts, _ := e.Payload["timestamp"].(string)
		if !isRFC3339(ts) {
			t.Errorf("step.started: timestamp %q is not valid RFC 3339", ts)
		}
		break
	}
	if !found {
		t.Error("pipeline.step.started event not received")
	}
}

// TestDevOpsEventPayload_StepOutputKeys verifies that pipeline.step.output
// carries: run_id, pipeline_slug, step, step_index, text, stream, timestamp (RFC 3339).
func TestDevOpsEventPayload_StepOutputKeys(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"multi-step-payload.yaml": pipelineMultiStepPayload,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectPayloadEvents(t, env, "multi-step-payload", 15*time.Second)

	var found bool
	for _, e := range events {
		if e.Type != "pipeline.step.output" {
			continue
		}
		found = true

		if runID, _ := e.Payload["run_id"].(string); runID == "" {
			t.Error("step.output: missing 'run_id'")
		}
		if slug, _ := e.Payload["pipeline_slug"].(string); slug == "" {
			t.Error("step.output: missing 'pipeline_slug'")
		}
		if step, _ := e.Payload["step"].(string); step == "" {
			t.Error("step.output: missing 'step'")
		}
		if _, ok := e.Payload["step_index"]; !ok {
			t.Error("step.output: missing 'step_index'")
		}
		if text, _ := e.Payload["text"].(string); text == "" {
			t.Error("step.output: missing or empty 'text'")
		}
		stream, _ := e.Payload["stream"].(string)
		if stream != "stdout" && stream != "stderr" {
			t.Errorf("step.output: stream = %q, want 'stdout' or 'stderr'", stream)
		}
		ts, _ := e.Payload["timestamp"].(string)
		if !isRFC3339(ts) {
			t.Errorf("step.output: timestamp %q is not valid RFC 3339", ts)
		}
		break
	}
	if !found {
		t.Error("pipeline.step.output event not received")
	}
}

// TestDevOpsEventPayload_StepCompletedKeys verifies that pipeline.step.completed
// carries: run_id, pipeline_slug, step, step_index, status, exit_code, duration_seconds.
func TestDevOpsEventPayload_StepCompletedKeys(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"multi-step-payload.yaml": pipelineMultiStepPayload,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectPayloadEvents(t, env, "multi-step-payload", 15*time.Second)

	var found bool
	for _, e := range events {
		if e.Type != "pipeline.step.completed" {
			continue
		}
		found = true

		if runID, _ := e.Payload["run_id"].(string); runID == "" {
			t.Error("step.completed: missing 'run_id'")
		}
		if slug, _ := e.Payload["pipeline_slug"].(string); slug == "" {
			t.Error("step.completed: missing 'pipeline_slug'")
		}
		if step, _ := e.Payload["step"].(string); step == "" {
			t.Error("step.completed: missing 'step'")
		}
		if _, ok := e.Payload["step_index"]; !ok {
			t.Error("step.completed: missing 'step_index'")
		}
		if status, _ := e.Payload["status"].(string); status == "" {
			t.Error("step.completed: missing 'status'")
		}
		if _, ok := e.Payload["exit_code"]; !ok {
			t.Error("step.completed: missing 'exit_code'")
		}
		if _, ok := e.Payload["duration_seconds"]; !ok {
			t.Error("step.completed: missing 'duration_seconds'")
		}
		break
	}
	if !found {
		t.Error("pipeline.step.completed event not received")
	}
}

// TestDevOpsEventPayload_RunCompletedKeys verifies that pipeline.run.completed
// carries: run_id, pipeline_slug, project, status, duration_seconds.
func TestDevOpsEventPayload_RunCompletedKeys(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"multi-step-payload.yaml": pipelineMultiStepPayload,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectPayloadEvents(t, env, "multi-step-payload", 15*time.Second)

	var found bool
	for _, e := range events {
		if e.Type != "pipeline.run.completed" {
			continue
		}
		found = true

		if runID, _ := e.Payload["run_id"].(string); runID == "" {
			t.Error("run.completed: missing 'run_id'")
		}
		if slug, _ := e.Payload["pipeline_slug"].(string); slug == "" {
			t.Error("run.completed: missing 'pipeline_slug'")
		}
		if project, _ := e.Payload["project"].(string); project == "" {
			t.Error("run.completed: missing 'project'")
		}
		if status, _ := e.Payload["status"].(string); status == "" {
			t.Error("run.completed: missing 'status'")
		}
		if _, ok := e.Payload["duration_seconds"]; !ok {
			t.Error("run.completed: missing 'duration_seconds'")
		}
		break
	}
	if !found {
		t.Error("pipeline.run.completed event not received")
	}
}

// TestDevOpsEventPayload_TimestampsAreRFC3339 verifies that every timestamp
// field across all received events parses as a valid RFC 3339 string.
func TestDevOpsEventPayload_TimestampsAreRFC3339(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"multi-step-payload.yaml": pipelineMultiStepPayload,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectPayloadEvents(t, env, "multi-step-payload", 15*time.Second)

	for _, e := range events {
		ts, hasTS := e.Payload["timestamp"]
		if !hasTS {
			// Not all event types carry a timestamp; skip those.
			continue
		}
		tsStr, _ := ts.(string)
		if !isRFC3339(tsStr) {
			t.Errorf("event %q: timestamp %q is not valid RFC 3339", e.Type, tsStr)
		}
	}
}

// TestDevOpsEventPayload_ANSIStripped verifies that when a step emits
// ANSI-coloured output, the text field in pipeline.step.output contains no
// ANSI escape sequences.
func TestDevOpsEventPayload_ANSIStripped(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"ansi-output.yaml": pipelineANSIOutput,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectPayloadEvents(t, env, "ansi-output", 15*time.Second)

	var checkedOutput bool
	for _, e := range events {
		if e.Type != "pipeline.step.output" {
			continue
		}
		text, _ := e.Payload["text"].(string)
		if text == "" {
			continue
		}
		checkedOutput = true
		if containsANSI(text) {
			t.Errorf("step.output text contains ANSI escape sequences: %q", text)
		}
	}

	if !checkedOutput {
		t.Error("no pipeline.step.output events with non-empty text were received")
	}
}

// TestDevOpsEventPayload_RunIDConsistentAcrossEvents verifies that every
// pipeline event for a single run carries the same run_id that was returned by
// the trigger API.
func TestDevOpsEventPayload_RunIDConsistentAcrossEvents(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"multi-step-payload.yaml": pipelineMultiStepPayload,
	})
	env.login("admin@test.local", "admin-pass-123")

	ch := make(chan []byte, 256)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/multi-step-payload/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	apiData := readJSON(t, resp)
	expectedRunID, _ := apiData["run_id"].(string)
	if expectedRunID == "" {
		t.Fatal("trigger API did not return run_id")
	}

	var events []wsEvent
	deadline := time.After(15 * time.Second)
COLLECT:
	for {
		select {
		case raw := <-ch:
			var evt wsEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if len(evt.Type) < 9 || evt.Type[:9] != "pipeline." {
				continue
			}
			events = append(events, evt)
			if evt.Type == "pipeline.run.completed" {
				break COLLECT
			}
		case <-deadline:
			t.Error("timed out waiting for pipeline.run.completed")
			break COLLECT
		}
	}

	if len(events) == 0 {
		t.Fatal("no pipeline events received")
	}

	for _, e := range events {
		runID, _ := e.Payload["run_id"].(string)
		if runID == "" {
			t.Errorf("event %q: missing run_id in payload", e.Type)
			continue
		}
		if runID != expectedRunID {
			t.Errorf("event %q: run_id = %q, want %q (from trigger API)", e.Type, runID, expectedRunID)
		}
	}
}
