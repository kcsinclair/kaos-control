// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/kaos-control/kaos-control/internal/architecture"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/project"
	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// adrNumberRe extracts the zero-padded number from an ADR filename.
var adrNumberRe = regexp.MustCompile(`adr-(\d{4})-`)

// handlePromoteArchitecture handles POST /api/p/{project}/architecture/promote
func (s *Server) handlePromoteArchitecture(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	if !requireRole(w, r, p, RolesArtifactEditors...) {
		return
	}

	var req struct {
		ArchitecturePath string `json:"architecture_path"`
		TechStackPath    string `json:"tech_stack_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	result, err := architecture.Promote(p.Entry.Path, architecture.PromotionRequest{
		ArchitectureCatalogPath: req.ArchitecturePath,
		TechStackCatalogPath:    req.TechStackPath,
	})
	if err != nil {
		if errors.Is(err, sandbox.ErrPathTraversal) || errors.Is(err, sandbox.ErrAbsolutePath) || errors.Is(err, os.ErrNotExist) {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", err.Error()))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	// Synchronous re-index of everything the promotion touched, mirroring the
	// PUT /artifacts/*path re-index-before-responding pattern (NFR-2).
	reindexPath(p, result.PromotedArchitecture)
	reindexPath(p, result.PromotedTechStack)
	for _, archivedPath := range result.Archived {
		reindexPath(p, archivedPath)
	}

	p.Hub.Broadcast(hub.Event{
		Type: "artifact.indexed",
		Payload: map[string]any{
			"action":                "promoted",
			"promoted_architecture": result.PromotedArchitecture,
			"promoted_tech_stack":   result.PromotedTechStack,
			"archived":              result.Archived,
		},
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"promoted_architecture": result.PromotedArchitecture,
		"promoted_tech_stack":   result.PromotedTechStack,
		"archived":              result.Archived,
	})
}

// handleCreateADR handles POST /api/p/{project}/architecture/adrs
func (s *Server) handleCreateADR(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}
	if !requireRole(w, r, p, RolesArtifactEditors...) {
		return
	}

	var req struct {
		Slug   string `json:"slug"`
		Title  string `json:"title"`
		Status string `json:"status"`
		Body   string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}
	if req.Slug == "" || req.Title == "" {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "slug and title are required"))
		return
	}

	relPath, err := architecture.CreateADR(p.Entry.Path, architecture.ADRRequest{
		Slug:   req.Slug,
		Title:  req.Title,
		Status: req.Status,
		Body:   req.Body,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	reindexPath(p, relPath)

	p.Hub.Broadcast(hub.Event{
		Type:    "artifact.indexed",
		Payload: map[string]string{"path": relPath, "action": "created"},
	})

	number := 0
	if m := adrNumberRe.FindStringSubmatch(filepath.Base(relPath)); m != nil {
		number, _ = strconv.Atoi(m[1])
	}
	writeJSON(w, http.StatusCreated, map[string]any{"path": relPath, "number": number})
}

// handleNextADRNumber handles GET /api/p/{project}/architecture/adrs/next
func (s *Server) handleNextADRNumber(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	number, err := architecture.NextADRNumber(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"number": number})
}

// handleArchitectureOverview handles GET /api/p/{project}/architecture/overview.
// Read-only: requires an authenticated session but no editor role, mirroring
// handleArchitectureMap. Missing/absent parts of the architecture zone
// degrade to empty/null fields in the payload rather than 5xx (NFR-5).
func (s *Server) handleArchitectureOverview(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	overview, err := architecture.LoadOverview(p.Entry.Path, p.Idx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, overview)
}

// reindexPath synchronously re-indexes a repo-relative artifact path,
// best-effort — the fsnotify watcher will pick up anything missed here.
func reindexPath(p *project.Project, relPath string) {
	if relPath == "" {
		return
	}
	absPath := filepath.Join(p.Entry.Path, filepath.FromSlash(relPath))
	_ = p.Idx.IndexFile(absPath)
}
