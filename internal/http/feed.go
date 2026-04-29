package http

import (
	"net/http"
	"strconv"
	"strings"
)

// handleGetFeed handles GET /api/p/:project/feed
// Query params:
//   - limit  int    default 50, max 200
//   - before int64  cursor: return events with id < before
//   - types  string comma-separated event type filter
func (s *Server) handleGetFeed(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	q := r.URL.Query()

	limit := 50
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 200 {
		limit = 200
	}

	var beforeID int64
	if v := q.Get("before"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			beforeID = n
		}
	}

	var types []string
	if v := q.Get("types"); v != "" {
		for _, t := range strings.Split(v, ",") {
			if t = strings.TrimSpace(t); t != "" {
				types = append(types, t)
			}
		}
	}

	events, err := p.Idx.ListEvents(limit, beforeID, types)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	var nextCursor *int64
	if len(events) == limit && len(events) > 0 {
		id := events[len(events)-1].ID
		nextCursor = &id
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"events":      events,
		"next_cursor": nextCursor,
	})
}
