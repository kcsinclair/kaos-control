package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kaos-control/kaos-control/internal/artifact"
	"github.com/kaos-control/kaos-control/internal/hub"
	"github.com/kaos-control/kaos-control/internal/index"
	"github.com/kaos-control/kaos-control/internal/release"
)

// handleListReleases handles GET /api/p/:project/releases
func (s *Server) handleListReleases(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	store := release.NewStore(p.Idx.DB())
	releases, err := store.List(p.Entry.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if releases == nil {
		releases = []*release.Release{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"releases": releases})
}

// createReleaseRequest is the JSON body for POST /releases.
type createReleaseRequest struct {
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
	Duration  *string `json:"duration"` // e.g. "14d" or "2w"
}

// handleCreateRelease handles POST /api/p/:project/releases
func (s *Server) handleCreateRelease(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	var req createReleaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	if req.Status == "" {
		req.Status = "planned"
	}

	rel := &release.Release{
		ProjectID: p.Entry.Name,
		Name:      req.Name,
		Status:    req.Status,
	}

	if req.StartDate != nil {
		t, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid start_date: "+err.Error()))
			return
		}
		rel.StartDate = &t
	}

	if req.EndDate != nil {
		t, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid end_date: "+err.Error()))
			return
		}
		rel.EndDate = &t
	} else if req.Duration != nil && rel.StartDate != nil {
		dur, err := parseDuration(*req.Duration)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid duration: "+err.Error()))
			return
		}
		end := rel.StartDate.Add(dur)
		rel.EndDate = &end
	}

	if err := rel.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("validation_error", err.Error()))
		return
	}

	store := release.NewStore(p.Idx.DB())
	if err := store.Create(rel); err != nil {
		if isDuplicateError(err) {
			writeJSON(w, http.StatusConflict, apiError("conflict", fmt.Sprintf("release %q already exists in this project", rel.Name)))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	p.Hub.Broadcast(hub.Event{Type: "release.created", Payload: map[string]any{"release": rel}})
	writeJSON(w, http.StatusCreated, map[string]any{"release": rel})
}

// handleGetRelease handles GET /api/p/:project/releases/{releaseID}
func (s *Server) handleGetRelease(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "releaseID"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid release ID"))
		return
	}

	store := release.NewStore(p.Idx.DB())
	rel, err := store.Get(p.Entry.Name, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if rel == nil {
		writeJSON(w, http.StatusNotFound, apiError("not_found", "release not found"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"release": rel})
}

// updateReleaseRequest is the JSON body for PUT /releases/{releaseID}.
type updateReleaseRequest struct {
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	StartDate *string `json:"start_date"`
	EndDate   *string `json:"end_date"`
	Duration  *string `json:"duration"`
}

// handleUpdateRelease handles PUT /api/p/:project/releases/{releaseID}
func (s *Server) handleUpdateRelease(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "releaseID"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid release ID"))
		return
	}

	var req updateReleaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid JSON: "+err.Error()))
		return
	}

	rel := &release.Release{
		ID:        id,
		ProjectID: p.Entry.Name,
		Name:      req.Name,
		Status:    req.Status,
	}

	if req.StartDate != nil {
		t, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid start_date: "+err.Error()))
			return
		}
		rel.StartDate = &t
	}

	if req.EndDate != nil {
		t, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid end_date: "+err.Error()))
			return
		}
		rel.EndDate = &t
	} else if req.Duration != nil && rel.StartDate != nil {
		dur, err := parseDuration(*req.Duration)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid duration: "+err.Error()))
			return
		}
		end := rel.StartDate.Add(dur)
		rel.EndDate = &end
	}

	if err := rel.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("validation_error", err.Error()))
		return
	}

	store := release.NewStore(p.Idx.DB())
	oldName, err := store.Update(rel)
	if err != nil {
		if isDuplicateError(err) {
			writeJSON(w, http.StatusConflict, apiError("conflict", fmt.Sprintf("release %q already exists in this project", rel.Name)))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	// If the name changed, propagate the rename to all assigned artifacts.
	renamed := 0
	if oldName != rel.Name && p.Git != nil {
		n, propErr := release.PropagateRename(p.Entry.Path, oldName, rel.Name, p.Idx, p.Git)
		if propErr != nil {
			// Log but do not fail the request; the DB update succeeded.
			_ = propErr
		}
		renamed = n
	}

	p.Hub.Broadcast(hub.Event{Type: "release.updated", Payload: map[string]any{
		"release":         rel,
		"old_name":        oldName,
		"artifacts_renamed": renamed,
	}})
	writeJSON(w, http.StatusOK, map[string]any{"release": rel, "artifacts_renamed": renamed})
}

// handleDeleteRelease handles DELETE /api/p/:project/releases/{releaseID}
func (s *Server) handleDeleteRelease(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "releaseID"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid release ID"))
		return
	}

	store := release.NewStore(p.Idx.DB())

	// Optional: reassign artifacts to another release before deleting.
	reassignTo := r.URL.Query().Get("reassign_to")
	if reassignTo != "" {
		reassignID, err := strconv.ParseInt(reassignTo, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid reassign_to value"))
			return
		}
		target, err := store.Get(p.Entry.Name, reassignID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
			return
		}
		if target == nil {
			writeJSON(w, http.StatusNotFound, apiError("not_found", "reassign_to release not found"))
			return
		}

		// Get current release name so we can find its artifacts.
		current, err := store.Get(p.Entry.Name, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
			return
		}
		if current == nil {
			writeJSON(w, http.StatusNotFound, apiError("not_found", "release not found"))
			return
		}

		// Propagate the rename from current.Name → target.Name on disk.
		if p.Git != nil {
			_, _ = release.PropagateRename(p.Entry.Path, current.Name, target.Name, p.Idx, p.Git)
		} else {
			// No git: update files in place without a commit.
			_ = rewriteReleaseField(p.Entry.Path, current.Name, target.Name, p.Idx)
		}
	}

	deletedName, orphaned, err := store.Delete(p.Entry.Name, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	p.Hub.Broadcast(hub.Event{Type: "release.deleted", Payload: map[string]any{
		"id":   id,
		"name": deletedName,
	}})
	writeJSON(w, http.StatusOK, map[string]any{"orphaned_artifact_count": orphaned})
}

// handleListReleaseArtifacts handles GET /api/p/:project/releases/{releaseID}/artifacts
func (s *Server) handleListReleaseArtifacts(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	id, err := strconv.ParseInt(chi.URLParam(r, "releaseID"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid release ID"))
		return
	}

	store := release.NewStore(p.Idx.DB())
	items, err := store.ListArtifacts(p.Entry.Name, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	if items == nil {
		items = []*index.ArtifactRow{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

// handleRoadmapGraph handles GET /api/p/:project/releases/graph
func (s *Server) handleRoadmapGraph(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	store := release.NewStore(p.Idx.DB())
	releases, err := store.List(p.Entry.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	// Build node set and edge list.
	var nodes []map[string]any
	var edges []map[string]any

	// Track paths of artifact nodes for edge filtering.
	artifactNodeSet := map[string]bool{}

	for i, rel := range releases {
		// Add release node.
		nodes = append(nodes, map[string]any{
			"id":     fmt.Sprintf("release:%d", rel.ID),
			"title":  rel.Name,
			"type":   "release",
			"status": rel.Status,
			"stage":  "",
			"lineage": "",
			"slug":   "",
			"index":  0,
			"labels": []string{},
			"start_date": rel.StartDate,
			"end_date":   rel.EndDate,
		})

		// Timeline edge to next release.
		if i+1 < len(releases) {
			edges = append(edges, map[string]any{
				"source": fmt.Sprintf("release:%d", rel.ID),
				"target": fmt.Sprintf("release:%d", releases[i+1].ID),
				"kind":   "timeline",
			})
		}

		// Fetch assigned ideas and defects.
		artifacts, _, err := p.Idx.List(index.Filter{
			Release:   rel.Name,
			Unlimited: true,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
			return
		}

		for _, a := range artifacts {
			if a.Type != "idea" && a.Type != "defect" {
				continue
			}
			artifactNodeSet[a.Path] = true
			nodes = append(nodes, map[string]any{
				"id":      a.Path,
				"title":   a.Title,
				"type":    a.Type,
				"status":  a.Status,
				"stage":   a.Stage,
				"lineage": a.Lineage,
				"slug":    a.Slug,
				"index":   a.Index,
				"labels":  a.FM.Labels,
			})
			edges = append(edges, map[string]any{
				"source": fmt.Sprintf("release:%d", rel.ID),
				"target": a.Path,
				"kind":   "assigned",
			})
		}
	}

	// Add existing depends_on / blocks edges between included artifact nodes.
	graphData, err := p.Idx.Graph(index.Filter{Unlimited: true})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	for _, e := range graphData.Edges {
		if artifactNodeSet[e.Source] && artifactNodeSet[e.Target] {
			edges = append(edges, map[string]any{
				"source": e.Source,
				"target": e.Target,
				"kind":   e.Kind,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"nodes": nodes,
		"edges": edges,
	})
}

// ----- helpers -----

// parseDuration parses a simple duration string like "14d" or "2w" into a time.Duration.
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return 0, fmt.Errorf("duration %q too short", s)
	}
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid duration number in %q", s)
	}
	switch unit {
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit %q; use 'd' (days) or 'w' (weeks)", string(unit))
	}
}

// isDuplicateError returns true when err is a SQLite unique constraint violation.
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// rewriteReleaseField updates the release frontmatter field on all artifact
// files assigned to oldName, writing them to disk and re-indexing.
// Used when git is unavailable (no commit is created).
func rewriteReleaseField(projectRoot, oldName, newName string, idx *index.Index) error {
	rows, _, err := idx.List(index.Filter{Release: oldName, Unlimited: true})
	if err != nil {
		return err
	}
	for _, row := range rows {
		absPath := filepath.Join(projectRoot, row.Path)
		raw, err := os.ReadFile(absPath)
		if err != nil {
			continue
		}
		patched, ok := artifact.PatchFrontmatterField(raw, "release", newName)
		if !ok {
			continue
		}
		_ = os.WriteFile(absPath, patched, 0o644)
		_ = idx.IndexFile(absPath)
	}
	return nil
}
