// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// Milestone 5 — Clean Baseline, Performance Benchmarking & Determinism
// (lifecycle/test-plans/frontend-lint-gap-5-test.md)
//
// Runs the real `make lint` / `make lint-frontend` targets against the repo
// itself (not a fixture project) to verify: a clean, zero-finding baseline;
// the < 5.0s NFR-1 performance budget for lint-frontend with a warm ESLint
// cache; and NFR-2 offline operation (no network dependency).
package cli_test

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestMakeLint_CleanBaseline verifies that `make lint` passes cleanly (exit
// code 0) across the whole repository, per Milestone 5's acceptance
// criteria ("make lint passes cleanly ... with --max-warnings 0").
func TestMakeLint_CleanBaseline(t *testing.T) {
	root := repoRoot(t)
	cmd := exec.Command("make", "lint")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("`make lint` did not exit cleanly: %v\noutput:\n%s", err, out)
	}
}

// TestMakeLintFrontend_PerformanceBudget verifies that `make lint-frontend`
// completes in under 5.0 seconds (NFR-1) once ESLint's cache is warm — the
// steady-state case for a developer running lint repeatedly during local
// iteration. `pnpm run lint` in web/package.json passes `--cache
// --cache-location .eslintcache`, so a cold first run is not representative.
func TestMakeLintFrontend_PerformanceBudget(t *testing.T) {
	root := repoRoot(t)

	warm := exec.Command("make", "lint-frontend")
	warm.Dir = root
	if out, err := warm.CombinedOutput(); err != nil {
		t.Fatalf("warm-up `make lint-frontend` run failed: %v\noutput:\n%s", err, out)
	}

	timed := exec.Command("make", "lint-frontend")
	timed.Dir = root
	start := time.Now()
	out, err := timed.CombinedOutput()
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("timed `make lint-frontend` run failed: %v\noutput:\n%s", err, out)
	}

	if elapsed >= 5*time.Second {
		t.Errorf("`make lint-frontend` took %v with a warm cache, want < 5.0s (NFR-1)", elapsed)
	}
}

// TestMakeLintFrontend_OfflineExecution verifies NFR-2 (deterministic
// offline operation): `make lint-frontend` must succeed even when the only
// route out is an unreachable proxy, proving it makes no network calls.
func TestMakeLintFrontend_OfflineExecution(t *testing.T) {
	root := repoRoot(t)

	// Port 1 on loopback refuses connections immediately (nothing listens
	// there), so any attempted network call fails fast instead of hanging.
	const unreachableProxy = "http://127.0.0.1:1"
	cmd := exec.Command("make", "lint-frontend")
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"http_proxy="+unreachableProxy, "https_proxy="+unreachableProxy,
		"HTTP_PROXY="+unreachableProxy, "HTTPS_PROXY="+unreachableProxy,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("`make lint-frontend` failed with only an unreachable proxy available "+
			"(suggests a network dependency, violating NFR-2): %v\noutput:\n%s", err, out)
	}
}
