//go:build integration

package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// newSchedulerTestEnv creates a testEnv with the scheduler goroutines started
// so that TriggerNow dispatches jobs and proj.Close() does not deadlock.
func newSchedulerTestEnv(t *testing.T) *testEnv {
	t.Helper()
	env := newTestEnv(t, nil)
	env.proj.StartScheduler(context.Background())
	return env
}

// schedulerPath builds an API path under /api/p/testproject/scheduler.
func schedulerPath(parts ...string) string {
	p := "/api/p/testproject/scheduler"
	for _, part := range parts {
		p += "/" + part
	}
	return p
}

// shellJobBody returns a JSON-serialisable body for a shell cron job.
func shellJobBody(name, target, cron string) map[string]any {
	return map[string]any{
		"name":        name,
		"target_type": "shell",
		"target":      target,
		"schedule": map[string]any{
			"kind": "cron",
			"cron": cron,
		},
		"enabled":     true,
		"priority":    5,
		"timeout_sec": 30,
	}
}

// waitForSchedulerRun polls GET /runs until at least one completed run exists,
// then returns the first completed run's data map.
func waitForSchedulerRun(t *testing.T, env *testEnv, jobName string, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp := env.doRequest("GET", schedulerPath("jobs", jobName, "runs"), nil)
		data := readJSON(t, resp)
		runs, _ := data["runs"].([]any)
		for _, r := range runs {
			run, _ := r.(map[string]any)
			status, _ := run["status"].(string)
			if status != "" && status != "running" {
				return run
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timeout: no completed run for job %q within %v", jobName, timeout)
	return nil
}

// ----- Test cases -----

// TestSchedulerListJobsEmpty verifies GET /scheduler/jobs returns an empty list
// when no jobs have been created.
func TestSchedulerListJobsEmpty(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", schedulerPath("jobs"), nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	jobs, _ := data["jobs"].([]any)
	if len(jobs) != 0 {
		t.Errorf("expected empty jobs list, got %d items", len(jobs))
	}
}

// TestSchedulerCreateJobValid verifies POST /scheduler/jobs creates a job and
// returns 201 with the job object. Then the job appears in the list.
func TestSchedulerCreateJobValid(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	body := shellJobBody("my-job", "true", "0 2 * * *")
	resp := env.doRequest("POST", schedulerPath("jobs"), body)
	requireStatus(t, resp, http.StatusCreated)
	data := readJSON(t, resp)
	job, _ := data["job"].(map[string]any)
	if job["name"] != "my-job" {
		t.Errorf("name: got %v want my-job", job["name"])
	}
	if job["target_type"] != "shell" {
		t.Errorf("target_type: got %v want shell", job["target_type"])
	}
	if job["enabled"] != true {
		t.Errorf("enabled: got %v want true", job["enabled"])
	}

	// Confirm it appears in the list.
	resp2 := env.doRequest("GET", schedulerPath("jobs"), nil)
	requireStatus(t, resp2, http.StatusOK)
	data2 := readJSON(t, resp2)
	if jobs, _ := data2["jobs"].([]any); len(jobs) != 1 {
		t.Errorf("expected 1 job in list, got %d", len(jobs))
	}
}

// TestSchedulerCreateJobDuplicate verifies that creating a job with a duplicate
// name returns 409 Conflict.
func TestSchedulerCreateJobDuplicate(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	body := shellJobBody("dup-job", "true", "0 2 * * *")
	env.doRequest("POST", schedulerPath("jobs"), body).Body.Close()

	resp := env.doRequest("POST", schedulerPath("jobs"), body)
	requireStatus(t, resp, http.StatusConflict)
	resp.Body.Close()
}

// TestSchedulerCreateJobInvalidCron verifies that a malformed cron expression
// returns 400 with an error payload.
func TestSchedulerCreateJobInvalidCron(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	body := shellJobBody("bad-cron", "true", "not a cron expression")
	resp := env.doRequest("POST", schedulerPath("jobs"), body)
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestSchedulerCreateJobShellPathTraversal verifies that a shell target containing
// path traversal sequences is rejected with 400.
func TestSchedulerCreateJobShellPathTraversal(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	body := map[string]any{
		"name":        "traversal-job",
		"target_type": "shell",
		"target":      "../../etc/passwd",
		"schedule":    map[string]any{"kind": "cron", "cron": "0 2 * * *"},
		"enabled":     true, "priority": 5, "timeout_sec": 30,
	}
	resp := env.doRequest("POST", schedulerPath("jobs"), body)
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestSchedulerCreateJobInvalidAgentRole verifies that targeting an agent that is
// not configured in the project returns 400.
func TestSchedulerCreateJobInvalidAgentRole(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	body := map[string]any{
		"name":        "bad-agent-job",
		"target_type": "agent",
		"target":      "nonexistent-agent",
		"schedule":    map[string]any{"kind": "cron", "cron": "0 2 * * *"},
		"enabled":     true, "priority": 5, "timeout_sec": 30,
	}
	resp := env.doRequest("POST", schedulerPath("jobs"), body)
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestSchedulerCreateJobPriorityOutOfRange verifies that priority values outside
// [1, 10] return 400.
func TestSchedulerCreateJobPriorityOutOfRange(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	for _, pri := range []int{11, -1} {
		body := map[string]any{
			"name":        fmt.Sprintf("pri%d", pri),
			"target_type": "shell",
			"target":      "true",
			"schedule":    map[string]any{"kind": "cron", "cron": "0 2 * * *"},
			"enabled":     true, "priority": pri, "timeout_sec": 30,
		}
		resp := env.doRequest("POST", schedulerPath("jobs"), body)
		requireStatus(t, resp, http.StatusBadRequest)
		resp.Body.Close()
	}
}

// TestSchedulerGetJobDetail verifies GET /scheduler/jobs/:name returns the job
// with a "runs" array attached.
func TestSchedulerGetJobDetail(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	env.doRequest("POST", schedulerPath("jobs"), shellJobBody("detail-job", "true", "0 2 * * *")).Body.Close()

	resp := env.doRequest("GET", schedulerPath("jobs", "detail-job"), nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	job, _ := data["job"].(map[string]any)
	if job["name"] != "detail-job" {
		t.Errorf("name: got %v want detail-job", job["name"])
	}
	if _, ok := data["runs"]; !ok {
		t.Error("response missing 'runs' key")
	}
}

// TestSchedulerGetJobNotFound verifies GET /scheduler/jobs/:name returns 404 for
// an unknown job.
func TestSchedulerGetJobNotFound(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	resp := env.doRequest("GET", schedulerPath("jobs", "ghost-job"), nil)
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

// TestSchedulerUpdateJob verifies PUT /scheduler/jobs/:name updates mutable fields
// and returns 200 with the updated job.
func TestSchedulerUpdateJob(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	env.doRequest("POST", schedulerPath("jobs"), shellJobBody("upd-job", "true", "0 2 * * *")).Body.Close()

	update := map[string]any{
		"target_type": "shell",
		"target":      "false",
		"schedule":    map[string]any{"kind": "cron", "cron": "0 3 * * *"},
		"enabled":     false, "priority": 9, "timeout_sec": 60,
	}
	resp := env.doRequest("PUT", schedulerPath("jobs", "upd-job"), update)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	job, _ := data["job"].(map[string]any)
	if job["target"] != "false" {
		t.Errorf("target: got %v want false", job["target"])
	}
	if job["enabled"] != false {
		t.Errorf("enabled: got %v want false", job["enabled"])
	}
	if pri, _ := job["priority"].(float64); int(pri) != 9 {
		t.Errorf("priority: got %v want 9", job["priority"])
	}
}

// TestSchedulerDeleteJob verifies DELETE /scheduler/jobs/:name returns 204 and
// the job is subsequently not found.
func TestSchedulerDeleteJob(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	env.doRequest("POST", schedulerPath("jobs"), shellJobBody("del-job", "true", "0 2 * * *")).Body.Close()

	resp := env.doRequest("DELETE", schedulerPath("jobs", "del-job"), nil)
	requireStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	resp2 := env.doRequest("GET", schedulerPath("jobs", "del-job"), nil)
	requireStatus(t, resp2, http.StatusNotFound)
	resp2.Body.Close()
}

// TestSchedulerTriggerJob verifies POST /scheduler/jobs/:name/trigger returns
// 200 and the job actually executes.
func TestSchedulerTriggerJob(t *testing.T) {
	env := newSchedulerTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	env.doRequest("POST", schedulerPath("jobs"), shellJobBody("trig-job", "true", "0 2 * * *")).Body.Close()

	resp := env.doRequest("POST", schedulerPath("jobs", "trig-job", "trigger"), nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	if data["triggered"] != "trig-job" {
		t.Errorf("triggered: got %v want trig-job", data["triggered"])
	}

	run := waitForSchedulerRun(t, env, "trig-job", 10*time.Second)
	if run["status"] != "success" {
		t.Errorf("run status: got %v want success", run["status"])
	}
}

// TestSchedulerPauseJob verifies POST /scheduler/jobs/:name/pause returns 200 and
// the job is stored with enabled=false.
func TestSchedulerPauseJob(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	env.doRequest("POST", schedulerPath("jobs"), shellJobBody("pause-job", "true", "0 2 * * *")).Body.Close()

	resp := env.doRequest("POST", schedulerPath("jobs", "pause-job", "pause"), nil)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	resp2 := env.doRequest("GET", schedulerPath("jobs", "pause-job"), nil)
	requireStatus(t, resp2, http.StatusOK)
	data := readJSON(t, resp2)
	job, _ := data["job"].(map[string]any)
	if job["enabled"] != false {
		t.Errorf("expected enabled=false after pause, got %v", job["enabled"])
	}
}

// TestSchedulerResumeJob verifies POST /scheduler/jobs/:name/resume returns 200
// and the job is re-enabled.
func TestSchedulerResumeJob(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	env.doRequest("POST", schedulerPath("jobs"), shellJobBody("res-job", "true", "0 2 * * *")).Body.Close()
	env.doRequest("POST", schedulerPath("jobs", "res-job", "pause"), nil).Body.Close()

	resp := env.doRequest("POST", schedulerPath("jobs", "res-job", "resume"), nil)
	requireStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	resp2 := env.doRequest("GET", schedulerPath("jobs", "res-job"), nil)
	requireStatus(t, resp2, http.StatusOK)
	data := readJSON(t, resp2)
	job, _ := data["job"].(map[string]any)
	if job["enabled"] != true {
		t.Errorf("expected enabled=true after resume, got %v", job["enabled"])
	}
}

// TestSchedulerListRuns verifies GET /scheduler/jobs/:name/runs returns paginated
// run records.
func TestSchedulerListRuns(t *testing.T) {
	env := newSchedulerTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	env.doRequest("POST", schedulerPath("jobs"), shellJobBody("runs-job", "true", "0 2 * * *")).Body.Close()

	// Trigger three runs sequentially.
	for i := 0; i < 3; i++ {
		env.doRequest("POST", schedulerPath("jobs", "runs-job", "trigger"), nil).Body.Close()
		waitForSchedulerRun(t, env, "runs-job", 10*time.Second)
		time.Sleep(100 * time.Millisecond)
	}

	resp := env.doRequest("GET", schedulerPath("jobs", "runs-job", "runs")+"?page=1&per_page=2", nil)
	requireStatus(t, resp, http.StatusOK)
	data := readJSON(t, resp)
	if total, _ := data["total"].(float64); int(total) < 3 {
		t.Errorf("total: got %v want >=3", total)
	}
	if runs, _ := data["runs"].([]any); len(runs) != 2 {
		t.Errorf("runs on page 1: got %d want 2", len(runs))
	}
}

// TestSchedulerGetRunLog verifies GET /scheduler/jobs/:name/runs/:id/log serves
// the log content with Content-Type text/plain.
func TestSchedulerGetRunLog(t *testing.T) {
	env := newSchedulerTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	env.doRequest("POST", schedulerPath("jobs"), shellJobBody("log-job", "echo LOG_CONTENT", "0 2 * * *")).Body.Close()
	env.doRequest("POST", schedulerPath("jobs", "log-job", "trigger"), nil).Body.Close()
	run := waitForSchedulerRun(t, env, "log-job", 10*time.Second)

	runID := fmt.Sprintf("%d", int64(run["id"].(float64)))
	resp := env.doRequest("GET", schedulerPath("jobs", "log-job", "runs", runID, "log"), nil)
	requireStatus(t, resp, http.StatusOK)
	ct := resp.Header.Get("Content-Type")
	if ct != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type: got %q want text/plain; charset=utf-8", ct)
	}
	resp.Body.Close()
}

// TestSchedulerGetRunLogPruned verifies that GET /runs/:id/log returns 404 when
// the underlying log file has been removed from disk.
func TestSchedulerGetRunLogPruned(t *testing.T) {
	env := newSchedulerTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	env.doRequest("POST", schedulerPath("jobs"), shellJobBody("pruned-job", "echo hi", "0 2 * * *")).Body.Close()
	env.doRequest("POST", schedulerPath("jobs", "pruned-job", "trigger"), nil).Body.Close()
	run := waitForSchedulerRun(t, env, "pruned-job", 10*time.Second)

	logPath, _ := run["log_path"].(string)
	if logPath == "" {
		t.Skip("no log_path on run — cannot test pruned-log scenario")
	}
	if err := os.Remove(logPath); err != nil {
		t.Fatal(err)
	}

	runID := fmt.Sprintf("%d", int64(run["id"].(float64)))
	resp := env.doRequest("GET", schedulerPath("jobs", "pruned-job", "runs", runID, "log"), nil)
	requireStatus(t, resp, http.StatusNotFound)
	resp.Body.Close()
}

// TestSchedulerUnauthenticated verifies that all scheduler endpoints reject
// requests without a valid session. GET endpoints return 401; mutating endpoints
// (POST/PUT/DELETE) may return 401 or 403 depending on whether the CSRF
// middleware fires before the auth middleware.
func TestSchedulerUnauthenticated(t *testing.T) {
	env := newTestEnv(t, nil)
	// No login.

	endpoints := []struct{ method, path string }{
		{"GET", schedulerPath("jobs")},
		{"POST", schedulerPath("jobs")},
		{"GET", schedulerPath("jobs", "x")},
		{"PUT", schedulerPath("jobs", "x")},
		{"DELETE", schedulerPath("jobs", "x")},
		{"POST", schedulerPath("jobs", "x", "trigger")},
		{"POST", schedulerPath("jobs", "x", "pause")},
		{"POST", schedulerPath("jobs", "x", "resume")},
		{"GET", schedulerPath("jobs", "x", "runs")},
		{"GET", schedulerPath("jobs", "x", "runs", "1", "log")},
	}
	for _, ep := range endpoints {
		req, err := http.NewRequest(ep.method, env.baseURL+ep.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		// CSRF middleware fires before auth for mutating requests, so POST/PUT/DELETE
		// without credentials return 403 (CSRF) rather than 401 (auth).  Both
		// indicate the request was correctly rejected.
		if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s: expected 401 or 403, got %d", ep.method, ep.path, resp.StatusCode)
		}
	}
}

// TestSchedulerCSRFEnforcement verifies that mutating endpoints without the
// X-CSRF-Token header return 403.
func TestSchedulerCSRFEnforcement(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// POST without CSRF token (cookies present but no X-CSRF-Token header).
	req, err := http.NewRequest("POST", env.baseURL+schedulerPath("jobs"), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for _, c := range env.cookies {
		req.AddCookie(c)
	}
	// Deliberately omit the X-CSRF-Token header.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for missing CSRF token, got %d", resp.StatusCode)
	}
}
