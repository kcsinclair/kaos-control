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
		if err == release.ErrNotFound {
			writeJSON(w, http.StatusNotFound, apiError("not_found", "release not found"))
			return
		}
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
		n, propErr := release.PropagateRename(p.Entry.Path, oldName, rel.Name, p.Idx, p.Git, p.Hub)
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
			_, _ = release.PropagateRename(p.Entry.Path, current.Name, target.Name, p.Idx, p.Git, p.Hub)
		} else {
			// No git: update files in place without a commit.
			_ = rewriteReleaseField(p.Entry.Path, current.Name, target.Name, p.Idx, p.Hub)
		}
	}

	deletedName, orphaned, err := store.Delete(p.Entry.Name, id)
	if err != nil {
		if err == release.ErrNotFound {
			writeJSON(w, http.StatusNotFound, apiError("not_found", "release not found"))
			return
		}
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

	// Synthetic Backlog node — always present as the chain root.
	const backlogID = "release:backlog"
	nodes = append(nodes, map[string]any{
		"id":        backlogID,
		"title":     "Backlog",
		"type":      "release",
		"status":    "",
		"stage":     "",
		"lineage":   "",
		"slug":      "",
		"index":     0,
		"labels":    []string{},
		"synthetic": true,
	})

	// Partition releases into scheduled (have a start_date) and unscheduled.
	// store.List already returns them ordered: scheduled by start_date ASC, name ASC;
	// then unscheduled by name ASC.
	var scheduled, unscheduled []*release.Release
	for _, rel := range releases {
		if rel.StartDate != nil {
			scheduled = append(scheduled, rel)
		} else {
			unscheduled = append(unscheduled, rel)
		}
	}

	// Track paths of artifact nodes for edge filtering.
	artifactNodeSet := map[string]bool{}

	// addReleaseNode appends a release node and its assigned artifact nodes/edges.
	addReleaseNode := func(rel *release.Release) error {
		releaseNodeID := fmt.Sprintf("release:%d", rel.ID)
		nodes = append(nodes, map[string]any{
			"id":         releaseNodeID,
			"title":      rel.Name,
			"type":       "release",
			"status":     rel.Status,
			"stage":      "",
			"lineage":    "",
			"slug":       "",
			"index":      0,
			"labels":     []string{},
			"start_date": rel.StartDate,
			"end_date":   rel.EndDate,
		})
		artifacts, _, err := p.Idx.List(index.Filter{Release: rel.Name, Unlimited: true})
		if err != nil {
			return err
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
				"source": releaseNodeID,
				"target": a.Path,
				"kind":   "assigned",
			})
		}
		return nil
	}

	// Build directed chain: Backlog → scheduled[0] → scheduled[1] → …
	prevID := backlogID
	for i, rel := range scheduled {
		nodeID := fmt.Sprintf("release:%d", rel.ID)
		if err := addReleaseNode(rel); err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
			return
		}
		label := ""
		if i > 0 {
			label = humanDuration(*scheduled[i-1].StartDate, *rel.StartDate)
		}
		edges = append(edges, map[string]any{
			"source": prevID,
			"target": nodeID,
			"kind":   "timeline",
			"label":  label,
		})
		prevID = nodeID
	}

	// Unscheduled releases are terminal leaves appended after the last scheduled.
	// prevID is the tail of the scheduled chain (or backlogID if none scheduled).
	for _, rel := range unscheduled {
		nodeID := fmt.Sprintf("release:%d", rel.ID)
		if err := addReleaseNode(rel); err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
			return
		}
		edges = append(edges, map[string]any{
			"source": prevID,
			"target": nodeID,
			"kind":   "timeline",
			"label":  "",
		})
		prevID = nodeID
	}

	// Artifacts with no release assignment attach as "assigned" edges from the Backlog node.
	unassigned, _, err := p.Idx.List(index.Filter{Release: "__unassigned__", Unlimited: true})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	for _, a := range unassigned {
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
			"source": backlogID,
			"target": a.Path,
			"kind":   "assigned",
		})
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

	if nodes == nil {
		nodes = []map[string]any{}
	}
	if edges == nil {
		edges = []map[string]any{}
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

// humanDuration returns a human-readable string for the duration between two dates.
// Uses the largest appropriate unit: days (< 8), weeks (< 5), months (< 13), years.
func humanDuration(from, to time.Time) string {
	days := int(to.Sub(from).Hours() / 24)
	if days < 0 {
		days = -days
	}
	if days < 8 {
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
	weeks := days / 7
	if weeks < 5 {
		if weeks == 1 {
			return "1 week"
		}
		return fmt.Sprintf("%d weeks", weeks)
	}
	months := days / 30
	if months < 13 {
		if months == 1 {
			return "1 month"
		}
		return fmt.Sprintf("%d months", months)
	}
	years := days / 365
	if years == 1 {
		return "1 year"
	}
	return fmt.Sprintf("%d years", years)
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
// h may be nil; when non-nil an "artifact.indexed" hub event is broadcast for
// each successfully re-indexed artifact.
func rewriteReleaseField(projectRoot, oldName, newName string, idx *index.Index, h *hub.Hub) error {
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
		if err := idx.IndexFile(absPath); err != nil {
			continue
		}
		if h != nil {
			h.Broadcast(hub.Event{
				Type:    "artifact.indexed",
				Payload: map[string]string{"path": row.Path, "action": "updated"},
			})
		}
	}
	return nil
}
