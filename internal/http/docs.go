// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/docs"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// handleListDocs handles GET /api/p/{project}/docs
func (s *Server) handleListDocs(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	entries, err := docs.List(p.Entry.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	if entries == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"docs":             []any{},
			"docs_dir_present": false,
		})
		return
	}

	// Sort server-side: title ascending (case-insensitive), ties broken by path.
	sort.Slice(entries, func(i, j int) bool {
		ti := strings.ToLower(entries[i].Title)
		tj := strings.ToLower(entries[j].Title)
		if ti != tj {
			return ti < tj
		}
		return entries[i].Path < entries[j].Path
	})

	type docItem struct {
		Path       string `json:"path"`
		Title      string `json:"title"`
		Summary    string `json:"summary"`
		IsMarkdown bool   `json:"is_markdown"`
		SubDir     string `json:"sub_dir"`
	}
	items := make([]docItem, len(entries))
	for i, e := range entries {
		items[i] = docItem{
			Path:       e.Path,
			Title:      e.Title,
			Summary:    e.Summary,
			IsMarkdown: e.IsMarkdown,
			SubDir:     e.SubDir,
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"docs":             items,
		"docs_dir_present": true,
	})
}

// handleGetDoc handles GET /api/p/{project}/docs/*path
func (s *Server) handleGetDoc(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	relPath := chi.URLParam(r, "*")

	raw, err := docs.Read(p.Entry.Path, relPath)
	if err != nil {
		if errors.Is(err, sandbox.ErrPathTraversal) {
			writeJSON(w, http.StatusBadRequest, apiError("path_traversal", err.Error()))
			return
		}
		if errors.Is(err, docs.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, apiError("not_found", "doc not found"))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	sum := sha256.Sum256(raw)
	fileSHA := hex.EncodeToString(sum[:])

	ext := strings.ToLower(filepath.Ext(relPath))
	isMarkdown := ext == ".md" || ext == ".markdown"

	if isMarkdown {
		writeJSON(w, http.StatusOK, map[string]any{
			"path":        relPath,
			"body":        string(raw),
			"file_sha":    fileSHA,
			"is_markdown": true,
		})
		return
	}

	// Non-markdown: base64-encode the body and detect MIME type.
	detectionBuf := raw
	if len(detectionBuf) > 512 {
		detectionBuf = raw[:512]
	}
	mime := http.DetectContentType(detectionBuf)

	writeJSON(w, http.StatusOK, map[string]any{
		"path":        relPath,
		"body_base64": base64.StdEncoding.EncodeToString(raw),
		"mime":        mime,
		"file_sha":    fileSHA,
		"is_markdown": false,
	})
}

// handlePutDoc handles PUT /api/p/{project}/docs/*path
func (s *Server) handlePutDoc(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	if !requireRole(w, r, p, RolesArtifactEditors...) {
		return
	}

	relPath := chi.URLParam(r, "*")

	// Only markdown files are writable.
	ext := strings.ToLower(filepath.Ext(relPath))
	if ext != ".md" && ext != ".markdown" {
		writeJSON(w, http.StatusUnsupportedMediaType, apiError("not_markdown", "only markdown files can be edited"))
		return
	}

	var req struct {
		Body        string `json:"body"`
		ExpectedSHA string `json:"expected_sha"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	// Optimistic concurrency: if expected_sha is provided, validate against current file.
	if req.ExpectedSHA != "" {
		current, err := docs.Read(p.Entry.Path, relPath)
		if err != nil {
			if errors.Is(err, sandbox.ErrPathTraversal) {
				writeJSON(w, http.StatusBadRequest, apiError("path_traversal", err.Error()))
				return
			}
			if errors.Is(err, docs.ErrNotFound) {
				writeJSON(w, http.StatusNotFound, apiError("not_found", "doc not found"))
				return
			}
			writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
			return
		}
		currentSum := sha256.Sum256(current)
		if hex.EncodeToString(currentSum[:]) != req.ExpectedSHA {
			writeJSON(w, http.StatusConflict, apiError("sha_mismatch", "doc has been modified since last read"))
			return
		}
	}

	if err := docs.Write(p.Entry.Path, relPath, []byte(req.Body)); err != nil {
		if errors.Is(err, sandbox.ErrPathTraversal) {
			writeJSON(w, http.StatusBadRequest, apiError("path_traversal", err.Error()))
			return
		}
		if errors.Is(err, docs.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, apiError("not_found", "doc not found"))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("fs_error", err.Error()))
		return
	}

	// Synchronous broadcast removes the 150 ms watcher debounce perceived delay.
	p.Hub.Broadcast(hub.Event{
		Type:    "doc.changed",
		Payload: map[string]string{"path": relPath},
	})

	newSum := sha256.Sum256([]byte(req.Body))
	writeJSON(w, http.StatusOK, map[string]any{
		"file_sha": hex.EncodeToString(newSum[:]),
	})
}
