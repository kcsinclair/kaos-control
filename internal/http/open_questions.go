// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/config"
)

// openQuestionsHeading is the exact heading text matched by HasOpenQuestions
// and ParseOpenQuestions.
const openQuestionsHeading = "## Open Questions"

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
		"heading":   openQuestionsHeading,
		"format":    cfg.OpenQuestions.EffectiveFormat(),
		"questions": questions,
	})
}
