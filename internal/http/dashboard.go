package http

import (
	"net/http"
	"strconv"
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

	stats, err := p.Idx.DashboardStats(isoWeekStart(), p.Cfg.Dashboard.TrackedTypes)
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

	distribution, err := p.Idx.StatusDistribution(p.Cfg.Dashboard.TrackedTypes)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"distribution": distribution})
}

// handleGetVelocity handles GET /api/p/:project/dashboard/velocity
// Query params:
//   - granularity  string  daily|weekly|monthly  (default: weekly)
//   - days         int     lookback window        (default: 90, max: 365)
func (s *Server) handleGetVelocity(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())
	if p == nil {
		writeJSON(w, http.StatusInternalServerError, apiError("no_project", "no project in context"))
		return
	}

	q := r.URL.Query()

	granularity := q.Get("granularity")
	switch granularity {
	case "daily", "weekly", "monthly":
	default:
		granularity = "weekly"
	}

	days := 90
	if v := q.Get("days"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			days = n
		}
	}

	buckets, err := p.Idx.CompletionVelocity(granularity, days, p.Cfg.Dashboard.TrackedTypes)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"granularity": granularity,
		"buckets":     buckets,
	})
}
