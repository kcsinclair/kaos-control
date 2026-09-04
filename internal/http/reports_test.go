// SPDX-License-Identifier: AGPL-3.0-or-later

package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaos-control/kaos-control/internal/reports"
)

// TestHandleGetFailoverReport verifies the failover report route aggregates
// operations.yaml history via reports.FailoverUsage (FR-10.2).
func TestHandleGetFailoverReport(t *testing.T) {
	p := newTestProjectForProviderSwitch(t)
	s := testServerWithAppProviders()

	if err := p.SwitchAgentProvider("failed-over-agent", "gemini-cloud", "gemini-2.5-flash", "seed failover", true); err != nil {
		t.Fatalf("seeding failover state: %v", err)
	}
	if err := p.RestoreAgentProvider("failed-over-agent"); err != nil {
		t.Fatalf("seeding restore: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = withProjectAndUser(req, p, "po@test")
	w := httptest.NewRecorder()
	s.handleGetFailoverReport(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, body: %s", w.Code, w.Body.String())
	}
	var report reports.FailoverReport
	if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(report.PerAgent) != 1 || report.PerAgent[0].Agent != "failed-over-agent" || report.PerAgent[0].FailoverCount != 1 || report.PerAgent[0].RestoreCount != 1 {
		t.Fatalf("unexpected per-agent report: %+v", report.PerAgent)
	}
	if len(report.PerProvider) != 1 || report.PerProvider[0].Provider != "anthropic-cloud" || report.PerProvider[0].FailoverCount != 1 {
		t.Fatalf("unexpected per-provider report: %+v", report.PerProvider)
	}
}
