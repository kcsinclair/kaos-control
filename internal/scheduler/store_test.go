// SPDX-License-Identifier: AGPL-3.0-or-later

package scheduler

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// newTestDB creates a temporary file-backed SQLite database with the
// scheduler_jobs and scheduler_runs tables for use in unit tests.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

	// Enable cascade deletes (must be set per connection).
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE scheduler_jobs (
		name                TEXT PRIMARY KEY,
		target_type         TEXT NOT NULL,
		target              TEXT NOT NULL,
		args_json           TEXT,
		schedule            TEXT NOT NULL,
		preconditions_json  TEXT,
		enabled             INTEGER NOT NULL DEFAULT 1,
		priority            INTEGER NOT NULL DEFAULT 5,
		timeout_sec         INTEGER NOT NULL,
		created_at          TEXT NOT NULL,
		updated_at          TEXT NOT NULL
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE scheduler_runs (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		job_name    TEXT NOT NULL REFERENCES scheduler_jobs(name) ON DELETE CASCADE,
		start_time  TEXT NOT NULL,
		end_time    TEXT,
		status      TEXT NOT NULL,
		log_path    TEXT,
		created_at  TEXT NOT NULL
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE INDEX idx_runs_job_start ON scheduler_runs(job_name, start_time DESC)`); err != nil {
		t.Fatal(err)
	}
	return db
}

// sampleJob returns a minimal valid Job for use in store tests.
func sampleJob(name string) *Job {
	return &Job{
		Name:       name,
		TargetType: "shell",
		Target:     "echo hello",
		Schedule:   ScheduleSpec{Kind: ScheduleKindCron, Cron: "0 2 * * *"},
		Enabled:    true,
		Priority:   5,
		TimeoutSec: 30,
	}
}

// TestCreateJob inserts a job and verifies all fields are persisted and retrievable.
func TestCreateJob(t *testing.T) {
	s := NewStore(newTestDB(t))
	j := &Job{
		Name:       "create-test",
		TargetType: "shell",
		Target:     "echo world",
		Args:       map[string]string{"key": "val"},
		Schedule:   ScheduleSpec{Kind: ScheduleKindInterval, Interval: 10 * time.Minute},
		Enabled:    true,
		Priority:   7,
		TimeoutSec: 60,
	}
	if err := s.CreateJob(j); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetJob("create-test")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected job, got nil")
	}
	if got.Name != "create-test" {
		t.Errorf("name: got %q want %q", got.Name, "create-test")
	}
	if got.TargetType != "shell" {
		t.Errorf("target_type: got %q want shell", got.TargetType)
	}
	if got.Target != "echo world" {
		t.Errorf("target: got %q want %q", got.Target, "echo world")
	}
	if got.Priority != 7 {
		t.Errorf("priority: got %d want 7", got.Priority)
	}
	if got.TimeoutSec != 60 {
		t.Errorf("timeout_sec: got %d want 60", got.TimeoutSec)
	}
	if !got.Enabled {
		t.Error("expected enabled=true")
	}
	if got.Args["key"] != "val" {
		t.Errorf("args: got %v", got.Args)
	}
	if got.Schedule.Kind != ScheduleKindInterval {
		t.Errorf("schedule kind: got %q want interval", got.Schedule.Kind)
	}
	if got.Schedule.Interval != 10*time.Minute {
		t.Errorf("schedule interval: got %v want 10m", got.Schedule.Interval)
	}
}

// TestCreateJobDuplicateName verifies that inserting a job with an existing name
// returns an error.
func TestCreateJobDuplicateName(t *testing.T) {
	s := NewStore(newTestDB(t))
	if err := s.CreateJob(sampleJob("dup")); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateJob(sampleJob("dup")); err == nil {
		t.Fatal("expected error for duplicate name, got nil")
	}
}

// TestGetJobNotFound verifies that GetJob returns nil for a non-existent job.
func TestGetJobNotFound(t *testing.T) {
	s := NewStore(newTestDB(t))
	got, err := s.GetJob("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("expected nil for missing job, got %+v", got)
	}
}

// TestUpdateJob verifies that mutable fields are updated and persisted.
func TestUpdateJob(t *testing.T) {
	s := NewStore(newTestDB(t))
	j := sampleJob("update-me")
	if err := s.CreateJob(j); err != nil {
		t.Fatal(err)
	}

	j.Priority = 9
	j.Enabled = false
	j.TimeoutSec = 120
	j.Schedule = ScheduleSpec{Kind: ScheduleKindInterval, Interval: 5 * time.Minute}
	if err := s.UpdateJob(j); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetJob("update-me")
	if err != nil {
		t.Fatal(err)
	}
	if got.Priority != 9 {
		t.Errorf("priority: got %d want 9", got.Priority)
	}
	if got.Enabled {
		t.Error("expected enabled=false after update")
	}
	if got.TimeoutSec != 120 {
		t.Errorf("timeout_sec: got %d want 120", got.TimeoutSec)
	}
	if got.Schedule.Kind != ScheduleKindInterval || got.Schedule.Interval != 5*time.Minute {
		t.Errorf("schedule not updated: %+v", got.Schedule)
	}
}

// TestDeleteJobCascades verifies that deleting a job removes all its run records.
func TestDeleteJobCascades(t *testing.T) {
	s := NewStore(newTestDB(t))
	if err := s.CreateJob(sampleJob("cascade-me")); err != nil {
		t.Fatal(err)
	}
	// Insert two runs.
	for i := 0; i < 2; i++ {
		r := &Run{JobName: "cascade-me", StartTime: time.Now(), Status: RunStatusSuccess}
		if err := s.InsertRun(r); err != nil {
			t.Fatal(err)
		}
	}

	// Delete the job.
	if err := s.DeleteJob("cascade-me"); err != nil {
		t.Fatal(err)
	}

	// Job should be gone.
	if got, _ := s.GetJob("cascade-me"); got != nil {
		t.Error("job should be deleted")
	}

	// Runs should be cascade-deleted (check raw DB since ListRuns needs the job).
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM scheduler_runs WHERE job_name='cascade-me'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("expected 0 cascade-deleted runs, got %d", count)
	}
}

// TestListJobs verifies that all jobs are returned, ordered by name.
func TestListJobs(t *testing.T) {
	s := NewStore(newTestDB(t))
	for _, name := range []string{"charlie", "alpha", "bravo"} {
		if err := s.CreateJob(sampleJob(name)); err != nil {
			t.Fatal(err)
		}
	}
	jobs, err := s.ListJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs))
	}
	want := []string{"alpha", "bravo", "charlie"}
	for i, w := range want {
		if jobs[i].Name != w {
			t.Errorf("jobs[%d].Name = %q, want %q", i, jobs[i].Name, w)
		}
	}
}

// TestInsertRunAndListRuns verifies paginated retrieval ordered by start_time DESC.
func TestInsertRunAndListRuns(t *testing.T) {
	s := NewStore(newTestDB(t))
	if err := s.CreateJob(sampleJob("runs-job")); err != nil {
		t.Fatal(err)
	}

	base := time.Now().Truncate(time.Second)
	for i := 0; i < 3; i++ {
		r := &Run{
			JobName:   "runs-job",
			StartTime: base.Add(time.Duration(i) * time.Second),
			Status:    RunStatusSuccess,
		}
		if err := s.InsertRun(r); err != nil {
			t.Fatal(err)
		}
	}

	runs, total, err := s.ListRuns("runs-job", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Errorf("total: got %d want 3", total)
	}
	if len(runs) != 3 {
		t.Errorf("len(runs): got %d want 3", len(runs))
	}
	// Most recent first.
	if !runs[0].StartTime.After(runs[1].StartTime) {
		t.Error("runs not ordered start_time DESC")
	}
	if !runs[1].StartTime.After(runs[2].StartTime) {
		t.Error("runs not ordered start_time DESC (middle pair)")
	}
}

// TestListRunsPagination verifies that offset/limit returns correct pages.
func TestListRunsPagination(t *testing.T) {
	s := NewStore(newTestDB(t))
	if err := s.CreateJob(sampleJob("page-job")); err != nil {
		t.Fatal(err)
	}

	base := time.Now().Truncate(time.Second)
	for i := 0; i < 5; i++ {
		r := &Run{
			JobName:   "page-job",
			StartTime: base.Add(time.Duration(i) * time.Second),
			Status:    RunStatusSuccess,
		}
		if err := s.InsertRun(r); err != nil {
			t.Fatal(err)
		}
	}

	page1, total, err := s.ListRuns("page-job", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 5 {
		t.Errorf("total: got %d want 5", total)
	}
	if len(page1) != 2 {
		t.Errorf("page1 len: got %d want 2", len(page1))
	}

	page2, _, err := s.ListRuns("page-job", 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 2 {
		t.Errorf("page2 len: got %d want 2", len(page2))
	}
	if page1[0].ID == page2[0].ID || page1[1].ID == page2[0].ID {
		t.Error("pages overlap: same run ID appears on both pages")
	}

	page3, _, err := s.ListRuns("page-job", 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page3) != 1 {
		t.Errorf("page3 len: got %d want 1 (last item)", len(page3))
	}
}

// TestPruneOldRuns verifies that only runs older than the retention window are removed.
func TestPruneOldRuns(t *testing.T) {
	s := NewStore(newTestDB(t))
	if err := s.CreateJob(sampleJob("prune-job")); err != nil {
		t.Fatal(err)
	}

	old := time.Now().AddDate(0, 0, -10) // 10 days ago
	for i := 0; i < 3; i++ {
		r := &Run{
			JobName:   "prune-job",
			StartTime: old.Add(time.Duration(i) * time.Second),
			Status:    RunStatusSuccess,
		}
		if err := s.InsertRun(r); err != nil {
			t.Fatal(err)
		}
	}
	recentRun := &Run{
		JobName:   "prune-job",
		StartTime: time.Now().Add(-time.Hour),
		Status:    RunStatusSuccess,
	}
	if err := s.InsertRun(recentRun); err != nil {
		t.Fatal(err)
	}

	// Retain 7 days — the 3 old runs should be deleted, the recent one kept.
	if err := s.PruneOldRuns(7); err != nil {
		t.Fatal(err)
	}

	_, total, err := s.ListRuns("prune-job", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Errorf("expected 1 run after pruning, got %d", total)
	}
}

// TestPruneOldRunsDeletesLogFiles verifies that log files referenced by pruned runs
// are removed from disk.
func TestPruneOldRunsDeletesLogFiles(t *testing.T) {
	s := NewStore(newTestDB(t))
	if err := s.CreateJob(sampleJob("logprune-job")); err != nil {
		t.Fatal(err)
	}

	// Create a real log file on disk.
	logFile := filepath.Join(t.TempDir(), "old.log")
	if err := os.WriteFile(logFile, []byte("old log content"), 0o644); err != nil {
		t.Fatal(err)
	}

	old := time.Now().AddDate(0, 0, -10)
	r := &Run{
		JobName:   "logprune-job",
		StartTime: old,
		Status:    RunStatusSuccess,
		LogPath:   logFile,
	}
	if err := s.InsertRun(r); err != nil {
		t.Fatal(err)
	}

	if err := s.PruneOldRuns(7); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(logFile); !os.IsNotExist(err) {
		t.Error("expected log file to be deleted by pruner")
	}
}
