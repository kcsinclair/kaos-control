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
	// parsed to extract a reset time (typically quota/usage limits which
	// reset on the hour or daily cycle — a long pause is the safer default).
	FallbackPause time.Duration
	// OverloadPause is the pause duration for transient server-overload
	// events (HTTP 529, "overloaded_error"). These typically clear within
	// minutes — Claude's own internal retry budget runs ~3.5 minutes — so a
	// shorter pause is appropriate. Defaults to 5 minutes when unset.
	OverloadPause time.Duration
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

func (c *Config) overloadPause() time.Duration {
	if c.OverloadPause > 0 {
		return c.OverloadPause
	}
	return 5 * time.Minute
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

	// ---- automated provider failover (switch-provider-3-be Milestone 3) ----
	// All four fields below are optional; a nil field disables auto-failover
	// for this project (the dispatcher falls back to its standard rate-limit
	// pause), so callers that don't wire failover (e.g. tests) are unaffected.

	// FailoverPolicy returns the project's effective provider-failover policy.
	FailoverPolicy func() FailoverPolicy
	// AgentFailoverInfo returns the named agent's configured fallback
	// provider/model, or ok=false when the agent is unknown or has none.
	AgentFailoverInfo func(agentName string) (AgentFailoverInfo, bool)
	// ProbeProviderHealth performs a fast, bounded reachability check of the
	// named provider, honouring ctx's deadline.
	ProbeProviderHealth func(ctx context.Context, providerName string) bool
	// SwitchAgentProvider switches agentName to provider/model. isFailover
	// distinguishes an automated failover from a manual operator switch (it
	// controls whether the current provider is stashed as "primary" for
	// later restore).
	SwitchAgentProvider func(agentName, provider, model, reason string, isFailover bool) error

	// ---- project-wide failover (agent-switchover-and-failover Milestone 4) ----
	// All three fields below are optional; a nil field disables project-wide
	// failover for this project (tryProjectWideFailover falls back to the
	// standard pause_queue action).

	// AgentActiveProvider returns agentName's current effective active
	// provider (operations.yaml override, else config-declared), or
	// ok=false when agentName is unknown.
	AgentActiveProvider func(agentName string) (provider string, ok bool)
	// IsAgentFailedOver reports whether agentName is currently active on a
	// provider other than its primary — used to enforce the one-level
	// failover cap (NFR-6): an agent already failed over that fails again
	// routes to pause_queue instead of a second failover.
	IsAgentFailedOver func(agentName string) bool
	// FailoverProviderWide performs FR-3.1: every agent whose effective
	// active provider is provider moves to its own configured secondary in
	// one action. Agents with no secondary are recorded as partially paused
	// (FR-3.4, returned in noSecondary) instead. Returns the agent names
	// actually switched and left partially paused.
	FailoverProviderWide func(provider, reason string, resetsAtUnix int64, bucket string) (switched, noSecondary []string, err error)

	// ProviderDisconnectCountLastHour returns the number of recorded
	// provider_disconnected occurrences for providerName within the last
	// rolling hour (Milestone 5, FR-6.3/6.4), sourced from operations.yaml.
	// Optional — nil disables the threshold override, leaving
	// "provider_disconnected" resolved as plain retry_in_place always.
	ProviderDisconnectCountLastHour func(providerName string) int

	// ---- restart semantics & the partial-commit race (Milestone 7) ----

	// DetectPartialCommit reports whether the project's repository shows any
	// commits authored at or after sinceUnix (an interrupted job's
	// StartedAt) — evidence the run reached its commit step before failing
	// (FR-7.1). Optional; nil (or a job with a zero StartedAt) always
	// results in a clean restart (FR-7.2), the pre-Milestone-7 behaviour.
	DetectPartialCommit func(sinceUnix int64) (bool, error)
	// MarkAwaitingOperatorDecision records that jobID for agentName was held
	// instead of auto-restarted because DetectPartialCommit found a
	// suspected partial commit (FR-7.3). Only called when DetectPartialCommit
	// is set and returns true.
	MarkAwaitingOperatorDecision func(agentName, jobID string) error
}

// FailoverPolicy is the subset of config.ProviderFailoverConfig /
// config.EffectiveSwitchoverPolicy the dispatcher needs, decoupled from the
// config package so this package has no dependency on it.
type FailoverPolicy struct {
	Enabled            bool
	AutoSwitch         bool
	SwitchOnKinds      []string
	MaxFailoversPerRun int
	// Actions maps a classified failure reason to its resolved action
	// (agent-switchover-and-failover Milestone 3): "failover", "pause_queue",
	// "retry_in_place", or "fail_run" — mirrors
	// config.EffectiveSwitchoverPolicy().Actions, complete by construction
	// (one entry per reason the system classifies).
	Actions map[string]string
}

// ActionFor returns the resolved action for reason, defaulting to
// "pause_queue" — the safest fail-closed behaviour — if reason is absent
// from Actions (e.g. Actions is unset because the project doesn't wire
// switchover policy at all).
func (fp FailoverPolicy) ActionFor(reason string) string {
	if a, ok := fp.Actions[reason]; ok {
		return a
	}
	return "pause_queue"
}

// AgentFailoverInfo is the subset of config.AgentConfig the dispatcher needs
// to evaluate automated failover for one agent.
type AgentFailoverInfo struct {
	FallbackProvider string
	FallbackModel    string
}

// ProjectLookup maps a project name to its runtime access handle.
type ProjectLookup func(name string) (ProjectAccess, bool)

// Dispatcher is the single-goroutine queue worker.
type Dispatcher struct {
	store  *Store
	lookup ProjectLookup
	appHub *hub.Hub // app-level hub for queue.* events
	cfg    Config

	mu          sync.Mutex
	pausedUntil time.Time // zero = not rate-limit-paused
	manualPause bool      // true when paused via Pause(); cleared only by Resume()

	// blockedAgents returns the (project, agent) pairs currently partially
	// paused (FR-3.4) across every open project, so Dequeue can skip their
	// jobs while other agents' work proceeds. nil disables the skip-logic
	// (Dequeue behaves as before) — set via SetBlockedAgentsFunc.
	blockedAgents func() []AgentKey
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

// SetBlockedAgentsFunc wires the callback the dispatcher uses to skip
// partially-paused agents' jobs (FR-3.4) when dequeuing. Optional — a nil
// (or never-called) setter leaves Dequeue unfiltered.
func (d *Dispatcher) SetBlockedAgentsFunc(f func() []AgentKey) {
	d.blockedAgents = f
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
	var blocked []AgentKey
	if d.blockedAgents != nil {
		blocked = d.blockedAgents()
	}
	job, err := d.store.DequeueSkipping(blocked)
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
		// Rate-limit retries (attempts > 1) have already been validated at first
		// enqueue. The active_status transition performed by StartRun (e.g.
		// approved → clarifying) should not block re-runs after a rate-limit pause.
		if job.Attempts <= 1 {
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
		slog.Debug("queue: rate-limit retry bypassing status gate",
			"job_id", job.ID, "artifact", job.ArtifactPath, "status", status,
			"attempts", job.Attempts)
	}

	d.broadcastJobEvent("queue.started", job, "running")

	// Subscribe to the project hub BEFORE starting the run so we don't miss
	// any events emitted between Start and our subscription. The watcher is
	// gated by runIDCh: it ignores all events until we know our run_id, then
	// only accepts events whose payload run_id matches. Without this filter,
	// a concurrent agent run (e.g. a manually-started UI run, or a previous
	// queue iteration that completed slightly late) emitting agent.finished
	// would be mis-attributed to *this* job — the dispatcher would mark this
	// job completed and start the next one while the real run is still
	// holding its lineage lock, surfacing as a "lock conflict" on the next
	// job's StartRun.
	runDone := make(chan runResult, 1)
	runIDCh := make(chan string, 1)
	if pa.Hub != nil {
		evCh := make(chan []byte, 64)
		pa.Hub.Register(evCh) // registers the send side
		go d.watchRunEvents(ctx, evCh, pa.Hub, runDone, runIDCh)
	}

	runID, err := pa.StartRun(ctx, job.AgentName, job.ArtifactPath)
	if err != nil {
		slog.Warn("queue: start run failed", "job_id", job.ID, "err", err)
		_ = d.store.MarkTerminal(job.ID, StateFailed, "start_failed:"+err.Error())
		d.broadcastJobEvent("queue.finished", job, "failed")
		if pa.Hub != nil {
			// Unblock the watcher (so it exits cleanly), then close runDone.
			close(runIDCh)
			select {
			case runDone <- runResult{kind: "cancelled"}:
			default:
			}
		}
		return
	}

	// StartRun may itself commit (e.g. an active-status transition) before
	// returning. Re-stamp StartedAt to now, after that commit has already
	// landed, so the Milestone 7 partial-commit check (FR-7.1) only looks at
	// commits made by the run itself — not this framework bookkeeping
	// commit, which would otherwise be indistinguishable from "the run
	// committed its own work" and wrongly hold every retry for an operator
	// decision.
	job.StartedAt = d.cfg.clock()

	// Hand the run_id to the watcher so it can start accepting events.
	if pa.Hub != nil {
		runIDCh <- runID
		close(runIDCh)
	} else {
		// No hub (e.g. in tests): send a synthetic done.
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
		reason := result.rlKind
		if reason == "" {
			reason = "rate_limit"
		}
		d.handleClassifiedFailure(ctx, job, pa, reason, result)
	case "auth_error":
		// Claude Code's own OAuth token rotation (queue.auth_error) — a
		// transient, per-run condition distinct from a provider's
		// FailureReasonAuthError (expired/invalid API key), which is
		// handled via the "failed" branch below. Always retry immediately
		// without pausing; the freshly-rotated token applies to the next run.
		d.handleAuthError(job, pa)
	case "cancelled":
		_ = d.store.MarkTerminal(job.ID, StateFailed, "cancelled")
		d.broadcastJobEvent("queue.finished", job, "failed")
	default:
		if result.FailureReason != "" {
			d.handleClassifiedFailure(ctx, job, pa, result.FailureReason, result)
			return
		}
		_ = d.store.MarkTerminal(job.ID, StateFailed, result.reason)
		d.broadcastJobEvent("queue.finished", job, "failed")
	}
}

// runResult captures the outcome of one agent run.
type runResult struct {
	kind    string // "completed", "failed", "rate_limit", "cancelled"
	reason  string
	rawText string // for rate_limit: the raw rate-limit message text
	// rlKind classifies the underlying transient-error variant for the
	// rate_limit kind: "rate_limit" (default) or "overloaded". The dispatcher
	// uses this to pick FallbackPause vs OverloadPause when the rawText
	// doesn't carry an explicit reset time.
	rlKind string
	// ResetsAtUnix is the precise Unix-UTC reset time from the most recent
	// rate_limit_event observed during the run (FR7), when > 0. When 0,
	// handleRateLimit falls back to parsing rawText with ParseResetTime.
	ResetsAtUnix int64
	// Bucket is the rate-limit reset bucket ("five_hour" | "weekly" |
	// "unknown") from the most recent rate_limit_event, when known
	// (agent-switchover-and-failover FR-3.3).
	Bucket string
	// FailureReason is the classified failure reason for an "agent.failed"
	// terminal event (agent.FailureReason* — e.g. "auth_error",
	// "model_not_found", "timeout"), when the run supervisor classified one.
	// Empty for an unclassified failure. Used to resolve the
	// agent-switchover-and-failover event -> action policy.
	FailureReason string
}

// watchRunEvents listens on evCh for agent.finished / agent.failed /
// queue.rate_limit events scoped to ourRunID (received via runIDCh after
// StartRun returns). Events that lack a payload.run_id, or whose run_id does
// not match ourRunID, are ignored — they belong to other concurrent runs.
//
// Before ourRunID is known we still drain evCh (so the hub channel never
// blocks Broadcast), but we never signal `done` for terminal events received
// in that window, since those terminal events are by definition for an
// earlier or parallel run, not the one we're about to start.
func (d *Dispatcher) watchRunEvents(ctx context.Context, evCh chan []byte, h *hub.Hub, done chan<- runResult, runIDCh <-chan string) {
	defer h.Unregister(evCh)
	var ourRunID string
	for {
		select {
		case <-ctx.Done():
			select {
			case done <- runResult{kind: "cancelled"}:
			default:
			}
			return
		case rid, ok := <-runIDCh:
			if !ok {
				// runIDCh closed without a value — StartRun errored. Exit.
				return
			}
			ourRunID = rid
			runIDCh = nil // disable this case after first delivery
		case data, ok := <-evCh:
			if !ok {
				return
			}
			var evt struct {
				Type    string `json:"type"`
				Payload struct {
					Status        string `json:"status"`
					RunID         string `json:"run_id"`
					RawText       string `json:"raw_text"` // for rate_limit stream events (M4)
					Kind          string `json:"kind"`     // "rate_limit" | "overloaded" | "unreachable"
					ResetsAtUnix  int64  `json:"resets_at_unix"`
					Bucket        string `json:"bucket"`         // "five_hour" | "weekly" | "unknown"
					FailureReason string `json:"failure_reason"` // agent.FailureReason* on agent.failed
				} `json:"payload"`
			}
			if err := json.Unmarshal(data, &evt); err != nil {
				continue
			}
			// Filter: terminal/dispatch-relevant events must match ourRunID.
			// Until we know our run_id, drop these — they cannot be ours.
			isRelevant := evt.Type == "agent.finished" ||
				evt.Type == "agent.failed" ||
				evt.Type == "queue.rate_limit" ||
				evt.Type == "queue.auth_error"
			if isRelevant {
				if ourRunID == "" || evt.Payload.RunID != ourRunID {
					continue
				}
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
				// already sent to done. Otherwise treat as a generic failure,
				// carrying the classified failure reason (if any) so the
				// dispatcher can resolve the switchover event -> action policy.
				select {
				case done <- runResult{kind: "failed", reason: evt.Payload.Status, FailureReason: evt.Payload.FailureReason}:
				default:
				}
				return
			case "queue.rate_limit": // M4: emitted by agent stream watcher
				select {
				case done <- runResult{
					kind:         "rate_limit",
					rawText:      evt.Payload.RawText,
					rlKind:       evt.Payload.Kind,
					ResetsAtUnix: evt.Payload.ResetsAtUnix,
					Bucket:       evt.Payload.Bucket,
				}:
				default:
				}
				return
			case "queue.auth_error": // emitted by agent stream watcher on OAuth token rotation
				select {
				case done <- runResult{kind: "auth_error"}:
				default:
				}
				return
			}
		}
	}
}

// providerDisconnectThreshold is the FR-6.3 rolling-hour pause threshold:
// the 4th distinct provider_disconnected occurrence for a provider within
// an hour pauses the queue.
const providerDisconnectThreshold = 3

// handleClassifiedFailure resolves the agent-switchover-and-failover event
// -> action policy for reason and dispatches accordingly (Milestone 4):
//   - "failover": attempt project-wide failover (tryProjectWideFailover);
//     falls through to the pause_queue behaviour if it can't engage.
//   - "pause_queue": the standard pause-preserving-order-and-restart-first
//     path (handleRateLimit works for any reason, not just rate limits).
//   - "retry_in_place": the in-loop backoff retry already happened inside
//     the run (Milestone 5); reaching the dispatcher as a terminal failure
//     means that budget was exhausted — re-enqueue immediately without
//     pausing, same as an auth-token rotation.
//   - "fail_run": fail the job outright — no switch, no re-enqueue, no pause.
//
// When the project doesn't wire FailoverPolicy at all (e.g. tests), this
// defaults to pause_queue — the safest fail-closed behaviour, and exactly
// the dispatcher's pre-Milestone-3 default when failover wasn't configured.
func (d *Dispatcher) handleClassifiedFailure(ctx context.Context, job *Job, pa ProjectAccess, reason string, result runResult) {
	action := "pause_queue"
	if pa.FailoverPolicy != nil {
		action = pa.FailoverPolicy().ActionFor(reason)
	}

	switch action {
	case "failover":
		if d.tryProjectWideFailover(ctx, job, pa, reason, result) {
			return
		}
		d.handleRateLimit(job, pa, result.rawText, result.rlKind, result.ResetsAtUnix)
	case "retry_in_place":
		// FR-6.3: more than 3 provider_disconnected occurrences for this
		// provider within a rolling hour pauses the queue instead of
		// retrying immediately again. The driver's own in-loop backoff
		// retry (2s/8s/30s) has already run by the time a run reaches this
		// terminal-failure path; this is the queue-level bound on top of it.
		if reason == "provider_disconnected" && pa.ProviderDisconnectCountLastHour != nil && pa.AgentActiveProvider != nil {
			if provider, ok := pa.AgentActiveProvider(job.AgentName); ok && pa.ProviderDisconnectCountLastHour(provider) > providerDisconnectThreshold {
				d.handleRateLimit(job, pa, result.rawText, result.rlKind, result.ResetsAtUnix)
				return
			}
		}
		d.handleImmediateRetry(job, pa, reason, reason+"_retry") // immediate re-enqueue, no pause
	case "fail_run":
		_ = d.store.MarkTerminal(job.ID, StateFailed, reason)
		d.broadcastJobEvent("queue.finished", job, "failed")
	default: // "pause_queue"
		d.handleRateLimit(job, pa, result.rawText, result.rlKind, result.ResetsAtUnix)
	}
}

// tryProjectWideFailover attempts automated, project-wide provider failover
// (FR-3): every agent whose effective active provider matches the failing
// job's agent's provider moves to its own secondary in one action. It
// returns true when failover engaged (whether or not job.AgentName itself
// had a secondary — see below), so the caller skips the pause_queue
// fallback. It returns false — falling through to pause_queue — when: the
// project doesn't wire project-wide failover support, the triggering
// agent's active provider can't be resolved, the triggering agent is
// already in a failover state (NFR-6 one-level cap: a second failure for an
// already-failed-over agent must pause, not fail over again), or the
// project-wide switch errors.
func (d *Dispatcher) tryProjectWideFailover(ctx context.Context, job *Job, pa ProjectAccess, reason string, result runResult) bool {
	if pa.AgentActiveProvider == nil || pa.IsAgentFailedOver == nil || pa.FailoverProviderWide == nil {
		return false
	}

	provider, ok := pa.AgentActiveProvider(job.AgentName)
	if !ok || provider == "" {
		return false
	}
	if pa.IsAgentFailedOver(job.AgentName) {
		// One-level cap (FR-3.5/NFR-6): the secondary also failed — no third
		// target exists, so this falls through to pause_queue.
		slog.Info("queue: agent already failed over; failover capped at one level, falling back to pause",
			"job_id", job.ID, "agent", job.AgentName, "provider", provider)
		return false
	}

	// Cheap pre-switch guard: probe the triggering agent's own secondary
	// before committing the whole project to switching. Other agents bound
	// to provider may have different secondaries (each is switched to its
	// own configured fallback inside FailoverProviderWide); an unhealthy
	// secondary for any one of them is caught the same way on its own next
	// failure, via the one-level cap above.
	if pa.AgentFailoverInfo != nil && pa.ProbeProviderHealth != nil {
		if info, ok := pa.AgentFailoverInfo(job.AgentName); ok && info.FallbackProvider != "" {
			probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			healthy := pa.ProbeProviderHealth(probeCtx, info.FallbackProvider)
			cancel()
			if !healthy {
				slog.Warn("queue: fallback provider failed pre-switch health probe; falling back to standard pause",
					"job_id", job.ID, "agent", job.AgentName, "fallback_provider", info.FallbackProvider)
				return false
			}
		}
	}

	switched, noSecondary, err := pa.FailoverProviderWide(provider, reason, result.ResetsAtUnix, result.Bucket)
	if err != nil {
		slog.Error("queue: project-wide failover failed; falling back to standard pause",
			"job_id", job.ID, "agent", job.AgentName, "provider", provider, "err", err)
		return false
	}
	if len(switched) == 0 && len(noSecondary) == 0 {
		// job.AgentName was supposedly bound to provider but nothing moved —
		// treat as failover not having engaged.
		return false
	}

	now := d.cfg.clock()

	// 1. Mark the interrupted job failed.
	_ = d.store.MarkTerminal(job.ID, StateFailed, "failover_triggered")
	d.broadcastJobEvent("queue.finished", job, "failed")

	// 2. Restart it (FR-3.2, subject to FR-7's partial-commit check). If the
	// triggering agent itself had no secondary (FR-3.4), it's now partially
	// paused — SetBlockedAgentsFunc will keep Dequeue from serving its jobs,
	// so re-enqueueing normally (rather than skipping) is enough; other
	// agents' work still proceeds. The failover itself has already engaged
	// regardless of the partial-commit outcome below — only the interrupted
	// job's automatic restart is gated.
	if d.restartOrHold(job, pa) {
		requeue := *job
		requeue.ID = newID()
		requeue.State = StatePending
		requeue.Attempts = job.Attempts + 1
		requeue.Position = d.store.MinPosition() - 1
		requeue.EnqueuedAt = now
		if err := d.store.EnqueueDirect(requeue); err != nil {
			slog.Error("queue: re-enqueue after project-wide failover failed", "err", err)
		} else {
			d.broadcast("queue.added", map[string]any{
				"id":       requeue.ID,
				"position": requeue.Position,
				"attempts": requeue.Attempts,
				"reason":   "failover_retry",
			})
		}
	}

	slog.Info("queue: project-wide automated failover engaged",
		"job_id", job.ID, "provider", provider, "reason", reason,
		"switched", switched, "paused_no_secondary", noSecondary)
	return true
}

// restartOrHold implements the FR-7 partial-commit race check that must run
// before any automatic re-run of an interrupted job: an agent may complete
// and commit work moments before its process dies, and blindly re-running
// would duplicate that work. When pa.DetectPartialCommit finds evidence the
// run reached its commit step before failing, the job is held for an
// operator decision (FR-7.3) instead of being re-enqueued — neither
// auto-rerun nor auto-rollback. It returns true when the caller should
// proceed with its normal clean restart (FR-7.2), including when the check
// is unavailable (pa.DetectPartialCommit is nil, or job.StartedAt is zero,
// e.g. a test harness that doesn't wire it) or errors — those cases can't
// distinguish "partial" from "clean", and a clean restart is the
// pre-Milestone-7 default.
func (d *Dispatcher) restartOrHold(job *Job, pa ProjectAccess) bool {
	if pa.DetectPartialCommit == nil || job.StartedAt.IsZero() {
		return true
	}
	suspected, err := pa.DetectPartialCommit(job.StartedAt.Unix())
	if err != nil {
		slog.Warn("queue: partial-commit detection failed; restarting normally",
			"job_id", job.ID, "agent", job.AgentName, "err", err)
		return true
	}
	if !suspected {
		return true
	}

	slog.Warn("queue: suspected partial commit; holding job for operator decision instead of auto-restarting",
		"job_id", job.ID, "agent", job.AgentName)
	if pa.MarkAwaitingOperatorDecision != nil {
		if err := pa.MarkAwaitingOperatorDecision(job.AgentName, job.ID); err != nil {
			slog.Error("queue: recording awaiting-operator-decision failed", "job_id", job.ID, "err", err)
		}
	}
	d.broadcastJobEvent("queue.awaiting_operator_decision", job, "awaiting_operator_decision")
	return false
}

// handleRateLimit processes a rate-limit / overloaded failure: marks the job
// failed, re-enqueues at the head (unless max-attempts exceeded), and pauses
// the queue. When resetsAtUnix is > 0 (FR7 — a precise reset observed via a
// prior rate_limit_event), it is used directly as the reset time without
// calling ParseResetTime. Otherwise the kind argument picks the appropriate
// fallback when rawText has no parseable reset time — overloads (HTTP 529)
// get OverloadPause (default 5 min) since they typically clear within
// minutes; rate limits and quotas get FallbackPause (default 30 min) since
// those usually align with hourly / daily reset cycles.
func (d *Dispatcher) handleRateLimit(job *Job, pa ProjectAccess, rawText, kind string, resetsAtUnix int64) {
	now := d.cfg.clock()
	var resetTime time.Time
	if resetsAtUnix > 0 {
		resetTime = time.Unix(resetsAtUnix, 0)
	} else {
		var ok bool
		resetTime, ok = ParseResetTime(rawText, now)
		if !ok {
			pause := d.cfg.fallbackPause()
			if kind == "overloaded" {
				pause = d.cfg.overloadPause()
			}
			slog.Warn("queue: rate-limit text not parsed; using fallback pause",
				"job_id", job.ID, "raw_text", rawText, "kind", kind, "pause", pause)
			resetTime = now.Add(pause)
		}
	}
	pausedUntil := resetTime.Add(d.cfg.resumeGrace())

	// 1. Mark current job failed.
	_ = d.store.MarkTerminal(job.ID, StateFailed, "rate_limit")

	// 2. Re-enqueue at head if within max-attempts and no partial commit is
	// suspected (FR-7).
	if job.Attempts >= d.cfg.maxAttempts() {
		slog.Warn("queue: job exceeded max attempts; not re-enqueueing",
			"job_id", job.ID, "attempts", job.Attempts, "max", d.cfg.maxAttempts())
		d.broadcast("queue.skipped", map[string]any{
			"id":     job.ID,
			"reason": "max_attempts",
		})
	} else if d.restartOrHold(job, pa) {
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

// handleAuthError processes an OAuth token rotation failure: marks the job
// failed and re-enqueues it immediately (bounded by max-attempts) without
// pausing the queue. An auth rotation is transient and per-run — subsequent
// runs will pick up the freshly-rotated token, so no queue-wide pause is needed.
func (d *Dispatcher) handleAuthError(job *Job, pa ProjectAccess) {
	d.handleImmediateRetry(job, pa, "auth_error", "auth_error_retry")
}

// handleImmediateRetry marks job failed with reason, re-enqueues it at the
// head if within max-attempts, and does NOT pause the queue — used for
// transient, per-run conditions where a pause would help nothing: an OAuth
// token rotation (handleAuthError) and, since Milestone 5, a
// provider_disconnected failure resolved to retry_in_place once the
// driver's own in-loop backoff retry has been exhausted. requeueReason
// labels the queue.added broadcast so the two callers stay distinguishable
// in the event stream.
func (d *Dispatcher) handleImmediateRetry(job *Job, pa ProjectAccess, reason, requeueReason string) {
	now := d.cfg.clock()

	// 1. Mark current job failed.
	_ = d.store.MarkTerminal(job.ID, StateFailed, reason)

	// 2. Re-enqueue at head if within max-attempts and no partial commit is
	// suspected (FR-7).
	if job.Attempts >= d.cfg.maxAttempts() {
		slog.Warn("queue: job exceeded max attempts; not re-enqueueing",
			"job_id", job.ID, "reason", reason, "attempts", job.Attempts, "max", d.cfg.maxAttempts())
		d.broadcast("queue.skipped", map[string]any{
			"id":     job.ID,
			"reason": "max_attempts",
		})
	} else if d.restartOrHold(job, pa) {
		requeue := *job
		requeue.ID = newID()
		requeue.State = StatePending
		requeue.Attempts = job.Attempts + 1
		requeue.Position = d.store.MinPosition() - 1
		requeue.EnqueuedAt = now
		if err := d.store.EnqueueDirect(requeue); err != nil {
			slog.Error("queue: re-enqueue failed", "reason", reason, "err", err)
		} else {
			d.broadcast("queue.added", map[string]any{
				"id":       requeue.ID,
				"position": requeue.Position,
				"attempts": requeue.Attempts,
				"reason":   requeueReason,
			})
		}
	}

	// 3. Do NOT pause — this is a transient, per-run condition.
	d.broadcastJobEvent("queue.finished", job, "failed")

	slog.Info("queue: re-enqueued job without pausing", "job_id", job.ID, "reason", reason, "attempts", job.Attempts)
}

// ---- public store-proxy methods (used by HTTP handlers) ----

// Enqueue adds a job to the queue, delegating to the store, and broadcasts
// queue.added carrying the full persisted job record on success.
func (d *Dispatcher) Enqueue(j Job) error {
	if err := d.store.Enqueue(j); err != nil {
		return err
	}
	if job, err := d.store.FindActiveByPath(j.Project, j.ArtifactPath); err == nil && job != nil {
		d.broadcast("queue.added", job)
	}
	return nil
}

// GetByID returns a single job by ID.
func (d *Dispatcher) GetByID(id string) (*Job, error) { return d.store.GetByID(id) }

// FindActiveByPath returns the active pending/running job for a project+path.
func (d *Dispatcher) FindActiveByPath(project, path string) (*Job, error) {
	return d.store.FindActiveByPath(project, path)
}

// Cancel cancels a pending job (returns ErrCannotCancelRunning for running
// jobs), and broadcasts queue.cancelled on success.
func (d *Dispatcher) Cancel(id string) error {
	if err := d.store.Cancel(id); err != nil {
		return err
	}
	if job, err := d.store.GetByID(id); err == nil && job != nil {
		d.broadcastJobEvent("queue.cancelled", job, "cancelled")
	}
	return nil
}

// StateSnapshot assembles the GET /api/queue response payload.
// PendingJobs returns all jobs currently in the pending state (across
// projects). Used to surface queued artefacts, which have no agent_runs row
// (those are created only when a run starts).
func (d *Dispatcher) PendingJobs() ([]*Job, error) {
	return d.store.ListByState(StatePending)
}

// RunningJobs returns the jobs currently running for project, used to guard
// a manual provider switch against in-flight runs (agent-switchover-and-
// failover FR-8.2): an operator-requested switch while any run is executing
// is rejected with a warning listing these jobs.
func (d *Dispatcher) RunningJobs(project string) ([]*Job, error) {
	running, err := d.store.ListByState(StateRunning)
	if err != nil {
		return nil, err
	}
	out := make([]*Job, 0, len(running))
	for _, j := range running {
		if j.Project == project {
			out = append(out, j)
		}
	}
	return out, nil
}

func (d *Dispatcher) StateSnapshot() (*queueSnapshot, error) {
	running, err := d.store.ListByState(StateRunning)
	if err != nil {
		return nil, err
	}
	pending, err := d.store.ListByState(StatePending)
	if err != nil {
		return nil, err
	}
	recent, err := d.store.ListRecent(10)
	if err != nil {
		return nil, err
	}
	paused, until, reason, err := d.store.GetPauseState()
	if err != nil {
		return nil, err
	}

	var runningJob *Job
	if len(running) > 0 {
		runningJob = running[0]
	}

	// Normalise nil slices to empty so the JSON payload always emits `[]`
	// rather than `null`. SPA consumers do `pending.find(...)` /
	// `pending.length` directly and crash on null.
	if pending == nil {
		pending = []*Job{}
	}
	if recent == nil {
		recent = []*Job{}
	}

	snap := &queueSnapshot{
		Running:     runningJob,
		Pending:     pending,
		Recent:      recent,
		Paused:      paused,
		PauseReason: reason,
	}
	if !until.IsZero() {
		s := until.UTC().Format(time.RFC3339)
		snap.PausedUntil = &s
	}
	return snap, nil
}

// queueSnapshot is the structured response for GET /api/queue.
type queueSnapshot struct {
	Running     *Job    `json:"running"`
	Pending     []*Job  `json:"pending"`
	Recent      []*Job  `json:"recent"`
	Paused      bool    `json:"paused"`
	PausedUntil *string `json:"paused_until"`
	PauseReason string  `json:"pause_reason"`
}

// ---- broadcast helpers ----

func (d *Dispatcher) broadcast(eventType string, payload any) {
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
