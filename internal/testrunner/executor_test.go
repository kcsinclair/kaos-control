// SPDX-License-Identifier: AGPL-3.0-or-later

package testrunner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

// TestHelperProcess is not a real test. It is invoked as a subprocess by
// executor tests to produce predictable stdout without running actual tools.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		os.Exit(1)
	}
	switch args[0] {
	case "go-pass":
		fmt.Print(`{"Action":"run","Package":"example","Test":"TestFoo"}` + "\n")
		fmt.Print(`{"Action":"pass","Package":"example","Test":"TestFoo","Elapsed":0.001}` + "\n")
		os.Exit(0)
	case "go-fail":
		fmt.Print(`{"Action":"run","Package":"example","Test":"TestBar"}` + "\n")
		fmt.Print(`{"Action":"output","Package":"example","Test":"TestBar","Output":"    bar_test.go:10: broken\n"}` + "\n")
		fmt.Print(`{"Action":"fail","Package":"example","Test":"TestBar","Elapsed":0.002}` + "\n")
		os.Exit(1)
	case "vitest-pass":
		fmt.Print(`{"numTotalTests":1,"numPassedTests":1,"numFailedTests":0,"numPendingTests":0,"testResults":[]}` + "\n")
		os.Exit(0)
	case "playwright-fail":
		fmt.Print(`{"suites":[],"stats":{"total":1,"expected":0,"unexpected":1,"skipped":0}}` + "\n")
		os.Exit(1)
	default:
		os.Exit(1)
	}
}

// helperCmd returns a *exec.Cmd that runs TestHelperProcess with the given scenario.
func helperCmd(ctx context.Context, scenario string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--", scenario)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

// TestExecutor_SkipsPlaywrightWhenDirAbsent verifies that Playwright is not run
// when tests/e2e/ does not exist.
func TestExecutor_SkipsPlaywrightWhenDirAbsent(t *testing.T) {
	dir := t.TempDir()
	// No tests/e2e/ subdirectory exists.

	callCount := 0
	e := &Executor{
		cmdFunc: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			callCount++
			return helperCmd(ctx, "go-pass")
		},
	}

	results, err := e.RunAll(context.Background(), dir)
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	// Only the Go suite should have run (Vitest and Playwright dirs absent).
	if len(results) != 1 {
		t.Errorf("len(results) = %d, want 1 (only go)", len(results))
	}
	if len(results) > 0 && results[0].Suite != "go" {
		t.Errorf("results[0].Suite = %q, want go", results[0].Suite)
	}
}

// TestExecutor_ContinuesAfterSuiteFailure verifies all suites run even when
// an earlier suite exits non-zero.
func TestExecutor_ContinuesAfterSuiteFailure(t *testing.T) {
	dir := t.TempDir()

	// Create tests/web/ so Vitest runs.
	if err := os.MkdirAll(dir+"/tests/web", 0o755); err != nil {
		t.Fatal(err)
	}

	calls := 0
	e := &Executor{
		cmdFunc: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			calls++
			switch calls {
			case 1:
				return helperCmd(ctx, "go-fail") // go suite fails
			default:
				return helperCmd(ctx, "vitest-pass")
			}
		},
	}

	results, err := e.RunAll(context.Background(), dir)
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	// Both go and vitest should have been attempted.
	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2 (go + vitest)", len(results))
	}
	// Go result should have a failure.
	if len(results) > 0 && results[0].Failed == 0 && results[0].RawError == "" {
		t.Error("expected go suite to report a failure")
	}
}

// TestExecutor_ElapsedRecorded verifies wall-clock elapsed time is set.
func TestExecutor_ElapsedRecorded(t *testing.T) {
	dir := t.TempDir()

	e := &Executor{
		cmdFunc: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return helperCmd(ctx, "go-pass")
		},
	}

	results, err := e.RunAll(context.Background(), dir)
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("no results returned")
	}
	if results[0].Elapsed <= 0 {
		t.Errorf("Elapsed = %f, want > 0", results[0].Elapsed)
	}
}

// TestExecutor_NonJSONOutputBecomesRawError verifies that invalid JSON output
// is captured in RawError rather than causing a crash.
func TestExecutor_NonJSONOutputBecomesRawError(t *testing.T) {
	dir := t.TempDir()

	e := &Executor{
		cmdFunc: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			// Script that outputs non-JSON to stdout.
			cmd := exec.CommandContext(ctx, "echo", "not valid json output")
			return cmd
		},
	}

	results, err := e.RunAll(context.Background(), dir)
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("no results")
	}
	goResult := results[0]
	if goResult.RawError == "" {
		t.Error("expected RawError for non-JSON output")
	}
}
