// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import (
	"os"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/devops"
	kgit "github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/sandbox"
)

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
func (Scaffolder) Available(projectRoot, archSlug, stackSlug string) ([]architecture.ScaffoldStep, bool) {
	return []architecture.ScaffoldStep{{
		Key:   scaffoldStepKey,
		Title: "Agent directives + devops pipelines",
		Description: "Generate AGENTS.md (with CLAUDE.md and GEMINI.md pointing to it), tune the " +
			"agent prompts in lifecycle/config.yaml — repo layout, per-role write paths, and " +
			"build/lint/test commands — to the chosen stack, and bootstrap build/lint/test " +
			"pipelines under lifecycle/devops/ from the stack profile.",
		Present: directivesPresent(projectRoot),
	}}, true
}

// directivesPresent reports whether a run of the agent-directives step would
// have nothing left to do: AGENTS.md and CLAUDE.md already exist, and
// GEMINI.md exists too if a gemini driver is configured (mirrors the files
// Generate writes — see hasGeminiDriver). Read-only: it never calls
// Generate, and resolves each filename through sandbox.Resolve before
// os.Stat (FR-6). A config-load error is treated fail-safe as "not present"
// so the step is still offered rather than erroring a read-only call.
func directivesPresent(projectRoot string) bool {
	if !scaffoldFileExists(projectRoot, agentsFile) || !scaffoldFileExists(projectRoot, claudeFile) {
		return false
	}
	drivers, err := configuredDrivers(projectRoot)
	if err != nil {
		return false
	}
	if hasGeminiDriver(drivers) {
		return scaffoldFileExists(projectRoot, geminiFile)
	}
	return true
}

// scaffoldFileExists resolves name (a fixed root-level filename) against
// projectRoot through the sandbox and reports whether it exists. An
// unresolvable/out-of-root path fails closed to false (FR-6).
func scaffoldFileExists(projectRoot, name string) bool {
	resolved, err := sandbox.Resolve(projectRoot, name)
	if err != nil {
		return false
	}
	_, err = os.Stat(resolved)
	return err == nil
}

// Run generates the directive set and config patch under projectRoot, mapping
// the write report onto the wizard's applied/skipped result. It scaffolds
// only when the agent-directives choice has Selected == true; otherwise it
// writes nothing and returns a zero ScaffoldResult (FR-10/FR-11).
func (Scaffolder) Run(projectRoot, archSlug, stackSlug string, choices []architecture.ScaffoldChoice) (architecture.ScaffoldResult, error) {
	if !selected(choices, scaffoldStepKey) {
		return architecture.ScaffoldResult{}, nil
	}

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

	// Bootstrap build/lint/test devops pipelines from the promoted stack's
	// stack_profile (skip-if-exists). Best-effort: a project without a promoted
	// stack simply gets no pipelines rather than failing the whole scaffold.
	if profile, _, perr := architecture.LoadPromotedStackProfile(projectRoot); perr == nil {
		pipelines, berr := devops.BootstrapPipelines(projectRoot, profile)
		if berr != nil {
			return out, berr
		}
		out.Applied = append(out.Applied, pipelines...)
	}

	// Track the generated files under git per the new-folder policy:
	// auto-commit for a repo kaos-control created, else hand back the commands.
	committed, cmds, err := kgit.TrackGenerated(projectRoot, out.Applied, "kaos-control: agent directives + devops pipelines")
	if err != nil {
		return out, err
	}
	out.Committed = committed
	out.GitCommands = cmds
	return out, nil
}

// selected reports whether choices contains an entry for stepKey with
// Selected == true. A missing entry, or one with Selected == false, means
// "do not scaffold this step" (FR-9/FR-11).
func selected(choices []architecture.ScaffoldChoice, stepKey string) bool {
	for _, c := range choices {
		if c.StepKey == stepKey && c.Selected {
			return true
		}
	}
	return false
}
