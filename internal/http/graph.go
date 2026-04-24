package http

import (
	"net/http"

	"github.com/kaos-control/kaos-control/internal/index"
)

// handleGraph handles GET /api/p/:project/graph
func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	f := index.Filter{
		Stage:   r.URL.Query().Get("stage"),
		Status:  r.URL.Query().Get("status"),
		Label:   r.URL.Query().Get("label"),
		Lineage: r.URL.Query().Get("lineage"),
		Type:    r.URL.Query().Get("type"),
	}

	data, err := p.Idx.Graph(f)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// handleLabels handles GET /api/p/:project/labels
func (s *Server) handleLabels(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	labels, err := p.Idx.Labels()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"labels": labels})
}

// handleLineages handles GET /api/p/:project/lineages
func (s *Server) handleLineages(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	lineages, err := p.Idx.Lineages()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lineages": lineages})
}

// handleParseErrors handles GET /api/p/:project/parse-errors
func (s *Server) handleParseErrors(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	errs, err := p.Idx.ParseErrors()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"errors": errs})
}
