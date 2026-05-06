package http

import (
	"net/http"
	"time"
)

// isoWeekStart returns midnight (local time) on the Monday that begins the
// current ISO week.
func isoWeekStart() time.Time {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7 in ISO 8601
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, now.Location())
}

// handleGetDashboardStats handles GET /api/p/:project/dashboard/stats
func (s *Server) handleGetDashboardStats(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	stats, err := p.Idx.DashboardStats(isoWeekStart())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// handleGetStatusDistribution handles GET /api/p/:project/dashboard/status-distribution
func (s *Server) handleGetStatusDistribution(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	distribution, err := p.Idx.StatusDistribution()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"distribution": distribution})
}
