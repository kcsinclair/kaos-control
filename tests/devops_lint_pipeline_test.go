// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration

// Milestone 4 — DevOps Pipeline Execution & Timeout Verification
// (lifecycle/test-plans/frontend-lint-gap-5-test.md)
//
// Verifies that lifecycle/devops/test-lint.yaml and lifecycle/devops/all-tests.yaml
// load correctly and that the shared "Lint" step (`make lint`) actually runs to
// completion through the real internal/devops execution engine, against the
// real repo (not a synthetic fixture project — `make lint` only works from the
// real repo root). This deliberately runs the single-step test-lint.yaml
// pipeline rather than the full 5-step all-tests.yaml: the other four steps
// (unit/frontend/integration/e2e suites) would themselves spawn long-running,
// resource-heavy nested test runs, which is out of scope for verifying the
// Lint step's wiring. all-tests.yaml is instead checked at the metadata level
// to confirm its step 1 is the same Lint/make-lint step.
package cli_test

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kaos-control/kaos-control/internal/devops"
)

// repoRoot resolves the repository root. The working directory for
// `go test ./tests/` is tests/, so "../" is the repo root.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// TestDevopsLintPipeline_LoadsRealDefinitions verifies that both
// lifecycle/devops/test-lint.yaml and lifecycle/devops/all-tests.yaml parse
// successfully from disk and that each declares "Lint" / `make lint` as step 1.
func TestDevopsLintPipeline_LoadsRealDefinitions(t *testing.T) {
	root := repoRoot(t)
	pipelines, warnings := devops.Discover(filepath.Join(root, "lifecycle", "devops"))
	if len(warnings) != 0 {
		t.Fatalf("unexpected parse warnings loading lifecycle/devops/*.yaml: %v", warnings)
	}

	byLine := make(map[string]devops.Pipeline, len(pipelines))
	for _, p := range pipelines {
		byLine[p.Slug] = p
	}

	testLint, ok := byLine["test-lint"]
	if !ok {
		t.Fatal("lifecycle/devops/test-lint.yaml did not load as pipeline slug \"test-lint\"")
	}
	if len(testLint.Steps) != 1 {
		t.Fatalf("test-lint: expected 1 step, got %d", len(testLint.Steps))
	}
	if testLint.Steps[0].Name != "Lint" {
		t.Errorf("test-lint step[0].Name = %q, want %q", testLint.Steps[0].Name, "Lint")
	}
	if testLint.Steps[0].Command != "make lint" {
		t.Errorf("test-lint step[0].Command = %q, want %q", testLint.Steps[0].Command, "make lint")
	}

	allTests, ok := byLine["all-tests"]
	if !ok {
		t.Fatal("lifecycle/devops/all-tests.yaml did not load as pipeline slug \"all-tests\"")
	}
	if len(allTests.Steps) == 0 {
		t.Fatal("all-tests: expected at least 1 step, got 0")
	}
	if allTests.Steps[0].Name != "Lint" {
		t.Errorf("all-tests step[0].Name = %q, want %q", allTests.Steps[0].Name, "Lint")
	}
	if allTests.Steps[0].Command != "make lint" {
		t.Errorf("all-tests step[0].Command = %q, want %q", allTests.Steps[0].Command, "make lint")
	}
}

// TestDevopsLintPipeline_ExecutesAgainstRealRepo drives the real test-lint.yaml
// pipeline through internal/devops.Runner with the actual repo root as the
// working directory, so `make lint` runs for real (go vet/staticcheck/
// govulncheck/gosec/gitleaks, then ESLint/vue-tsc). It asserts the runner
// executes the Lint step to completion, well within the 2-minute budget
// (NFR-1/Milestone 4 acceptance criteria), and that step output carries
// diagnostics from the Go toolchain. Frontend diagnostics are asserted only
// when the Go phase passes, since `make lint`'s targets are fail-fast
// (lint: lint-go lint-frontend halts on the first failure — see Milestone 1);
// a failing lint-go run never reaches lint-frontend, and that is accurately
// reported as a test failure below rather than papered over.
func TestDevopsLintPipeline_ExecutesAgainstRealRepo(t *testing.T) {
	root := repoRoot(t)
	pipelines, warnings := devops.Discover(filepath.Join(root, "lifecycle", "devops"))
	if len(warnings) != 0 {
		t.Fatalf("unexpected parse warnings: %v", warnings)
	}
	var pipeline devops.Pipeline
	found := false
	for _, p := range pipelines {
		if p.Slug == "test-lint" {
			pipeline = p
			found = true
			break
		}
	}
	if !found {
		t.Fatal("lifecycle/devops/test-lint.yaml not found by Discover")
	}

	var mu sync.Mutex
	var started, completed bool
	var stepCompletedPayload devops.StepCompletedPayload
	var stdout string

	runner := devops.NewRunner()
	runner.SetEventHook(func(_ string, eventType string, payload any) {
		mu.Lock()
		defer mu.Unlock()
		switch eventType {
		case devops.EventStepStarted:
			if p, ok := payload.(devops.StepStartedPayload); ok && p.Step == "Lint" {
				started = true
			}
		case devops.EventStepOutput:
			if p, ok := payload.(devops.StepOutputPayload); ok && p.Stream == "stdout" {
				stdout += p.Text + "\n"
			}
		case devops.EventStepCompleted:
			if p, ok := payload.(devops.StepCompletedPayload); ok && p.Step == "Lint" {
				completed = true
				stepCompletedPayload = p
			}
		}
	})

	runID, err := runner.Start(pipeline, root, nil, "kaos-control-self-test", nil)
	if err != nil {
		t.Fatalf("starting test-lint pipeline: %v", err)
	}

	deadline := time.Now().Add(pipeline.Steps[0].Timeout + 30*time.Second)
	for runner.IsRunningID(runID) {
		if time.Now().After(deadline) {
			t.Fatalf("test-lint pipeline did not complete within %v", pipeline.Steps[0].Timeout+30*time.Second)
		}
		time.Sleep(100 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()

	// Pipeline runner successfully executes the Lint step (runs it to
	// completion, regardless of pass/fail verdict).
	if !started {
		t.Error("runner never emitted pipeline.step.started for the Lint step")
	}
	if !completed {
		t.Fatal("runner never emitted pipeline.step.completed for the Lint step")
	}

	// Execution completes well under the 2-minute timeout budget.
	if d := time.Duration(stepCompletedPayload.DurationSeconds * float64(time.Second)); d >= 2*time.Minute {
		t.Errorf("Lint step took %v, want well under 2m", d)
	}

	// Step output includes Go lint diagnostics (go vet is always the first
	// tool `make lint-go` runs, so its echoed command line is always present).
	if !strings.Contains(stdout, "go vet") {
		t.Error("Lint step stdout does not contain Go lint diagnostics (\"go vet\")")
	}

	if stepCompletedPayload.Status != "passed" {
		t.Errorf("Lint step status = %q (exit code %d); make lint currently fails before reaching "+
			"lint-frontend, so frontend diagnostics cannot appear in step output — this reflects a real "+
			"gap in the repo's lint baseline (see Milestone 5's clean-baseline test), not a bug in this test.\nstdout:\n%s",
			stepCompletedPayload.Status, stepCompletedPayload.ExitCode, stdout)
		return
	}

	// Only reachable once make lint-go passes: frontend diagnostics must
	// also appear in the same step's output (make lint's frontend half).
	if !strings.Contains(stdout, "pnpm run lint") {
		t.Error("Lint step stdout does not contain frontend lint diagnostics (\"pnpm run lint\")")
	}
}
