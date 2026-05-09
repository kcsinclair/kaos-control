// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestone 4 — Step Execution, Timeout & Cancellation Tests
//
// Tests verifying step-level behaviour:
//   - Steps execute in declared order (verified via hub output events)
//   - Non-zero exit stops the run; subsequent steps are not executed
//   - A step exceeding its timeout is killed; run is marked failed
//   - POST .../cancel on an active run stops execution
//   - POST .../cancel on a pipeline with no active run returns 404
//   - Cancelled run's remaining steps are not executed

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestDevopsSteps_ExecuteInOrder verifies that steps execute sequentially in
// declared order by inspecting the step names in pipeline.step.started events.
func TestDevopsSteps_ExecuteInOrder(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"ordered.yaml": pipelineOrderedOutput,
	})
	env.login("admin@test.local", "admin-pass-123")

	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/ordered/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	resp.Body.Close()

	// Collect step.started events in order.
	var stepOrder []string
	timeout := time.After(15 * time.Second)

COLLECT:
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			switch evt.Type {
			case "pipeline.step.started":
				if step, _ := evt.Payload["step"].(string); step != "" {
					stepOrder = append(stepOrder, step)
				}
			case "pipeline.run.completed":
				break COLLECT
			}
		case <-timeout:
			t.Fatal("timed out waiting for pipeline.run.completed")
		}
	}

	want := []string{"First", "Second", "Third"}
	if len(stepOrder) != len(want) {
		t.Fatalf("got %d step.started events, want %d: %v", len(stepOrder), len(want), stepOrder)
	}
	for i, name := range want {
		if stepOrder[i] != name {
			t.Errorf("step[%d] = %q, want %q", i, stepOrder[i], name)
		}
	}
}

// TestDevopsSteps_NonZeroExitStopsPipeline verifies that when a step exits
// with a non-zero code, the pipeline stops and subsequent steps are not run.
func TestDevopsSteps_NonZeroExitStopsPipeline(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"multi-fail.yaml": pipelineMultiStepWithFail,
	})
	env.login("admin@test.local", "admin-pass-123")

	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/multi-fail/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	resp.Body.Close()

	var startedSteps []string
	var runStatus string
	timeout := time.After(15 * time.Second)

COLLECT:
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			switch evt.Type {
			case "pipeline.step.started":
				if step, _ := evt.Payload["step"].(string); step != "" {
					startedSteps = append(startedSteps, step)
				}
			case "pipeline.run.completed":
				runStatus, _ = evt.Payload["status"].(string)
				break COLLECT
			}
		case <-timeout:
			t.Fatal("timed out waiting for pipeline.run.completed")
		}
	}

	// Only "First OK" and "Then Fail" should have started; "Never Runs" must not.
	if len(startedSteps) != 2 {
		t.Errorf("expected 2 steps to start (stopping after failure), got %d: %v", len(startedSteps), startedSteps)
	}
	for _, s := range startedSteps {
		if s == "Never Runs" {
			t.Error("'Never Runs' step should not have started after failure")
		}
	}

	if runStatus != "failed" {
		t.Errorf("run status = %q, want %q", runStatus, "failed")
	}
}

// TestDevopsSteps_TimeoutKillsStep verifies that a step exceeding its configured
// timeout is killed and the run is marked as failed.
func TestDevopsSteps_TimeoutKillsStep(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"timeout-step.yaml": pipelineTimeoutStep,
	})
	env.login("admin@test.local", "admin-pass-123")

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/timeout-step/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	resp.Body.Close()

	var runStatus string
	// The step has a 1s timeout, so the run should complete quickly.
	timeout := time.After(15 * time.Second)

COLLECT:
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "pipeline.run.completed" {
				runStatus, _ = evt.Payload["status"].(string)
				break COLLECT
			}
		case <-timeout:
			t.Fatal("timed out waiting for run.completed after step timeout")
		}
	}

	if runStatus != "failed" {
		t.Errorf("run status = %q after timeout, want %q", runStatus, "failed")
	}
}

// TestDevopsSteps_CancelActiveRun verifies that cancelling an active run stops
// execution and the run.completed event reports status=cancelled.
func TestDevopsSteps_CancelActiveRun(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"slow-step.yaml": pipelineSlowStep,
	})
	env.login("admin@test.local", "admin-pass-123")

	ch := make(chan []byte, 64)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	// Start the slow pipeline.
	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	resp.Body.Close()

	// Wait until step has actually started (to make cancel meaningful).
	stepStarted := make(chan struct{}, 1)
	go func() {
		for raw := range ch {
			var evt struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "pipeline.step.started" {
				select {
				case stepStarted <- struct{}{}:
				default:
				}
				return
			}
		}
	}()

	select {
	case <-stepStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for step to start before cancel")
	}

	// Cancel the run.
	cancelResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/cancel", nil)
	requireStatus(t, cancelResp, http.StatusOK)
	data := readJSON(t, cancelResp)
	if cancelled, _ := data["cancelled"].(bool); !cancelled {
		t.Error("expected cancelled=true in cancel response")
	}

	// Wait for run to finish.
	waitForRunComplete(t, env, "slow-step", 10*time.Second)

	// Collect the run.completed event.
	var runStatus string
	collectTimeout := time.After(5 * time.Second)
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "pipeline.run.completed" {
				runStatus, _ = evt.Payload["status"].(string)
				goto done
			}
		case <-collectTimeout:
			goto done
		}
	}
done:

	if runStatus != "cancelled" {
		t.Errorf("run status after cancel = %q, want %q", runStatus, "cancelled")
	}
}

// TestDevopsSteps_CancelNoActiveRun verifies that cancelling a pipeline with no
// active run returns 404.
func TestDevopsSteps_CancelNoActiveRun(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// No run has been started — cancel should return 404.
	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/cancel", nil)
	requireStatus(t, resp, http.StatusNotFound)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "not_found" {
		t.Errorf("expected error code 'not_found', got %q", code)
	}
}

// TestDevopsSteps_CancelledRunSkipsRemainingSteps verifies that after cancellation,
// remaining steps are not started.
func TestDevopsSteps_CancelledRunSkipsRemainingSteps(t *testing.T) {
	// Use a two-step pipeline where the first step sleeps (giving us time to cancel)
	// and the second step would echo a marker.
	const cancelTestPipeline = `name: Cancel Multi
type: build
steps:
  - name: Slow First
    command: sleep 30
  - name: Second Should Not Run
    command: echo SHOULD_NOT_RUN
`
	env := newDevopsTestEnv(t, map[string]string{
		"cancel-multi.yaml": cancelTestPipeline,
	})
	env.login("admin@test.local", "admin-pass-123")

	ch := make(chan []byte, 128)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/cancel-multi/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	resp.Body.Close()

	// Wait for first step to start.
	stepStarted := make(chan struct{}, 1)
	go func() {
		for raw := range ch {
			var evt struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "pipeline.step.started" {
				select {
				case stepStarted <- struct{}{}:
				default:
				}
				return
			}
		}
	}()

	select {
	case <-stepStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for first step to start")
	}

	// Cancel the run.
	cancelResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/cancel-multi/cancel", nil)
	cancelResp.Body.Close()
	waitForRunComplete(t, env, "cancel-multi", 10*time.Second)

	// Drain remaining events and confirm no second step.started was received.
	var secondStepStarted bool
	drain := time.After(1 * time.Second)
DRAIN:
	for {
		select {
		case raw := <-ch:
			var evt struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			}
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "pipeline.step.started" {
				if step, _ := evt.Payload["step"].(string); step == "Second Should Not Run" {
					secondStepStarted = true
				}
			}
		case <-drain:
			break DRAIN
		}
	}

	if secondStepStarted {
		t.Error("second step should not have started after cancellation")
	}
}
