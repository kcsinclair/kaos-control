package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kaos-control/kaos-control/internal/ideachat"
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

	modelCfg, err := resolveIdeaCaptureConfig(p, templateKey)
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
