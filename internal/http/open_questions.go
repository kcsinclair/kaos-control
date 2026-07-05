// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/config"
)

// handleGetOpenQuestions handles GET /api/p/:project/artifacts/*path/open-questions.
func (s *Server) handleGetOpenQuestions(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	param := chi.URLParam(r, "*")
	relPath := filepath.ToSlash(strings.TrimSuffix(param, "/open-questions"))

	row, err := p.Idx.Get(relPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if row == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "artifact not found"))
		return
	}

	absPath := filepath.Join(p.Entry.Path, relPath)
	raw, err := os.ReadFile(absPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("read_error", err.Error()))
		return
	}
	body := artifact.Parse(raw, relPath, row.Mtime).Body

	cfg, err := config.LoadProject(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("config_error", err.Error()))
		return
	}

	questions, _ := artifact.ParseOpenQuestions(body)
	if questions == nil {
		questions = []artifact.Question{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"heading":   "## Open Questions",
		"format":    cfg.OpenQuestions.EffectiveFormat(),
		"questions": questions,
	})
}

// handlePreviewOpenQuestions handles
// POST /api/p/:project/artifacts/*path/open-questions/preview. It is a
// compute-only endpoint: it builds and returns the new body that would
// result from writing the given answers, without writing to disk. The
// actual persistence happens via the existing PUT /artifacts/*path.
func (s *Server) handlePreviewOpenQuestions(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	param := chi.URLParam(r, "*")
	relPath := filepath.ToSlash(strings.TrimSuffix(param, "/open-questions/preview"))

	row, err := p.Idx.Get(relPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if row == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "artifact not found"))
		return
	}

	var req struct {
		Answers  map[string]string `json:"answers"`
		Complete bool              `json:"complete"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	answers := make(map[int]string, len(req.Answers))
	for k, v := range req.Answers {
		idx, err := strconv.Atoi(k)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", "answers keys must be question indices: "+k))
			return
		}
		answers[idx] = v
	}

	absPath := filepath.Join(p.Entry.Path, relPath)
	raw, err := os.ReadFile(absPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("read_error", err.Error()))
		return
	}
	body := artifact.Parse(raw, relPath, row.Mtime).Body

	cfg, err := config.LoadProject(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("config_error", err.Error()))
		return
	}

	newBody, err := artifact.ApplyAnswers(body, answers, cfg.OpenQuestions.EffectiveFormat(), req.Complete)
	if err != nil {
		if errors.Is(err, artifact.ErrIncompleteAnswers) {
			writeJSON(w, http.StatusUnprocessableEntity, apiError("incomplete_answers", err.Error()))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("apply_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"body": newBody})
}
