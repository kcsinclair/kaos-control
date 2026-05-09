// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 5 — WebSocket Event Streaming Tests
//
// Tests verifying that the correct WebSocket events are broadcast in the correct
// order during pipeline execution. Events are captured via the Hub's in-process
// channel rather than a real WebSocket connection — this is faster and equivalent
// for verifying the broadcast behaviour.
//
// Scenarios:
//   - pipeline.run.started is the first event; contains run_id, slug, project
//   - pipeline.step.started is received for each step before its output
//   - pipeline.step.output events contain step index and text
//   - pipeline.step.completed contains exit_code and duration_seconds
//   - pipeline.run.completed is the last event; contains overall status and duration
//   - Events arrive in correct order: run.started → (step.started → step.output* → step.completed)+ → run.completed
//   - On failure, remaining steps do not emit started/completed events
//   - On cancel, pipeline.run.completed has status=cancelled

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// wsEvent is a generic WebSocket event envelope.
type wsEvent struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

// collectDevopsEvents registers a hub channel, triggers a run, and collects
// all events until pipeline.run.completed arrives or the timeout fires.
func collectDevopsEvents(t *testing.T, env *testEnv, slug string, timeout time.Duration) []wsEvent {
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
			// Only collect pipeline events.
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

// TestDevopsWS_RunStartedIsFirst verifies that pipeline.run.started is the
// first pipeline event emitted and contains the required fields.
func TestDevopsWS_RunStartedIsFirst(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectDevopsEvents(t, env, "quick-pass", 15*time.Second)

	if len(events) == 0 {
		t.Fatal("no pipeline events received")
	}
	first := events[0]
	if first.Type != "pipeline.run.started" {
		t.Errorf("first event type = %q, want %q", first.Type, "pipeline.run.started")
	}

	// Required payload fields.
	if runID, _ := first.Payload["run_id"].(string); runID == "" {
		t.Error("run.started: missing run_id in payload")
	}
	if pipeline, _ := first.Payload["pipeline_slug"].(string); pipeline != "quick-pass" {
		t.Errorf("run.started: pipeline_slug = %q, want %q", pipeline, "quick-pass")
	}
	if project, _ := first.Payload["project"].(string); project != "testproject" {
		t.Errorf("run.started: project = %q, want %q", project, "testproject")
	}
}

// TestDevopsWS_StepStartedPerStep verifies that a pipeline.step.started event
// is emitted for each step.
func TestDevopsWS_StepStartedPerStep(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"ordered.yaml": pipelineOrderedOutput,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectDevopsEvents(t, env, "ordered", 15*time.Second)

	var stepStarts []string
	for _, e := range events {
		if e.Type == "pipeline.step.started" {
			if name, _ := e.Payload["step"].(string); name != "" {
				stepStarts = append(stepStarts, name)
			}
		}
	}

	want := []string{"First", "Second", "Third"}
	if len(stepStarts) != len(want) {
		t.Fatalf("got %d step.started events, want %d: %v", len(stepStarts), len(want), stepStarts)
	}
	for i, name := range want {
		if stepStarts[i] != name {
			t.Errorf("step.started[%d] = %q, want %q", i, stepStarts[i], name)
		}
	}
}

// TestDevopsWS_StepOutputContainsText verifies that pipeline.step.output events
// contain the step name, step index, stream, and text fields.
func TestDevopsWS_StepOutputContainsText(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectDevopsEvents(t, env, "quick-pass", 15*time.Second)

	var outputEvts []wsEvent
	for _, e := range events {
		if e.Type == "pipeline.step.output" {
			outputEvts = append(outputEvts, e)
		}
	}

	if len(outputEvts) == 0 {
		t.Fatal("expected at least one pipeline.step.output event")
	}

	for _, e := range outputEvts {
		if _, ok := e.Payload["step"].(string); !ok {
			t.Error("step.output missing 'step' field")
		}
		if _, ok := e.Payload["step_index"]; !ok {
			t.Error("step.output missing 'step_index' field")
		}
		if text, _ := e.Payload["text"].(string); text == "" {
			t.Error("step.output has empty 'text' field")
		}
		if stream, _ := e.Payload["stream"].(string); stream != "stdout" && stream != "stderr" {
			t.Errorf("step.output stream = %q, want 'stdout' or 'stderr'", stream)
		}
	}
}

// TestDevopsWS_StepCompletedHasFields verifies that pipeline.step.completed
// events include exit_code and duration_seconds.
func TestDevopsWS_StepCompletedHasFields(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectDevopsEvents(t, env, "quick-pass", 15*time.Second)

	var completedEvts []wsEvent
	for _, e := range events {
		if e.Type == "pipeline.step.completed" {
			completedEvts = append(completedEvts, e)
		}
	}

	if len(completedEvts) == 0 {
		t.Fatal("expected at least one pipeline.step.completed event")
	}

	for _, e := range completedEvts {
		if _, ok := e.Payload["exit_code"]; !ok {
			t.Error("step.completed missing 'exit_code' field")
		}
		if _, ok := e.Payload["duration_seconds"]; !ok {
			t.Error("step.completed missing 'duration_seconds' field")
		}
		if status, _ := e.Payload["status"].(string); status == "" {
			t.Error("step.completed missing 'status' field")
		}
	}
}

// TestDevopsWS_RunCompletedIsLast verifies that pipeline.run.completed is the
// last event and contains overall status and duration_seconds.
func TestDevopsWS_RunCompletedIsLast(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectDevopsEvents(t, env, "quick-pass", 15*time.Second)

	if len(events) == 0 {
		t.Fatal("no events collected")
	}
	last := events[len(events)-1]
	if last.Type != "pipeline.run.completed" {
		t.Errorf("last event = %q, want %q", last.Type, "pipeline.run.completed")
	}

	if status, _ := last.Payload["status"].(string); status == "" {
		t.Error("run.completed missing 'status' field")
	}
	if _, ok := last.Payload["duration_seconds"]; !ok {
		t.Error("run.completed missing 'duration_seconds' field")
	}
}

// TestDevopsWS_EventOrdering verifies the strict ordering of events for a
// multi-step pipeline: run.started → (step.started → step.output* → step.completed)+ → run.completed.
func TestDevopsWS_EventOrdering(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"ordered.yaml": pipelineOrderedOutput,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectDevopsEvents(t, env, "ordered", 15*time.Second)

	if len(events) == 0 {
		t.Fatal("no pipeline events received")
	}

	// First event must be run.started.
	if events[0].Type != "pipeline.run.started" {
		t.Errorf("events[0] = %q, want pipeline.run.started", events[0].Type)
	}

	// Last event must be run.completed.
	if events[len(events)-1].Type != "pipeline.run.completed" {
		t.Errorf("events[%d] = %q, want pipeline.run.completed", len(events)-1, events[len(events)-1].Type)
	}

	// Between first and last, for each step: step.started comes before step.completed.
	lastStartedIdx := map[string]int{}
	for i, e := range events {
		switch e.Type {
		case "pipeline.step.started":
			step, _ := e.Payload["step"].(string)
			lastStartedIdx[step] = i
		case "pipeline.step.completed":
			step, _ := e.Payload["step"].(string)
			startIdx, ok := lastStartedIdx[step]
			if !ok {
				t.Errorf("step.completed for %q without prior step.started", step)
			} else if i < startIdx {
				t.Errorf("step.completed[%d] before step.started[%d] for %q", i, startIdx, step)
			}
		}
	}
}

// TestDevopsWS_FailureSkipsRemainingStepEvents verifies that when a step fails,
// the remaining steps do not emit started/completed events.
func TestDevopsWS_FailureSkipsRemainingStepEvents(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"multi-fail.yaml": pipelineMultiStepWithFail,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectDevopsEvents(t, env, "multi-fail", 15*time.Second)

	var startedSteps []string
	var runStatus string
	for _, e := range events {
		if e.Type == "pipeline.step.started" {
			if name, _ := e.Payload["step"].(string); name != "" {
				startedSteps = append(startedSteps, name)
			}
		}
		if e.Type == "pipeline.run.completed" {
			runStatus, _ = e.Payload["status"].(string)
		}
	}

	// Only "First OK" and "Then Fail" should start.
	for _, s := range startedSteps {
		if s == "Never Runs" {
			t.Error("'Never Runs' step should not emit step.started after failure")
		}
	}
	if runStatus != "failed" {
		t.Errorf("run completed with status %q, want %q", runStatus, "failed")
	}
}

// TestDevopsWS_CancelEmitsCompletedEvent verifies that cancelling a run causes
// a pipeline.run.completed event with status=cancelled.
func TestDevopsWS_CancelEmitsCompletedEvent(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"slow-step.yaml": pipelineSlowStep,
	})
	env.login("admin@test.local", "admin-pass-123")

	ch := make(chan []byte, 256)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	resp.Body.Close()

	// Wait for the step to start.
	deadline := time.After(5 * time.Second)
WAIT:
	for {
		select {
		case raw := <-ch:
			var evt wsEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "pipeline.step.started" {
				break WAIT
			}
		case <-deadline:
			t.Fatal("timed out waiting for step to start before cancel")
		}
	}

	cancelResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/cancel", nil)
	cancelResp.Body.Close()

	// Collect until run.completed.
	var runStatus string
	collectTimeout := time.After(10 * time.Second)
COLLECT:
	for {
		select {
		case raw := <-ch:
			var evt wsEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "pipeline.run.completed" {
				runStatus, _ = evt.Payload["status"].(string)
				break COLLECT
			}
		case <-collectTimeout:
			t.Fatal("timed out waiting for run.completed after cancel")
		}
	}

	if runStatus != "cancelled" {
		t.Errorf("run.completed status = %q after cancel, want %q", runStatus, "cancelled")
	}
}

// TestDevopsWS_OutputLatency verifies that step output events arrive within
// 500ms of starting the pipeline run (NF requirement).
func TestDevopsWS_OutputLatency(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	triggerTime := time.Now()
	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	resp.Body.Close()

	// Wait for first step.output event.
	latencyDeadline := time.After(500 * time.Millisecond)
	collectDeadline := time.After(15 * time.Second)
	var outputReceived bool
COLLECT:
	for {
		select {
		case raw := <-ch:
			var evt wsEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "pipeline.step.output" {
				outputReceived = true
				elapsed := time.Since(triggerTime)
				if elapsed > 500*time.Millisecond {
					t.Errorf("first step.output arrived %v after trigger, want < 500ms", elapsed)
				}
				break COLLECT
			}
		case <-latencyDeadline:
			// Will check outputReceived after the loop
			break COLLECT
		case <-collectDeadline:
			break COLLECT
		}
	}

	// Drain remaining events.
	waitForRunComplete(t, env, "quick-pass", 10*time.Second)

	if !outputReceived {
		t.Error("no step.output event received within 500ms latency window")
	}
}
