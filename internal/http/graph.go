// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"net/http"

	"github.com/kaos-control/kaos-control/internal/index"
)

// handleGraph handles GET /api/p/:project/graph
//
// Optional query parameters:
//   - stage, status, label, lineage, type, release — filter artifact nodes.
//   - include_releases=true — merge release overlay data (release nodes,
//     timeline edges, assignment edges) into the artifact graph response.
//     Release nodes are always included; artifact nodes from the overlay that
//     are not already in the filtered artifact graph are not added (they must
//     pass the same filters). Duplicate nodes are suppressed by ID.
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
		Release: r.URL.Query().Get("release"),
	}

	data, err := p.Idx.Graph(f)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	if r.URL.Query().Get("include_releases") == "true" {
		roadmap, err := buildRoadmapGraph(p)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
			return
		}

		// Build the set of artifact node IDs already present in the filtered graph.
		artifactIDs := make(map[string]bool, len(data.Nodes))
		for _, n := range data.Nodes {
			artifactIDs[n.ID] = true
		}

		// Merge release overlay nodes:
		//   - Release nodes (type=="release") are always added.
		//   - Artifact nodes from the overlay are only added if they already
		//     exist in the filtered artifact graph (dedup by ID).
		mergedIDs := make(map[string]bool, len(data.Nodes)+len(roadmap.Nodes))
		for id := range artifactIDs {
			mergedIDs[id] = true
		}
		for _, n := range roadmap.Nodes {
			if mergedIDs[n.ID] {
				continue // already present — skip duplicate
			}
			if n.Type == "release" {
				// Release nodes are always included when include_releases=true.
				data.Nodes = append(data.Nodes, n)
				mergedIDs[n.ID] = true
			}
			// Non-release artifact nodes that aren't in the filtered graph are
			// intentionally omitted so that existing filters still apply.
		}

		// Append overlay edges where both endpoints are in the merged node set.
		for _, e := range roadmap.Edges {
			if mergedIDs[e.Source] && mergedIDs[e.Target] {
				data.Edges = append(data.Edges, e)
			}
		}
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

// handlePriorities handles GET /api/p/:project/priorities
func (s *Server) handlePriorities(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	priorities, err := p.Idx.Priorities()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"priorities": priorities})
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
