// SPDX-License-Identifier: AGPL-3.0-or-later

package devops

// WebSocket event type constants for pipeline execution.
const (
	EventRunStarted    = "pipeline.run.started"
	EventStepStarted   = "pipeline.step.started"
	EventStepOutput    = "pipeline.step.output"
	EventStepCompleted = "pipeline.step.completed"
	EventRunCompleted  = "pipeline.run.completed"
)

// RunStartedPayload is the payload for pipeline.run.started events.
type RunStartedPayload struct {
	RunID    string `json:"run_id"`
	Pipeline string `json:"pipeline_slug"`
	Project  string `json:"project"`
}

// StepStartedPayload is the payload for pipeline.step.started events.
type StepStartedPayload struct {
	RunID     string `json:"run_id"`
	Pipeline  string `json:"pipeline_slug"`
	Step      string `json:"step"`
	StepIndex int    `json:"step_index"`
	Timestamp string `json:"timestamp"` // RFC 3339
}

// StepOutputPayload is the payload for pipeline.step.output events.
// Stream is "stdout" or "stderr".
type StepOutputPayload struct {
	RunID     string `json:"run_id"`
	Pipeline  string `json:"pipeline_slug"`
	Step      string `json:"step"`
	StepIndex int    `json:"step_index"`
	Text      string `json:"text"`
	Stream    string `json:"stream"`
	Timestamp string `json:"timestamp"` // RFC 3339
}

// StepCompletedPayload is the payload for pipeline.step.completed events.
type StepCompletedPayload struct {
	RunID           string  `json:"run_id"`
	Pipeline        string  `json:"pipeline_slug"`
	Step            string  `json:"step"`
	StepIndex       int     `json:"step_index"`
	Status          string  `json:"status"` // "passed", "failed", "cancelled"
	ExitCode        int     `json:"exit_code"`
	DurationSeconds float64 `json:"duration_seconds"`
}

// RunCompletedPayload is the payload for pipeline.run.completed events.
type RunCompletedPayload struct {
	RunID           string  `json:"run_id"`
	Pipeline        string  `json:"pipeline_slug"`
	Project         string  `json:"project"`
	Status          string  `json:"status"` // "passed", "failed", "cancelled"
	DurationSeconds float64 `json:"duration_seconds"`
}
