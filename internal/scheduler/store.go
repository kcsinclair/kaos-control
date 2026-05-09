// SPDX-License-Identifier: AGPL-3.0-or-later

package scheduler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Store provides CRUD access to scheduler_jobs and scheduler_runs via the
// shared project SQLite database. The caller must ensure concurrent writes are
// serialised externally (the index already sets MaxOpenConns=1 on the DB).
type Store struct {
	db *sql.DB
}

// NewStore wraps the given *sql.DB (shared with the artifact index).
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ----- Job CRUD -----

// ListJobs returns all jobs ordered by name.
func (s *Store) ListJobs() ([]*Job, error) {
	rows, err := s.db.Query(`
		SELECT name, target_type, target, args_json, schedule,
		       preconditions_json, enabled, priority, timeout_sec, created_at, updated_at
		FROM scheduler_jobs ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []*Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// GetJob returns a single job by name, or nil if not found.
func (s *Store) GetJob(name string) (*Job, error) {
	rows, err := s.db.Query(`
		SELECT name, target_type, target, args_json, schedule,
		       preconditions_json, enabled, priority, timeout_sec, created_at, updated_at
		FROM scheduler_jobs WHERE name = ?`, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	return scanJob(rows)
}

// CreateJob inserts a new job. Returns an error if the name already exists.
func (s *Store) CreateJob(j *Job) error {
	argsJSON, err := marshalNullableJSON(j.Args)
	if err != nil {
		return fmt.Errorf("marshalling args: %w", err)
	}
	schedJSON, err := json.Marshal(j.Schedule)
	if err != nil {
		return fmt.Errorf("marshalling schedule: %w", err)
	}
	preJSON, err := marshalNullableJSON(j.Preconditions)
	if err != nil {
		return fmt.Errorf("marshalling preconditions: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`
		INSERT INTO scheduler_jobs
			(name, target_type, target, args_json, schedule, preconditions_json,
			 enabled, priority, timeout_sec, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		j.Name, j.TargetType, j.Target, argsJSON, string(schedJSON), preJSON,
		boolInt(j.Enabled), j.Priority, j.TimeoutSec, now, now,
	)
	return err
}

// UpdateJob overwrites all mutable fields for the named job.
func (s *Store) UpdateJob(j *Job) error {
	argsJSON, err := marshalNullableJSON(j.Args)
	if err != nil {
		return fmt.Errorf("marshalling args: %w", err)
	}
	schedJSON, err := json.Marshal(j.Schedule)
	if err != nil {
		return fmt.Errorf("marshalling schedule: %w", err)
	}
	preJSON, err := marshalNullableJSON(j.Preconditions)
	if err != nil {
		return fmt.Errorf("marshalling preconditions: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.Exec(`
		UPDATE scheduler_jobs SET
			target_type=?, target=?, args_json=?, schedule=?, preconditions_json=?,
			enabled=?, priority=?, timeout_sec=?, updated_at=?
		WHERE name=?`,
		j.TargetType, j.Target, argsJSON, string(schedJSON), preJSON,
		boolInt(j.Enabled), j.Priority, j.TimeoutSec, now,
		j.Name,
	)
	return err
}

// DeleteJob removes the job and cascade-deletes its runs.
func (s *Store) DeleteJob(name string) error {
	_, err := s.db.Exec(`DELETE FROM scheduler_jobs WHERE name=?`, name)
	return err
}

// SetEnabled sets the enabled flag for a job.
func (s *Store) SetEnabled(name string, enabled bool) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE scheduler_jobs SET enabled=?, updated_at=? WHERE name=?`,
		boolInt(enabled), now, name)
	return err
}

// ----- Run CRUD -----

// InsertRun inserts a new run record and sets r.ID from LastInsertId.
func (s *Store) InsertRun(r *Run) error {
	now := time.Now().UTC().Format(time.RFC3339)
	start := r.StartTime.UTC().Format(time.RFC3339)
	res, err := s.db.Exec(`
		INSERT INTO scheduler_runs (job_name, start_time, status, log_path, created_at)
		VALUES (?,?,?,?,?)`,
		r.JobName, start, string(r.Status), nullableString(r.LogPath), now,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	r.ID = id
	return nil
}

// UpdateRun updates the mutable fields of an existing run (end_time, status).
func (s *Store) UpdateRun(r *Run) error {
	var endTime *string
	if r.EndTime != nil {
		s := r.EndTime.UTC().Format(time.RFC3339)
		endTime = &s
	}
	_, err := s.db.Exec(`
		UPDATE scheduler_runs SET end_time=?, status=?, log_path=?
		WHERE id=?`,
		endTime, string(r.Status), nullableString(r.LogPath), r.ID,
	)
	return err
}

// GetRun returns a run by ID, or nil if not found.
func (s *Store) GetRun(id int64) (*Run, error) {
	rows, err := s.db.Query(`
		SELECT id, job_name, start_time, end_time, status, log_path, created_at
		FROM scheduler_runs WHERE id=?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	return scanRun(rows)
}

// ListRuns returns paginated runs for a job ordered by start_time DESC.
// page is 1-based; perPage must be positive.
func (s *Store) ListRuns(jobName string, page, perPage int) ([]*Run, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM scheduler_runs WHERE job_name=?`, jobName).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * perPage
	rows, err := s.db.Query(`
		SELECT id, job_name, start_time, end_time, status, log_path, created_at
		FROM scheduler_runs WHERE job_name=?
		ORDER BY start_time DESC LIMIT ? OFFSET ?`, jobName, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, r)
	}
	return out, total, rows.Err()
}

// LastRunForJob returns the most recent run for a job, or nil.
func (s *Store) LastRunForJob(jobName string) (*Run, error) {
	rows, err := s.db.Query(`
		SELECT id, job_name, start_time, end_time, status, log_path, created_at
		FROM scheduler_runs WHERE job_name=?
		ORDER BY start_time DESC LIMIT 1`, jobName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, nil
	}
	return scanRun(rows)
}

// MarkStaleRunsFailed sets status=failure for any run in status=running.
// Called on startup to recover from a crash.
func (s *Store) MarkStaleRunsFailed() error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE scheduler_runs SET status=?, end_time=? WHERE status=?`,
		string(RunStatusFailure), now, string(RunStatusRunning))
	return err
}

// PruneOldRuns deletes runs older than retentionDays days and removes their log files.
func (s *Store) PruneOldRuns(retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays).UTC().Format(time.RFC3339)

	// Collect log paths before deleting.
	rows, err := s.db.Query(`SELECT log_path FROM scheduler_runs WHERE start_time < ? AND log_path IS NOT NULL`, cutoff)
	if err != nil {
		return err
	}
	var paths []string
	for rows.Next() {
		var p sql.NullString
		if err := rows.Scan(&p); err != nil {
			rows.Close()
			return err
		}
		if p.Valid && p.String != "" {
			paths = append(paths, p.String)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	if _, err := s.db.Exec(`DELETE FROM scheduler_runs WHERE start_time < ?`, cutoff); err != nil {
		return err
	}

	// Best-effort log file cleanup.
	for _, p := range paths {
		_ = os.Remove(p)
	}
	return nil
}

// ----- helpers -----

func scanJob(rows *sql.Rows) (*Job, error) {
	var j Job
	var argsJSON, preJSON sql.NullString
	var schedJSON string
	var enabledInt int
	var createdStr, updatedStr string

	err := rows.Scan(
		&j.Name, &j.TargetType, &j.Target,
		&argsJSON, &schedJSON, &preJSON,
		&enabledInt, &j.Priority, &j.TimeoutSec,
		&createdStr, &updatedStr,
	)
	if err != nil {
		return nil, err
	}
	j.Enabled = enabledInt != 0
	if argsJSON.Valid && argsJSON.String != "" {
		_ = json.Unmarshal([]byte(argsJSON.String), &j.Args)
	}
	if err := json.Unmarshal([]byte(schedJSON), &j.Schedule); err != nil {
		return nil, fmt.Errorf("unmarshal schedule for job %q: %w", j.Name, err)
	}
	if preJSON.Valid && preJSON.String != "" {
		_ = json.Unmarshal([]byte(preJSON.String), &j.Preconditions)
	}
	if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
		j.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, updatedStr); err == nil {
		j.UpdatedAt = t
	}
	return &j, nil
}

func scanRun(rows *sql.Rows) (*Run, error) {
	var r Run
	var endStr sql.NullString
	var logPath sql.NullString
	var startStr, createdStr string
	var statusStr string

	err := rows.Scan(
		&r.ID, &r.JobName, &startStr, &endStr, &statusStr, &logPath, &createdStr,
	)
	if err != nil {
		return nil, err
	}
	r.Status = RunStatus(statusStr)
	if t, err := time.Parse(time.RFC3339, startStr); err == nil {
		r.StartTime = t
	}
	if endStr.Valid && endStr.String != "" {
		if t, err := time.Parse(time.RFC3339, endStr.String); err == nil {
			r.EndTime = &t
			ms := t.Sub(r.StartTime).Milliseconds()
			r.DurationMS = &ms
		}
	}
	if logPath.Valid {
		r.LogPath = logPath.String
	}
	if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
		r.CreatedAt = t
	}
	return &r, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func marshalNullableJSON(v any) (*string, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	s := string(b)
	// Don't store bare null or empty objects/arrays as non-null.
	if s == "null" || s == "{}" || s == "[]" {
		return nil, nil
	}
	return &s, nil
}
