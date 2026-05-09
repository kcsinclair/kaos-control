// SPDX-License-Identifier: AGPL-3.0-or-later

package devops

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"time"

	"github.com/kaos-control/kaos-control/internal/hub"
)

// activeRun holds the cancellation function and a done channel for one running
// pipeline instance.
type activeRun struct {
	runID  string
	cancel context.CancelFunc
	done   <-chan struct{}
}

// Runner manages concurrently active pipeline runs. At most one run per
// pipeline slug may be active at a time.
type Runner struct {
	mu      sync.Mutex
	bySlug  map[string]*activeRun // slug → active run
	byRunID map[string]string     // run_id → slug

	// onEvent is called for every event emitted during a run, in addition to
	// broadcasting to the hub. Used by the log persistence layer.
	onEvent func(runID string, eventType string, payload any)
}

// NewRunner creates an idle Runner with no active runs.
func NewRunner() *Runner {
	return &Runner{
		bySlug:  make(map[string]*activeRun),
		byRunID: make(map[string]string),
	}
}

// SetEventHook registers a callback invoked for every pipeline event before
// it is broadcast. This is used for log persistence. Only one hook is
// supported; calling again replaces the previous one.
func (r *Runner) SetEventHook(fn func(runID, eventType string, payload any)) {
	r.mu.Lock()
	r.onEvent = fn
	r.mu.Unlock()
}

// IsRunning reports whether a pipeline with the given slug is currently active.
func (r *Runner) IsRunning(slug string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.bySlug[slug]
	return ok
}

// ActiveRunID returns the run_id of the currently active run for slug, and
// true. Returns "", false if no run is active.
func (r *Runner) ActiveRunID(slug string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ar, ok := r.bySlug[slug]
	if !ok {
		return "", false
	}
	return ar.runID, true
}

// Start launches pipeline in a background goroutine. It returns the run_id
// immediately. Returns an error if the pipeline slug is already running.
func (r *Runner) Start(pipeline Pipeline, projectDir string, h *hub.Hub, projectID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.bySlug[pipeline.Slug]; ok {
		return "", fmt.Errorf("pipeline %q is already running", pipeline.Slug)
	}

	runID := newRunID()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	ar := &activeRun{runID: runID, cancel: cancel, done: done}
	r.bySlug[pipeline.Slug] = ar
	r.byRunID[runID] = pipeline.Slug

	// Capture the hook at launch time to avoid holding the lock during execution.
	hook := r.onEvent

	go func() {
		defer close(done)
		defer func() {
			r.mu.Lock()
			delete(r.bySlug, pipeline.Slug)
			delete(r.byRunID, runID)
			r.mu.Unlock()
		}()
		executeRun(ctx, runID, pipeline, projectDir, h, projectID, hook)
	}()

	return runID, nil
}

// Cancel signals the active run identified by runID to stop, waits for it to
// finish, and returns nil. Returns an error if runID is unknown.
func (r *Runner) Cancel(runID string) error {
	r.mu.Lock()
	slug, ok := r.byRunID[runID]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("no active run with id %q", runID)
	}
	ar := r.bySlug[slug]
	r.mu.Unlock()

	ar.cancel()
	<-ar.done
	return nil
}

// broadcast emits an event to the hub and, if a hook is set, to the hook.
func broadcast(h *hub.Hub, hook func(string, string, any), runID, eventType string, payload any) {
	if h != nil {
		h.Broadcast(hub.Event{Type: eventType, Payload: payload})
	}
	if hook != nil {
		hook(runID, eventType, payload)
	}
}

// executeRun runs all steps of pipeline sequentially in projectDir.
// It broadcasts hub events and calls the optional hook for each event.
func executeRun(
	ctx context.Context,
	runID string,
	pipeline Pipeline,
	projectDir string,
	h *hub.Hub,
	projectID string,
	hook func(string, string, any),
) {
	start := time.Now()
	slog.Info("devops: run started", "run_id", runID, "pipeline", pipeline.Slug, "project", projectID)

	broadcast(h, hook, runID, EventRunStarted, RunStartedPayload{
		RunID:    runID,
		Pipeline: pipeline.Slug,
		Project:  projectID,
	})

	overallStatus := string(StepPassed)
	skip := false // set true after any failure or cancellation

	for i, step := range pipeline.Steps {
		if skip {
			// Remaining steps are left as pending; no events emitted.
			continue
		}

		broadcast(h, hook, runID, EventStepStarted, StepStartedPayload{
			RunID:     runID,
			Pipeline:  pipeline.Slug,
			Step:      step.Name,
			StepIndex: i,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		})

		stepStart := time.Now()
		stepStatus, exitCode := runStep(ctx, runID, pipeline.Slug, step, i, projectDir, h, hook)

		duration := time.Since(stepStart).Seconds()
		slog.Info("devops: step completed",
			"run_id", runID, "pipeline", pipeline.Slug,
			"step", step.Name, "status", stepStatus, "exit_code", exitCode)

		broadcast(h, hook, runID, EventStepCompleted, StepCompletedPayload{
			RunID:           runID,
			Pipeline:        pipeline.Slug,
			Step:            step.Name,
			StepIndex:       i,
			Status:          string(stepStatus),
			ExitCode:        exitCode,
			DurationSeconds: duration,
		})

		if stepStatus != StepPassed {
			overallStatus = string(stepStatus)
			skip = true
		}
	}

	totalDuration := time.Since(start).Seconds()
	slog.Info("devops: run completed",
		"run_id", runID, "pipeline", pipeline.Slug,
		"status", overallStatus, "duration_s", totalDuration)

	broadcast(h, hook, runID, EventRunCompleted, RunCompletedPayload{
		RunID:           runID,
		Pipeline:        pipeline.Slug,
		Project:         projectID,
		Status:          overallStatus,
		DurationSeconds: totalDuration,
	})
}

// runStep executes a single step and streams its output as hub events.
// Returns the step's final status and exit code.
func runStep(
	ctx context.Context,
	runID, pipelineSlug string,
	step Step,
	stepIdx int,
	workDir string,
	h *hub.Hub,
	hook func(string, string, any),
) (StepStatus, int) {
	// Apply per-step timeout on top of any parent cancellation.
	stepCtx, cancel := context.WithTimeout(ctx, step.Timeout)
	defer cancel()

	// Commands come exclusively from YAML files on disk; no request data is
	// interpolated, so there is no shell injection risk.
	cmd := exec.CommandContext(stepCtx, "sh", "-c", step.Command)
	cmd.Dir = workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("devops: stdout pipe", "run_id", runID, "step", step.Name, "err", err)
		return StepFailed, -1
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		slog.Error("devops: stderr pipe", "run_id", runID, "step", step.Name, "err", err)
		return StepFailed, -1
	}

	if err := cmd.Start(); err != nil {
		slog.Error("devops: start command", "run_id", runID, "step", step.Name, "err", err)
		return StepFailed, -1
	}

	// Stream stdout and stderr concurrently.
	var wg sync.WaitGroup
	streamLines := func(r io.Reader, stream string) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			broadcast(h, hook, runID, EventStepOutput, StepOutputPayload{
				RunID:     runID,
				Pipeline:  pipelineSlug,
				Step:      step.Name,
				StepIndex: stepIdx,
				Text:      StripANSI(scanner.Text()),
				Stream:    stream,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
		}
	}

	wg.Add(2)
	go streamLines(stdout, "stdout")
	go streamLines(stderr, "stderr")
	wg.Wait()

	err = cmd.Wait()

	// Determine status: distinguish context cancellation (parent cancel vs timeout).
	if err != nil {
		if stepCtx.Err() != nil {
			// Either the parent context was cancelled or the step timed out.
			if ctx.Err() != nil {
				return StepCancelled, -1
			}
			// Timeout.
			slog.Warn("devops: step timed out", "run_id", runID, "step", step.Name, "timeout", step.Timeout)
			return StepFailed, -1
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return StepFailed, exitErr.ExitCode()
		}
		return StepFailed, -1
	}

	return StepPassed, 0
}

func newRunID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
