//go:build integration

package integration

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSchedulerShellPathTraversalRelative verifies that creating a shell job with
// a relative path that escapes the project root (e.g. "../../etc/passwd") is
// rejected with 400 at the API level.
func TestSchedulerShellPathTraversalRelative(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	body := map[string]any{
		"name":        "traversal-rel",
		"target_type": "shell",
		"target":      "../../../etc/passwd",
		"schedule":    map[string]any{"kind": "cron", "cron": "0 2 * * *"},
		"enabled":     true, "priority": 5, "timeout_sec": 30,
	}
	resp := env.doRequest("POST", schedulerPath("jobs"), body)
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestSchedulerShellPathTraversalAbsolute verifies that an absolute path is
// rejected with 400 at the API level.
func TestSchedulerShellPathTraversalAbsolute(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	body := map[string]any{
		"name":        "traversal-abs",
		"target_type": "shell",
		"target":      "/etc/passwd",
		"schedule":    map[string]any{"kind": "cron", "cron": "0 2 * * *"},
		"enabled":     true, "priority": 5, "timeout_sec": 30,
	}
	resp := env.doRequest("POST", schedulerPath("jobs"), body)
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestSchedulerShellPathTraversalSymlink verifies that a shell target that
// resolves via a symlink to a path outside the project root is rejected with 400.
func TestSchedulerShellPathTraversalSymlink(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	// Create a symlink inside the project root pointing outside.
	symlinkPath := filepath.Join(env.projectRoot, "evil-link")
	if err := os.Symlink("/etc/passwd", symlinkPath); err != nil {
		t.Skip("cannot create symlink (OS restriction):", err)
	}
	t.Cleanup(func() { os.Remove(symlinkPath) })

	body := map[string]any{
		"name":        "traversal-sym",
		"target_type": "shell",
		"target":      "evil-link",
		"schedule":    map[string]any{"kind": "cron", "cron": "0 2 * * *"},
		"enabled":     true, "priority": 5, "timeout_sec": 30,
	}
	resp := env.doRequest("POST", schedulerPath("jobs"), body)
	// The sandbox resolves symlinks and must reject targets outside the root.
	requireStatus(t, resp, http.StatusBadRequest)
	resp.Body.Close()
}

// TestSchedulerShellEnvironmentIsolation verifies that shell jobs run with a
// minimal environment that does not include arbitrary server process env vars.
func TestSchedulerShellEnvironmentIsolation(t *testing.T) {
	env := newSchedulerTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	// Set a sensitive env var in the test process.
	t.Setenv("KAOS_TEST_SECRET", "should-not-leak")

	// Shell job prints $KAOS_TEST_SECRET; if env is isolated it will be empty.
	body := map[string]any{
		"name":        "env-isolation-job",
		"target_type": "shell",
		"target":      `echo "SECRET=${KAOS_TEST_SECRET}"`,
		"schedule":    map[string]any{"kind": "cron", "cron": "0 2 * * *"},
		"enabled":     true, "priority": 5, "timeout_sec": 10,
	}
	env.doRequest("POST", schedulerPath("jobs"), body).Body.Close()
	env.doRequest("POST", schedulerPath("jobs", "env-isolation-job", "trigger"), nil).Body.Close()

	run := waitForSchedulerRun(t, env, "env-isolation-job", 10*time.Second)
	logPath, _ := run["log_path"].(string)
	if logPath == "" {
		t.Skip("no log_path on run — cannot verify env isolation")
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}
	output := string(data)
	if strings.Contains(output, "should-not-leak") {
		t.Errorf("secret value leaked into shell job environment; log:\n%s", output)
	}
	// Should print "SECRET=" (empty value).
	if !strings.Contains(output, "SECRET=") {
		t.Errorf("expected SECRET= in output; log:\n%s", output)
	}
}

// TestSchedulerShellRunsInProjectRoot verifies that the working directory of a
// shell job matches the project root.
func TestSchedulerShellRunsInProjectRoot(t *testing.T) {
	env := newSchedulerTestEnv(t)
	env.login("admin@test.local", "admin-pass-123")

	body := map[string]any{
		"name":        "pwd-job",
		"target_type": "shell",
		"target":      "pwd",
		"schedule":    map[string]any{"kind": "cron", "cron": "0 2 * * *"},
		"enabled":     true, "priority": 5, "timeout_sec": 10,
	}
	env.doRequest("POST", schedulerPath("jobs"), body).Body.Close()
	env.doRequest("POST", schedulerPath("jobs", "pwd-job", "trigger"), nil).Body.Close()

	run := waitForSchedulerRun(t, env, "pwd-job", 10*time.Second)
	logPath, _ := run["log_path"].(string)
	if logPath == "" {
		t.Skip("no log_path on run — cannot verify working directory")
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log: %v", err)
	}

	// `pwd` output and projectRoot may both have symlinks resolved differently,
	// so compare the real paths.
	realProjectRoot, _ := filepath.EvalSymlinks(env.projectRoot)
	output := strings.TrimSpace(string(data))

	// Strip header lines (scheduler logs a comment at the top of each log file).
	lines := strings.Split(output, "\n")
	var pwdLine string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "#") {
			pwdLine = l
			break
		}
	}
	realPWD, _ := filepath.EvalSymlinks(pwdLine)

	if realPWD != realProjectRoot {
		t.Errorf("shell job cwd: got %q want %q", realPWD, realProjectRoot)
	}
}

// TestSchedulerAgentRoleValidation verifies that attempting to create a job with
// an agent target that is not configured in the project is rejected with 400 at
// creation time (not deferred to execution).
func TestSchedulerAgentRoleValidation(t *testing.T) {
	env := newTestEnv(t, nil)
	env.login("admin@test.local", "admin-pass-123")

	body := map[string]any{
		"name":        "bad-role-job",
		"target_type": "agent",
		"target":      "imaginary-agent",
		"schedule":    map[string]any{"kind": "cron", "cron": "0 2 * * *"},
		"enabled":     true, "priority": 5, "timeout_sec": 30,
	}
	resp := env.doRequest("POST", schedulerPath("jobs"), body)
	requireStatus(t, resp, http.StatusBadRequest)
	data := readJSON(t, resp)

	// Confirm the job was NOT created.
	resp2 := env.doRequest("GET", schedulerPath("jobs", "bad-role-job"), nil)
	defer func() {
		b, _ := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		_ = b
	}()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("job should not exist after rejected creation, got status %d (err: %v)", resp2.StatusCode, data)
	}
}
