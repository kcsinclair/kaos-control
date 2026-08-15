// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture

// ScaffoldNameField is one naming choice a scaffolding step needs, with a
// computed "decide for me" default so a less-technical user can proceed
// without deciding anything (FR-18).
type ScaffoldNameField struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	DefaultValue string `json:"default_value"`
}

// ScaffoldStep describes one available scaffolding offering (config,
// pipelines, agent directives, repo skeleton) for a chosen architecture +
// stack (FR-17).
type ScaffoldStep struct {
	Key         string              `json:"key"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	NameFields  []ScaffoldNameField `json:"name_fields,omitempty"`
}

// ScaffoldChoice is the caller's answer for one ScaffoldStep: explicit
// values per NameField key, or UseDefaults to accept the "decide for me"
// defaults verbatim.
type ScaffoldChoice struct {
	StepKey     string            `json:"step_key"`
	Values      map[string]string `json:"values,omitempty"`
	UseDefaults bool              `json:"use_defaults"`
}

// ScaffoldResult reports what a Scaffolder run actually did.
type ScaffoldResult struct {
	Applied []string `json:"applied"`
	Skipped []string `json:"skipped"`
}

// Scaffolder generates scaffolding for a chosen architecture + stack. It is
// deliberately not built by this milestone — [[architecture-templates]] §4
// and [[agent-directives-generation]] own the concrete generators — this
// interface + registry is only the seam they plug into later, so the wizard
// never has a hard dependency on either landing (FR-17/FR-18).
type Scaffolder interface {
	// Available reports the scaffolding steps offered for archSlug/stackSlug,
	// or ok=false if this combination isn't supported.
	Available(archSlug, stackSlug string) (steps []ScaffoldStep, ok bool)
	// Run applies the chosen scaffolding steps under projectRoot.
	Run(projectRoot, archSlug, stackSlug string, choices []ScaffoldChoice) (ScaffoldResult, error)
}

// activeScaffolder holds the process-wide registered Scaffolder, set once at
// startup by whichever generator package is wired in (nil until one is).
var activeScaffolder Scaffolder

// RegisterScaffolder installs s as the active Scaffolder. Intended to be
// called once during startup wiring, not per-request.
func RegisterScaffolder(s Scaffolder) {
	activeScaffolder = s
}

// ActiveScaffolder returns the currently registered Scaffolder, or nil if
// none has been registered (the default, until a generator package lands).
func ActiveScaffolder() Scaffolder {
	return activeScaffolder
}
