// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import "github.com/kaos-control/kaos-control/internal/architecture"

// scaffoldStepKey identifies the agent-directives offering in the wizard's
// optional scaffolding step.
const scaffoldStepKey = "agent-directives"

// Scaffolder adapts directive generation to the architecture.Scaffolder seam
// that the Architecture Wizard's optional final step calls (FR-17/FR-18).
// Register it once at startup via architecture.RegisterScaffolder; until then
// the wizard reports scaffolding as unavailable.
type Scaffolder struct{}

var _ architecture.Scaffolder = Scaffolder{}

// Available offers the single agent-directives step for any chosen
// architecture + stack. Generation reads the promoted stack profile from disk
// (written by the wizard's commit), so it needs no naming choices up front.
func (Scaffolder) Available(archSlug, stackSlug string) ([]architecture.ScaffoldStep, bool) {
	return []architecture.ScaffoldStep{{
		Key:   scaffoldStepKey,
		Title: "Agent directive files",
		Description: "Generate AGENTS.md (with CLAUDE.md and GEMINI.md pointing to it) and tune " +
			"the agent prompts in lifecycle/config.yaml — repo layout, per-role write paths, and " +
			"build/lint/test commands — to the chosen stack.",
	}}, true
}

// Run generates the directive set and config patch under projectRoot, mapping
// the write report onto the wizard's applied/skipped result. Choices are
// ignored: this step has no naming fields.
func (Scaffolder) Run(projectRoot, archSlug, stackSlug string, choices []architecture.ScaffoldChoice) (architecture.ScaffoldResult, error) {
	res, err := Generate(projectRoot, GenerateOptions{})
	if err != nil {
		return architecture.ScaffoldResult{}, err
	}

	out := architecture.ScaffoldResult{}
	for _, f := range res.Files {
		switch {
		case f.Created || f.Changed:
			out.Applied = append(out.Applied, f.Path)
		case f.Skipped || f.Diff != "":
			out.Skipped = append(out.Skipped, f.Path)
		}
	}
	out.Skipped = append(out.Skipped, res.Skipped...)
	return out, nil
}
