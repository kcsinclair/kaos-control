// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

package integration

// Milestones 2 and 4 — Run history listing endpoint and live completion.
//
// Tests for GET /api/p/{project}/devops/pipelines/{slug}/runs (F2; NF1, NF5)
// and the live-update behaviour driven by the pipeline.run.completed WebSocket
// event (F6).
//
// Run with: go test ./tests/... -tags integration -run TestRunHistory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/auth"
	"github.com/kaos-control/kaos-control/internal/config"
	"github.com/kaos-control/kaos-control/internal/devops"
	kaoshttp "github.com/kaos-control/kaos-control/internal/http"
	"github.com/kaos-control/kaos-control/internal/project"
)

// TestRunHistory_ListNewestFirst runs a pipeline once (passed), seeds a second
// older record with status=failed, then asserts the listing returns both runs
// newest-first with the required five fields.
func TestRunHistory_ListNewestFirst(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Trigger a real run so we get a "passed" record at approximately now.
	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	data := readJSON(t, resp)
	runID1, _ := data["run_id"].(string)
	if runID1 == "" {
		t.Fatal("expected non-empty run_id")
	}
	waitForRunComplete(t, env, "quick-pass", 15*time.Second)

	// Seed a second (older) record manually so we control the status.
	olderStart := time.Now().UTC().Add(-2 * time.Hour)
	env.proj.DevopsLogs.WriteRecord("testproject", devops.RunRecord{
		RunID:      "ffffffffffffffff",
		Slug:       "quick-pass",
		StartedAt:  olderStart.Format(time.RFC3339),
		EndedAt:    olderStart.Add(5 * time.Second).Format(time.RFC3339),
		DurationMs: 5000,
		Status:     "failed",
		LogRef:     "ffffffffffffffff.log",
	})

	runsResp := env.doRequest(http.MethodGet, devopsPipelineRunsPath("quick-pass"), nil)
	requireStatus(t, runsResp, http.StatusOK)
	runsData := readJSON(t, runsResp)

	runs, _ := runsData["runs"].([]any)
	if len(runs) < 2 {
		t.Fatalf("expected ≥2 runs, got %d", len(runs))
	}

	// Newest run (index 0) must be the just-completed "passed" run.
	newest := runs[0].(map[string]any)
	if newest["run_id"] != runID1 {
		t.Errorf("runs[0].run_id = %q, want %q", newest["run_id"], runID1)
	}
	if newest["status"] != "passed" {
		t.Errorf("runs[0].status = %q, want passed", newest["status"])
	}

	// Second run must be the seeded "failed" run.
	second := runs[1].(map[string]any)
	if second["status"] != "failed" {
		t.Errorf("runs[1].status = %q, want failed", second["status"])
	}

	// Each run must have the five required fields.
	for i, r := range runs[:2] {
		row := r.(map[string]any)
		for _, field := range []string{"run_id", "status", "started_at", "ended_at", "duration_ms"} {
			if _, ok := row[field]; !ok {
				t.Errorf("runs[%d] missing field %q", i, field)
			}
		}
		if sa, _ := row["started_at"].(string); !isRFC3339(sa) {
			t.Errorf("runs[%d].started_at %q is not valid RFC 3339", i, sa)
		}
	}
}

// TestRunHistory_LimitDefaultAndCap seeds 55 records and verifies that the
// default returns 10, ?limit=2 returns 2, and ?limit=999 is capped at 50.
func TestRunHistory_LimitDefaultAndCap(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	base := time.Now().UTC()
	for i := 0; i < 55; i++ {
		startedAt := base.Add(-time.Duration(i) * time.Minute)
		env.proj.DevopsLogs.WriteRecord("testproject", devops.RunRecord{
			RunID:      fmt.Sprintf("cccc%012x", i),
			Slug:       "quick-pass",
			StartedAt:  startedAt.Format(time.RFC3339),
			EndedAt:    startedAt.Add(5 * time.Second).Format(time.RFC3339),
			DurationMs: 5000,
			Status:     "passed",
			LogRef:     fmt.Sprintf("cccc%012x.log", i),
		})
	}

	// Default (no limit param) → 10.
	r1 := env.doRequest(http.MethodGet, devopsPipelineRunsPath("quick-pass"), nil)
	requireStatus(t, r1, http.StatusOK)
	d1 := readJSON(t, r1)
	runs1, _ := d1["runs"].([]any)
	if len(runs1) != 10 {
		t.Errorf("default limit: got %d runs, want 10", len(runs1))
	}

	// ?limit=2 → 2.
	r2 := env.doRequest(http.MethodGet, devopsPipelineRunsPath("quick-pass")+"?limit=2", nil)
	requireStatus(t, r2, http.StatusOK)
	d2 := readJSON(t, r2)
	runs2, _ := d2["runs"].([]any)
	if len(runs2) != 2 {
		t.Errorf("limit=2: got %d runs, want 2", len(runs2))
	}

	// ?limit=999 → capped at 50.
	r3 := env.doRequest(http.MethodGet, devopsPipelineRunsPath("quick-pass")+"?limit=999", nil)
	requireStatus(t, r3, http.StatusOK)
	d3 := readJSON(t, r3)
	runs3, _ := d3["runs"].([]any)
	if len(runs3) > 50 {
		t.Errorf("limit=999 cap: got %d runs, want ≤50", len(runs3))
	}
	if len(runs3) < 50 {
		t.Errorf("limit=999 cap: got %d runs, want 50 (we seeded 55)", len(runs3))
	}
}

// TestRunHistory_EmptyPipeline asserts that a known pipeline with no runs
// returns 200 and an empty array.
func TestRunHistory_EmptyPipeline(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodGet, devopsPipelineRunsPath("quick-pass"), nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)

	runs, ok := data["runs"]
	if !ok {
		t.Fatal("response missing 'runs' key")
	}
	switch v := runs.(type) {
	case []any:
		if len(v) != 0 {
			t.Errorf("expected empty runs array, got %d elements", len(v))
		}
	case nil:
		// JSON null is acceptable for an empty list.
	default:
		t.Errorf("expected array for 'runs', got %T", runs)
	}
}

// TestRunHistory_UnknownSlug404 asserts that listing runs for a non-existent
// pipeline returns 404.
func TestRunHistory_UnknownSlug404(t *testing.T) {
	env := newDevopsTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest(http.MethodGet, devopsPipelineRunsPath("nonexistent-pipe"), nil)
	requireStatus(t, resp, http.StatusNotFound)
	data := readJSON(t, resp)

	errObj, _ := data["error"].(map[string]any)
	if code, _ := errObj["code"].(string); code != "not_found" {
		t.Errorf("error.code = %q, want not_found", code)
	}
}

// TestRunHistory_ForbiddenRole asserts that non-devops/non-owner roles get 403
// and unauthenticated requests get 401.
func TestRunHistory_ForbiddenRole(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})

	// qa role → 403.
	env.login("qa@test.local", "qa-pass-123")
	resp := env.doRequest(http.MethodGet, devopsPipelineRunsPath("quick-pass"), nil)
	requireStatus(t, resp, http.StatusForbidden)
	resp.Body.Close()

	// Unauthenticated → 401.
	env.logout()
	resp2 := env.doRequest(http.MethodGet, devopsPipelineRunsPath("quick-pass"), nil)
	requireStatus(t, resp2, http.StatusUnauthorized)
	resp2.Body.Close()
}

// TestRunHistory_CancelledRecorded starts a slow pipeline, cancels it, then
// asserts the listing shows status=cancelled for the most recent run.
func TestRunHistory_CancelledRecorded(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"slow-step.yaml": pipelineSlowStep,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Start the slow pipeline.
	startResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/run", nil)
	requireStatus(t, startResp, http.StatusAccepted)
	startResp.Body.Close()

	// Wait briefly then cancel.
	time.Sleep(150 * time.Millisecond)
	cancelResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/slow-step/cancel", nil)
	cancelResp.Body.Close()
	waitForRunComplete(t, env, "slow-step", 10*time.Second)

	// List runs and check the newest has status=cancelled.
	runsResp := env.doRequest(http.MethodGet, devopsPipelineRunsPath("slow-step"), nil)
	requireStatus(t, runsResp, http.StatusOK)
	data := readJSON(t, runsResp)

	runs, _ := data["runs"].([]any)
	if len(runs) == 0 {
		t.Fatal("expected at least one run after cancel")
	}
	newest := runs[0].(map[string]any)
	if newest["status"] != "cancelled" {
		t.Errorf("newest run status = %q, want cancelled", newest["status"])
	}
}

// TestRunHistory_PersistsAcrossRestart runs a pipeline, shuts down the project
// environment, reopens it against the same data directory, and verifies the
// run record is still listed — proving disk (not memory) is the source of
// truth.
func TestRunHistory_PersistsAcrossRestart(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Run the pipeline and record the run ID.
	startResp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, startResp, http.StatusAccepted)
	runID, _ := readJSON(t, startResp)["run_id"].(string)
	if runID == "" {
		t.Fatal("expected non-empty run_id")
	}
	waitForRunComplete(t, env, "quick-pass", 15*time.Second)

	// Capture the persistent dirs before shutting down env1.
	savedProjectRoot := env.projectRoot
	savedDataDir := env.dataDir

	// Stop the first environment's HTTP server.
	env.cancel()
	time.Sleep(150 * time.Millisecond)

	// Reopen the project on the same data directory (simulates server restart).
	env2 := reopenDevopsTestEnvAt(t, savedProjectRoot, savedDataDir)
	env2.login("admin@test.local", "admin-pass-123")

	// The run must still appear in the listing after restart.
	runsResp := env2.doRequest(http.MethodGet, devopsPipelineRunsPath("quick-pass"), nil)
	requireStatus(t, runsResp, http.StatusOK)
	data := readJSON(t, runsResp)

	runs, _ := data["runs"].([]any)
	if len(runs) == 0 {
		t.Fatal("expected at least one run after restart, got none")
	}
	found := false
	for _, r := range runs {
		row, _ := r.(map[string]any)
		if row["run_id"] == runID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("run %q not found in listing after restart", runID)
	}
}

// TestRunHistory_Performance50Runs seeds 50 records and asserts the GET
// responds within 200 ms (NF1).
func TestRunHistory_Performance50Runs(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	base := time.Now().UTC()
	for i := 0; i < 50; i++ {
		startedAt := base.Add(-time.Duration(i) * time.Minute)
		env.proj.DevopsLogs.WriteRecord("testproject", devops.RunRecord{
			RunID:      fmt.Sprintf("perf%012x", i),
			Slug:       "quick-pass",
			StartedAt:  startedAt.Format(time.RFC3339),
			EndedAt:    startedAt.Add(5 * time.Second).Format(time.RFC3339),
			DurationMs: 5000,
			Status:     "passed",
			LogRef:     fmt.Sprintf("perf%012x.log", i),
		})
	}

	start := time.Now()
	resp := env.doRequest(http.MethodGet, devopsPipelineRunsPath("quick-pass"), nil)
	elapsed := time.Since(start)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	const maxLatency = 200 * time.Millisecond
	if elapsed > maxLatency {
		t.Errorf("GET /runs with 50 records took %v, want < %v", elapsed, maxLatency)
	}
}

// ── Milestone 4 — Live update via WebSocket (F6) ─────────────────────────────

// TestRunHistory_LiveCompletionAppears registers a Hub channel, triggers a
// run, awaits pipeline.run.completed, then immediately GETs the listing and
// asserts the just-completed run is present and newest.
func TestRunHistory_LiveCompletionAppears(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	// Register the Hub channel before triggering so we don't miss the event.
	ch := make(chan []byte, 256)
	env.proj.Hub.Register(ch)
	defer env.proj.Hub.Unregister(ch)

	resp := env.doRequest(http.MethodPost, "/api/p/testproject/devops/pipelines/quick-pass/run", nil)
	requireStatus(t, resp, http.StatusAccepted)
	runID, _ := readJSON(t, resp)["run_id"].(string)

	// Wait for pipeline.run.completed WS event.
	deadline := time.After(15 * time.Second)
WAIT:
	for {
		select {
		case raw := <-ch:
			var evt wsEvent
			if err := json.Unmarshal(raw, &evt); err != nil {
				continue
			}
			if evt.Type == "pipeline.run.completed" {
				break WAIT
			}
		case <-deadline:
			t.Fatal("timed out waiting for pipeline.run.completed")
		}
	}

	// Immediately list runs — the completed run must already be present.
	runsResp := env.doRequest(http.MethodGet, devopsPipelineRunsPath("quick-pass"), nil)
	requireStatus(t, runsResp, http.StatusOK)
	data := readJSON(t, runsResp)

	runs, _ := data["runs"].([]any)
	if len(runs) == 0 {
		t.Fatal("listing returned no runs immediately after pipeline.run.completed")
	}
	newest, _ := runs[0].(map[string]any)
	if newest["run_id"] != runID {
		t.Errorf("newest run_id = %q, want %q", newest["run_id"], runID)
	}
}

// TestRunHistory_NoNewEventTypes verifies that the event stream for a run
// contains only the five pre-existing pipeline.* event types — guarding
// against accidental introduction of pipeline.history.* variants (F6).
func TestRunHistory_NoNewEventTypes(t *testing.T) {
	env := newDevopsTestEnv(t, map[string]string{
		"quick-pass.yaml": pipelineQuickPass,
	})
	env.login("admin@test.local", "admin-pass-123")

	events := collectDevopsEvents(t, env, "quick-pass", 15*time.Second)

	allowed := map[string]bool{
		"pipeline.run.started":    true,
		"pipeline.step.started":   true,
		"pipeline.step.output":    true,
		"pipeline.step.completed": true,
		"pipeline.run.completed":  true,
	}
	for _, e := range events {
		if !allowed[e.Type] {
			t.Errorf("unexpected event type %q in pipeline run stream", e.Type)
		}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// reopenDevopsTestEnvAt creates a new testEnv against an already-initialised
// project root and data directory, simulating a server restart. Unlike
// newDevopsTestEnv it skips git-init and user creation.
func reopenDevopsTestEnvAt(t *testing.T, projectRoot, dataDir string) *testEnv {
	t.Helper()

	authDBPath := dataDir + "/auth.db"
	authStore, err := auth.Open(authDBPath, 24*time.Hour)
	if err != nil {
		t.Fatalf("reopenDevopsTestEnvAt: open auth: %v", err)
	}
	t.Cleanup(func() { authStore.Close() })

	entry := &config.ProjectEntry{
		Name:        "testproject",
		Path:        projectRoot,
		Description: "integration test project (reopened)",
	}
	proj, err := project.Open(entry, dataDir, project.OpenOptions{
		MaxConcurrentAgents: 2,
		DevopsLogDir:        dataDir,
	})
	if err != nil {
		t.Fatalf("reopenDevopsTestEnvAt: open project: %v", err)
	}
	t.Cleanup(func() { proj.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	proj.StartWatcher(ctx)
	proj.StartLockReaper(ctx)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		t.Fatalf("reopenDevopsTestEnvAt: listen: %v", err)
	}
	addr := ln.Addr().String()

	srv := kaoshttp.New(kaoshttp.ServerConfig{
		Listener: ln,
		Auth:     authStore,
	}, map[string]*project.Project{"testproject": proj})

	srvDone := make(chan error, 1)
	go func() { srvDone <- srv.ListenAndServe(ctx) }()

	baseURL := "http://" + addr
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
	}

	env := &testEnv{
		t:           t,
		projectRoot: projectRoot,
		dataDir:     dataDir,
		baseURL:     baseURL,
		cancel:      cancel,
		authStore:   authStore,
		proj:        proj,
	}
	t.Cleanup(func() {
		cancel()
		select {
		case <-srvDone:
		case <-time.After(5 * time.Second):
		}
	})

	return env
}

// devopsRunsField extracts the "runs" array from a JSON response body.
// Provided for brevity in assertions.
func devopsRunsField(t *testing.T, resp *http.Response) []any {
	t.Helper()
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("devopsRunsField: unmarshal: %v", err)
	}
	runs, _ := m["runs"].([]any)
	return runs
}
