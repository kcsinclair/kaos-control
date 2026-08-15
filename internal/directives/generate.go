// SPDX-License-Identifier: AGPL-3.0-or-later

package directives

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/config"
)

// Canonical directive filenames, always at the project root.
const (
	agentsFile = "AGENTS.md"
	claudeFile = "CLAUDE.md"
	geminiFile = "GEMINI.md"
)

// GenerateResult reports what Generate wrote, skipped, and which standard
// agents ended up disabled for the stack.
type GenerateResult struct {
	// Files lists every AGENTS.md/CLAUDE.md/GEMINI.md write attempt, with
	// project-root-relative Path. Generate has no project/index handle of
	// its own — a caller that does have one should re-index each
	// Created/Changed file afterward so the graph reflects it (FR-15).
	Files []FileWrite
	// DisabledAgents lists standard agent names disabled because their
	// stack role is required: false (from PatchAgentConfig).
	DisabledAgents []string
	// Skipped names files intentionally not written this run (GEMINI.md
	// with no gemini driver configured — FR-12).
	Skipped []string
}

// GenerateOptions configures Generate.
type GenerateOptions struct {
	// Force allows overwriting a file whose managed-region markers are
	// missing (see writeFile) instead of withholding it behind a Diff.
	Force bool
	// Drivers lists the agent drivers configured for this project (e.g.
	// "claude-code-cli", "gemini-cli"). When nil, Generate reads them from
	// lifecycle/config.yaml itself. GEMINI.md is only written when a
	// "gemini-cli" or "gemini" driver is present (FR-12).
	Drivers []string
}

// Generate ties BuildModel, RenderAgents/RenderPointer, and
// PatchAgentConfig into one deterministic, idempotent operation (FR-10/
// FR-14, NFR-1/NFR-2 — no clock or network involved): it always writes
// AGENTS.md (canonical) and CLAUDE.md (an @AGENTS.md pointer), writes
// GEMINI.md only when a gemini driver is configured (driver-based
// selectivity, FR-12), and patches the six standard agents' config.yaml
// entries to match. No promoted stack is not an error: it falls back to
// GenericAgents (see Risk notes — "No promoted stack") so migration can run
// before the Architecture Wizard.
func Generate(projectRoot string, opts GenerateOptions) (GenerateResult, error) {
	drivers := opts.Drivers
	if drivers == nil {
		var err error
		drivers, err = configuredDrivers(projectRoot)
		if err != nil {
			return GenerateResult{}, err
		}
	}

	m, err := BuildModel(projectRoot)
	generic := false
	if err != nil {
		if !errors.Is(err, architecture.ErrNoPromotedStack) {
			return GenerateResult{}, fmt.Errorf("building directive model: %w", err)
		}
		generic = true
	}

	var agentsBody []byte
	if generic {
		agentsBody, err = GenericAgents(filepath.Base(filepath.Clean(projectRoot)), "")
	} else {
		agentsBody, err = RenderAgents(m)
	}
	if err != nil {
		return GenerateResult{}, fmt.Errorf("rendering %s: %w", agentsFile, err)
	}

	var result GenerateResult

	fw, err := writeFile(filepath.Join(projectRoot, agentsFile), agentsBody, opts.Force)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("writing %s: %w", agentsFile, err)
	}
	result.Files = append(result.Files, relativizeFileWrite(fw, agentsFile))

	pointer := RenderPointer(agentsFile)
	fw, err = writeFile(filepath.Join(projectRoot, claudeFile), pointer, opts.Force)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("writing %s: %w", claudeFile, err)
	}
	result.Files = append(result.Files, relativizeFileWrite(fw, claudeFile))

	if hasGeminiDriver(drivers) {
		fw, err = writeFile(filepath.Join(projectRoot, geminiFile), pointer, opts.Force)
		if err != nil {
			return GenerateResult{}, fmt.Errorf("writing %s: %w", geminiFile, err)
		}
		result.Files = append(result.Files, relativizeFileWrite(fw, geminiFile))
	} else {
		result.Skipped = append(result.Skipped, geminiFile)
	}

	if !generic {
		patchRes, err := PatchAgentConfig(projectRoot, m)
		if err != nil {
			return GenerateResult{}, fmt.Errorf("patching lifecycle/config.yaml: %w", err)
		}
		result.DisabledAgents = patchRes.Disabled
	}

	return result, nil
}

// GenericAgents renders the pre-wizard fallback AGENTS.md: the standing
// content (lineage convention, frontmatter vocab, commit conventions,
// roles) but with a placeholder repo-layout section and no stack-tuned
// commands, for projects that run migration before promoting an
// architecture.
func GenericAgents(projectName, language string) ([]byte, error) {
	m := DirectiveModel{
		ProjectName: projectName,
		Language:    language,
		RepoLayout: []architecture.RepoLayoutEntry{
			{Path: "lifecycle/", Note: "artifact store (ideas → requirements → plans → releases)"},
		},
		ArchitecturePointer: false,
	}
	return RenderAgents(m)
}

// relativizeFileWrite replaces fw.Path (an absolute path, as written by
// writeFile) with relName, the project-root-relative name Generate already
// knows it wrote.
func relativizeFileWrite(fw FileWrite, relName string) FileWrite {
	fw.Path = relName
	return fw
}

// configuredDrivers returns the distinct, non-empty Agent.Driver values
// configured in projectRoot's lifecycle/config.yaml, in file order.
func configuredDrivers(projectRoot string) ([]string, error) {
	cfg, err := config.LoadProject(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}
	seen := make(map[string]bool, len(cfg.Agents))
	var drivers []string
	for _, a := range cfg.Agents {
		if a.Driver == "" || seen[a.Driver] {
			continue
		}
		seen[a.Driver] = true
		drivers = append(drivers, a.Driver)
	}
	return drivers, nil
}

// hasGeminiDriver reports whether drivers includes a gemini-cli/gemini
// entry (FR-12).
func hasGeminiDriver(drivers []string) bool {
	for _, d := range drivers {
		if d == "gemini-cli" || d == "gemini" {
			return true
		}
	}
	return false
}
