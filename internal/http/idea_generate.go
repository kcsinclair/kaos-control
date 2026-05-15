// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/ideachat"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/project"
)

// handleIdeaGenerate handles POST /api/p/:project/ideas/generate.
//
// Request:  { "input": string, "type"?: "idea" | "defect" }
// Response: { "slug": string, "title": string, "labels": [...], "body": string,
//
//	"frontmatter": {...}, "target_dir": string }
//
// No artifact is written to disk; the response is preview-only.
func (s *Server) handleIdeaGenerate(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	user := userFromCtx(r.Context())
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	var req struct {
		Input          string `json:"input"`
		ArtifactType   string `json:"type"`
		SourceLineage  string `json:"source_lineage"`
		SourcePath     string `json:"source_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.Input == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "input is required"))
		return
	}
	if len(strings.Fields(req.Input)) < 5 {
		writeJSON(w, http.StatusBadRequest, apiError("input_too_short", "Please provide at least 5 words describing your idea."))
		return
	}

	artifactType := req.ArtifactType
	if artifactType == "" {
		artifactType = "idea"
	}

	var templateKey string
	switch artifactType {
	case "defect":
		templateKey = "defect-generate"
	case "doc":
		templateKey = "doc-generate"
	default:
		templateKey = "idea-generate"
	}

	modelCfg, err := resolveIdeaCaptureConfig(p, templateKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("config_error", err.Error()))
		return
	}

	// Gather label vocabulary from the index.
	existingLabels, _ := p.Idx.Labels()

	// Gather slugs: merge index slugs with disk slugs for thorough collision detection.
	var targetDir string
	switch artifactType {
	case "defect":
		targetDir = "lifecycle/defects"
	case "doc":
		targetDir = "lifecycle/docs"
	default:
		targetDir = "lifecycle/ideas"
	}
	diskSlugs, _ := ideachat.CollectDiskSlugs(p.Entry.Path, targetDir)
	indexSlugs, _ := collectSlugs(p)
	allSlugs := mergeSlugs(diskSlugs, indexSlugs)

	// For doc generation with a source lineage, assemble context from the source
	// artifact and its originating idea. Context is capped to those two artifacts.
	var sourceContext string
	if artifactType == "doc" && req.SourceLineage != "" {
		sourceContext = buildDocSourceContext(p, req.SourceLineage, req.SourcePath)
	}

	result, err := ideachat.Generate(r.Context(), ideachat.GenerateOptions{
		Input:          req.Input,
		ArtifactType:   artifactType,
		SourceLineage:  req.SourceLineage,
		SourcePath:     req.SourcePath,
		SourceContext:  sourceContext,
		ExistingLabels: existingLabels,
		ExistingSlugs:  allSlugs,
		ModelCfg:       modelCfg,
	})
	if err != nil {
		if errors.Is(err, ideachat.ErrInputTooShort) {
			writeJSON(w, http.StatusBadRequest, apiError("input_too_short", "Please provide at least 5 words describing your idea."))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("generate_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"slug":        result.Slug,
		"title":       result.Title,
		"labels":      result.Labels,
		"body":        result.Body,
		"frontmatter": result.Frontmatter,
		"target_dir":  result.TargetDir,
	})
}

// buildDocSourceContext reads the source artifact and its lineage's originating
// idea from disk and assembles them as context sections for the doc-generate
// prompt. Context is capped to the originating idea + the source artifact (per
// spec FR1 / milestone 5 — not all plans/tests/defects).
func buildDocSourceContext(p *project.Project, sourceLineage, sourcePath string) string {
	var sb strings.Builder

	// Helper to read and parse an artifact from disk, returning its body.
	readBody := func(relPath string) string {
		absPath := filepath.Join(p.Entry.Path, relPath)
		raw, err := os.ReadFile(absPath)
		if err != nil {
			return ""
		}
		info, err := os.Stat(absPath)
		if err != nil {
			return ""
		}
		a := artifact.Parse(raw, relPath, info.ModTime())
		return a.Body
	}

	// 1. Originating idea: find the idea-type artifact in this lineage.
	ideaRows, _, err := p.Idx.List(index.Filter{
		Lineage:   sourceLineage,
		Type:      "idea",
		Unlimited: true,
	})
	if err == nil && len(ideaRows) > 0 {
		// Use the first (usually only) idea artifact.
		body := readBody(ideaRows[0].Path)
		if body != "" {
			sb.WriteString(fmt.Sprintf("## Source: Originating Idea\n\nFile: %s\n\n%s", ideaRows[0].Path, body))
		}
	}

	// 2. The source artifact itself (e.g. a requirement).
	if sourcePath != "" {
		body := readBody(sourcePath)
		if body != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(fmt.Sprintf("## Source: Referenced Artifact\n\nFile: %s\n\n%s", sourcePath, body))
		}
	}

	return sb.String()
}

// mergeSlugs returns a deduplicated union of two slug slices.
func mergeSlugs(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	for _, s := range a {
		seen[s] = true
	}
	for _, s := range b {
		seen[s] = true
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	return out
}
