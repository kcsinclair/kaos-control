// SPDX-License-Identifier: AGPL-3.0-or-later

package ideachat

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ErrInputTooShort is returned by Generate when the input has fewer than 5 words.
var ErrInputTooShort = errors.New("input too short: provide at least 5 words describing your idea")

// GenerateOptions configures a single-shot idea, defect, or doc generation call.
type GenerateOptions struct {
	Input          string
	ArtifactType   string   // "idea", "defect", or "doc"; defaults to "idea"
	SourceLineage  string   // optional: lineage slug of the source artifact (doc flow)
	SourcePath     string   // optional: project-relative path of the source artifact (doc flow)
	SourceContext  string   // optional: pre-assembled context text from source lineage to inject into prompt
	ExistingLabels []string // label vocabulary to constrain LLM choices
	ExistingSlugs  []string // existing slugs for collision detection
	ModelCfg       ModelConfig
	SourcePriority string // optional: priority inherited from the source/parent artifact (FR-6)
	SourceRelease  string // optional: release inherited from the source/parent artifact (FR-6)
}

// GenerateResult holds the LLM-proposed artifact, ready for the caller to preview or persist.
type GenerateResult struct {
	Slug        string
	Title       string
	Labels      []string
	Body        string
	Frontmatter map[string]any
	TargetDir   string
}

// Generate sends opts.Input to the LLM in a single round-trip and returns a
// fully-formed artifact proposal without writing anything to disk.
func Generate(ctx context.Context, opts GenerateOptions) (*GenerateResult, error) {
	// 1. Validate input length.
	if countWords(opts.Input) < 5 {
		return nil, ErrInputTooShort
	}

	// 2. Validate artifact type.
	artifactType := opts.ArtifactType
	if artifactType == "" {
		artifactType = "idea"
	}
	switch artifactType {
	case "idea", "defect", "doc":
		// valid
	default:
		return nil, fmt.Errorf("unknown artifact type %q: must be \"idea\", \"defect\", or \"doc\"", artifactType)
	}

	// 3. Build user message with label vocabulary hint and optional source context.
	userContent := opts.Input
	if len(opts.ExistingLabels) > 0 {
		userContent += "\n\nAvailable label vocabulary: " + strings.Join(opts.ExistingLabels, ", ")
	}
	if opts.SourceContext != "" {
		userContent += "\n\n" + opts.SourceContext
	}
	llmMsgs := []LLMMessage{
		{Role: "user", Content: userContent},
	}

	// 4. Call LLM (single round-trip — no multi-turn conversation).
	raw, err := CallLLM(ctx, opts.ModelCfg, llmMsgs)
	if err != nil {
		return nil, fmt.Errorf("Generate: LLM call failed: %w", err)
	}

	// 5. Parse structured JSON response.
	action, err := parseAction(raw)
	if err != nil {
		return nil, fmt.Errorf("Generate: parsing LLM response: %w", err)
	}
	if action.Action != "propose" {
		return nil, fmt.Errorf("Generate: expected action \"propose\", got %q", action.Action)
	}

	// 6. Sanitise then resolve slug against existing slugs.
	cleanSlug := sanitiseSlug(action.Slug)
	slug, err := resolveSlug(ctx, cleanSlug, opts.ExistingSlugs, opts.ModelCfg, nil)
	if err != nil {
		return nil, fmt.Errorf("Generate: resolving slug: %w", err)
	}

	// 7. Filter labels to existing vocabulary.
	labels := filterLabels(action.Labels, opts.ExistingLabels)
	if labels == nil {
		labels = []string{}
	}
	// For defects, ensure "defect" label is always present.
	if artifactType == "defect" {
		hasDefect := false
		for _, l := range labels {
			if l == "defect" {
				hasDefect = true
				break
			}
		}
		if !hasDefect {
			labels = append(labels, "defect")
		}
	}

	// 8. Construct frontmatter map.
	// For doc artifacts with a source lineage, inherit the lineage from the source.
	lineage := slug
	if artifactType == "doc" && opts.SourceLineage != "" {
		lineage = opts.SourceLineage
	}
	priority := opts.SourcePriority
	if priority == "" {
		priority = "normal"
	}
	fm := map[string]any{
		"title":    action.Title,
		"type":     artifactType,
		"status":   "draft",
		"lineage":  lineage,
		"labels":   labels,
		"priority": priority,
	}
	if opts.SourceRelease != "" {
		fm["release"] = opts.SourceRelease
	}
	if artifactType == "doc" && opts.SourcePath != "" {
		fm["parent"] = opts.SourcePath
	}

	// 9. Set target directory.
	targetDir := "lifecycle/ideas"
	switch artifactType {
	case "defect":
		targetDir = "lifecycle/defects"
	case "doc":
		targetDir = "lifecycle/docs"
	}

	return &GenerateResult{
		Slug:        slug,
		Title:       action.Title,
		Labels:      labels,
		Body:        action.Body,
		Frontmatter: fm,
		TargetDir:   targetDir,
	}, nil
}

// CollectDiskSlugs globs all .md files in projectPath/targetDir and returns
// the deduplicated set of base slugs, stripping lineage index and stage suffixes.
func CollectDiskSlugs(projectPath, targetDir string) ([]string, error) {
	pattern := filepath.Join(projectPath, targetDir, "*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("CollectDiskSlugs: glob %q: %w", pattern, err)
	}
	seen := make(map[string]bool, len(matches))
	for _, m := range matches {
		base := filepath.Base(m)
		name := strings.TrimSuffix(base, ".md")
		// Strip known stage suffixes before index stripping.
		for _, sfx := range []string{"-be", "-fe", "-test"} {
			if strings.HasSuffix(name, sfx) {
				name = strings.TrimSuffix(name, sfx)
				break
			}
		}
		// Strip trailing "-<digits>" lineage index if present.
		slug := stripLineageIndex(name)
		if slug != "" {
			seen[slug] = true
		}
	}
	slugs := make([]string, 0, len(seen))
	for s := range seen {
		slugs = append(slugs, s)
	}
	return slugs, nil
}

// stripLineageIndex removes a trailing "-<N>" numeric suffix from name.
// For example "my-idea-2" → "my-idea", "my-idea" → "my-idea".
func stripLineageIndex(name string) string {
	idx := strings.LastIndex(name, "-")
	if idx < 0 {
		return name
	}
	suffix := name[idx+1:]
	if len(suffix) == 0 {
		return name
	}
	for _, c := range suffix {
		if c < '0' || c > '9' {
			return name
		}
	}
	// All characters after last hyphen are digits — strip the suffix.
	base := name[:idx]
	if base == "" {
		return name // don't produce empty string
	}
	return base
}

// countWords returns the number of whitespace-separated tokens in s.
func countWords(s string) int {
	return len(strings.Fields(s))
}
