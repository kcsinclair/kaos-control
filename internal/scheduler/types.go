// Package scheduler implements a tick-based job scheduler with SQLite persistence,
// precondition evaluation, priority queuing, and concurrency control.
package scheduler

import (
	"time"
)

// RunStatus is the terminal or in-flight state of a scheduler run.
type RunStatus string

const (
	RunStatusRunning RunStatus = "running"
	RunStatusSuccess RunStatus = "success"
	RunStatusFailure RunStatus = "failure"
	RunStatusTimeout RunStatus = "timeout"
	RunStatusSkipped RunStatus = "skipped"
)

// ScheduleKind distinguishes the three scheduling modes.
type ScheduleKind string

const (
	ScheduleKindCron     ScheduleKind = "cron"
	ScheduleKindInterval ScheduleKind = "interval"
	ScheduleKindOneOff   ScheduleKind = "one_off"
)

// ScheduleSpec describes when a job should fire.
// Exactly one of Cron, Interval, or At should be set, corresponding to Kind.
type ScheduleSpec struct {
	Kind     ScheduleKind  `json:"kind"`
	Cron     string        `json:"cron,omitempty"`     // 5 or 6-field cron expression
	Interval time.Duration `json:"interval,omitempty"` // for "interval" schedules
	At       time.Time     `json:"at,omitempty"`       // for "one_off" schedules
}

// PreconditionKind is the type of a precondition gate.
type PreconditionKind string

const (
	PreconditionAfterJob   PreconditionKind = "after_job"
	PreconditionFileExists PreconditionKind = "file_exists"
	PreconditionHTTPOk     PreconditionKind = "http_ok"
	PreconditionShell      PreconditionKind = "shell"
)

// Precondition is one gate that must pass before a job executes.
type Precondition struct {
	Kind    PreconditionKind `json:"kind"`
	JobName string           `json:"job_name,omitempty"` // for after_job
	Path    string           `json:"path,omitempty"`     // for file_exists
	URL     string           `json:"url,omitempty"`      // for http_ok
	Command string           `json:"command,omitempty"`  // for shell
}

// Job is a persisted scheduled-task definition.
type Job struct {
	Name          string            `json:"name"`
	TargetType    string            `json:"target_type"`       // "agent" | "shell"
	Target        string            `json:"target"`            // agent name or shell script path
	Args          map[string]string `json:"args,omitempty"`    // arbitrary key/value args
	Schedule      ScheduleSpec      `json:"schedule"`
	Preconditions []Precondition    `json:"preconditions,omitempty"`
	Enabled       bool              `json:"enabled"`
	Priority      int               `json:"priority"` // 1 (lowest) – 10 (highest)
	TimeoutSec    int               `json:"timeout_sec"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`

	// NextRunAt is computed at query time and not persisted.
	NextRunAt *time.Time `json:"next_run_at,omitempty"`
}

// Run is one execution record for a Job.
type Run struct {
	ID        int64      `json:"id"`
	JobName   string     `json:"job_name"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Status    RunStatus  `json:"status"`
	LogPath   string     `json:"log_path,omitempty"`
	CreatedAt time.Time  `json:"created_at"`

	// DurationMS is populated when EndTime is set.
	DurationMS *int64 `json:"duration_ms,omitempty"`
}
