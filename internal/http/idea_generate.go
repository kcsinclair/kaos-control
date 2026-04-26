package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/kaos-control/kaos-control/internal/ideachat"
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
		Input        string `json:"input"`
		ArtifactType string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.Input == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "input is required"))
		return
	}

	artifactType := req.ArtifactType
	if artifactType == "" {
		artifactType = "idea"
	}

	templateKey := "idea-generate"
	if artifactType == "defect" {
		templateKey = "defect-generate"
	}

	modelCfg, err := resolveGenerateConfig(p, templateKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("config_error", err.Error()))
		return
	}

	// Gather label vocabulary from the index.
	existingLabels, _ := p.Idx.Labels()

	// Gather slugs: merge index slugs with disk slugs for thorough collision detection.
	targetDir := "lifecycle/ideas"
	if artifactType == "defect" {
		targetDir = "lifecycle/defects"
	}
	diskSlugs, _ := ideachat.CollectDiskSlugs(p.Entry.Path, targetDir)
	indexSlugs, _ := collectSlugs(p)
	allSlugs := mergeSlugs(diskSlugs, indexSlugs)

	result, err := ideachat.Generate(r.Context(), ideachat.GenerateOptions{
		Input:          req.Input,
		ArtifactType:   artifactType,
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

// resolveGenerateConfig looks up the idea-capture agent and returns a ModelConfig
// using the specified template key. This is refactored in Milestone 4 to share
// logic with resolveIdeaCaptureConfig.
func resolveGenerateConfig(p *project.Project, templateKey string) (ideachat.ModelConfig, error) {
	for _, a := range p.Cfg.Agents {
		if a.Name == "idea-capture" {
			prompt, ok := a.PromptTemplates[templateKey]
			if !ok {
				return ideachat.ModelConfig{}, fmt.Errorf("idea-capture agent has no template %q", templateKey)
			}
			model := a.Model
			if model == "" {
				model = "claude-sonnet-4-6"
			}
			return ideachat.ModelConfig{
				Model:        model,
				SystemPrompt: prompt,
			}, nil
		}
	}
	return ideachat.ModelConfig{}, fmt.Errorf("idea-capture agent not configured")
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
