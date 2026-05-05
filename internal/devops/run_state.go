package devops

import "time"

// StepStatus is the execution state of a single pipeline step.
type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepPassed    StepStatus = "passed"
	StepFailed    StepStatus = "failed"
	StepCancelled StepStatus = "cancelled"
)

// StepState records the runtime state of one step within a run.
type StepState struct {
	Name      string
	Status    StepStatus
	StartTime time.Time
	EndTime   time.Time
	ExitCode  int
}

// RunState tracks the overall runtime state of a pipeline run.
type RunState struct {
	RunID     string
	Pipeline  string
	Steps     []StepState
	StartTime time.Time
	EndTime   time.Time
}
