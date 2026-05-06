package scheduler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/kaos-control/kaos-control/internal/sandbox"
)

// EvaluatePreconditions returns true if all preconditions for job are met.
// projectRoot is used to resolve file_exists and shell paths safely.
func EvaluatePreconditions(ctx context.Context, preconditions []Precondition, store *Store, projectRoot string) (bool, error) {
	for _, p := range preconditions {
		ok, err := evaluateOne(ctx, p, store, projectRoot)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func evaluateOne(ctx context.Context, p Precondition, store *Store, projectRoot string) (bool, error) {
	switch p.Kind {
	case PreconditionAfterJob:
		return evalAfterJob(store, p.JobName)
	case PreconditionFileExists:
		return evalFileExists(p.Path, projectRoot)
	case PreconditionHTTPOk:
		return evalHTTPOk(ctx, p.URL)
	case PreconditionShell:
		return evalShell(ctx, p.Command, projectRoot)
	default:
		return false, fmt.Errorf("unknown precondition kind %q", p.Kind)
	}
}

// evalAfterJob returns true when the named job's most recent run has status success.
func evalAfterJob(store *Store, jobName string) (bool, error) {
	if jobName == "" {
		return false, fmt.Errorf("after_job precondition: job_name must not be empty")
	}
	run, err := store.LastRunForJob(jobName)
	if err != nil {
		return false, err
	}
	if run == nil {
		return false, nil // job has never run
	}
	return run.Status == RunStatusSuccess, nil
}

// evalFileExists returns true if the path exists within the project sandbox.
func evalFileExists(path, projectRoot string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("file_exists precondition: path must not be empty")
	}
	resolved, err := sandbox.Resolve(projectRoot, path)
	if err != nil {
		return false, fmt.Errorf("file_exists precondition: sandbox resolve: %w", err)
	}
	_, err = os.Stat(resolved)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// evalHTTPOk performs a GET with a 10-second timeout; returns true on 2xx.
func evalHTTPOk(ctx context.Context, rawURL string) (bool, error) {
	if rawURL == "" {
		return false, fmt.Errorf("http_ok precondition: url must not be empty")
	}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return false, fmt.Errorf("http_ok precondition: building request: %w", err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, nil // network error → not ok
	}
	_ = resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300, nil
}

// evalShell runs cmd in the project root with a 30-second timeout; returns true on exit 0.
func evalShell(ctx context.Context, cmd, projectRoot string) (bool, error) {
	if cmd == "" {
		return false, fmt.Errorf("shell precondition: command must not be empty")
	}
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	c := exec.CommandContext(runCtx, "sh", "-c", cmd)
	c.Dir = projectRoot
	c.Env = minimalEnv(projectRoot)
	err := c.Run()
	if err == nil {
		return true, nil
	}
	if runCtx.Err() != nil {
		return false, nil // timeout — treat as unmet, not an error
	}
	// Non-zero exit → precondition unmet.
	return false, nil
}

// minimalEnv returns a minimal environment for shell targets.
func minimalEnv(projectRoot string) []string {
	return []string{
		"PATH=/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
		"HOME=" + os.Getenv("HOME"),
		"PROJECT_ROOT=" + projectRoot,
	}
}
