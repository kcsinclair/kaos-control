// SPDX-License-Identifier: AGPL-3.0-or-later

package architecture

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// BreakingReq is one architecture-breaking requirement the questionnaire
// surfaced, and how the chosen architecture + stack satisfies it.
type BreakingReq struct {
	Label       string // the catalog decision-signal label the answer contributed
	Requirement string // plain-language statement of the requirement
	Mapping     string // how the chosen architecture + stack satisfies it
}

// QAPair is one question/answer pair from the wizard's Q&A trail.
type QAPair struct {
	Question string
	Answer   string
}

// SummaryInput is the content WriteSummary renders into
// architecture-summary.md. Architecture, TechStack, ADRPaths, and
// StandardPaths are repo-relative paths.
type SummaryInput struct {
	Architecture         string
	TechStack            string
	BreakingRequirements []BreakingReq
	QA                   []QAPair
	ADRPaths             []string
	StandardPaths        []string
}

// summaryFrontmatter is the frontmatter shape written for
// architecture-summary.md.
type summaryFrontmatter struct {
	Title   string `yaml:"title"`
	Type    string `yaml:"type"`
	Status  string `yaml:"status"`
	Created string `yaml:"created,omitempty"`
}

// WriteSummary deterministically (re)writes
// lifecycle/architecture/architecture-summary.md: the architecture-breaking
// requirements and their mapping to the chosen architecture + stack, the
// full Q&A trail, and links to the promoted architecture, stack, ADR(s), and
// any seeded standards (FR-14, FR-15). It is idempotent — re-writing with
// the same input overwrites in place rather than duplicating (NFR-2/NFR-3).
func WriteSummary(projectRoot string, in SummaryInput) (relPath string, err error) {
	fmBytes, err := yaml.Marshal(summaryFrontmatter{
		Title:   "Architecture Summary",
		Type:    "doc",
		Status:  "approved",
		Created: createdFor(filepath.Join(projectRoot, filepath.FromSlash(architectureDir), "architecture-summary.md")),
	})
	if err != nil {
		return "", fmt.Errorf("marshalling summary frontmatter: %w", err)
	}

	var body strings.Builder
	body.WriteString("---\n")
	body.Write(fmBytes)
	body.WriteString("---\n\n")
	body.WriteString("# Architecture Summary\n\n")

	body.WriteString("## Architecture-breaking requirements\n\n")
	if len(in.BreakingRequirements) == 0 {
		body.WriteString("None surfaced by the questionnaire.\n\n")
	} else {
		body.WriteString("| Requirement | Signal | How it's satisfied |\n")
		body.WriteString("| --- | --- | --- |\n")
		for _, req := range in.BreakingRequirements {
			body.WriteString("| " + tableCell(req.Requirement) + " | " + tableCell(req.Label) + " | " + tableCell(req.Mapping) + " |\n")
		}
		body.WriteString("\n")
	}

	body.WriteString("## Selection Q&A\n\n")
	if len(in.QA) == 0 {
		body.WriteString("No questions were answered.\n\n")
	} else {
		for _, qa := range in.QA {
			body.WriteString("- **Q:** " + qa.Question + "\n")
			body.WriteString("  **A:** " + qa.Answer + "\n")
		}
		body.WriteString("\n")
	}

	body.WriteString("## Links\n\n")
	body.WriteString("- Architecture: " + summaryLink(in.Architecture) + "\n")
	body.WriteString("- Tech stack: " + summaryLink(in.TechStack) + "\n")
	for _, adr := range in.ADRPaths {
		body.WriteString("- ADR: " + summaryLink(adr) + "\n")
	}
	for _, std := range in.StandardPaths {
		body.WriteString("- Standard: " + summaryLink(std) + "\n")
	}

	absDest := filepath.Join(projectRoot, filepath.FromSlash(architectureDir), "architecture-summary.md")
	if err := writeAtomic(absDest, []byte(body.String())); err != nil {
		return "", fmt.Errorf("writing architecture-summary.md: %w", err)
	}

	return path.Join(architectureDir, "architecture-summary.md"), nil
}

// tableCell escapes a value for embedding in a markdown table cell.
func tableCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

// summaryLink renders a repo-relative path as a markdown link relative to
// architecture-summary.md's own location (lifecycle/architecture/ root).
func summaryLink(relToProjectRoot string) string {
	if relToProjectRoot == "" {
		return "_none_"
	}
	target := relToProjectRoot
	if rel, ok := strings.CutPrefix(target, architectureDir+"/"); ok {
		target = rel
	}
	return "[" + filepath.Base(relToProjectRoot) + "](" + target + ")"
}
