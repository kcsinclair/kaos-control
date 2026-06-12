// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"errors"
	"net/http"
	"time"

	"github.com/kaos-control/kaos-control/internal/reports"
)

// handleGetAgentUsageReport aggregates agent_runs data and returns the
// summary + time-series analytics report.
// GET /api/p/:project/reports/agent-usage
func (s *Server) handleGetAgentUsageReport(w http.ResponseWriter, r *http.Request) {
	p := projectFromCtx(r.Context())

	now := time.Now().UTC()
	q := r.URL.Query()

	// Parse `from` (default: now-30d).
	from := now.AddDate(0, 0, -30)
	if raw := q.Get("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid from: "+err.Error()))
			return
		}
		from = t
	}

	// Parse `to` (default: now).
	to := now
	if raw := q.Get("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid to: "+err.Error()))
			return
		}
		to = t
	}

	if to.Before(from) {
		writeJSON(w, http.StatusBadRequest, apiError("bad_request", "to before from"))
		return
	}

	// Parse `bucket` (default: "day").
	bucket := q.Get("bucket")
	if bucket == "" {
		bucket = "day"
	}

	// Parse repeated `agent` query params.
	agents := q["agent"]

	// Parse repeated `status` query params.
	statuses := q["status"]

	// Parse `tz` (IANA timezone name; default UTC).
	loc := time.UTC
	if tz := q.Get("tz"); tz != "" {
		l, err := time.LoadLocation(tz)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", "invalid tz"))
			return
		}
		loc = l
	}

	f := reports.AgentUsageFilter{
		From:     from,
		To:       to,
		Agents:   agents,
		Statuses: statuses,
		Bucket:   bucket,
		Loc:      loc,
	}

	report, err := reports.BuildAgentUsageReport(p.Idx, f)
	if err != nil {
		var badFilter reports.ErrBadFilter
		if errors.As(err, &badFilter) {
			writeJSON(w, http.StatusBadRequest, apiError("bad_request", badFilter.Msg))
			return
		}
		writeJSON(w, http.StatusInternalServerError, apiError("db_error", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, report)
}
