// SPDX-License-Identifier: AGPL-3.0-or-later

// Package queue implements the agent work queue: a SQLite-backed FIFO of
// pending agent runs with rate-limit auto-pause and WebSocket event
// broadcasting.
package queue

import "time"

// JobState is the lifecycle state of a queued job.
type JobState string

const (
	StatePending   JobState = "pending"
	StateRunning   JobState = "running"
	StateCompleted JobState = "completed"
	StateFailed    JobState = "failed"
	StateSkipped   JobState = "skipped"
	StateCancelled JobState = "cancelled"
)

// terminalStates is the set of states from which no further transitions occur.
var terminalStates = map[JobState]bool{
	StateCompleted: true,
	StateFailed:    true,
	StateSkipped:   true,
	StateCancelled: true,
}

// isTerminal reports whether s is a terminal state.
func isTerminal(s JobState) bool { return terminalStates[s] }

// Job is one unit of work in the queue.
type Job struct {
	ID           string    `json:"id"`
	Project      string    `json:"project"`
	ArtifactPath string    `json:"artifact_path"`
	AgentName    string    `json:"agent_name"`
	State        JobState  `json:"state"`
	Reason       string    `json:"reason,omitempty"`
	Attempts     int       `json:"attempts"`
	EnqueuedAt   time.Time `json:"enqueued_at"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	FinishedAt   time.Time `json:"finished_at,omitempty"`
	Position     int64     `json:"position"`
	EnqueuedBy   string    `json:"enqueued_by"`
}
