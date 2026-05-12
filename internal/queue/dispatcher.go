// SPDX-License-Identifier: AGPL-3.0-or-later

package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/kaos-control/kaos-control/internal/hub"
)

// Config holds tunable parameters for the Dispatcher.
type Config struct {
	// FallbackPause is the pause duration when the rate-limit text cannot be
	// parsed to extract a reset time.
	FallbackPause time.Duration
	// ResumeGrace is added to the parsed reset time to compute paused_until.
	// Provides a small buffer so the rate limit has definitively cleared before
	// the queue resumes.
	ResumeGrace time.Duration
	// MaxAttempts is the maximum number of times a job is re-queued after a
	// rate-limit failure. Jobs that exceed this threshold are dropped.
	MaxAttempts int
	// TickInterval controls how often the dispatcher checks for pending work.
	// Defaults to 1 second.
	TickInterval time.Duration
	// ClockFn is the clock source. Defaults to time.Now when nil.
	// Inject a deterministic clock in tests.
	ClockFn func() time.Time
}

func (c *Config) clock() time.Time {
	if c.ClockFn != nil {
		return c.ClockFn()
	}
	return time.Now()
}

func (c *Config) tickInterval() time.Duration {
	if c.TickInterval > 0 {
		return c.TickInterval
	}
	return time.Second
}

func (c *Config) maxAttempts() int {
	if c.MaxAttempts > 0 {
		return c.MaxAttempts
	}
	return 5
}

func (c *Config) fallbackPause() time.Duration {
	if c.FallbackPause > 0 {
		return c.FallbackPause
	}
	return 30 * time.Minute
}

func (c *Config) resumeGrace() time.Duration {
	if c.ResumeGrace > 0 {
		return c.ResumeGrace
	}
	return 5 * time.Minute
}

// ProjectAccess provides the dispatcher with everything it needs from one project.
type ProjectAccess struct {
	// StartRun starts an agent run on the project; returns the run ID.
	StartRun func(ctx context.Context, agentName, targetPath string) (string, error)
	// ArtifactStatus returns the current status of the artifact at relPath,
	// or "" when the artifact is not found in the index.
	ArtifactStatus func(relPath string) string
	// Hub is the project's WebSocket hub; used to subscribe for run-completion
	// events (agent.finished / agent.failed).
	Hub *hub.Hub
}

// ProjectLookup maps a project name to its runtime access handle.
type ProjectLookup func(name string) (ProjectAccess, bool)

// Dispatcher is the single-goroutine queue worker.
type Dispatcher struct {
	store    *Store
	lookup   ProjectLookup
	appHub   *hub.Hub // app-level hub for queue.* events
	cfg      Config

	mu          sync.Mutex
	pausedUntil time.Time // zero = not rate-limit-paused
	manualPause bool      // true when paused via Pause(); cleared only by Resume()
}

// New creates a Dispatcher.
func New(store *Store, lookup ProjectLookup, appHub *hub.Hub, cfg Config) *Dispatcher {
	return &Dispatcher{
		store:  store,
		lookup: lookup,
		appHub: appHub,
		cfg:    cfg,
	}
}

// Start spawns the dispatcher goroutine. It returns immediately; the goroutine
// runs until ctx is cancelled.
func (d *Dispatcher) Start(ctx context.Context) {
	// Restore persisted pause state.
	if paused, until, _, err := d.store.GetPauseState(); err == nil && paused {
		d.mu.Lock()
		if !until.IsZero() {
			d.pausedUntil = until
		} else {
			d.manualPause = true
		}
		d.mu.Unlock()
	}

	go d.loop(ctx)
}

// Pause manually pauses the queue indefinitely. The queue will not resume
// until Resume() is called.
func (d *Dispatcher) Pause(reason string) {
	d.mu.Lock()
	d.manualPause = true
	d.pausedUntil = time.Time{} // clear any auto-resume time
	d.mu.Unlock()
	_ = d.store.SetPauseState(true, time.Time{}, reason)
	d.broadcast("queue.paused", map[string]any{"reason": reason, "manual": true})
}

// Resume clears both manual and rate-limit pause states.
func (d *Dispatcher) Resume() {
	d.mu.Lock()
	d.manualPause = false
	d.pausedUntil = time.Time{}
	d.mu.Unlock()
	_ = d.store.SetPauseState(false, time.Time{}, "")
	d.broadcast("queue.resumed", map[string]any{})
}

// paused returns true when the dispatcher should not dequeue work.
// It also auto-resumes when paused_until is reached.
func (d *Dispatcher) paused() bool {
	now := d.cfg.clock()
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.manualPause {
		return true
	}
	if d.pausedUntil.IsZero() {
		return false
	}
	if now.Before(d.pausedUntil) {
		return true
	}
	// Auto-resume: paused_until has passed.
	d.pausedUntil = time.Time{}
	go func() {
		_ = d.store.SetPauseState(false, time.Time{}, "")
		d.broadcast("queue.resumed", map[string]any{"auto": true})
	}()
	return false
}

// setPausedUntil sets the in-memory auto-resume time (used by handleRateLimit).
func (d *Dispatcher) setPausedUntil(t time.Time) {
	d.mu.Lock()
	d.pausedUntil = t
	d.mu.Unlock()
}

// loop is the main dispatcher goroutine.
func (d *Dispatcher) loop(ctx context.Context) {
	tick := time.NewTicker(d.cfg.tickInterval())
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			if d.paused() {
				continue
			}
			d.processNext(ctx)
		}
	}
}

// processNext dequeues one job (if available) and runs it to completion.
func (d *Dispatcher) processNext(ctx context.Context) {
	job, err := d.store.Dequeue()
	if err != nil {
		slog.Warn("queue: dequeue error", "err", err)
		return
	}
	if job == nil {
		return // queue empty
	}

	// FR7: verify artifact is still approved before launching.
	pa, ok := d.lookup(job.Project)
	if !ok {
		slog.Warn("queue: project not found for job", "job_id", job.ID, "project", job.Project)
		_ = d.store.MarkTerminal(job.ID, StateFailed, "project_not_found")
		d.broadcastJobEvent("queue.finished", job, "failed")
		return
	}

	if status := pa.ArtifactStatus(job.ArtifactPath); status != "approved" {
		reason := "status_changed_to:" + status
		if status == "" {
			reason = "artifact_not_found"
		}
		slog.Info("queue: skipping job — artifact no longer approved",
			"job_id", job.ID, "artifact", job.ArtifactPath, "status", status)
		_ = d.store.MarkTerminal(job.ID, StateSkipped, reason)
		d.broadcastJobEvent("queue.finished", job, "skipped")
		return
	}

	d.broadcastJobEvent("queue.started", job, "running")

	// Subscribe to the project hub BEFORE starting the run so we don't miss
	// any events emitted between Start and our subscription.
	runDone := make(chan runResult, 1)
	if pa.Hub != nil {
		evCh := make(chan []byte, 64)
		pa.Hub.Register(evCh) // registers the send side
		go d.watchRunEvents(ctx, evCh, pa.Hub, runDone)
	}

	runID, err := pa.StartRun(ctx, job.AgentName, job.ArtifactPath)
	if err != nil {
		slog.Warn("queue: start run failed", "job_id", job.ID, "err", err)
		_ = d.store.MarkTerminal(job.ID, StateFailed, "start_failed:"+err.Error())
		d.broadcastJobEvent("queue.finished", job, "failed")
		if pa.Hub != nil {
			// Close runDone so the watcher goroutine exits.
			select {
			case runDone <- runResult{kind: "cancelled"}:
			default:
			}
		}
		return
	}

	// If no hub is available (e.g. in tests without a hub), send a synthetic done.
	if pa.Hub == nil {
		go func() { runDone <- runResult{kind: "completed"} }()
	}

	slog.Info("queue: run started", "job_id", job.ID, "run_id", runID,
		"agent", job.AgentName, "artifact", job.ArtifactPath)

	// Block until the run finishes (or ctx is cancelled).
	var result runResult
	select {
	case <-ctx.Done():
		result = runResult{kind: "cancelled"}
	case result = <-runDone:
	}

	switch result.kind {
	case "completed":
		_ = d.store.MarkTerminal(job.ID, StateCompleted, "")
		d.broadcastJobEvent("queue.finished", job, "completed")
	case "rate_limit":
		d.handleRateLimit(job, result.rawText)
	case "cancelled":
		_ = d.store.MarkTerminal(job.ID, StateFailed, "cancelled")
		d.broadcastJobEvent("queue.finished", job, "failed")
	default:
		_ = d.store.MarkTerminal(job.ID, StateFailed, result.reason)
		d.broadcastJobEvent("queue.finished", job, "failed")
	}
}

// runResult captures the outcome of one agent run.
type runResult struct {
	kind    string // "completed", "failed", "rate_limit", "cancelled"
	reason  string
	rawText string // for rate_limit: the raw rate-limit message text
}

// watchRunEvents listens on evCh for agent.finished / agent.failed events and
// routes the result to runDone. When the appropriate event is received (or ctx
// is cancelled), it unregisters from the hub.
func (d *Dispatcher) watchRunEvents(ctx context.Context, evCh chan []byte, h *hub.Hub, done chan<- runResult) {
	defer h.Unregister(evCh)
	for {
		select {
		case <-ctx.Done():
			select {
			case done <- runResult{kind: "cancelled"}:
			default:
			}
			return
		case data, ok := <-evCh:
			if !ok {
				return
			}
			var evt struct {
				Type    string `json:"type"`
				Payload struct {
					Status  string `json:"status"`
					RunID   string `json:"run_id"`
					RawText string `json:"raw_text"` // for rate_limit stream events (M4)
				} `json:"payload"`
			}
			if err := json.Unmarshal(data, &evt); err != nil {
				continue
			}
			switch evt.Type {
			case "agent.finished":
				select {
				case done <- runResult{kind: "completed"}:
				default:
				}
				return
			case "agent.failed":
				// M4: rate_limit failures are delivered as stream events before
				// agent.failed; if we received a rate_limit event earlier, it was
				// already sent to done. Otherwise treat as a generic failure.
				select {
				case done <- runResult{kind: "failed", reason: evt.Payload.Status}:
				default:
				}
				return
			case "queue.rate_limit": // M4: emitted by agent stream watcher
				select {
				case done <- runResult{kind: "rate_limit", rawText: evt.Payload.RawText}:
				default:
				}
				return
			}
		}
	}
}

// handleRateLimit processes a rate-limit failure: marks the job failed,
// re-enqueues at the head (unless max-attempts exceeded), and pauses the queue.
func (d *Dispatcher) handleRateLimit(job *Job, rawText string) {
	now := d.cfg.clock()
	resetTime, ok := ParseResetTime(rawText, now)
	if !ok {
		slog.Warn("queue: rate-limit text not parsed; using fallback pause",
			"job_id", job.ID, "raw_text", rawText)
		resetTime = now.Add(d.cfg.fallbackPause())
	}
	pausedUntil := resetTime.Add(d.cfg.resumeGrace())

	// 1. Mark current job failed.
	_ = d.store.MarkTerminal(job.ID, StateFailed, "rate_limit")

	// 2. Re-enqueue at head if within max-attempts.
	if job.Attempts >= d.cfg.maxAttempts() {
		slog.Warn("queue: job exceeded max attempts; not re-enqueueing",
			"job_id", job.ID, "attempts", job.Attempts, "max", d.cfg.maxAttempts())
		d.broadcast("queue.skipped", map[string]any{
			"id":     job.ID,
			"reason": "max_attempts",
		})
	} else {
		requeue := *job
		requeue.ID = newID()
		requeue.State = StatePending
		requeue.Attempts = job.Attempts + 1
		requeue.Position = d.store.MinPosition() - 1
		requeue.EnqueuedAt = now
		if err := d.store.EnqueueDirect(requeue); err != nil {
			slog.Error("queue: re-enqueue after rate-limit failed", "err", err)
		} else {
			d.broadcast("queue.added", map[string]any{
				"id":       requeue.ID,
				"position": requeue.Position,
				"attempts": requeue.Attempts,
				"reason":   "rate_limit_retry",
			})
		}
	}

	// 3. Pause queue.
	_ = d.store.SetPauseState(true, pausedUntil, "rate_limit")
	d.setPausedUntil(pausedUntil)
	d.broadcast("queue.paused", map[string]any{
		"paused_until": pausedUntil.Format(time.RFC3339),
		"reset_time":   resetTime.Format(time.RFC3339),
		"raw_text":     rawText,
	})

	slog.Info("queue: paused due to rate limit",
		"job_id", job.ID,
		"reset_time", resetTime.Format(time.RFC3339),
		"paused_until", pausedUntil.Format(time.RFC3339))
}

// ---- broadcast helpers ----

func (d *Dispatcher) broadcast(eventType string, payload map[string]any) {
	if d.appHub == nil {
		return
	}
	d.appHub.Broadcast(hub.Event{Type: eventType, Payload: payload})
}

func (d *Dispatcher) broadcastJobEvent(eventType string, job *Job, status string) {
	d.broadcast(eventType, map[string]any{
		"id":            job.ID,
		"project":       job.Project,
		"artifact_path": job.ArtifactPath,
		"agent_name":    job.AgentName,
		"status":        status,
		"attempts":      job.Attempts,
		"enqueued_by":   job.EnqueuedBy,
	})
}
