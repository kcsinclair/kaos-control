package http

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/artifact"
	git "github.com/kaos-control/kaos-control/internal/git"
	"github.com/kaos-control/kaos-control/internal/index"
)

// handleListArtifacts handles GET /api/p/:project/artifacts
func (s *Server) handleListArtifacts(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	f := index.Filter{
		Stage:    r.URL.Query().Get("stage"),
		Status:   r.URL.Query().Get("status"),
		Label:    r.URL.Query().Get("label"),
		Lineage:  r.URL.Query().Get("lineage"),
		Type:     r.URL.Query().Get("type"),
		Priority: r.URL.Query().Get("priority"),
		Q:        r.URL.Query().Get("q"),
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		n, _ := strconv.Atoi(v)
		if n <= 0 {
			f.Unlimited = true
		} else {
			f.Limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		f.Offset, _ = strconv.Atoi(v)
	}

	items, total, err := p.Idx.List(f)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": total,
	})
}

// handleGetArtifact handles GET /api/p/:project/artifacts/*path
func (s *Server) handleGetArtifact(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	relPath := chi.URLParam(r, "*")
	row, err := p.Idx.Get(relPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if row == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "artifact not found"))
		return
	}

	// Read the raw file for body + HTML rendering.
	absPath := filepath.Join(p.Entry.Path, relPath)
	raw, err := os.ReadFile(absPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("read_error", err.Error()))
		return
	}

	body := artifact.Parse(raw, relPath, row.Mtime).Body
	bodyHTML := artifact.RenderHTML(body)
	sum := sha256.Sum256(raw)
	fileSHA := hex.EncodeToString(sum[:])

	writeJSON(w, http.StatusOK, map[string]any{
		"artifact": row,
		"body":     body,
		"body_html": bodyHTML,
		"file_sha": fileSHA,
	})
}

// handleGetArtifactHistory handles GET /api/p/:project/artifacts/*path/history
func (s *Server) handleGetArtifactHistory(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	param := chi.URLParam(r, "*")
	relPath := filepath.ToSlash(strings.TrimSuffix(param, "/history"))

	if p.Git == nil {
		writeJSON(w, http.StatusOK, map[string]any{"commits": []any{}, "note": "project is not a git repository"})
		return
	}

	limit := 20
	commits, err := p.Git.Log(relPath, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("git_error", err.Error()))
		return
	}
	if commits == nil {
		commits = []*git.CommitInfo{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"commits": commits})
}
