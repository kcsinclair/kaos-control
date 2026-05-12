// SPDX-License-Identifier: AGPL-3.0-or-later

package queue

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// ErrCannotCancelRunning is returned when Cancel is called on a running job.
var ErrCannotCancelRunning = errors.New("cannot cancel a running job")

// ErrDuplicateActive is returned when an identical active job already exists.
var ErrDuplicateActive = errors.New("an active job for this artifact already exists")

// Store is the SQLite-backed queue store.
type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS jobs (
  id            TEXT PRIMARY KEY,
  project       TEXT NOT NULL,
  artifact_path TEXT NOT NULL,
  agent_name    TEXT NOT NULL,
  state         TEXT NOT NULL CHECK(state IN ('pending','running','completed','failed','skipped','cancelled')),
  reason        TEXT,
  attempts      INTEGER NOT NULL DEFAULT 1,
  enqueued_at   INTEGER NOT NULL,
  started_at    INTEGER,
  finished_at   INTEGER,
  position      INTEGER NOT NULL,
  enqueued_by   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_jobs_state_position ON jobs(state, position);
CREATE INDEX IF NOT EXISTS idx_jobs_project_path ON jobs(project, artifact_path);

CREATE TABLE IF NOT EXISTS queue_state (
  k TEXT PRIMARY KEY,
  v TEXT NOT NULL
);
`

// Open opens (or creates) the queue database at path, applying the schema.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("queue: creating db dir: %w", err)
	}
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("queue: opening sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("queue: applying schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// newID generates a 16-character hex random ID.
func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Enqueue inserts a new job, enforcing the duplicate-active constraint atomically.
// It assigns a new ID and position if they are zero/empty.
func (s *Store) Enqueue(j Job) error {
	if j.ID == "" {
		j.ID = newID()
	}
	if j.EnqueuedAt.IsZero() {
		j.EnqueuedAt = time.Now()
	}
	if j.Attempts == 0 {
		j.Attempts = 1
	}

	// position: if not set, use next monotonic value.
	if j.Position == 0 {
		var maxPos sql.NullInt64
		row := s.db.QueryRow(`SELECT MAX(position) FROM jobs`)
		if err := row.Scan(&maxPos); err != nil {
			return fmt.Errorf("queue: enqueue max position: %w", err)
		}
		if maxPos.Valid {
			j.Position = maxPos.Int64 + 1
		} else {
			j.Position = 1
		}
	}

	// Atomic INSERT that fails if there is already a pending/running job for
	// this project+artifact_path (FR3 duplicate suppression).
	res, err := s.db.Exec(`
		INSERT INTO jobs (id, project, artifact_path, agent_name, state, reason,
		                  attempts, enqueued_at, started_at, finished_at, position, enqueued_by)
		SELECT ?,?,?,?,?,?,?,?,NULL,NULL,?,?
		WHERE NOT EXISTS (
			SELECT 1 FROM jobs
			WHERE project = ? AND artifact_path = ? AND state IN ('pending','running')
		)`,
		j.ID, j.Project, j.ArtifactPath, j.AgentName, string(StatePending), j.Reason,
		j.Attempts, j.EnqueuedAt.Unix(), j.Position, j.EnqueuedBy,
		j.Project, j.ArtifactPath,
	)
	if err != nil {
		return fmt.Errorf("queue: enqueue insert: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("queue: enqueue rows affected: %w", err)
	}
	if n == 0 {
		return ErrDuplicateActive
	}
	return nil
}

// EnqueueDirect inserts a job bypassing the duplicate check (used for
// re-queueing after rate-limit: position may be set to head).
func (s *Store) EnqueueDirect(j Job) error {
	if j.ID == "" {
		j.ID = newID()
	}
	if j.EnqueuedAt.IsZero() {
		j.EnqueuedAt = time.Now()
	}
	if j.Attempts == 0 {
		j.Attempts = 1
	}
	_, err := s.db.Exec(`
		INSERT INTO jobs (id, project, artifact_path, agent_name, state, reason,
		                  attempts, enqueued_at, started_at, finished_at, position, enqueued_by)
		VALUES (?,?,?,?,?,?,?,?,NULL,NULL,?,?)`,
		j.ID, j.Project, j.ArtifactPath, j.AgentName, string(StatePending), j.Reason,
		j.Attempts, j.EnqueuedAt.Unix(), j.Position, j.EnqueuedBy,
	)
	if err != nil {
		return fmt.Errorf("queue: enqueue direct: %w", err)
	}
	return nil
}

// MinPosition returns the smallest position value across all jobs, or 0 if
// the table is empty. Used to insert a re-queued job at the head.
func (s *Store) MinPosition() int64 {
	var v sql.NullInt64
	_ = s.db.QueryRow(`SELECT MIN(position) FROM jobs`).Scan(&v)
	if v.Valid {
		return v.Int64
	}
	return 0
}

// Dequeue selects the head pending job, marks it running, and returns it.
// Returns (nil, nil) when the queue is empty.
func (s *Store) Dequeue() (*Job, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("queue: dequeue begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRow(`
		SELECT id, project, artifact_path, agent_name, state, COALESCE(reason,''),
		       attempts, enqueued_at, started_at, finished_at, position, enqueued_by
		FROM jobs
		WHERE state = 'pending'
		ORDER BY position ASC
		LIMIT 1`)

	j, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("queue: dequeue scan: %w", err)
	}

	now := time.Now().Unix()
	if _, err := tx.Exec(`UPDATE jobs SET state='running', started_at=? WHERE id=?`, now, j.ID); err != nil {
		return nil, fmt.Errorf("queue: dequeue update: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("queue: dequeue commit: %w", err)
	}
	j.State = StateRunning
	j.StartedAt = time.Unix(now, 0)
	return j, nil
}

// MarkTerminal transitions a job to a terminal state.
func (s *Store) MarkTerminal(id string, state JobState, reason string) error {
	if !isTerminal(state) {
		return fmt.Errorf("queue: %q is not a terminal state", state)
	}
	now := time.Now().Unix()
	res, err := s.db.Exec(`UPDATE jobs SET state=?, reason=?, finished_at=? WHERE id=?`,
		string(state), reason, now, id)
	if err != nil {
		return fmt.Errorf("queue: mark terminal: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("queue: mark terminal: job %q not found", id)
	}
	return nil
}

// ListByState returns all jobs in the given states, ordered by position ASC.
func (s *Store) ListByState(states ...JobState) ([]*Job, error) {
	if len(states) == 0 {
		return nil, nil
	}
	args := make([]any, len(states))
	placeholders := ""
	for i, st := range states {
		args[i] = string(st)
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
	}
	rows, err := s.db.Query(`
		SELECT id, project, artifact_path, agent_name, state, COALESCE(reason,''),
		       attempts, enqueued_at, started_at, finished_at, position, enqueued_by
		FROM jobs WHERE state IN (`+placeholders+`) ORDER BY position ASC`, args...)
	if err != nil {
		return nil, fmt.Errorf("queue: list by state: %w", err)
	}
	defer rows.Close()
	return scanJobs(rows)
}

// ListRecent returns the last n terminal jobs ordered by finished_at DESC.
func (s *Store) ListRecent(n int) ([]*Job, error) {
	rows, err := s.db.Query(`
		SELECT id, project, artifact_path, agent_name, state, COALESCE(reason,''),
		       attempts, enqueued_at, started_at, finished_at, position, enqueued_by
		FROM jobs
		WHERE state IN ('completed','failed','skipped','cancelled')
		ORDER BY COALESCE(finished_at, enqueued_at) DESC
		LIMIT ?`, n)
	if err != nil {
		return nil, fmt.Errorf("queue: list recent: %w", err)
	}
	defer rows.Close()
	return scanJobs(rows)
}

// GetByID returns a single job by its ID.
func (s *Store) GetByID(id string) (*Job, error) {
	row := s.db.QueryRow(`
		SELECT id, project, artifact_path, agent_name, state, COALESCE(reason,''),
		       attempts, enqueued_at, started_at, finished_at, position, enqueued_by
		FROM jobs WHERE id=?`, id)
	j, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("queue: get by id: %w", err)
	}
	return j, nil
}

// FindActiveByPath returns a pending-or-running job for the given project+path,
// or nil when none exists.
func (s *Store) FindActiveByPath(project, path string) (*Job, error) {
	row := s.db.QueryRow(`
		SELECT id, project, artifact_path, agent_name, state, COALESCE(reason,''),
		       attempts, enqueued_at, started_at, finished_at, position, enqueued_by
		FROM jobs
		WHERE project=? AND artifact_path=? AND state IN ('pending','running')
		LIMIT 1`, project, path)
	j, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("queue: find active: %w", err)
	}
	return j, nil
}

// Cancel transitions a pending job to cancelled. Returns ErrCannotCancelRunning
// for running jobs.
func (s *Store) Cancel(id string) error {
	j, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if j == nil {
		return fmt.Errorf("queue: cancel: job %q not found", id)
	}
	if j.State == StateRunning {
		return ErrCannotCancelRunning
	}
	if isTerminal(j.State) {
		return fmt.Errorf("queue: cancel: job %q is already in terminal state %q", id, j.State)
	}
	return s.MarkTerminal(id, StateCancelled, "cancelled_by_user")
}

// RecoverOrphans is called at startup: any rows still in state=running are
// moved back to pending with attempts incremented (they were orphaned by a
// server crash).
func (s *Store) RecoverOrphans() error {
	_, err := s.db.Exec(`
		UPDATE jobs SET state='pending', started_at=NULL, attempts=attempts+1
		WHERE state='running'`)
	if err != nil {
		return fmt.Errorf("queue: recover orphans: %w", err)
	}
	return nil
}

// GetPauseState reads the queue pause state from the queue_state table.
func (s *Store) GetPauseState() (paused bool, until time.Time, reason string, _ error) {
	rows, err := s.db.Query(`SELECT k, v FROM queue_state WHERE k IN ('paused','paused_until','pause_reason')`)
	if err != nil {
		return false, time.Time{}, "", fmt.Errorf("queue: get pause state: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return false, time.Time{}, "", fmt.Errorf("queue: get pause state scan: %w", err)
		}
		switch k {
		case "paused":
			paused = v == "true"
		case "paused_until":
			if v != "" {
				t, err := time.Parse(time.RFC3339Nano, v)
				if err == nil {
					until = t
				}
			}
		case "pause_reason":
			reason = v
		}
	}
	return paused, until, reason, rows.Err()
}

// SetPauseState writes the queue pause state atomically.
func (s *Store) SetPauseState(paused bool, until time.Time, reason string) error {
	pausedStr := "false"
	if paused {
		pausedStr = "true"
	}
	untilStr := ""
	if !until.IsZero() {
		untilStr = until.UTC().Format(time.RFC3339Nano)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("queue: set pause state begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	upsert := `INSERT INTO queue_state(k,v) VALUES(?,?) ON CONFLICT(k) DO UPDATE SET v=excluded.v`
	for _, kv := range [][2]string{{"paused", pausedStr}, {"paused_until", untilStr}, {"pause_reason", reason}} {
		if _, err := tx.Exec(upsert, kv[0], kv[1]); err != nil {
			return fmt.Errorf("queue: set pause state upsert %q: %w", kv[0], err)
		}
	}
	return tx.Commit()
}

// ---- helpers ----

type scanner interface {
	Scan(dest ...any) error
}

func scanJob(row scanner) (*Job, error) {
	var (
		id, project, artifactPath, agentName, state, reason, enqueuedBy string
		attempts                                                         int
		enqueuedAt                                                       int64
		startedAt, finishedAt                                            sql.NullInt64
		position                                                         int64
	)
	err := row.Scan(&id, &project, &artifactPath, &agentName, &state, &reason,
		&attempts, &enqueuedAt, &startedAt, &finishedAt, &position, &enqueuedBy)
	if err != nil {
		return nil, err
	}
	j := &Job{
		ID:           id,
		Project:      project,
		ArtifactPath: artifactPath,
		AgentName:    agentName,
		State:        JobState(state),
		Reason:       reason,
		Attempts:     attempts,
		EnqueuedAt:   time.Unix(enqueuedAt, 0),
		Position:     position,
		EnqueuedBy:   enqueuedBy,
	}
	if startedAt.Valid {
		j.StartedAt = time.Unix(startedAt.Int64, 0)
	}
	if finishedAt.Valid {
		j.FinishedAt = time.Unix(finishedAt.Int64, 0)
	}
	return j, nil
}

func scanJobs(rows *sql.Rows) ([]*Job, error) {
	var out []*Job
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}
