// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/project"
)

// projectRuntimeDir returns the per-project runtime state directory (the
// same base directory the SQLite index and scheduler runs live under),
// which the wizard uses for scratch, resumable state — always outside
// lifecycle/, so it never counts as a write under lifecycle/architecture/
// (NFR-1).
func (s *Server) projectRuntimeDir(p *project.Project) string {
	return filepath.Join(s.dataDir, p.Entry.Name)
}

// priorRunInfo reports whether the Architecture Wizard has already run in
// this project (FR-2), by scanning lifecycle/architecture/ for a promoted
// architecture/tech-stack, architecture-summary.md, and adr-0001-*.md.
type priorRunInfo struct {
	Detected     bool   `json:"detected"`
	Architecture string `json:"architecture,omitempty"`
	TechStack    string `json:"tech_stack,omitempty"`
	ADRPath      string `json:"adr_path,omitempty"`
	SummaryPath  string `json:"summary_path,omitempty"`
}

func detectPriorRun(projectRoot string) (priorRunInfo, error) {
	var pr priorRunInfo

	archDirAbs := filepath.Join(projectRoot, "lifecycle", "architecture")
	entries, err := os.ReadDir(archDirAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return pr, nil
		}
		return pr, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if e.Name() == "architecture-summary.md" {
			pr.SummaryPath = path.Join("lifecycle/architecture", e.Name())
			continue
		}
		raw, rerr := os.ReadFile(filepath.Join(archDirAbs, e.Name()))
		if rerr != nil {
			continue
		}
		relPath := path.Join("lifecycle/architecture", e.Name())
		a := artifact.Parse(raw, relPath, time.Time{})
		switch a.FM.Type {
		case "architecture":
			pr.Architecture = relPath
		case "tech-stack":
			pr.TechStack = relPath
		}
	}

	if decisions, derr := os.ReadDir(filepath.Join(archDirAbs, "decisions")); derr == nil {
		for _, e := range decisions {
			if !e.IsDir() && strings.HasPrefix(e.Name(), "adr-0001-") && strings.HasSuffix(e.Name(), ".md") {
				pr.ADRPath = path.Join("lifecycle/architecture/decisions", e.Name())
				break
			}
		}
	}

	pr.Detected = pr.Architecture != "" || pr.TechStack != "" || pr.SummaryPath != "" || pr.ADRPath != ""
	return pr, nil
}

// handleGetArchitectureWizard handles GET /api/p/{project}/architecture/wizard
func (s *Server) handleGetArchitectureWizard(w http.ResponseWriter, r *http.Request) {
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

	prior, err := detectPriorRun(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	var resumable *architecture.WizardState
	if st, found, serr := architecture.LoadWizardState(s.projectRuntimeDir(p), user.Email); serr != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", serr.Error()))
		return
	} else if found {
		resumable = &st
	}

	cfg := p.Config().ArchitectureWizard
	writeJSON(w, http.StatusOK, map[string]any{
		"questions":            cfg.Questions,
		"default_architecture": cfg.DefaultArchitecture,
		"prior_run":            prior,
		"resumable_state":      resumable,
	})
}

// handleRecommendArchitecture handles POST /api/p/{project}/architecture/wizard/recommend
func (s *Server) handleRecommendArchitecture(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	if userFromCtx(r.Context()) == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	var req struct {
		Answers []architecture.Answer `json:"answers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	arches, _, err := architecture.LoadCatalog(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	recs, dropped, err := architecture.Recommend(arches, p.Config().ArchitectureWizard, req.Answers)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("recommend_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"recommendations":     recs,
		"dropped_constraints": dropped,
	})
}

// handleListWizardStacks handles GET /api/p/{project}/architecture/wizard/stacks
func (s *Server) handleListWizardStacks(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	if userFromCtx(r.Context()) == nil {
		writeJSON(w, http.StatusUnauthorized, apiError("unauthorized", "authentication required"))
		return
	}

	archSlug := r.URL.Query().Get("architecture")
	if archSlug == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "architecture query parameter is required"))
		return
	}
	language := r.URL.Query().Get("language")

	arches, stacks, err := architecture.LoadCatalog(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	var chosen *architecture.CatalogItem
	for i := range arches {
		if arches[i].Slug == archSlug {
			chosen = &arches[i]
			break
		}
	}
	if chosen == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "architecture not found in catalog: "+archSlug))
		return
	}

	ranked := architecture.RankStacks(*chosen, stacks, language)
	writeJSON(w, http.StatusOK, map[string]any{"stacks": ranked})
}

// handlePutWizardState handles PUT /api/p/{project}/architecture/wizard/state
func (s *Server) handlePutWizardState(w http.ResponseWriter, r *http.Request) {
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

	var st architecture.WizardState
	if err := json.NewDecoder(r.Body).Decode(&st); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	if err := architecture.SaveWizardState(s.projectRuntimeDir(p), user.Email, st); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"saved": true})
}

// handleDeleteWizardState handles DELETE /api/p/{project}/architecture/wizard/state
func (s *Server) handleDeleteWizardState(w http.ResponseWriter, r *http.Request) {
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

	if err := architecture.ClearWizardState(s.projectRuntimeDir(p), user.Email); err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cleared": true})
}
