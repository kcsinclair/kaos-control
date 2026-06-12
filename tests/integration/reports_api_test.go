// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/index"
)

// seedAgentRun inserts a minimal agent run into the test project's index.
func seedAgentRun(t *testing.T, env *testEnv, id, agent string, startedAt time.Time, status string) {
	t.Helper()
	row := &index.AgentRunRow{
		RunID:     id,
		AgentName: agent,
		Role:      "analyst",
		StartedAt: startedAt,
		Status:    status,
	}
	if err := env.proj.Idx.InsertAgentRun(row); err != nil {
		t.Fatalf("seedAgentRun(%s): %v", id, err)
	}
	if status != "running" {
		finished := startedAt.Add(time.Second)
		row.FinishedAt = &finished
		if err := env.proj.Idx.UpdateAgentRun(row); err != nil {
			t.Fatalf("UpdateAgentRun(%s): %v", id, err)
		}
	}
}

// seedRunWithMetrics inserts a run with full metrics into the test project's index.
func seedRunWithMetrics(
	t *testing.T, env *testEnv,
	id, agent, model string,
	startedAt time.Time,
	status string,
	cost float64,
	dur, inputTok, cacheCreate, cacheRead, outputTok int64,
) {
	t.Helper()
	seedAgentRun(t, env, id, agent, startedAt, status)
	m := index.AgentRunMetrics{
		Model:               model,
		TotalCostUSD:        cost,
		DurationApiMs:       dur,
		InputTokens:         inputTok,
		CacheCreationTokens: cacheCreate,
		CacheReadTokens:     cacheRead,
		OutputTokens:        outputTok,
	}
	if err := env.proj.Idx.UpdateAgentRunMetrics(id, m); err != nil {
		t.Fatalf("seedRunWithMetrics UpdateAgentRunMetrics(%s): %v", id, err)
	}
}

func reportsURL(env *testEnv) string {
	return "/api/p/testproject/reports/agent-usage"
}

// TestReportsAgentUsage_Defaults seeds 5 runs in the last 7 days and 2 runs
// 60 days ago. With the default 30-day window, only the 5 recent runs should
// appear.
func TestReportsAgentUsage_Defaults(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	now := time.Now()
	for i := 0; i < 5; i++ {
		seedAgentRun(t, env,
			fmt.Sprintf("recent-%02d", i), "qa",
			now.Add(-time.Duration(i+1)*24*time.Hour), "done")
	}
	// 2 old runs outside the default 30d window.
	for i := 0; i < 2; i++ {
		seedAgentRun(t, env,
			fmt.Sprintf("old-%02d", i), "qa",
			now.Add(-time.Duration(60+i)*24*time.Hour), "done")
	}

	resp := env.doRequest("GET", reportsURL(env), nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	summary, _ := data["summary"].(map[string]any)
	overall, _ := summary["overall"].(map[string]any)
	runCount, _ := overall["run_count"].(float64)
	if int(runCount) != 5 {
		t.Errorf("run_count: got %v, want 5 (old runs outside 30d window)", runCount)
	}
	series, _ := data["series"].([]any)
	if len(series) == 0 {
		t.Error("series should be non-empty array even when bucket-filling")
	}
}

// TestReportsAgentUsage_ResponseShape verifies the top-level JSON shape.
func TestReportsAgentUsage_ResponseShape(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", reportsURL(env), nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	if _, ok := data["summary"]; !ok {
		t.Error("response missing 'summary' key")
	}
	summary, _ := data["summary"].(map[string]any)
	if _, ok := summary["overall"]; !ok {
		t.Error("summary missing 'overall' key")
	}
	if _, ok := summary["per_model"]; !ok {
		t.Error("summary missing 'per_model' key")
	}
	if _, ok := summary["per_agent"]; !ok {
		t.Error("summary missing 'per_agent' key")
	}
	if _, ok := data["series"]; !ok {
		t.Error("response missing 'series' key")
	}
	if _, ok := data["series_by_model"]; !ok {
		t.Error("response missing 'series_by_model' key")
	}
	// series_by_agent is omitempty — should NOT appear without an agent filter.
	if _, ok := data["series_by_agent"]; ok {
		t.Error("series_by_agent should be absent when no agent filter is set")
	}
}

// TestReportsAgentUsage_FilterFrom_To seeds runs at known times and queries a
// narrow window with explicit from/to params.
func TestReportsAgentUsage_FilterFrom_To(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	base := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	// 3 runs inside the window
	for i := 0; i < 3; i++ {
		seedAgentRun(t, env,
			fmt.Sprintf("win-%02d", i), "qa",
			base.Add(time.Duration(i)*time.Hour), "done")
	}
	// 2 runs outside the window
	seedAgentRun(t, env, "before", "qa", base.Add(-2*24*time.Hour), "done")
	seedAgentRun(t, env, "after", "qa", base.Add(10*24*time.Hour), "done")

	from := base.Add(-time.Hour).Format(time.RFC3339)
	to := base.Add(4 * time.Hour).Format(time.RFC3339)
	url := fmt.Sprintf("%s?from=%s&to=%s", reportsURL(env), from, to)

	resp := env.doRequest("GET", url, nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	summary, _ := data["summary"].(map[string]any)
	overall, _ := summary["overall"].(map[string]any)
	runCount, _ := overall["run_count"].(float64)
	if int(runCount) != 3 {
		t.Errorf("run_count with explicit window: got %v, want 3", runCount)
	}
}

// TestReportsAgentUsage_FilterAgent filters by specific agents; verifies
// per_agent and series_by_agent are present.
func TestReportsAgentUsage_FilterAgent(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	now := time.Now()
	agents := []string{"qa", "qa", "backend-developer", "backend-developer", "frontend-developer"}
	for i, a := range agents {
		seedAgentRun(t, env,
			fmt.Sprintf("fagent-%02d", i), a,
			now.Add(-time.Duration(i+1)*time.Hour), "done")
	}

	url := reportsURL(env) + "?agent=qa&agent=backend-developer"
	resp := env.doRequest("GET", url, nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	summary, _ := data["summary"].(map[string]any)
	overall, _ := summary["overall"].(map[string]any)
	runCount, _ := overall["run_count"].(float64)
	if int(runCount) != 4 {
		t.Errorf("run_count with agent filter: got %v, want 4 (qa=2 + backend=2)", runCount)
	}

	perAgent, _ := summary["per_agent"].([]any)
	if len(perAgent) != 2 {
		t.Errorf("per_agent count: got %d, want 2", len(perAgent))
	}

	if _, ok := data["series_by_agent"]; !ok {
		t.Error("series_by_agent should be present when agent filter is active")
	}
}

// TestReportsAgentUsage_FilterStatus filters to only failed runs.
func TestReportsAgentUsage_FilterStatus(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	now := time.Now()
	seedAgentRun(t, env, "fstatus-done", "qa", now.Add(-time.Hour), "done")
	seedAgentRun(t, env, "fstatus-fail-0", "qa", now.Add(-2*time.Hour), "failed")
	seedAgentRun(t, env, "fstatus-fail-1", "qa", now.Add(-3*time.Hour), "failed")

	url := reportsURL(env) + "?status=failed"
	resp := env.doRequest("GET", url, nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	summary, _ := data["summary"].(map[string]any)
	overall, _ := summary["overall"].(map[string]any)
	runCount, _ := overall["run_count"].(float64)
	if int(runCount) != 2 {
		t.Errorf("run_count with status=failed: got %v, want 2", runCount)
	}
	successCount, _ := overall["success_count"].(float64)
	if int(successCount) != 0 {
		t.Errorf("success_count with status=failed filter: got %v, want 0", successCount)
	}
}

// TestReportsAgentUsage_BadTo_Returns400 verifies that to < from returns 400.
func TestReportsAgentUsage_BadTo_Returns400(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	url := reportsURL(env) + "?from=2026-01-02T00:00:00Z&to=2026-01-01T00:00:00Z"
	resp := env.doRequest("GET", url, nil)
	requireStatus(t, resp, http.StatusBadRequest)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "bad_request" {
		t.Errorf("error.code: got %q, want 'bad_request'", code)
	}
}

// TestReportsAgentUsage_UnknownBucket_Returns400 verifies that an unknown
// bucket parameter returns 400.
func TestReportsAgentUsage_UnknownBucket_Returns400(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", reportsURL(env)+"?bucket=year", nil)
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestReportsAgentUsage_BadTz_Returns400 verifies that an invalid timezone
// name returns 400.
func TestReportsAgentUsage_BadTz_Returns400(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", reportsURL(env)+"?tz=Mars%2FPhobos", nil)
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestReportsAgentUsage_MetricsUnavailableRuns verifies that
// metrics_unavailable_count is correctly reported.
func TestReportsAgentUsage_MetricsUnavailableRuns(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	now := time.Now()
	// 3 runs with metrics.
	for i := 0; i < 3; i++ {
		seedRunWithMetrics(t, env,
			fmt.Sprintf("meta-%02d", i), "qa", "claude-opus",
			now.Add(-time.Duration(i+1)*time.Hour), "done",
			0.01, 1000, 100, 0, 50, 200)
	}
	// 2 runs without metrics.
	for i := 0; i < 2; i++ {
		seedAgentRun(t, env,
			fmt.Sprintf("nometa-%02d", i), "qa",
			now.Add(-time.Duration(i+4)*time.Hour), "done")
	}

	resp := env.doRequest("GET", reportsURL(env), nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	summary, _ := data["summary"].(map[string]any)
	overall, _ := summary["overall"].(map[string]any)
	unavail, _ := overall["metrics_unavailable_count"].(float64)
	if int(unavail) != 2 {
		t.Errorf("metrics_unavailable_count: got %v, want 2", unavail)
	}
}

// TestReportsAgentUsage_Empty verifies that an empty project returns 200 with
// run_count=0 and a non-empty series (bucket-gap filling).
func TestReportsAgentUsage_Empty(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", reportsURL(env), nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	summary, _ := data["summary"].(map[string]any)
	overall, _ := summary["overall"].(map[string]any)
	runCount, _ := overall["run_count"].(float64)
	if int(runCount) != 0 {
		t.Errorf("run_count on empty project: got %v, want 0", runCount)
	}
	series, _ := data["series"].([]any)
	if len(series) == 0 {
		t.Error("series should be non-empty (continuous bucket sequence) even with no runs")
	}
}

// TestReportsAgentUsage_Performance10k seeds 10 000 runs and asserts the
// request completes in under 2 seconds. Skipped in short mode.
func TestReportsAgentUsage_Performance10k(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	now := time.Now()
	for i := 0; i < 10000; i++ {
		row := &index.AgentRunRow{
			RunID:     fmt.Sprintf("perf-%05d", i),
			AgentName: "qa",
			Role:      "analyst",
			StartedAt: now.Add(-time.Duration(i) * time.Minute),
			Status:    "done",
		}
		if err := env.proj.Idx.InsertAgentRun(row); err != nil {
			t.Fatalf("InsertAgentRun perf-%05d: %v", i, err)
		}
	}

	start := time.Now()
	resp := env.doRequest("GET", reportsURL(env), nil)
	elapsed := time.Since(start)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	if elapsed > 2*time.Second {
		t.Errorf("performance: request took %v, want < 2s", elapsed)
	}
}

// TestReportsAgentUsage_AuthRequired verifies that the endpoint requires
// authentication.
func TestReportsAgentUsage_AuthRequired(t *testing.T) {
	env := newTestEnv(t, nil)
	env.logout()

	resp := env.doRequest("GET", reportsURL(env), nil)
	requireStatus(t, resp, http.StatusUnauthorized)
	resp.Body.Close()
}
