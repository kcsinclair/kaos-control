package scheduler

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/hub"
)

// newTestScheduler creates a fully started Scheduler backed by a temp SQLite DB.
// t.Cleanup stops the scheduler when the test ends.
func newTestScheduler(t *testing.T) (*Scheduler, *Store) {
	t.Helper()
	return newTestSchedulerWorkers(t, 2)
}

// newTestSchedulerWorkers is like newTestScheduler but sets a specific worker count.
func newTestSchedulerWorkers(t *testing.T, maxWorkers int) (*Scheduler, *Store) {
	t.Helper()
	db := newTestDB(t)
	store := NewStore(db)
	h := hub.New()
	projectRoot := t.TempDir()
	logDir := t.TempDir()
	sc := New(store, nil, h, projectRoot, logDir, maxWorkers)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Cleanup(sc.Stop)
	sc.Start(ctx)
	return sc, store
}

// waitForRunStatus polls until the last run for jobName reaches a terminal
// status (anything other than "running"), or the timeout elapses.
func waitForRunStatus(t *testing.T, store *Store, jobName string, timeout time.Duration) *Run {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		run, err := store.LastRunForJob(jobName)
		if err != nil {
			t.Fatal(err)
		}
		if run != nil && run.Status != RunStatusRunning {
			return run
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timeout: job %q did not reach a terminal status within %v", jobName, timeout)
	return nil
}

// waitForRunCount polls until the total run count for jobName is at least n.
func waitForRunCount(t *testing.T, store *Store, jobName string, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, total, err := store.ListRuns(jobName, 1, 100)
		if err != nil {
			t.Fatal(err)
		}
		if total >= n {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	_, total, _ := store.ListRuns(jobName, 1, 100)
	t.Fatalf("timeout: job %q has %d runs, want at least %d", jobName, total, n)
}

// insertShellJob creates and persists a shell job via the store.
func insertShellJob(t *testing.T, store *Store, name, cmd string, priority, timeoutSec int) *Job {
	t.Helper()
	j := &Job{
		Name:       name,
		TargetType: "shell",
		Target:     cmd,
		// Schedule far in future so the tick loop doesn't fire it automatically.
		Schedule:   ScheduleSpec{Kind: ScheduleKindOneOff, At: time.Now().Add(time.Hour)},
		Enabled:    true,
		Priority:   priority,
		TimeoutSec: timeoutSec,
	}
	if err := store.CreateJob(j); err != nil {
		t.Fatal(err)
	}
	return j
}

// TestJobFiresOnSchedule verifies that TriggerNow dispatches a job and records a
// successful run.
func TestJobFiresOnSchedule(t *testing.T) {
	sc, store := newTestScheduler(t)
	insertShellJob(t, store, "fire-job", "true", 5, 10)
	if err := sc.TriggerNow("fire-job"); err != nil {
		t.Fatal(err)
	}
	run := waitForRunStatus(t, store, "fire-job", 5*time.Second)
	if run.Status != RunStatusSuccess {
		t.Errorf("expected success, got %q", run.Status)
	}
}

// TestJobDoesNotFireWhenPaused verifies that Pause prevents a job from being
// selected by the tick loop, and Resume re-enables it.
func TestJobDoesNotFireWhenPaused(t *testing.T) {
	sc, store := newTestScheduler(t)
	insertShellJob(t, store, "paused-job", "true", 5, 10)

	if err := sc.Pause("paused-job"); err != nil {
		t.Fatal(err)
	}

	// Verify the persisted enabled flag is false.
	job, err := store.GetJob("paused-job")
	if err != nil {
		t.Fatal(err)
	}
	if job.Enabled {
		t.Error("Pause should have set enabled=false")
	}

	// Wait briefly — the tick loop must not have dispatched the job.
	time.Sleep(100 * time.Millisecond)
	_, total, _ := store.ListRuns("paused-job", 1, 10)
	if total != 0 {
		t.Errorf("paused job should not have produced runs, got %d", total)
	}

	// Resume and trigger — job should now run.
	if err := sc.Resume("paused-job"); err != nil {
		t.Fatal(err)
	}
	if err := sc.TriggerNow("paused-job"); err != nil {
		t.Fatal(err)
	}
	run := waitForRunStatus(t, store, "paused-job", 5*time.Second)
	if run.Status != RunStatusSuccess {
		t.Errorf("expected success after resume, got %q", run.Status)
	}
}

// TestConcurrencyLimit verifies that with maxWorkers=1 two jobs run sequentially
// so the second job's start_time is not before the first job's end_time.
func TestConcurrencyLimit(t *testing.T) {
	sc, store := newTestSchedulerWorkers(t, 1)

	// Job A holds the worker slot for ~200ms.
	insertShellJob(t, store, "conc-a", "sleep 0.2", 5, 10)
	// Job B is instant.
	insertShellJob(t, store, "conc-b", "true", 5, 10)

	if err := sc.TriggerNow("conc-a"); err != nil {
		t.Fatal(err)
	}
	// Let A start before enqueuing B.
	time.Sleep(20 * time.Millisecond)
	if err := sc.TriggerNow("conc-b"); err != nil {
		t.Fatal(err)
	}

	runA := waitForRunStatus(t, store, "conc-a", 5*time.Second)
	runB := waitForRunStatus(t, store, "conc-b", 5*time.Second)

	if runA.EndTime == nil {
		t.Fatal("run A has no EndTime")
	}
	if runB.StartTime.Before(*runA.EndTime) {
		t.Errorf("concurrency violation: job B started (%v) before job A ended (%v)",
			runB.StartTime, *runA.EndTime)
	}
}

// TestPriorityOrdering verifies that the higher-priority job executes before the
// lower-priority job when both are enqueued simultaneously.
func TestPriorityOrdering(t *testing.T) {
	// Single worker so only one job runs at a time.
	sc, store := newTestSchedulerWorkers(t, 1)

	// Gate job occupies the worker while we enqueue the test pair.
	insertShellJob(t, store, "gate", "sleep 0.3", 5, 10)
	insertShellJob(t, store, "lo-pri", "true", 1, 10)
	insertShellJob(t, store, "hi-pri", "true", 10, 10)

	if err := sc.TriggerNow("gate"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond) // ensure gate holds the slot

	// Enqueue lo-pri then hi-pri while gate runs.
	if err := sc.TriggerNow("lo-pri"); err != nil {
		t.Fatal(err)
	}
	if err := sc.TriggerNow("hi-pri"); err != nil {
		t.Fatal(err)
	}

	runLo := waitForRunStatus(t, store, "lo-pri", 5*time.Second)
	runHi := waitForRunStatus(t, store, "hi-pri", 5*time.Second)

	// hi-pri must start before lo-pri.
	if runHi.StartTime.After(runLo.StartTime) {
		t.Errorf("priority violation: hi-pri started (%v) after lo-pri (%v)",
			runHi.StartTime, runLo.StartTime)
	}
}

// TestTriggerNow verifies that TriggerNow immediately executes a job regardless
// of its scheduled time.
func TestTriggerNow(t *testing.T) {
	sc, store := newTestScheduler(t)
	// Schedule far in the future — tick loop would never fire it.
	j := &Job{
		Name:       "trigger-now-job",
		TargetType: "shell",
		Target:     "true",
		Schedule:   ScheduleSpec{Kind: ScheduleKindOneOff, At: time.Now().Add(24 * time.Hour)},
		Enabled:    true,
		Priority:   5,
		TimeoutSec: 10,
	}
	if err := store.CreateJob(j); err != nil {
		t.Fatal(err)
	}
	if err := sc.TriggerNow("trigger-now-job"); err != nil {
		t.Fatal(err)
	}
	run := waitForRunStatus(t, store, "trigger-now-job", 5*time.Second)
	if run.Status != RunStatusSuccess {
		t.Errorf("expected success, got %q", run.Status)
	}
}

// TestTimeoutEnforcement verifies that a shell job exceeding its TimeoutSec is
// killed and its run is recorded with status timeout.
func TestTimeoutEnforcement(t *testing.T) {
	sc, store := newTestScheduler(t)
	insertShellJob(t, store, "timeout-job", "sleep 60", 5, 1) // 1-second timeout
	if err := sc.TriggerNow("timeout-job"); err != nil {
		t.Fatal(err)
	}
	run := waitForRunStatus(t, store, "timeout-job", 5*time.Second)
	if run.Status != RunStatusTimeout {
		t.Errorf("expected timeout, got %q", run.Status)
	}
}

// TestJobFailureRecovery verifies that after a job fails the scheduler continues
// operating.  This exercises the deferred cleanup in execute() — the running map
// and worker semaphore must be released even on failure so that subsequent jobs
// can run.  (Simulated "panic recovery": a non-zero exit is the observable
// failure mode without requiring an actual Go panic.)
func TestJobFailureRecovery(t *testing.T) {
	sc, store := newTestScheduler(t)

	// Job 1 fails.
	insertShellJob(t, store, "fail-job", "false", 5, 10)
	if err := sc.TriggerNow("fail-job"); err != nil {
		t.Fatal(err)
	}
	run1 := waitForRunStatus(t, store, "fail-job", 5*time.Second)
	if run1.Status != RunStatusFailure {
		t.Errorf("expected failure, got %q", run1.Status)
	}

	// Job 2 must still succeed — scheduler must still be alive.
	insertShellJob(t, store, "ok-job", "true", 5, 10)
	if err := sc.TriggerNow("ok-job"); err != nil {
		t.Fatal(err)
	}
	run2 := waitForRunStatus(t, store, "ok-job", 5*time.Second)
	if run2.Status != RunStatusSuccess {
		t.Errorf("expected success after previous failure, got %q", run2.Status)
	}
}

// TestStartupReconciliationStaleRunning verifies that Start() marks any run with
// status "running" as "failure" (crash recovery via MarkStaleRunsFailed).
func TestStartupReconciliationStaleRunning(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)

	if err := store.CreateJob(sampleJob("stale-job")); err != nil {
		t.Fatal(err)
	}
	staleRun := &Run{JobName: "stale-job", StartTime: time.Now(), Status: RunStatusRunning}
	if err := store.InsertRun(staleRun); err != nil {
		t.Fatal(err)
	}

	sc := New(store, nil, hub.New(), t.TempDir(), t.TempDir(), 1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Cleanup(sc.Stop)
	sc.Start(ctx)

	// Allow reconciliation to complete.
	time.Sleep(50 * time.Millisecond)

	got, err := store.GetRun(staleRun.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != RunStatusFailure {
		t.Errorf("expected stale run to be marked failure, got %q", got.Status)
	}
}

// TestStartupReconciliationMissedOneOff verifies that a one-off job with a past
// scheduled time and no existing runs is recorded as skipped on startup.
func TestStartupReconciliationMissedOneOff(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)

	j := &Job{
		Name:       "missed-oneoff",
		TargetType: "shell",
		Target:     "true",
		Schedule:   ScheduleSpec{Kind: ScheduleKindOneOff, At: time.Now().Add(-time.Hour)},
		Enabled:    true,
		Priority:   5,
		TimeoutSec: 10,
	}
	if err := store.CreateJob(j); err != nil {
		t.Fatal(err)
	}

	sc := New(store, nil, hub.New(), t.TempDir(), t.TempDir(), 1)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Cleanup(sc.Stop)
	sc.Start(ctx)

	time.Sleep(50 * time.Millisecond)

	run, err := store.LastRunForJob("missed-oneoff")
	if err != nil {
		t.Fatal(err)
	}
	if run == nil {
		t.Fatal("expected a skipped run to be inserted, got nil")
	}
	if run.Status != RunStatusSkipped {
		t.Errorf("expected skipped, got %q", run.Status)
	}
}

// TestPreconditionGating verifies that a job with an unsatisfied after_job
// precondition is not dispatched by the tick loop, and becomes eligible once the
// dependency has a successful run.
func TestPreconditionGating(t *testing.T) {
	sc, store := newTestScheduler(t)

	// Dependency job — no runs yet.
	if err := store.CreateJob(sampleJob("dep")); err != nil {
		t.Fatal(err)
	}

	// Gated job with a very short interval so the tick would fire it soon.
	gated := &Job{
		Name:       "gated",
		TargetType: "shell",
		Target:     "true",
		Schedule:   ScheduleSpec{Kind: ScheduleKindInterval, Interval: time.Millisecond},
		Preconditions: []Precondition{
			{Kind: PreconditionAfterJob, JobName: "dep"},
		},
		Enabled:    true,
		Priority:   5,
		TimeoutSec: 10,
	}
	if err := store.CreateJob(gated); err != nil {
		t.Fatal(err)
	}

	// The tick loop won't fire "gated" because dep has never run.
	time.Sleep(200 * time.Millisecond)
	_, total, _ := store.ListRuns("gated", 1, 10)
	if total != 0 {
		t.Errorf("gated job fired before dependency succeeded (%d runs)", total)
	}

	// Satisfy the dependency.
	depRun := &Run{JobName: "dep", StartTime: time.Now(), Status: RunStatusSuccess}
	if err := store.InsertRun(depRun); err != nil {
		t.Fatal(err)
	}

	// TriggerNow to exercise the after-precondition-met path.
	if err := sc.TriggerNow("gated"); err != nil {
		t.Fatal(err)
	}
	run := waitForRunStatus(t, store, "gated", 5*time.Second)
	if run.Status != RunStatusSuccess {
		t.Errorf("expected success after dependency met, got %q", run.Status)
	}
}

// TestAgentTargetDispatch verifies the agent dispatch code path.  With agents==nil
// the scheduler's runAgent guard returns failure, confirming the dispatch branch
// was exercised and the run was recorded.
func TestAgentTargetDispatch(t *testing.T) {
	sc, store := newTestScheduler(t) // agents is nil
	j := &Job{
		Name:       "agent-job",
		TargetType: "agent",
		Target:     "requirements-analyst",
		Args:       map[string]string{"role": "analyst", "target_path": "lifecycle/ideas/foo.md"},
		Schedule:   ScheduleSpec{Kind: ScheduleKindOneOff, At: time.Now().Add(time.Hour)},
		Enabled:    true,
		Priority:   5,
		TimeoutSec: 10,
	}
	if err := store.CreateJob(j); err != nil {
		t.Fatal(err)
	}
	if err := sc.TriggerNow("agent-job"); err != nil {
		t.Fatal(err)
	}
	run := waitForRunStatus(t, store, "agent-job", 5*time.Second)
	// No agent manager → runAgent returns failure.
	if run.Status != RunStatusFailure {
		t.Errorf("expected failure (no agent manager), got %q", run.Status)
	}
}

// TestShellTargetOutputCapture verifies that stdout/stderr from a shell job are
// written to the run's log file.
func TestShellTargetOutputCapture(t *testing.T) {
	sc, store := newTestScheduler(t)
	// Write a marker to stderr.
	insertShellJob(t, store, "output-job", "echo HELLO_MARKER 1>&2", 5, 10)
	if err := sc.TriggerNow("output-job"); err != nil {
		t.Fatal(err)
	}
	run := waitForRunStatus(t, store, "output-job", 5*time.Second)
	if run.Status != RunStatusSuccess {
		t.Errorf("expected success, got %q", run.Status)
	}
	if run.LogPath == "" {
		t.Fatal("expected LogPath to be set on the run")
	}
	data, err := os.ReadFile(run.LogPath)
	if err != nil {
		t.Fatalf("reading log file %q: %v", run.LogPath, err)
	}
	if !strings.Contains(string(data), "HELLO_MARKER") {
		t.Errorf("log file does not contain HELLO_MARKER; content:\n%s", data)
	}
}

// TestHubEventsFired verifies that scheduler.job.started and
// scheduler.job.completed events are broadcast via the hub.
func TestHubEventsFired(t *testing.T) {
	db := newTestDB(t)
	store := NewStore(db)
	h := hub.New()

	sc := New(store, nil, h, t.TempDir(), t.TempDir(), 2)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Cleanup(sc.Stop)
	sc.Start(ctx)

	ch := make(chan []byte, 32)
	h.Register(ch)
	defer h.Unregister(ch)

	if err := store.CreateJob(sampleJob("hub-job")); err != nil {
		t.Fatal(err)
	}
	if err := sc.TriggerNow("hub-job"); err != nil {
		t.Fatal(err)
	}

	var gotStarted, gotCompleted bool
	timeout := time.After(5 * time.Second)
	for !gotStarted || !gotCompleted {
		select {
		case raw := <-ch:
			var evt struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(raw, &evt)
			switch evt.Type {
			case "scheduler.job.started":
				gotStarted = true
			case "scheduler.job.completed":
				gotCompleted = true
			}
		case <-timeout:
			t.Fatalf("timeout waiting for hub events: started=%v completed=%v",
				gotStarted, gotCompleted)
		}
	}
}
