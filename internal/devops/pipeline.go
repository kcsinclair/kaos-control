// Package devops implements pipeline definition parsing, execution, and run
// log persistence for the DevOps feature.
package devops

import "time"

// Pipeline is a parsed pipeline definition from a YAML file in lifecycle/devops/.
type Pipeline struct {
	Slug  string // derived from the filename (without .yaml)
	Name  string
	Type  string
	Steps []Step
}

// Step is one executable unit within a Pipeline.
type Step struct {
	Name        string
	Description string
	Command     string
	Timeout     time.Duration // defaults to 60s when not specified in YAML
}
