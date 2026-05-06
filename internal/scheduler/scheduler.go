package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kaos-control/kaos-control/internal/agent"
	"github.com/kaos-control/kaos-control/internal/hub"
)

// ErrJobNotFound is returned when a job name cannot be resolved.
var ErrJobNotFound = errors.New("scheduler job not found")

const tickInterval = 15 * time.Second

// Scheduler is the core scheduler engine. It ticks every 15 seconds, evaluates
// which jobs are due, checks preconditions, and dispatches execution through a
// priority-aware worker pool.
type Scheduler struct {
	store       *Store
	agents      *agent.Manager
	hub         *hub.Hub
	projectRoot string
	maxWorkers  int
	logDir      string // base dir for per-run log files

	mu      sync.Mutex
	queue   *jobQueue
	running map[string]struct{} // set of job names currently executing
	seq     int64               // used by workers to avoid concurrent executions

	workerSem chan struct{} // sized to maxWorkers
	workCh    chan *Job    // unbufffered handoff from dispatcher to workers

	stopOnce sync.Once
	stopCh   chan struct{}
	doneCh   chan struct{}
	started  atomic.Bool // true once Start() has launched goroutines
}

// New creates a Scheduler. logDir is the base directory for per-run log files;
// sub-directories per job are created automatically.
func New(store *Store, agents *agent.Manager, h *hub.Hub, projectRoot, logDir string, maxWorkers int) *Scheduler {
	if maxWorkers <= 0 {
		maxWorkers = 2
	}
	return &Scheduler{
		store:       store,
		agents:      agents,
		hub:         h,
		projectRoot: projectRoot,
		maxWorkers:  maxWorkers,
		logDir:      logDir,
		queue:       newJobQueue(),
		running:     make(map[string]struct{}),
		workerSem:   make(chan struct{}, maxWorkers),
		workCh:      make(chan *Job),
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
	}
}

// Start begins the scheduler goroutines. It returns after startup is complete.
// ctx is used only to propagate cancellation into job execution; the scheduler
// itself is stopped via Stop().
func (sc *Scheduler) Start(ctx context.Context) {
	sc.started.Store(true)
	// Startup reconciliation.
	if err := sc.store.MarkStaleRunsFailed(); err != nil {
		slog.Warn("SCHEDULER: marking stale runs failed", "err", err)
	}
	if err := sc.markMissedOneOffSkipped(); err != nil {
		slog.Warn("SCHEDULER: marking missed one-off jobs skipped", "err", err)
	}

	// Start worker pool.
	for i := 0; i < sc.maxWorkers; i++ {
		go sc.worker(ctx)
	}

	// Start dispatcher loop.
	go sc.dispatchLoop(ctx)

	slog.Info("SCHEDULER: scheduler started", "workers", sc.maxWorkers, "project", sc.projectRoot)
}

// Stop signals the scheduler to stop and waits for it to drain.
// If Start was never called, Stop returns immediately.
func (sc *Scheduler) Stop() {
	sc.stopOnce.Do(func() {
		close(sc.stopCh)
	})
	if sc.started.Load() {
		<-sc.doneCh
	}
	slog.Info("SCHEDULER: scheduler stopped")
}

// TriggerNow enqueues a job for immediate execution, bypassing its schedule
// and preconditions. Respects the concurrency limit.
func (sc *Scheduler) TriggerNow(jobName string) error {
	job, err := sc.store.GetJob(jobName)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job %q: %w", jobName, ErrJobNotFound)
	}
	sc.mu.Lock()
	_, alreadyRunning := sc.running[jobName]
	sc.mu.Unlock()
	if alreadyRunning {
		return fmt.Errorf("job %q is already running", jobName)
	}
	sc.enqueue(job)
	return nil
}

// Pause disables a job (sets enabled=false and persists it).
func (sc *Scheduler) Pause(jobName string) error {
	return sc.store.SetEnabled(jobName, false)
}

// Resume enables a job (sets enabled=true and persists it).
func (sc *Scheduler) Resume(jobName string) error {
	return sc.store.SetEnabled(jobName, true)
}

// ----- internal -----

func (sc *Scheduler) dispatchLoop(ctx context.Context) {
	defer close(sc.doneCh)
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sc.stopCh:
			return
		case <-ticker.C:
			sc.tick(ctx)
		}
	}
}

func (sc *Scheduler) tick(ctx context.Context) {
	jobs, err := sc.store.ListJobs()
	if err != nil {
		slog.Error("SCHEDULER: listing jobs", "err", err)
		return
	}
	now := time.Now()
	for _, job := range jobs {
		if !job.Enabled {
			continue
		}
		sc.mu.Lock()
		_, running := sc.running[job.Name]
		queued, _ := sc.isQueued(job.Name)
		sc.mu.Unlock()
		if running || queued {
			continue
		}

		lastRun, err := sc.store.LastRunForJob(job.Name)
		if err != nil {
			slog.Warn("SCHEDULER: getting last run", "job", job.Name, "err", err)
			continue
		}
		var lastRunTime time.Time
		if lastRun != nil && lastRun.EndTime != nil {
			lastRunTime = *lastRun.EndTime
		}

		next := NextFireTime(job.Schedule, lastRunTime, now)
		if next.IsZero() || next.After(now) {
			continue
		}

		// Check preconditions.
		if len(job.Preconditions) > 0 {
			ok, err := EvaluatePreconditions(ctx, job.Preconditions, sc.store, sc.projectRoot)
			if err != nil {
				slog.Warn("SCHEDULER: precondition error", "job", job.Name, "err", err)
				continue
			}
			if !ok {
				slog.Debug("SCHEDULER: preconditions not met, skipping", "job", job.Name)
				continue
			}
		}

		sc.enqueue(job)
	}
}

// isQueued returns whether jobName is currently in the queue.
// Must be called with sc.mu held.
func (sc *Scheduler) isQueued(jobName string) (bool, int) {
	for i, item := range sc.queue.pq {
		if item.job.Name == jobName {
			return true, i
		}
	}
	return false, -1
}

func (sc *Scheduler) enqueue(job *Job) {
	sc.mu.Lock()
	sc.queue.Push(job)
	sc.mu.Unlock()
	// Signal workers non-blocking.
	select {
	case sc.workCh <- job:
	default:
	}
	// Try to hand work off to a worker directly.
	sc.tryDispatch()
}

// tryDispatch pops jobs from the queue and hands them to available workers.
func (sc *Scheduler) tryDispatch() {
	for {
		select {
		case sc.workerSem <- struct{}{}: // acquired a worker slot
		default:
			return // all workers busy
		}

		sc.mu.Lock()
		job := sc.queue.Pop()
		if job == nil {
			sc.mu.Unlock()
			<-sc.workerSem
			return
		}
		sc.running[job.Name] = struct{}{}
		sc.mu.Unlock()

		go sc.execute(job)
	}
}

func (sc *Scheduler) worker(ctx context.Context) {
	for {
		select {
		case <-sc.stopCh:
			return
		case <-sc.workCh:
			// A job was signalled; tryDispatch will pop it.
			sc.tryDispatch()
		}
	}
}

func (sc *Scheduler) execute(job *Job) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("SCHEDULER: panic in job execution", "job", job.Name, "panic", r)
		}
		sc.mu.Lock()
		delete(sc.running, job.Name)
		sc.mu.Unlock()
		<-sc.workerSem
		// Drain any queued jobs now that a worker slot is free.
		sc.tryDispatch()
	}()

	logPath := sc.runLogPath(job.Name, 0) // temp, will be updated after insert
	run := &Run{
		JobName:   job.Name,
		StartTime: time.Now(),
		Status:    RunStatusRunning,
		LogPath:   logPath,
		CreatedAt: time.Now(),
	}
	if err := sc.store.InsertRun(run); err != nil {
		slog.Error("SCHEDULER: inserting run record", "job", job.Name, "err", err)
		return
	}
	// Now we have the real ID; update the log path.
	run.LogPath = sc.runLogPath(job.Name, run.ID)
	if err := sc.store.UpdateRun(run); err != nil {
		slog.Warn("SCHEDULER: updating log path", "job", job.Name, "err", err)
	}

	slog.Info("SCHEDULER: JOB_START", "job", job.Name, "run_id", run.ID,
		"target_type", job.TargetType, "target", job.Target)

	sc.hub.Broadcast(hub.Event{
		Type:    "scheduler.job.started",
		Payload: map[string]any{"job": job.Name, "run_id": run.ID},
	})

	startTime := time.Now()
	finalStatus, execErr := sc.runTarget(job, run)
	endTime := time.Now()
	durationMS := endTime.Sub(startTime).Milliseconds()

	run.EndTime = &endTime
	run.Status = finalStatus
	if err := sc.store.UpdateRun(run); err != nil {
		slog.Error("SCHEDULER: updating run record", "job", job.Name, "run_id", run.ID, "err", err)
	}

	switch finalStatus {
	case RunStatusSuccess:
		slog.Info("SCHEDULER: JOB_SUCCESS", "job", job.Name, "run_id", run.ID, "duration_ms", durationMS)
	case RunStatusTimeout:
		slog.Warn("SCHEDULER: JOB_TIMEOUT", "job", job.Name, "run_id", run.ID, "duration_ms", durationMS)
	default:
		slog.Warn("SCHEDULER: JOB_FAIL", "job", job.Name, "run_id", run.ID,
			"duration_ms", durationMS, "err", execErr)
	}

	sc.hub.Broadcast(hub.Event{
		Type: "scheduler.job.completed",
		Payload: map[string]any{
			"job":         job.Name,
			"run_id":      run.ID,
			"status":      string(finalStatus),
			"duration_ms": durationMS,
		},
	})
}

func (sc *Scheduler) runTarget(job *Job, run *Run) (RunStatus, error) {
	timeout := time.Duration(job.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	switch job.TargetType {
	case "agent":
		return sc.runAgent(ctx, job, run)
	case "shell":
		return sc.runShell(ctx, job, run)
	default:
		return RunStatusFailure, fmt.Errorf("unknown target_type %q", job.TargetType)
	}
}

func (sc *Scheduler) runAgent(ctx context.Context, job *Job, run *Run) (RunStatus, error) {
	if sc.agents == nil {
		return RunStatusFailure, fmt.Errorf("no agent manager configured")
	}
	agentName := job.Target
	targetPath := job.Args["target_path"]
	role := job.Args["role"]

	_, err := sc.agents.StartRun(ctx, agentName, targetPath, role, nil)
	if err != nil {
		return RunStatusFailure, err
	}

	// agent.Manager runs asynchronously; for the scheduler we just treat
	// the enqueue as success. The agent supervisor updates its own run records.
	return RunStatusSuccess, nil
}

func (sc *Scheduler) runShell(ctx context.Context, job *Job, run *Run) (RunStatus, error) {
	if err := os.MkdirAll(filepath.Dir(run.LogPath), 0o755); err != nil {
		slog.Warn("SCHEDULER: creating log dir", "path", run.LogPath, "err", err)
	}

	logFile, err := os.OpenFile(run.LogPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		slog.Warn("SCHEDULER: opening log file", "path", run.LogPath, "err", err)
		logFile = nil
	}
	if logFile != nil {
		defer logFile.Close()
		fmt.Fprintf(logFile, "# scheduler run %d — job=%s target=%s\n# started=%s\n\n",
			run.ID, job.Name, job.Target, run.StartTime.Format(time.RFC3339))
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", job.Target)
	cmd.Dir = sc.projectRoot
	cmd.Env = minimalEnv(sc.projectRoot)
	if logFile != nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return RunStatusTimeout, err
		}
		return RunStatusFailure, err
	}
	return RunStatusSuccess, nil
}

func (sc *Scheduler) runLogPath(jobName string, runID int64) string {
	return filepath.Join(sc.logDir, jobName, fmt.Sprintf("%d.log", runID))
}

func (sc *Scheduler) markMissedOneOffSkipped() error {
	jobs, err := sc.store.ListJobs()
	if err != nil {
		return err
	}
	now := time.Now()
	for _, job := range jobs {
		if job.Schedule.Kind != ScheduleKindOneOff {
			continue
		}
		if job.Schedule.At.IsZero() || job.Schedule.At.After(now) {
			continue
		}
		// Check if there's already a run for this job.
		run, err := sc.store.LastRunForJob(job.Name)
		if err != nil || run != nil {
			continue
		}
		// Mark as skipped.
		r := &Run{
			JobName:   job.Name,
			StartTime: now,
			Status:    RunStatusSkipped,
			CreatedAt: now,
		}
		endTime := now
		r.EndTime = &endTime
		if err := sc.store.InsertRun(r); err != nil {
			slog.Warn("SCHEDULER: marking missed one-off skipped", "job", job.Name, "err", err)
			continue
		}
		_ = sc.store.UpdateRun(r)
		slog.Info("SCHEDULER: marked missed one-off job as skipped", "job", job.Name)
	}
	return nil
}

